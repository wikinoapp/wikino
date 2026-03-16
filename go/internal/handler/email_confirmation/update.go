package email_confirmation

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	emailconfirmationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/validator"
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

	// バリデーション（形式チェック + 状態チェック）
	result := h.updateValidator.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: emailConfirmationID,
		Code:                code,
	})

	// 形式バリデーションエラー（Err が nil でも FormErrors がある場合）
	if result.FormErrors != nil && result.Err == nil {
		h.renderEditForm(w, r, result.FormErrors, csrfToken)
		return
	}

	if result.Err != nil {
		switch {
		case errors.Is(result.Err, validator.ErrEmailConfirmationNotFound):
			slog.WarnContext(ctx, "メール確認情報が見つからない", "email_confirmation_id", emailConfirmationID)
			h.renderEditForm(w, r, result.FormErrors, csrfToken)
			return
		case errors.Is(result.Err, validator.ErrEmailConfirmationAlreadySucceeded):
			slog.WarnContext(ctx, "既に確認済み", "email_confirmation_id", emailConfirmationID)
			http.Redirect(w, r, "/accounts/new", http.StatusFound)
			return
		case errors.Is(result.Err, validator.ErrEmailConfirmationExpired):
			slog.WarnContext(ctx, "確認コードの有効期限切れ", "email_confirmation_id", emailConfirmationID)
			h.renderEditForm(w, r, result.FormErrors, csrfToken)
			return
		case errors.Is(result.Err, validator.ErrEmailConfirmationCodeMismatch):
			slog.WarnContext(ctx, "確認コードが一致しない", "email_confirmation_id", emailConfirmationID)
			h.renderEditForm(w, r, result.FormErrors, csrfToken)
			return
		default:
			slog.ErrorContext(ctx, "状態バリデーションでエラー", "error", result.Err, "email_confirmation_id", emailConfirmationID)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// メール確認を完了状態に更新
	if err := h.markEmailAsConfirmedUC.Execute(ctx, emailConfirmationID); err != nil {
		slog.ErrorContext(ctx, "メール確認の更新に失敗", "error", err, "email_confirmation_id", emailConfirmationID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
