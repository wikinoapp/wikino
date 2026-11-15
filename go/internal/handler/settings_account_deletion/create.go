package settings_account_deletion

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Createはアカウントを削除し、セッションのCookieを消してトップへリダイレクトします
// (POST /settings/account/deletion)。
//
// アットネームが一致しないときや、参加しているスペースがあるときは、確認画面を描き直します。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	atname := r.FormValue("atname")

	if err := h.discardAccountUC.Execute(ctx, usecase.DiscardAccountInput{
		User:   user,
		Atname: atname,
	}); err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.renderForm(w, r, user, http.StatusUnprocessableEntity, atname, ve)
			return
		}
		if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeResourceNotFound {
			httperror.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "アカウントの削除に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(ctx, "アカウントを削除しました", "user_id", user.ID)
	h.sessionMgr.DeleteSessionCookie(w)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_account_deleted"))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
