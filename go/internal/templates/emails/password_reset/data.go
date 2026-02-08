package password_reset

// Data はパスワードリセットメールテンプレートのデータ
type Data struct {
	Email    string
	ResetURL string
	AppURL   string
}
