package model

import (
	"time"
)

// EmailConfirmationEvent はメール確認イベントの種類を表す
type EmailConfirmationEvent int32

const (
	// EmailConfirmationEventSignUp はサインアップ時のメール確認
	EmailConfirmationEventSignUp EmailConfirmationEvent = 0
	// EmailConfirmationEventEmailUpdate はメールアドレス変更時の確認
	EmailConfirmationEventEmailUpdate EmailConfirmationEvent = 1
	// EmailConfirmationEventPasswordReset はパスワードリセット時の確認
	EmailConfirmationEventPasswordReset EmailConfirmationEvent = 2
)

// EmailConfirmation はメール確認のドメインモデル
type EmailConfirmation struct {
	ID          string
	Email       string
	Event       EmailConfirmationEvent
	Code        string
	StartedAt   time.Time
	SucceededAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsExpired は確認コードの有効期限が切れているかを返す（15分）
func (e *EmailConfirmation) IsExpired() bool {
	return time.Since(e.StartedAt) > 15*time.Minute
}

// IsSucceeded は確認が完了しているかを返す
func (e *EmailConfirmation) IsSucceeded() bool {
	return e.SucceededAt != nil
}
