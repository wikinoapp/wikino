package email_confirmation

import (
	"context"
	"net/mail"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// CreateRequest はメール確認コード送信リクエストのバリデーション用構造体
type CreateRequest struct {
	Email string
	Event string
}

// NewCreateRequest はリクエストからCreateRequestを生成します
func NewCreateRequest(email, event string) *CreateRequest {
	return &CreateRequest{
		Email: email,
		Event: event,
	}
}

// Validate はリクエストのバリデーションを行います
// エラーがある場合はFormErrorsを返します
func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// メールアドレスのバリデーション
	if r.Email == "" {
		errors.AddField("email", i18n.T(ctx, "validation_required"))
	} else if !isValidEmail(r.Email) {
		errors.AddField("email", i18n.T(ctx, "validation_email_invalid"))
	}

	// イベントタイプのバリデーション
	if r.Event == "" {
		errors.AddField("event", i18n.T(ctx, "validation_required"))
	} else if !isValidEvent(r.Event) {
		errors.AddField("event", i18n.T(ctx, "validation_invalid_event"))
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}

// GetEvent はイベントタイプを返します
func (r *CreateRequest) GetEvent() model.EmailConfirmationEvent {
	switch r.Event {
	case "signup":
		return model.EmailConfirmationEventSignUp
	case "email_update":
		return model.EmailConfirmationEventEmailUpdate
	case "password_reset":
		return model.EmailConfirmationEventPasswordReset
	default:
		return model.EmailConfirmationEventSignUp
	}
}

// isValidEmail はメールアドレスの形式をチェックします
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// isValidEvent はイベントタイプが有効かチェックします
func isValidEvent(event string) bool {
	validEvents := []string{"signup", "email_update", "password_reset"}
	for _, e := range validEvents {
		if event == e {
			return true
		}
	}
	return false
}

// UpdateRequest は確認コード検証リクエストのバリデーション用構造体
type UpdateRequest struct {
	Code string
}

// NewUpdateRequest はリクエストからUpdateRequestを生成します
func NewUpdateRequest(code string) *UpdateRequest {
	return &UpdateRequest{
		Code: code,
	}
}

// Validate はリクエストのバリデーションを行います
// エラーがある場合はFormErrorsを返します
func (r *UpdateRequest) Validate(ctx context.Context) *session.FormErrors {
	errors := session.NewFormErrors()

	// 確認コードのバリデーション
	if r.Code == "" {
		errors.AddField("code", i18n.T(ctx, "validation_required"))
	} else if len(r.Code) != 6 {
		errors.AddField("code", i18n.T(ctx, "validation_confirmation_code_invalid_length"))
	}

	if errors.HasErrors() {
		return errors
	}

	return nil
}
