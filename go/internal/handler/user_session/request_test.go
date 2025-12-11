package user_session_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/user_session"
)

func TestCreateRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		email          string
		password       string
		wantErrors     bool
		wantFieldError string
	}{
		{
			name:       "正常なリクエスト",
			email:      "test@example.com",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:           "メールアドレスが空",
			email:          "",
			password:       "password123",
			wantErrors:     true,
			wantFieldError: "email",
		},
		{
			name:           "メールアドレスが無効な形式",
			email:          "invalid-email",
			password:       "password123",
			wantErrors:     true,
			wantFieldError: "email",
		},
		{
			name:           "パスワードが空",
			email:          "test@example.com",
			password:       "",
			wantErrors:     true,
			wantFieldError: "password",
		},
		{
			name:           "両方が空",
			email:          "",
			password:       "",
			wantErrors:     true,
			wantFieldError: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			req := user_session.NewCreateRequest(tt.email, tt.password)
			errors := req.Validate(ctx)

			if tt.wantErrors && errors == nil {
				t.Error("expected errors, but got nil")
			}

			if !tt.wantErrors && errors != nil {
				t.Errorf("expected no errors, but got: %v", errors)
			}

			if tt.wantErrors && errors != nil && tt.wantFieldError != "" {
				if !errors.HasFieldError(tt.wantFieldError) {
					t.Errorf("expected field error for %s, but not found", tt.wantFieldError)
				}
			}
		})
	}
}
