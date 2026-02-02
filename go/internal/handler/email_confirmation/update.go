package email_confirmation

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	emailconfirmationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Update は確認コードを検証します (PATCH /email_confirmation)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// セッションから email_confirmation_id を取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// フォームをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// リクエストのバリデーション
	req := NewUpdateRequest(code)
	if formErrors := req.Validate(ctx); formErrors != nil {
		h.renderEditForm(w, r, formErrors, csrfToken)
		return
	}

	// 確認コードを検証
	err := h.verifyEmailConfirmationUC.Execute(ctx, usecase.VerifyEmailConfirmationInput{
		EmailConfirmationID: emailConfirmationID,
		Code:                code,
	})
	if err != nil {
		formErrors := session.NewFormErrors()

		switch {
		case errors.Is(err, usecase.ErrEmailConfirmationNotFound):
			slog.WarnContext(ctx, "メール確認情報が見つからない", "email_confirmation_id", emailConfirmationID)
			formErrors.AddGlobal(i18n.T(ctx, "validation_confirmation_not_found"))
		case errors.Is(err, usecase.ErrEmailConfirmationAlreadySucceeded):
			slog.WarnContext(ctx, "既に確認済み", "email_confirmation_id", emailConfirmationID)
			http.Redirect(w, r, "/accounts/new", http.StatusFound)
			return
		case errors.Is(err, usecase.ErrEmailConfirmationExpired):
			slog.WarnContext(ctx, "確認コードの有効期限切れ", "email_confirmation_id", emailConfirmationID)
			formErrors.AddGlobal(i18n.T(ctx, "validation_confirmation_code_expired"))
		case errors.Is(err, usecase.ErrEmailConfirmationCodeMismatch):
			slog.WarnContext(ctx, "確認コードが一致しない", "email_confirmation_id", emailConfirmationID)
			formErrors.AddField("code", i18n.T(ctx, "validation_confirmation_code_mismatch"))
		default:
			slog.ErrorContext(ctx, "確認コード検証でエラー", "error", err, "email_confirmation_id", emailConfirmationID)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		h.renderEditForm(w, r, formErrors, csrfToken)
		return
	}

	// 確認成功：アカウント作成ページにリダイレクト
	http.Redirect(w, r, "/accounts/new", http.StatusFound)
}

// renderEditForm は確認コード入力フォームをエラー付きでレンダリングします
func (h *Handler) renderEditForm(w http.ResponseWriter, r *http.Request, formErrors *session.FormErrors, csrfToken string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "email_confirmation_edit_title")

	content := emailconfirmationpages.Edit(emailconfirmationpages.EditPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
	})

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	w.WriteHeader(http.StatusUnprocessableEntity)

	err := layouts.Simple(meta, nil, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
