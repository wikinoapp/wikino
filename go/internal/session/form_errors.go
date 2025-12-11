package session

// FormErrors はフォームのバリデーションエラーを保持します
type FormErrors struct {
	// Global はフォーム全体のエラーメッセージ（特定のフィールドに紐づかないエラー）
	Global []string
	// Fields はフィールドごとのエラーメッセージ
	Fields map[string][]string
}

// NewFormErrors は新しい FormErrors を生成します
func NewFormErrors() *FormErrors {
	return &FormErrors{
		Global: []string{},
		Fields: make(map[string][]string),
	}
}

// AddGlobal はグローバルエラーを追加します
func (f *FormErrors) AddGlobal(message string) {
	f.Global = append(f.Global, message)
}

// AddField はフィールドエラーを追加します
func (f *FormErrors) AddField(field, message string) {
	if f.Fields == nil {
		f.Fields = make(map[string][]string)
	}
	f.Fields[field] = append(f.Fields[field], message)
}

// HasErrors はエラーがあるかどうかを返します
func (f *FormErrors) HasErrors() bool {
	if f == nil {
		return false
	}
	return len(f.Global) > 0 || len(f.Fields) > 0
}

// HasFieldError は指定されたフィールドにエラーがあるかどうかを返します
func (f *FormErrors) HasFieldError(field string) bool {
	if f == nil || f.Fields == nil {
		return false
	}
	return len(f.Fields[field]) > 0
}

// GetFieldErrors は指定されたフィールドのエラーメッセージを返します
func (f *FormErrors) GetFieldErrors(field string) []string {
	if f == nil || f.Fields == nil {
		return nil
	}
	return f.Fields[field]
}
