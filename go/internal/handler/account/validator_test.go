package account_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/handler/account"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// 形式バリデーションのテスト

func TestCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	queries := query.New(db).WithTx(tx)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	userRepo := repository.NewUserRepository(queries)

	// メール確認情報を作成（確認完了済み）
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     "test@example.com",
		Event:     model.EmailConfirmationEventSignUp,
		Code:      "123456",
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}
	if err := emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID); err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	validator := account.NewCreateValidator(emailConfirmationRepo, userRepo)

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
			atname:     "testuser8",
			password:   "12345678",
			wantErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, i18n.LangJa)

			result := validator.Validate(ctx, account.CreateValidatorInput{
				EmailConfirmationID: emailConfirmation.ID,
				Atname:              tt.atname,
				Password:            tt.password,
			})

			if tt.wantErrors {
				if result.FormErrors == nil {
					t.Error("expected errors, got nil")
					return
				}
				if !result.FormErrors.HasErrors() {
					t.Error("expected errors, got none")
				}
				if tt.expectedField != "" && !result.FormErrors.HasFieldError(tt.expectedField) {
					t.Errorf("expected field error for %q", tt.expectedField)
				}
			} else {
				if result.FormErrors != nil {
					t.Errorf("expected no errors, got %v", result.FormErrors)
				}
				if result.Err != nil {
					t.Errorf("expected no error, got %v", result.Err)
				}
			}
		})
	}
}

func TestCreateValidator_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	queries := query.New(db).WithTx(tx)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	userRepo := repository.NewUserRepository(queries)

	// メール確認情報を作成（確認完了済み）
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     "test@example.com",
		Event:     model.EmailConfirmationEventSignUp,
		Code:      "123456",
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}
	if err := emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID); err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	validator := account.NewCreateValidator(emailConfirmationRepo, userRepo)

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
			ctx := context.Background()
			if tt.locale == "ja" {
				ctx = i18n.SetLocale(ctx, i18n.LangJa)
			} else {
				ctx = i18n.SetLocale(ctx, i18n.LangEn)
			}

			result := validator.Validate(ctx, account.CreateValidatorInput{
				EmailConfirmationID: emailConfirmation.ID,
				Atname:              tt.atname,
				Password:            tt.password,
			})

			if result.FormErrors == nil {
				t.Fatal("expected errors, got nil")
			}

			// エラーメッセージが含まれているか確認
			found := false
			for _, errors := range result.FormErrors.Fields {
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

// 状態バリデーションのテスト

func TestCreateValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	queries := query.New(db).WithTx(tx)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	userRepo := repository.NewUserRepository(queries)

	// メール確認情報を作成（確認完了済み）
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     "test@example.com",
		Event:     model.EmailConfirmationEventSignUp,
		Code:      "123456",
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}
	// 確認完了に更新
	if err := emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID); err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	validator := account.NewCreateValidator(emailConfirmationRepo, userRepo)

	// コンテキストを作成
	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	// 新しいアットネームで検証
	result := validator.Validate(ctx, account.CreateValidatorInput{
		EmailConfirmationID: emailConfirmation.ID,
		Atname:              "newuser",
		Password:            "password123",
	})
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.FormErrors != nil {
		t.Errorf("expected no form errors, got: %v", result.FormErrors)
	}
	if result.EmailConfirmation == nil {
		t.Error("expected email confirmation, got nil")
	}
	if result.EmailConfirmation.Email != "test@example.com" {
		t.Errorf("expected email %q, got %q", "test@example.com", result.EmailConfirmation.Email)
	}
}

func TestCreateValidator_Validate_EmailConfirmationNotFound(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	queries := query.New(db).WithTx(tx)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	userRepo := repository.NewUserRepository(queries)

	validator := account.NewCreateValidator(emailConfirmationRepo, userRepo)

	// コンテキストを作成
	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	// 存在しないメール確認IDで検証（有効なUUID形式だが存在しないID）
	result := validator.Validate(ctx, account.CreateValidatorInput{
		EmailConfirmationID: "00000000-0000-0000-0000-000000000000",
		Atname:              "newuser",
		Password:            "password123",
	})
	if result.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(result.Err, account.ErrEmailConfirmationNotFound) {
		t.Errorf("expected ErrEmailConfirmationNotFound, got: %v", result.Err)
	}
	if result.EmailConfirmation != nil {
		t.Error("expected nil email confirmation")
	}
}

func TestCreateValidator_Validate_EmailNotConfirmed(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	queries := query.New(db).WithTx(tx)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	userRepo := repository.NewUserRepository(queries)

	// メール確認情報を作成（確認未完了）
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     "test@example.com",
		Event:     model.EmailConfirmationEventSignUp,
		Code:      "123456",
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	validator := account.NewCreateValidator(emailConfirmationRepo, userRepo)

	// コンテキストを作成
	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	// 確認未完了のメール確認IDで検証
	result := validator.Validate(ctx, account.CreateValidatorInput{
		EmailConfirmationID: emailConfirmation.ID,
		Atname:              "newuser",
		Password:            "password123",
	})
	if result.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(result.Err, account.ErrEmailNotConfirmed) {
		t.Errorf("expected ErrEmailNotConfirmed, got: %v", result.Err)
	}
	if result.EmailConfirmation != nil {
		t.Error("expected nil email confirmation")
	}
}

func TestCreateValidator_Validate_AtnameAlreadyTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTestDB(t)
	queries := query.New(db).WithTx(tx)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	userRepo := repository.NewUserRepository(queries)

	// メール確認情報を作成（確認完了済み）
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     "test@example.com",
		Event:     model.EmailConfirmationEventSignUp,
		Code:      "123456",
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}
	// 確認完了に更新
	if err := emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID); err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	// 既存のユーザーを作成
	_, err = userRepo.Create(t.Context(), repository.CreateUserInput{
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

	validator := account.NewCreateValidator(emailConfirmationRepo, userRepo)

	// コンテキストを作成
	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	// 既存のアットネームで検証
	result := validator.Validate(ctx, account.CreateValidatorInput{
		EmailConfirmationID: emailConfirmation.ID,
		Atname:              "existinguser",
		Password:            "password123",
	})
	if result.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(result.Err, account.ErrAtnameAlreadyTaken) {
		t.Errorf("expected ErrAtnameAlreadyTaken, got: %v", result.Err)
	}
	if result.FormErrors == nil {
		t.Fatal("expected form errors, got nil")
	}
	if !result.FormErrors.HasErrors() {
		t.Error("expected form errors to have errors")
	}
	if !result.FormErrors.HasFieldError("atname") {
		t.Error("expected atname field error")
	}
	if result.EmailConfirmation == nil {
		t.Error("expected email confirmation, got nil")
	}
}
