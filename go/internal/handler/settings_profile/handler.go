// Package settings_profileはプロフィールの編集のHTTPハンドラーを提供します
package settings_profile

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Handlerはプロフィールの編集画面と、その送信先の保存処理を提供します
type Handler struct {
	cfg             *config.Config
	flashMgr        *session.FlashManager
	updateProfileUC *usecase.UpdateProfileUsecase
}

// NewHandlerはプロフィールの編集のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	flashMgr *session.FlashManager,
	updateProfileUC *usecase.UpdateProfileUsecase,
) *Handler {
	return &Handler{
		cfg:             cfg,
		flashMgr:        flashMgr,
		updateProfileUC: updateProfileUC,
	}
}

// renderはプロフィールの編集画面を描画します。パンくずは設定のトップを通り、この画面自身の
// 非リンクな現在項目で締めます。
func (h *Handler) render(w http.ResponseWriter, r *http.Request, user *model.User, status int, data settingspages.ProfileData) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "settings_profile_show_title")

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSettingsProfile,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "settings_profile_show_breadcrumb_settings"),
					Path:  templates.SettingsPath(),
				},
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "settings_profile_show_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	// 設定の画面はログインした本人のための画面のため、検索エンジンにインデックスさせない
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := layouts.Default(layoutData, settingspages.Profile(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
