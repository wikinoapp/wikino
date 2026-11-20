// Package settings_email_confirmationは、メールアドレスの変更の確認コードを入力する画面と、
// 変更を確定する処理のHTTPハンドラーを提供します
package settings_email_confirmation

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Handlerは、メールアドレスの変更の確認コードを入力する画面と、その送信先の処理を提供します
type Handler struct {
	cfg           *config.Config
	sessionMgr    *session.Manager
	flashMgr      *session.FlashManager
	updateEmailUC *usecase.UpdateEmailUsecase
}

// NewHandlerは、メールアドレスの変更の確認コードのハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	updateEmailUC *usecase.UpdateEmailUsecase,
) *Handler {
	return &Handler{
		cfg:           cfg,
		sessionMgr:    sessionMgr,
		flashMgr:      flashMgr,
		updateEmailUC: updateEmailUC,
	}
}

// renderは確認コードを入力する画面を描画します
func (h *Handler) render(w http.ResponseWriter, r *http.Request, user *model.User, status int, formErrors *model.ValidationError) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "settings_email_confirmation_edit_title")

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSettingsEmailConfirmation,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "settings_email_confirmation_edit_breadcrumb_settings"),
					Path:  templates.SettingsPath(),
				},
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "settings_email_confirmation_edit_breadcrumb_email"),
					Path:  templates.SettingsEmailPath(),
				},
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "settings_email_confirmation_edit_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	// 設定の画面はログインした本人のための画面のため、検索エンジンにインデックスさせない
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	content := settingspages.EmailConfirmation(settingspages.EmailConfirmationData{
		CSRFToken:  middleware.GetCSRFTokenFromContext(ctx),
		FormErrors: formErrors,
	})
	if err := layouts.Default(layoutData, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
