package draft_page_revision

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	pagepages "github.com/wikinoapp/wikino/go/internal/templates/pages/page"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Update は下書きページを手動保存します (PATCH /s/{space_identifier}/pages/{page_number}/draft_page_revision)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	// タイトルのポインタ変換
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	// UseCase を実行
	saveOutput, err := h.manualSaveDraftPageUC.Execute(ctx, usecase.ManualSaveDraftPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
		Title:           titlePtr,
		Body:            body,
	})
	if err != nil {
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				handler.NotFound(w, r)
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

	// リダイレクト先を決定
	redirectTo := r.URL.Query().Get("redirect_to")
	if redirectTo == "suggestion_new" && saveOutput.DraftPage != nil {
		suggestionNewPath := fmt.Sprintf("%s?draft_page_ids=%s",
			string(templates.SuggestionNewPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), saveOutput.TopicNumber)),
			string(saveOutput.DraftPage.ID),
		)
		http.Redirect(w, r, suggestionNewPath, http.StatusSeeOther)
		return
	}

	// For htmx requests (the editor's save-draft button) respond without navigation: return an
	// OOB swap fragment that refreshes the saved-at indicator and both edit history columns.
	//
	// [Ja] htmx リクエスト (編集画面の「下書き保存」ボタン) には画面遷移なしで応答する。保存時刻
	// 表示と編集履歴カラム 2 箇所を更新する OOB スワップフラグメントを返す。
	if r.Header.Get("HX-Request") == "true" {
		h.renderUpdateFragment(w, r, spaceIdentifier, int32(pageNumber), user.ID, saveOutput)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_draft_page_saved"))
	http.Redirect(w, r, "/drafts", http.StatusSeeOther)
}

// renderUpdateFragment renders the OOB swap fragment returned after a manual save.
// [Ja] renderUpdateFragment は手動保存後の OOB スワップフラグメントをレンダリングします。
func (h *Handler) renderUpdateFragment(
	w http.ResponseWriter,
	r *http.Request,
	spaceIdentifier model.SpaceIdentifier,
	pageNumber int32,
	userID model.UserID,
	saveOutput *usecase.ManualSaveDraftPageOutput,
) {
	ctx := r.Context()

	// Re-read the revision list and total count so the response reflects the just-saved state
	// (including the skip case where no new revision was created).
	//
	// [Ja] 保存直後の状態 (新規リビジョンが作成されないスキップ時を含む) を反映するため、
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
		// The save above just succeeded, so the page cannot have disappeared except in races.
		// [Ja] 直前の保存が成功しているため、競合以外でページが消えていることはない。
		handler.NotFound(w, r)
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
