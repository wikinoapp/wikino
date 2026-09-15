package page_trash

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Createはページをゴミ箱へ入れます (POST /s/{space_identifier}/pages/{page_number}/trash)。
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
		handler.NotFound(w, r)
		return
	}

	output, err := h.trashPageUC.Execute(ctx, usecase.TrashPageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
		UserID:          user.ID,
	})
	if err != nil {
		// 「ゴミ箱に入れる権限が無い」と「存在しない」はどちらも404にし、開けないページの
		// 存在をレスポンスから読み取れないようにする。
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

		slog.ErrorContext(ctx, "ページのゴミ箱への移動に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ゴミ箱に入れてもページは読めるため、操作した対象をそのまま見せる。そこに出るアラートが
	// ゴミ箱そのものへの導線を兼ねる。
	// パスはリクエストURLではなく
	// 保存済みの識別子から組み立て、大文字小文字が異なるリクエストでも正規のアドレスへ着地させる。
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_page_moved_to_trash"))
	pagePath := templates.PagePath(viewmodel.NewSpaceIdentifier(output.Space.Identifier), viewmodel.PageNumber(pageNumber))
	http.Redirect(w, r, string(pagePath), http.StatusSeeOther)
}
