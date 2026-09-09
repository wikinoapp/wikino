package validator_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestAccountCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := query.New(db).WithTx(tx)
	userRepo := repository.NewUserRepository(queries)

	v := validator.NewAccountCreateValidator(userRepo)

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
			name:          "password too long",
			atname:        "testuser",
			password:      strings.Repeat("a", 129),
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:          "password with multibyte characters",
			atname:        "testuser",
			password:      "パスワード123",
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:          "password with space",
			atname:        "testuser",
			password:      "pass word",
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
			atname:     "testuser8",
			password:   "12345678",
			wantErrors: false,
		},
		{
			name:       "password exactly 128 chars",
			atname:     "testuser",
			password:   strings.Repeat("a", 128),
			wantErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, i18n.LangJa)

			err := v.Validate(ctx, validator.AccountCreateValidatorInput{
				Atname:   tt.atname,
				Password: tt.password,
			})

			if tt.wantErrors {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Error("expected ValidationError, got nil")
					return
				}
				if !ve.HasErrors() {
					t.Error("expected errors, got none")
				}
				if tt.expectedField != "" && !ve.HasFieldError(tt.expectedField) {
					t.Errorf("expected field error for %q", tt.expectedField)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestAccountCreateValidator_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := query.New(db).WithTx(tx)
	userRepo := repository.NewUserRepository(queries)

	v := validator.NewAccountCreateValidator(userRepo)

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
		{
			name:            "password too long ja",
			atname:          "testuser",
			password:        strings.Repeat("a", 129),
			locale:          "ja",
			expectedMessage: "パスワードは128文字以内で入力してください",
		},
		{
			name:            "password too long en",
			atname:          "testuser",
			password:        strings.Repeat("a", 129),
			locale:          "en",
			expectedMessage: "Password must be at most 128 characters",
		},
		{
			name:            "password invalid chars ja",
			atname:          "testuser",
			password:        "パスワード123",
			locale:          "ja",
			expectedMessage: "パスワードには印字可能なASCII文字のみ使用できます",
		},
		{
			name:            "password invalid chars en",
			atname:          "testuser",
			password:        "パスワード123",
			locale:          "en",
			expectedMessage: "Password can only contain printable ASCII characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.locale == "ja" {
				ctx = i18n.SetLocale(ctx, i18n.LangJa)
			} else {
				ctx = i18n.SetLocale(ctx, i18n.LangEn)
			}

			err := v.Validate(ctx, validator.AccountCreateValidatorInput{
				Atname:   tt.atname,
				Password: tt.password,
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("expected ValidationError, got nil")
			}

			// エラーメッセージが含まれているか確認
			found := false
			for _, messages := range ve.Fields {
				for _, msg := range messages {
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

func TestAccountCreateValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := query.New(db).WithTx(tx)
	userRepo := repository.NewUserRepository(queries)

	v := validator.NewAccountCreateValidator(userRepo)

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	err := v.Validate(ctx, validator.AccountCreateValidatorInput{
		Atname:   "newuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountCreateValidator_Validate_AtnameAlreadyTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := query.New(db).WithTx(tx)
	userRepo := repository.NewUserRepository(queries)

	// 既存のユーザーを作成
	now := time.Now()
	_, err := userRepo.Create(t.Context(), repository.CreateUserInput{
		Email:       "existing@example.com",
		Atname:      "existinguser",
		Name:        "",
		Description: "",
		Locale:      model.LocaleJa,
		TimeZone:    "Asia/Tokyo",
		JoinedAt:    now,
	})
	if err != nil {
		t.Fatalf("failed to create existing user: %v", err)
	}

	v := validator.NewAccountCreateValidator(userRepo)

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	err = v.Validate(ctx, validator.AccountCreateValidatorInput{
		Atname:   "existinguser",
		Password: "password123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasErrors() {
		t.Error("expected form errors to have errors")
	}
	if !ve.HasFieldError("atname") {
		t.Error("expected atname field error")
	}
}

func TestIsValidAtname(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		atname string
		want   bool
	}{
		{name: "英数字とアンダースコア", atname: "seed_user1", want: true},
		{name: "大文字を含む", atname: "SeedUser1", want: true},
		{name: "上限ちょうど", atname: strings.Repeat("a", validator.AtnameMaxLength), want: true},
		{name: "上限を1文字超える", atname: strings.Repeat("a", validator.AtnameMaxLength+1), want: false},
		{name: "空文字列", atname: "", want: false},
		{name: "ハイフンを含む", atname: "seed-user1", want: false},
		{name: "空白を含む", atname: "seed user1", want: false},
		{name: "日本語を含む", atname: "シードユーザー", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := validator.IsValidAtname(tt.atname); got != tt.want {
				t.Errorf("IsValidAtname(%q) = %v であることを期待したが %v だった", tt.atname, tt.want, got)
			}
		})
	}
}
