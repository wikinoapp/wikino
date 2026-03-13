package validator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateValidator のテスト

func TestEmailConfirmationCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		email      string
		event      model.EmailConfirmationEvent
		wantError  bool
		errorField string
	}{
		{
			name:      "正常なメールアドレス",
			email:     "test@example.com",
			event:     model.EmailConfirmationEventSignUp,
			wantError: false,
		},
		{
			name:       "空のメールアドレス",
			email:      "",
			event:      model.EmailConfirmationEventSignUp,
			wantError:  true,
			errorField: "email",
		},
		{
			name:       "不正な形式のメールアドレス",
			email:      "invalid-email",
			event:      model.EmailConfirmationEventSignUp,
			wantError:  true,
			errorField: "email",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			userRepo := repository.NewUserRepository(q)
			v := validator.NewEmailConfirmationCreateValidator(userRepo)

			ctx := i18n.SetLocale(context.Background(), "ja")
			result := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
				Email: tc.email,
				Event: tc.event,
			})

			if tc.wantError {
				if result.FormErrors == nil {
					t.Error("expected form errors, but got nil")
				} else if !result.FormErrors.HasFieldError(tc.errorField) {
					t.Errorf("expected field error for %s, but not found", tc.errorField)
				}
			}
		})
	}
}

func TestEmailConfirmationCreateValidator_Validate_SignUp_NewEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	v := validator.NewEmailConfirmationCreateValidator(userRepo)

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 新規メールアドレスで signup イベント → 成功
	result := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: "newuser@example.com",
		Event: model.EmailConfirmationEventSignUp,
	})

	if result.Err != nil {
		t.Fatalf("Validate() error = %v, want nil", result.Err)
	}
	if result.FormErrors != nil {
		t.Errorf("FormErrors should be nil, got %v", result.FormErrors)
	}
}

func TestEmailConfirmationCreateValidator_Validate_SignUp_ExistingEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	v := validator.NewEmailConfirmationCreateValidator(userRepo)

	// 既存ユーザーを作成
	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("existing@example.com").
		WithAtname("existinguser").
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 既存メールアドレスで signup イベント → エラー
	result := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: "existing@example.com",
		Event: model.EmailConfirmationEventSignUp,
	})

	if !errors.Is(result.Err, validator.ErrEmailAlreadyRegistered) {
		t.Errorf("Validate() error = %v, want %v", result.Err, validator.ErrEmailAlreadyRegistered)
	}
	if result.FormErrors == nil || !result.FormErrors.HasErrors() {
		t.Error("FormErrors should have errors")
	}
	if !result.FormErrors.HasFieldError("email") {
		t.Error("FormErrors should have field error for 'email'")
	}
}

func TestEmailConfirmationCreateValidator_Validate_PasswordReset_ExistingEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	v := validator.NewEmailConfirmationCreateValidator(userRepo)

	// 既存ユーザーを作成
	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("resetuser@example.com").
		WithAtname("resetuser").
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 既存メールアドレスで password_reset イベント → 成功（重複チェックをスキップ）
	result := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: "resetuser@example.com",
		Event: model.EmailConfirmationEventPasswordReset,
	})

	if result.Err != nil {
		t.Fatalf("Validate() error = %v, want nil", result.Err)
	}
	if result.FormErrors != nil {
		t.Errorf("FormErrors should be nil, got %v", result.FormErrors)
	}
}

func TestEmailConfirmationCreateValidator_Validate_EmailUpdate_ExistingEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	v := validator.NewEmailConfirmationCreateValidator(userRepo)

	// 既存ユーザーを作成
	_ = testutil.NewUserBuilder(t, tx).
		WithEmail("updateuser@example.com").
		WithAtname("updateuser").
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 既存メールアドレスで email_update イベント → 成功（重複チェックをスキップ）
	result := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: "updateuser@example.com",
		Event: model.EmailConfirmationEventEmailUpdate,
	})

	if result.Err != nil {
		t.Fatalf("Validate() error = %v, want nil", result.Err)
	}
	if result.FormErrors != nil {
		t.Errorf("FormErrors should be nil, got %v", result.FormErrors)
	}
}

// UpdateValidator のテスト

func TestEmailConfirmationUpdateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		code       string
		wantError  bool
		errorField string
	}{
		{
			name:      "正常な6桁のコード",
			code:      "ABC123",
			wantError: false,
		},
		{
			name:       "空のコード",
			code:       "",
			wantError:  true,
			errorField: "code",
		},
		{
			name:       "5文字のコード（短すぎる）",
			code:       "ABC12",
			wantError:  true,
			errorField: "code",
		},
		{
			name:       "7文字のコード（長すぎる）",
			code:       "ABC1234",
			wantError:  true,
			errorField: "code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			repo := repository.NewEmailConfirmationRepository(q)
			v := validator.NewEmailConfirmationUpdateValidator(repo)

			ctx := i18n.SetLocale(context.Background(), "ja")
			result := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
				EmailConfirmationID: "dummy-id",
				Code:                tc.code,
			})

			if tc.wantError {
				if result.FormErrors == nil {
					t.Error("expected form errors, but got nil")
				} else if !result.FormErrors.HasFieldError(tc.errorField) {
					t.Errorf("expected field error for %s, but not found", tc.errorField)
				}
			}
		})
	}
}

func TestEmailConfirmationUpdateValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	// テストデータを作成（有効な確認コード）
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("validator-success@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("ABC123").
		WithStartedAt(time.Now()).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	result := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "ABC123",
	})

	if result.Err != nil {
		t.Fatalf("Validate() error = %v, want nil", result.Err)
	}
	if result.EmailConfirmation == nil {
		t.Error("EmailConfirmation should not be nil")
	}
	if result.FormErrors != nil {
		t.Errorf("FormErrors should be nil, got %v", result.FormErrors)
	}
}

func TestEmailConfirmationUpdateValidator_Validate_CaseInsensitive(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	// テストデータを作成（大文字のコード）
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("case-insensitive@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("XYZ789").
		WithStartedAt(time.Now()).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 小文字で入力しても検証が成功することを確認
	result := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "xyz789",
	})

	if result.Err != nil {
		t.Fatalf("Validate() error = %v, want nil（小文字でも成功すべき）", result.Err)
	}
}

func TestEmailConfirmationUpdateValidator_Validate_NotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 存在しないIDで検証
	result := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: "00000000-0000-0000-0000-000000000000",
		Code:                "ABC123",
	})

	if !errors.Is(result.Err, validator.ErrEmailConfirmationNotFound) {
		t.Errorf("Validate() error = %v, want %v", result.Err, validator.ErrEmailConfirmationNotFound)
	}
	if result.FormErrors == nil || !result.FormErrors.HasErrors() {
		t.Error("FormErrors should have errors")
	}
}

func TestEmailConfirmationUpdateValidator_Validate_AlreadySucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	// 既に確認済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("already-succeeded@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("DEF456").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 検証しようとするとエラーになる
	result := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "DEF456",
	})

	if !errors.Is(result.Err, validator.ErrEmailConfirmationAlreadySucceeded) {
		t.Errorf("Validate() error = %v, want %v", result.Err, validator.ErrEmailConfirmationAlreadySucceeded)
	}
	// 既に確認済みの場合は FormErrors は nil（リダイレクトするため）
	if result.FormErrors != nil {
		t.Errorf("FormErrors should be nil for already succeeded, got %v", result.FormErrors)
	}
	// EmailConfirmation は返される（リダイレクト先の判断に使用）
	if result.EmailConfirmation == nil {
		t.Error("EmailConfirmation should not be nil for already succeeded")
	}
}

func TestEmailConfirmationUpdateValidator_Validate_Expired(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	// 16分前のテストデータを作成（15分で有効期限切れ）
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("expired@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("GHI789").
		WithStartedAt(time.Now().Add(-16 * time.Minute)).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 検証しようとするとエラーになる
	result := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "GHI789",
	})

	if !errors.Is(result.Err, validator.ErrEmailConfirmationExpired) {
		t.Errorf("Validate() error = %v, want %v", result.Err, validator.ErrEmailConfirmationExpired)
	}
	if result.FormErrors == nil || !result.FormErrors.HasErrors() {
		t.Error("FormErrors should have errors")
	}
}

func TestEmailConfirmationUpdateValidator_Validate_CodeMismatch(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	// テストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("code-mismatch@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("JKL012").
		WithStartedAt(time.Now()).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 間違ったコードで検証
	result := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "WRONG1",
	})

	if !errors.Is(result.Err, validator.ErrEmailConfirmationCodeMismatch) {
		t.Errorf("Validate() error = %v, want %v", result.Err, validator.ErrEmailConfirmationCodeMismatch)
	}
	if result.FormErrors == nil || !result.FormErrors.HasErrors() {
		t.Error("FormErrors should have errors")
	}
	// コード不一致の場合、フィールドエラーとして "code" が設定される
	if !result.FormErrors.HasFieldError("code") {
		t.Error("FormErrors should have field error for 'code'")
	}
}

func TestEmailConfirmationUpdateValidator_Validate_PasswordResetEvent(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	// パスワードリセットイベントのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("password-reset@example.com").
		WithEvent(model.EmailConfirmationEventPasswordReset).
		WithCode("MNO345").
		WithStartedAt(time.Now()).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 確認コードを検証（イベント種別に関係なく検証できる）
	result := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "MNO345",
	})

	if result.Err != nil {
		t.Fatalf("Validate() error = %v, want nil", result.Err)
	}
}
