// Package settings_two_factor_auth_recovery_codeは二要素認証のリカバリーコードの画面と、作り直す処理のHTTPハンドラーを提供します
package settings_two_factor_auth_recovery_code

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Handlerは二要素認証のリカバリーコードの画面と、作り直す処理を提供します
type Handler struct {
	cfg                               *config.Config
	flashMgr                          *session.FlashManager
	limiter                           *ratelimit.Limiter
	getTwoFactorAuthRecoveryCodesUC   *usecase.GetTwoFactorAuthRecoveryCodesUsecase
	regenerateTwoFactorAuthRecoveryUC *usecase.RegenerateTwoFactorAuthRecoveryCodesUsecase
}

// NewHandlerは二要素認証のリカバリーコードのハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	limiter *ratelimit.Limiter,
	getTwoFactorAuthRecoveryCodesUC *usecase.GetTwoFactorAuthRecoveryCodesUsecase,
	regenerateTwoFactorAuthRecoveryUC *usecase.RegenerateTwoFactorAuthRecoveryCodesUsecase,
) *Handler {
	return &Handler{
		cfg:                               cfg,
		flashMgr:                          flashMgr,
		limiter:                           limiter,
		getTwoFactorAuthRecoveryCodesUC:   getTwoFactorAuthRecoveryCodesUC,
		regenerateTwoFactorAuthRecoveryUC: regenerateTwoFactorAuthRecoveryUC,
	}
}

// renderはリカバリーコードの画面を指定したステータスで描画します
func (h *Handler) render(w http.ResponseWriter, r *http.Request, user *model.User, status int, data settingspages.TwoFactorAuthRecoveryCodesData) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "settings_two_factor_auth_recovery_code_index_title")

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSettingsTwoFactorAuthRecoveryCodes,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "settings_two_factor_auth_recovery_code_index_breadcrumb_settings"),
					Path:  templates.SettingsPath(),
				},
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "settings_two_factor_auth_recovery_code_index_breadcrumb_two_factor_auth"),
					Path:  templates.SettingsTwoFactorAuthPath(),
				},
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "settings_two_factor_auth_recovery_code_index_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	// 設定の画面はログインした本人のための画面のため、検索エンジンにインデックスさせない
	w.Header().Set("X-Robots-Tag", "noindex")
	// リカバリーコードを含むため、戻る操作や共用の端末で読み返されないようキャッシュに残させない
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := layouts.Default(layoutData, settingspages.TwoFactorAuthRecoveryCodes(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderCodesは、今のリカバリーコードを読み直して画面を描画します。
// 二要素認証が有効でなくなっていたときは、設定の画面へ送ります。
func (h *Handler) renderCodes(w http.ResponseWriter, r *http.Request, user *model.User, status int, formErrors *model.ValidationError) {
	ctx := r.Context()

	output, err := h.getTwoFactorAuthRecoveryCodesUC.Execute(ctx, usecase.GetTwoFactorAuthRecoveryCodesInput{UserID: user.ID})
	if err != nil {
		slog.ErrorContext(ctx, "リカバリーコードの取得に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !output.Enabled {
		h.flashMgr.SetWarning(w, i18n.T(ctx, "flash_two_factor_auth_not_enabled"))
		http.Redirect(w, r, string(templates.SettingsTwoFactorAuthPath()), http.StatusSeeOther)
		return
	}

	h.render(w, r, user, status, settingspages.TwoFactorAuthRecoveryCodesData{
		CSRFToken:     middleware.GetCSRFTokenFromContext(ctx),
		FormErrors:    formErrors,
		RecoveryCodes: output.RecoveryCodes,
	})
}
