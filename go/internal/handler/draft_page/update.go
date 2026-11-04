package draft_page

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Updateは下書きページを自動保存します (PATCH /s/{space_identifier}/pages/{page_number}/draft_page)
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
	title := r.FormValue("pages_edit_form[title]")
	body := r.FormValue("pages_edit_form[body]")

	// 編集画面が保存の前提にした下書きの版。前提の分からない保存は、別の場所の変更を上書きしうるため受け付けない。
	// 版を送らない編集画面でも保存の失敗として文面を表示できるよう、422と再読み込みを促す文面で断る
	draftPageDigests, ok := r.Form["pages_edit_form[draft_page_digest]"]
	if !ok || len(draftPageDigests) == 0 {
		http.Error(w, i18n.T(ctx, "validation_draft_page_digest_missing"), http.StatusUnprocessableEntity)
		return
	}

	// タイトルのポインタ変換 (空文字列の場合もポインタとして渡す)
	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	// UseCaseを実行
	output, err := h.autoSaveDraftPageUC.Execute(ctx, usecase.AutoSaveDraftPageInput{
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

		slog.ErrorContext(ctx, "下書きの自動保存に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("ETag", draftPageETag(output.DraftPage.ContentDigest()))
	w.WriteHeader(http.StatusNoContent)
}

// draftPageETagは下書きの版を、引用符で囲んだ強いエンティティタグにする
func draftPageETag(digest string) string {
	return `"` + digest + `"`
}
