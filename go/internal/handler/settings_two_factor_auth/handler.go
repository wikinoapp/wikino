// Package settings_two_factor_authは二要素認証の設定の画面と、認証アプリへの登録・有効化のHTTPハンドラーを提供します
package settings_two_factor_auth

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/qrcode"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Handlerは二要素認証の設定の画面と、認証アプリへの登録・有効化の処理を提供します
type Handler struct {
	cfg                        *config.Config
	flashMgr                   *session.FlashManager
	limiter                    *ratelimit.Limiter
	getTwoFactorAuthSettingsUC *usecase.GetTwoFactorAuthSettingsUsecase
	prepareTwoFactorAuthUC     *usecase.PrepareTwoFactorAuthUsecase
	getPendingTwoFactorAuthUC  *usecase.GetPendingTwoFactorAuthUsecase
	enableTwoFactorAuthUC      *usecase.EnableTwoFactorAuthUsecase
}

// NewHandlerは二要素認証の設定のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	limiter *ratelimit.Limiter,
	getTwoFactorAuthSettingsUC *usecase.GetTwoFactorAuthSettingsUsecase,
	prepareTwoFactorAuthUC *usecase.PrepareTwoFactorAuthUsecase,
	getPendingTwoFactorAuthUC *usecase.GetPendingTwoFactorAuthUsecase,
	enableTwoFactorAuthUC *usecase.EnableTwoFactorAuthUsecase,
) *Handler {
	return &Handler{
		cfg:                        cfg,
		flashMgr:                   flashMgr,
		limiter:                    limiter,
		getTwoFactorAuthSettingsUC: getTwoFactorAuthSettingsUC,
		prepareTwoFactorAuthUC:     prepareTwoFactorAuthUC,
		getPendingTwoFactorAuthUC:  getPendingTwoFactorAuthUC,
		enableTwoFactorAuthUC:      enableTwoFactorAuthUC,
	}
}

// pageはレイアウトに入れて描く二要素認証の画面の種類
type page struct {
	titleKey string
	// breadcrumbSettingsKeyはパンくずの設定のトップの文言のキー
	breadcrumbSettingsKey string
	// breadcrumbParentKeyは、パンくずで二要素認証の設定の画面を親として挟むときの文言のキー。挟まないときは空
	breadcrumbParentKey string
	breadcrumbKey       string
	currentPageName     templates.PageName
}

var (
	showPage = page{
		titleKey:              "settings_two_factor_auth_show_title",
		breadcrumbSettingsKey: "settings_two_factor_auth_show_breadcrumb_settings",
		breadcrumbKey:         "settings_two_factor_auth_show_breadcrumb",
		currentPageName:       templates.PageNameSettingsTwoFactorAuth,
	}
	newPage = page{
		titleKey:              "settings_two_factor_auth_new_title",
		breadcrumbSettingsKey: "settings_two_factor_auth_new_breadcrumb_settings",
		breadcrumbParentKey:   "settings_two_factor_auth_new_breadcrumb_two_factor_auth",
		breadcrumbKey:         "settings_two_factor_auth_new_breadcrumb",
		currentPageName:       templates.PageNameSettingsTwoFactorAuthNew,
	}
)

// renderは二要素認証の画面をレイアウトに入れて、指定したステータスで描画します
func (h *Handler) render(w http.ResponseWriter, r *http.Request, user *model.User, status int, p page, content templ.Component) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, p.titleKey)

	items := append(components.HomeBreadcrumbItems(ctx, true), components.BreadcrumbItem{
		Label: i18n.T(ctx, p.breadcrumbSettingsKey),
		Path:  templates.SettingsPath(),
	})
	if p.breadcrumbParentKey != "" {
		items = append(items, components.BreadcrumbItem{
			Label: i18n.T(ctx, p.breadcrumbParentKey),
			Path:  templates.SettingsTwoFactorAuthPath(),
		})
	}
	items = append(items, components.BreadcrumbItem{
		Label:     i18n.T(ctx, p.breadcrumbKey),
		IsCurrent: true,
	})

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: p.currentPageName,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items:         items,
		},
	}

	setResponseHeaders(w)
	w.WriteHeader(status)

	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// setResponseHeadersは、設定の画面を検索やキャッシュに残さないヘッダーを設定する。
// 本文を返す応答とHEADで、秘密鍵を扱うページのキャッシュ方針を揃える。
func setResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

// renderNewは、秘密鍵のQRコードを作り、認証アプリへ登録する画面を指定したステータスで描画します。
// 表示 (200) と、パスワードかコードを受け付けなかったときの描き直し (422) で共有します。
func (h *Handler) renderNew(w http.ResponseWriter, r *http.Request, user *model.User, status int, setup *usecase.TwoFactorAuthSetup, csrfToken string, formErrors *model.ValidationError) {
	ctx := r.Context()

	qrCode, err := qrcode.Encode(setup.OTPAuthURL)
	if err != nil {
		slog.ErrorContext(ctx, "二要素認証の登録のQRコードの作成に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.render(w, r, user, status, newPage, settingspages.TwoFactorAuthNew(settingspages.TwoFactorAuthNewData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		Secret:     setup.Secret,
		QRCode:     qrCode,
	}))
}
