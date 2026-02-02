package account_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/account"
	"github.com/wikinoapp/wikino/go/internal/i18n"
)

func TestCreateRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		atname        string
		password      string
		wantErrors    bool
		expectedField string
	}{
		{
			name:       "valid request",
			atname:     "testuser",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:          "empty atname",
			atname:        "",
			password:      "password123",
			wantErrors:    true,
			expectedField: "atname",
		},
		{
			name:          "empty password",
			atname:        "testuser",
			password:      "",
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:       "both empty",
			atname:     "",
			password:   "",
			wantErrors: true,
		},
		{
			name:          "atname too long",
			atname:        "verylongusernameover20",
			password:      "password123",
			wantErrors:    true,
			expectedField: "atname",
		},
		{
			name:          "atname with invalid characters",
			atname:        "test-user!@",
			password:      "password123",
			wantErrors:    true,
			expectedField: "atname",
		},
		{
			name:          "password too short",
			atname:        "testuser",
			password:      "short",
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:       "atname with underscore",
			atname:     "test_user",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:       "atname with numbers",
			atname:     "user123",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:       "atname exactly 20 chars",
			atname:     "12345678901234567890",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:       "password exactly 8 chars",
			atname:     "testuser",
			password:   "12345678",
			wantErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, i18n.LangJa)

			req := account.NewCreateRequest(tt.atname, tt.password)
			formErrors := req.Validate(ctx)

			if tt.wantErrors {
				if formErrors == nil {
					t.Error("expected errors, got nil")
					return
				}
				if !formErrors.HasErrors() {
					t.Error("expected errors, got none")
				}
				if tt.expectedField != "" && !formErrors.HasFieldError(tt.expectedField) {
					t.Errorf("expected field error for %q", tt.expectedField)
				}
			} else {
				if formErrors != nil {
					t.Errorf("expected no errors, got %v", formErrors)
				}
			}
		})
	}
}

func TestCreateRequest_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		atname          string
		password        string
		locale          string
		expectedMessage string
	}{
		{
			name:            "atname required ja",
			atname:          "",
			password:        "password123",
			locale:          "ja",
			expectedMessage: "アットネームを入力してください",
		},
		{
			name:            "atname required en",
			atname:          "",
			password:        "password123",
			locale:          "en",
			expectedMessage: "Please enter a username",
		},
		{
			name:            "atname too long ja",
			atname:          "verylongusernameover20",
			password:        "password123",
			locale:          "ja",
			expectedMessage: "アットネームは20文字以内で入力してください",
		},
		{
			name:            "atname too long en",
			atname:          "verylongusernameover20",
			password:        "password123",
			locale:          "en",
			expectedMessage: "Username must be 20 characters or less",
		},
		{
			name:            "atname invalid format ja",
			atname:          "test-user!",
			password:        "password123",
			locale:          "ja",
			expectedMessage: "アットネームは英数字とアンダースコアのみ使用できます",
		},
		{
			name:            "atname invalid format en",
			atname:          "test-user!",
			password:        "password123",
			locale:          "en",
			expectedMessage: "Username can only contain letters, numbers, and underscores",
		},
		{
			name:            "password required ja",
			atname:          "testuser",
			password:        "",
			locale:          "ja",
			expectedMessage: "パスワードを入力してください",
		},
		{
			name:            "password required en",
			atname:          "testuser",
			password:        "",
			locale:          "en",
			expectedMessage: "Please enter a password",
		},
		{
			name:            "password too short ja",
			atname:          "testuser",
			password:        "short",
			locale:          "ja",
			expectedMessage: "パスワードは8文字以上で入力してください",
		},
		{
			name:            "password too short en",
			atname:          "testuser",
			password:        "short",
			locale:          "en",
			expectedMessage: "Password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tt.locale == "ja" {
				ctx = i18n.SetLocale(ctx, i18n.LangJa)
			} else {
				ctx = i18n.SetLocale(ctx, i18n.LangEn)
			}

			req := account.NewCreateRequest(tt.atname, tt.password)
			formErrors := req.Validate(ctx)

			if formErrors == nil {
				t.Fatal("expected errors, got nil")
			}

			// エラーメッセージが含まれているか確認
			found := false
			for _, errors := range formErrors.Fields {
				for _, msg := range errors {
					if strings.Contains(msg, tt.expectedMessage) {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("expected message %q not found in errors", tt.expectedMessage)
			}
		})
	}
}
