package password_reset

// Dataはパスワードリセットメールテンプレートのデータ
type Data struct {
	Email    string
	ResetURL string
	AppURL   string
}
