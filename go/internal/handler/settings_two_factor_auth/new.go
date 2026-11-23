package settings_two_factor_auth

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Newは認証アプリへ登録する新しい秘密鍵を作り、登録の画面を表示します (GET /settings/two_factor_auth/new)。
//
// 既に有効にしているときは、秘密鍵を差し替えずに設定の画面へ送ります。
// GETで行を書き換えることになりますが、書き換えるのは有効にする前の秘密鍵だけで、
// ログインした本人のno-storeの画面のため許容します。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	output, err := h.prepareTwoFactorAuthUC.Execute(ctx, usecase.PrepareTwoFactorAuthInput{UserID: user.ID, Email: user.Email})
	if err != nil {
		slog.ErrorContext(ctx, "二要素認証の登録の準備に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if output.Setup == nil {
		h.flashMgr.SetWarning(w, i18n.T(ctx, "flash_two_factor_auth_already_enabled"))
		http.Redirect(w, r, string(templates.SettingsTwoFactorAuthPath()), http.StatusFound)
		return
	}

	h.renderNew(w, r, user, http.StatusOK, output.Setup, middleware.GetCSRFTokenFromContext(ctx), nil)
}

// HeadNewは登録の画面のヘッダーだけを返します (HEAD /settings/two_factor_auth/new)。
// GETと同じ準備の処理を呼ぶと、表示中の秘密鍵が使えなくなるため、有効かどうかだけを読み取ります。
func (h *Handler) HeadNew(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	output, err := h.getTwoFactorAuthSettingsUC.Execute(ctx, usecase.GetTwoFactorAuthSettingsInput{UserID: user.ID})
	if err != nil {
		slog.ErrorContext(ctx, "二要素認証の設定の取得に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	setResponseHeaders(w)
	if output.Enabled {
		http.Redirect(w, r, string(templates.SettingsTwoFactorAuthPath()), http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}
