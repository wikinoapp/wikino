// Package settings_account_deletionはアカウントの削除の確認画面と削除処理のHTTPハンドラーを提供します。
package settings_account_deletion

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

// Handlerはアカウントの削除の確認画面と、その送信先の削除処理を提供します。
type Handler struct {
	cfg                     *config.Config
	sessionMgr              *session.Manager
	flashMgr                *session.FlashManager
	getAccountDeletionNewUC *usecase.GetAccountDeletionNewUsecase
	discardAccountUC        *usecase.DiscardAccountUsecase
}

// NewHandlerはアカウントの削除のハンドラーを生成します
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	getAccountDeletionNewUC *usecase.GetAccountDeletionNewUsecase,
	discardAccountUC *usecase.DiscardAccountUsecase,
) *Handler {
	return &Handler{
		cfg:                     cfg,
		sessionMgr:              sessionMgr,
		flashMgr:                flashMgr,
		getAccountDeletionNewUC: getAccountDeletionNewUC,
		discardAccountUC:        discardAccountUC,
	}
}

// renderFormは、閲覧者が参加しているスペースを読み直して確認画面を描画します。
// 参加しているスペースがあれば、フォームの代わりに削除できない理由とその一覧を出します。
func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, user *model.User, status int, atname string, formErrors *model.ValidationError) {
	ctx := r.Context()

	output, err := h.getAccountDeletionNewUC.Execute(ctx, usecase.GetAccountDeletionNewInput{UserID: user.ID})
	if err != nil {
		slog.ErrorContext(ctx, "アカウントの削除の確認画面のデータの取得に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.render(w, r, user, status, settingspages.AccountDeletionNewData{
		CSRFToken:    middleware.GetCSRFTokenFromContext(ctx),
		UserAtname:   user.Atname,
		Atname:       atname,
		ActiveSpaces: viewmodel.NewSpaces(output.ActiveSpaces),
		FormErrors:   formErrors,
	})
}

// renderは確認画面を、設定の他の画面と共有するレイアウトへ描画します。
func (h *Handler) render(w http.ResponseWriter, r *http.Request, user *model.User, status int, data settingspages.AccountDeletionNewData) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "settings_account_deletion_new_title")

	layoutData := layouts.DefaultLayoutData{
		Meta: meta,

		GlobalNav: components.GlobalNavData{
			CurrentPageName: templates.PageNameSettingsAccountDeletionNew,
			SignedIn:        true,
			UserAtname:      user.Atname,
		},

		BreadcrumbHeader: components.BreadcrumbHeaderData{
			MaxWidthClass: "max-w-2xl",
			Items: append(components.HomeBreadcrumbItems(ctx, true),
				components.BreadcrumbItem{
					Label: i18n.T(ctx, "settings_account_deletion_new_breadcrumb_settings"),
					Path:  templates.SettingsPath(),
				},
				components.BreadcrumbItem{
					Label:     i18n.T(ctx, "settings_account_deletion_new_breadcrumb"),
					IsCurrent: true,
				},
			),
		},
	}

	// 設定の画面はログインした本人のための画面のため、検索エンジンにインデックスさせない
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := layouts.Default(layoutData, settingspages.AccountDeletionNew(data)).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
