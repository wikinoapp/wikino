package draft_page_revision

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Updateは下書きページを手動保存します (PATCH /s/{space_identifier}/pages/{page_number}/draft_page_revision)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 保存の応答のETagは編集画面が次の保存と公開の前提にする版のため、前段の中継が応答を加工してETagを消さないよう、
	// 加工を禁じる。保存の失敗を含むすべての応答に付ける
	w.Header().Set("Cache-Control", "no-transform")

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	pageNumberStr := chi.URLParam(r, "page_number")

	pageNumber, err := strconv.ParseInt(pageNumberStr, 10, 32)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// フォームパラメータを取得
	title := r.FormValue("title")
	body := r.FormValue("body")

	// 編集画面が保存の前提にした下書きの版。前提の分からない保存は、別の場所の変更を上書きしうるため受け付けない。
	// 版を送らない編集画面でも保存の失敗として文面を表示できるよう、422と再読み込みを促す文面で断る
	draftPageDigests, ok := r.Form["draft_page_digest"]
	if !ok || len(draftPageDigests) == 0 {
		http.Error(w, i18n.T(ctx, "validation_draft_page_digest_missing"), http.StatusUnprocessableEntity)
		return
	}

	// タイトルのポインタ変換
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	// UseCaseを実行
	saveOutput, err := h.manualSaveDraftPageUC.Execute(ctx, usecase.ManualSaveDraftPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		Title:           titlePtr,
		Body:            body,
		DraftPageDigest: draftPageDigests[0],
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			// 保存できなかった理由を、利用者に示せる文面で応答の本文に入れる
			http.Error(w, strings.Join(ve.GetFieldErrors("body"), "\n"), http.StatusUnprocessableEntity)
			return
		}

		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				httperror.NotFound(w, r)
			case model.AppErrCodeConflict:
				// 下書きが別の場所で更新されていた。編集画面が今の版を前提に上書きし直せるよう、今の版を添える
				var conflict *usecase.DraftPageConflict
				if errors.As(err, &conflict) {
					w.Header().Set("ETag", draftPageETag(conflict.CurrentDigest))
				}
				http.Error(w, ae.UserMsg, http.StatusConflict)
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		slog.ErrorContext(ctx, "下書きの手動保存に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 編集画面が次の保存の前提にする、保存後の下書きの版
	w.Header().Set("ETag", draftPageETag(saveOutput.DraftPage.ContentDigest()))

	// リダイレクト先を決定
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "suggestion_new" && saveOutput.DraftPage != nil {
		suggestionNewPath := fmt.Sprintf("%s?draft_page_ids=%s",
			string(templates.SuggestionNewPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), saveOutput.TopicNumber)),
			string(saveOutput.DraftPage.ID),
		)
		// 編集画面の「下書き保存して編集提案を作成」はhtmxで送り、保存に失敗したときは編集画面に留まる。
		// fetchはリダイレクトを追ってしまうため、htmxには移動先をHX-Redirectで伝える
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", suggestionNewPath)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, suggestionNewPath, http.StatusSeeOther)
		return
	}

	// htmxリクエスト (編集画面の「下書き保存」ボタン) には画面遷移なしで応答する。保存時刻
	// 表示と編集履歴カラム2箇所を更新するOOBスワップフラグメントを返す。
	if r.Header.Get("HX-Request") == "true" {
		h.renderUpdateFragment(w, r, spaceIdentifier, int32(pageNumber), user.ID, saveOutput)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_draft_page_saved"))
	http.Redirect(w, r, "/drafts", http.StatusSeeOther)
}

// draftPageETagは下書きの版を、引用符で囲んだ強いエンティティタグにする
func draftPageETag(digest string) string {
	return `"` + digest + `"`
}

// renderUpdateFragmentは手動保存後のOOBスワップフラグメントをレンダリングします。
func (h *Handler) renderUpdateFragment(
	w http.ResponseWriter,
	r *http.Request,
	spaceIdentifier model.SpaceIdentifier,
	pageNumber int32,
	userID model.UserID,
	saveOutput *usecase.ManualSaveDraftPageOutput,
) {
	ctx := r.Context()

	// 保存直後の状態 (新規リビジョンが作成されないスキップ時を含む) を反映するため、
	// リビジョン一覧と総件数を取得し直す。
	detail, err := h.getPageDetailUC.Execute(ctx, usecase.GetPageDetailInput{
		SpaceIdentifier:       spaceIdentifier,
		PageNumber:            pageNumber,
		UserID:                userID,
		IncludeDraftRevisions: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "ページ詳細の取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if detail == nil {
		// 直前の保存が成功しているため、競合以外でページが消えていることはない。
		httperror.NotFound(w, r)
		return
	}

	responseData := pagepages.DraftPageRevisionUpdateResponseData{
		SavedAt:         saveOutput.DraftPage.ModifiedAt,
		SpaceIdentifier: viewmodel.NewSpaceIdentifier(spaceIdentifier),
		PageNumber:      pageNumber,
		DraftRevisions:  viewmodel.NewDraftPageRevisions(detail.DraftPageRevisions, detail.DraftPageRevisionTotalCount),
	}
	if err := pagepages.DraftPageRevisionUpdateResponse(responseData).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "手動保存レスポンスのレンダリングに失敗", "error", err)
	}
}
