package password

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	passwordPage "github.com/wikinoapp/wikino/go/internal/templates/pages/password"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Edit はパスワード編集フォームを表示します (GET /password/edit)
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// トークンをクエリパラメータから取得
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderInvalidTokenError(w, r)
		return
	}

	// トークンを検証
	_, err := h.getTokenDataUC.Execute(ctx, usecase.GetPasswordResetTokenDataInput{
		Token: token,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrPasswordResetTokenUsed):
			h.renderTokenUsedError(w, r)
		case errors.Is(err, usecase.ErrPasswordResetTokenExpired):
			h.renderTokenExpiredError(w, r)
		case errors.Is(err, usecase.ErrPasswordResetTokenNotFound):
			h.renderInvalidTokenError(w, r)
		default:
			slog.ErrorContext(ctx, "トークンの検証に失敗しました", "error", err)
			h.renderInvalidTokenError(w, r)
		}
		return
	}

	// フォームを表示
	h.renderEditForm(w, r, token, nil)
}

// renderEditForm は編集フォームをレンダリングします
func (h *Handler) renderEditForm(w http.ResponseWriter, r *http.Request, token string, formErrors *session.FormErrors) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_edit_title")

	data := passwordPage.EditPageData{
		CSRFToken:  csrfToken,
		Token:      token,
		FormErrors: formErrors,
	}

	content := passwordPage.Edit(data)
	if err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "テンプレートのレンダリングに失敗しました", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderInvalidTokenError は無効なトークンエラーを表示します
func (h *Handler) renderInvalidTokenError(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	formErrors := session.NewFormErrors()
	formErrors.AddGlobal(i18n.T(ctx, "validation_token_invalid"))

	h.renderEditForm(w, r, "", formErrors)
}

// renderTokenUsedError は使用済みトークンエラーを表示します
func (h *Handler) renderTokenUsedError(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	formErrors := session.NewFormErrors()
	formErrors.AddGlobal(i18n.T(ctx, "validation_token_used"))

	h.renderEditForm(w, r, "", formErrors)
}

// renderTokenExpiredError は期限切れトークンエラーを表示します
func (h *Handler) renderTokenExpiredError(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	formErrors := session.NewFormErrors()
	formErrors.AddGlobal(i18n.T(ctx, "validation_token_expired"))

	h.renderEditForm(w, r, "", formErrors)
}
