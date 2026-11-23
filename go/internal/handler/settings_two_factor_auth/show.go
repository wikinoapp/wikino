package settings_two_factor_auth

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Showは二要素認証の設定の画面を表示します (GET /settings/two_factor_auth)。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
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

	h.renderShow(w, r, user, http.StatusOK, output, nil)
}

// renderShowは二要素認証の設定の画面を指定したステータスで描画します。
// 表示 (200) と、無効にするフォームのパスワードを受け付けなかったときの描き直し (422) で共有します。
func (h *Handler) renderShow(w http.ResponseWriter, r *http.Request, user *model.User, status int, settings *usecase.GetTwoFactorAuthSettingsOutput, formErrors *model.ValidationError) {
	h.render(w, r, user, status, showPage, settingspages.TwoFactorAuth(settingspages.TwoFactorAuthData{
		CSRFToken:  middleware.GetCSRFTokenFromContext(r.Context()),
		FormErrors: formErrors,
		Enabled:    settings.Enabled,
		EnabledAt:  settings.EnabledAt,
	}))
}
