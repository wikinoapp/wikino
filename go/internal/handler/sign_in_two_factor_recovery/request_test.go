package sign_in_two_factor_recovery_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor_recovery"
	"github.com/wikinoapp/wikino/go/internal/i18n"
)

func TestCreateRequest_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		recoveryCode string
		wantError    bool
		errorField   string
	}{
		{
			name:         "正常なリカバリーコード",
			recoveryCode: "code1234",
			wantError:    false,
		},
		{
			name:         "正常なリカバリーコード（全て数字）",
			recoveryCode: "12345678",
			wantError:    false,
		},
		{
			name:         "正常なリカバリーコード（全て小文字）",
			recoveryCode: "abcdefgh",
			wantError:    false,
		},
		{
			name:         "空のリカバリーコード",
			recoveryCode: "",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "7文字（短すぎる）",
			recoveryCode: "code123",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "9文字（長すぎる）",
			recoveryCode: "code12345",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "大文字を含む",
			recoveryCode: "CODE1234",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "一部大文字",
			recoveryCode: "Code1234",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "記号を含む",
			recoveryCode: "code123!",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "空白を含む",
			recoveryCode: "code 123",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "ハイフンを含む",
			recoveryCode: "code-123",
			wantError:    true,
			errorField:   "recovery_code",
		},
		{
			name:         "アンダースコアを含む",
			recoveryCode: "code_123",
			wantError:    true,
			errorField:   "recovery_code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// テスト用のコンテキストを作成（日本語ロケール）
			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			req := sign_in_two_factor_recovery.NewCreateRequest(tc.recoveryCode)
			errors := req.Validate(ctx)

			if tc.wantError {
				if errors == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if !errors.HasFieldError(tc.errorField) {
					t.Errorf("expected error for field %s but not found", tc.errorField)
				}
			} else {
				if errors != nil {
					t.Errorf("unexpected error: %v", errors)
				}
			}
		})
	}
}
