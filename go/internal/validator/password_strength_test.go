package validator

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TestAddPasswordStrengthError は addPasswordStrengthError ヘルパーが
// auth.ValidatePasswordStrength の sentinel error を `password` フィールドの
// バリデーションエラーへ正しく変換することを検証する。
func TestAddPasswordStrengthError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		password         string
		wantFieldError   bool
		wantMessageJa    string
		wantMessageEn    string
		wantPlaceholders []string
	}{
		{
			name:           "正常系: 8文字ちょうど",
			password:       "12345678",
			wantFieldError: false,
		},
		{
			name:           "正常系: 128文字ちょうど",
			password:       strings.Repeat("a", 128),
			wantFieldError: false,
		},
		{
			name:             "短すぎる",
			password:         "abc",
			wantFieldError:   true,
			wantMessageJa:    "パスワードは8文字以上で入力してください",
			wantMessageEn:    "Password must be at least 8 characters",
			wantPlaceholders: []string{"8"},
		},
		{
			name:             "長すぎる",
			password:         strings.Repeat("a", 129),
			wantFieldError:   true,
			wantMessageJa:    "パスワードは128文字以内で入力してください",
			wantMessageEn:    "Password must be at most 128 characters",
			wantPlaceholders: []string{"128"},
		},
		{
			name:           "マルチバイト文字を含む",
			password:       "パスワード123",
			wantFieldError: true,
			wantMessageJa:  "パスワードには印字可能なASCII文字のみ使用できます",
			wantMessageEn:  "Password can only contain printable ASCII characters",
		},
		{
			name:           "スペースを含む",
			password:       "pass word",
			wantFieldError: true,
			wantMessageJa:  "パスワードには印字可能なASCII文字のみ使用できます",
			wantMessageEn:  "Password can only contain printable ASCII characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, locale := range []string{i18n.LangJa, i18n.LangEn} {
				ctx := i18n.SetLocale(context.Background(), locale)
				ve := model.NewValidationError()

				addPasswordStrengthError(ctx, ve, tt.password)

				gotFieldError := ve.HasFieldError("password")
				if gotFieldError != tt.wantFieldError {
					t.Errorf("locale=%s: HasFieldError(\"password\") = %v, want %v", locale, gotFieldError, tt.wantFieldError)
				}

				if !tt.wantFieldError {
					continue
				}

				wantMessage := tt.wantMessageJa
				if locale == i18n.LangEn {
					wantMessage = tt.wantMessageEn
				}

				errs := ve.GetFieldErrors("password")
				if len(errs) == 0 {
					t.Fatalf("locale=%s: expected at least one password field error", locale)
				}

				found := false
				for _, msg := range errs {
					if strings.Contains(msg, wantMessage) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("locale=%s: expected message containing %q, got %v", locale, wantMessage, errs)
				}

				for _, ph := range tt.wantPlaceholders {
					placeholderFound := false
					for _, msg := range errs {
						if strings.Contains(msg, ph) {
							placeholderFound = true
							break
						}
					}
					if !placeholderFound {
						t.Errorf("locale=%s: expected placeholder %q in message, got %v", locale, ph, errs)
					}
				}
			}
		})
	}
}

// TestAddPasswordStrengthError_DoesNotAddOtherFields は強度違反のエラーが
// `password` 以外のフィールドに紛れ込まないことを検証する。
func TestAddPasswordStrengthError_DoesNotAddOtherFields(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	ve := model.NewValidationError()

	addPasswordStrengthError(ctx, ve, "abc")

	if !ve.HasFieldError("password") {
		t.Fatal("expected password field error")
	}
	if ve.HasFieldError("password_confirmation") {
		t.Error("did not expect password_confirmation field error")
	}
	if len(ve.Global) != 0 {
		t.Errorf("did not expect global errors, got %v", ve.Global)
	}
}
