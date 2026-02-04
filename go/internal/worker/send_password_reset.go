package worker

import (
	"github.com/riverqueue/river"
)

// SendPasswordResetArgs はパスワードリセットメール送信ジョブの引数です
type SendPasswordResetArgs struct {
	Email    string `json:"email"`
	ResetURL string `json:"reset_url"`
	AppURL   string `json:"app_url"`
	Locale   string `json:"locale"`
}

// Kind はジョブの種類を返します
func (SendPasswordResetArgs) Kind() string {
	return "send_password_reset"
}

// InsertOpts はジョブのInsertオプションを返します
func (SendPasswordResetArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5,
	}
}
