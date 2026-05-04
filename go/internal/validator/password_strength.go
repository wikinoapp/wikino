package validator

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// addPasswordStrengthError は auth.ValidatePasswordStrength の結果を
// `password` フィールドのバリデーションエラーとして翻訳済みメッセージで追加する。
//
// auth は i18n に依存しない設計のため、sentinel error → 翻訳キーの解決はここで行う。
// 文字数制限のメッセージは auth パッケージの定数 (Min/MaxPasswordLength) をプレースホルダー
// として埋め込み、定数変更時に翻訳本文との不整合が起きないようにしている。
func addPasswordStrengthError(ctx context.Context, ve *model.ValidationError, password string) {
	err := auth.ValidatePasswordStrength(password)
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, auth.ErrPasswordTooShort):
		ve.AddField("password", i18n.T(ctx, "validation_password_too_short", map[string]any{
			"MinLength": auth.MinPasswordLength,
		}))
	case errors.Is(err, auth.ErrPasswordTooLong):
		ve.AddField("password", i18n.T(ctx, "validation_password_too_long", map[string]any{
			"MaxLength": auth.MaxPasswordLength,
		}))
	case errors.Is(err, auth.ErrPasswordInvalidChars):
		ve.AddField("password", i18n.T(ctx, "validation_password_invalid_chars"))
	}
}
