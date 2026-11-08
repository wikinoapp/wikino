package page_pin

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Createはページにピンを打ちます (POST /s/{space_identifier}/pages/{page_number}/pin)。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	pageNumber, err := strconv.ParseInt(chi.URLParam(r, "page_number"), 10, 32)
	if err != nil {
		httperror.NotFound(w, r)
		return
	}

	output, err := h.pinPageUC.Execute(ctx, usecase.PinPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
		writeError(w, r, err, "ページのピン留めに失敗")
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_page_pinned"))
	redirectToPage(w, r, output.Space, int32(pageNumber))
}

// writeErrorはUseCaseのエラーを応答にする。「権限が無い」と「存在しない」はどちらも404にし、
// 開けないページの存在をレスポンスから読み取れないようにする。
func writeError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	ctx := r.Context()

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			httperror.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, logMsg, "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// redirectToPageは操作したページへ戻す。パスはリクエストURLではなく保存済みの識別子から組み立て、
// 大文字小文字が異なるリクエストでも正規のアドレスへ着地させる。
func redirectToPage(w http.ResponseWriter, r *http.Request, space *model.Space, pageNumber int32) {
	pagePath := templates.PagePath(viewmodel.NewSpaceIdentifier(space.Identifier), viewmodel.PageNumber(pageNumber))
	http.Redirect(w, r, string(pagePath), http.StatusSeeOther)
}
