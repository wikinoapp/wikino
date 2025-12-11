package sign_in_two_factor_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/sign_in_two_factor"
)

func TestCreateRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		totpCode       string
		wantErrors     bool
		wantFieldError string
	}{
		{
			name:       "正常な6桁コード",
			totpCode:   "123456",
			wantErrors: false,
		},
		{
			name:           "TOTPコードが空",
			totpCode:       "",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:           "5桁のコード",
			totpCode:       "12345",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:           "7桁のコード",
			totpCode:       "1234567",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:           "英字を含むコード",
			totpCode:       "12345a",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:           "スペースを含むコード",
			totpCode:       "123 456",
			wantErrors:     true,
			wantFieldError: "totp_code",
		},
		{
			name:       "全て0のコード",
			totpCode:   "000000",
			wantErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			req := sign_in_two_factor.NewCreateRequest(tt.totpCode)
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
