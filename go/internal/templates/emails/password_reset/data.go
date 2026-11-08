// Package password_resetはパスワードリセットメールのテンプレートを提供します
package password_reset

// Dataはパスワードリセットメールテンプレートのデータ
type Data struct {
	Email    string
	ResetURL string
	AppURL   string
}
