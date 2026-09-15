package validator_test

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateValidatorのテスト

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
			err := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
				Email: tc.email,
				Event: tc.event,
			})

			if tc.wantError {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Error("バリデーションエラーではない (nilまたは別の型のエラー)")
				} else if !ve.HasFieldError(tc.errorField) {
					t.Errorf("%sのフィールドエラーが見つからない", tc.errorField)
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
	// 新規メールアドレスでsignupイベント → 成功
	err := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: "newuser@example.com",
		Event: model.EmailConfirmationEventSignUp,
	})

	if err != nil {
		t.Fatalf("Validate()のエラー = %v、期待値 = nil", err)
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
	// 既存メールアドレスでsignupイベント → エラー
	err := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: "existing@example.com",
		Event: model.EmailConfirmationEventSignUp,
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("ValidationErrorを期待したが、nilか別の型のエラーだった")
	}
	if !ve.HasFieldError("email") {
		t.Error("'email'のフィールドエラーが無い")
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
	// 既存メールアドレスでpassword_resetイベント → 成功 (重複チェックをスキップ)
	err := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: "resetuser@example.com",
		Event: model.EmailConfirmationEventPasswordReset,
	})

	if err != nil {
		t.Fatalf("Validate()のエラー = %v、期待値 = nil", err)
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
	// 既存メールアドレスでemail_updateイベント → 成功 (重複チェックをスキップ)
	err := v.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: "updateuser@example.com",
		Event: model.EmailConfirmationEventEmailUpdate,
	})

	if err != nil {
		t.Fatalf("Validate()のエラー = %v、期待値 = nil", err)
	}
}

// UpdateValidatorのテスト

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
			name:       "5文字のコード (短すぎる)",
			code:       "ABC12",
			wantError:  true,
			errorField: "code",
		},
		{
			name:       "7文字のコード (長すぎる)",
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
			_, err := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
				EmailConfirmationID: "dummy-id",
				Code:                tc.code,
			})

			if tc.wantError {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Error("バリデーションエラーではない (nilまたは別の型のエラー)")
				} else if !ve.HasFieldError(tc.errorField) {
					t.Errorf("%sのフィールドエラーが見つからない", tc.errorField)
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

	// テストデータを作成 (有効な確認コード)
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("validator-success@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("ABC123").
		WithStartedAt(time.Now()).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	confirmation, err := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "ABC123",
	})

	if err != nil {
		t.Fatalf("Validate()のエラー = %v、期待値 = nil", err)
	}
	if confirmation == nil {
		t.Error("EmailConfirmationがnil")
	}
}

func TestEmailConfirmationUpdateValidator_Validate_CaseInsensitive(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	// テストデータを作成 (大文字のコード)
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("case-insensitive@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("XYZ789").
		WithStartedAt(time.Now()).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 小文字で入力しても検証が成功することを確認
	_, err := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "xyz789",
	})

	if err != nil {
		t.Fatalf("Validate()のエラー = %v、期待値 = nil (小文字でも成功する)", err)
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
	_, err := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: "00000000-0000-0000-0000-000000000000",
		Code:                "ABC123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("ValidationErrorを期待したが、nilか別の型のエラーだった")
	}
	if !ve.HasErrors() {
		t.Error("ValidationErrorにエラーが無い")
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
	// 検証しようとするとAppErrorが返る
	_, err := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "DEF456",
	})

	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatal("AppErrorを期待したが、nilか別の型のエラーだった")
	}
	if ae.Code != model.AppErrCodeConflict {
		t.Errorf("AppError.Code = %d、期待値 = %d (AppErrCodeConflict)", ae.Code, model.AppErrCodeConflict)
	}
}

func TestEmailConfirmationUpdateValidator_Validate_Expired(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	v := validator.NewEmailConfirmationUpdateValidator(repo)

	// 16分前のテストデータを作成 (15分で有効期限切れ)
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("expired@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("GHI789").
		WithStartedAt(time.Now().Add(-16 * time.Minute)).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")
	// 検証しようとするとエラーになる
	_, err := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "GHI789",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("ValidationErrorを期待したが、nilか別の型のエラーだった")
	}
	if !ve.HasErrors() {
		t.Error("ValidationErrorにエラーが無い")
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
	_, err := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "WRONG1",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("ValidationErrorを期待したが、nilか別の型のエラーだった")
	}
	if !ve.HasFieldError("code") {
		t.Error("ValidationErrorに'code'のフィールドエラーが無い")
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
	// 確認コードを検証 (イベント種別に関係なく検証できる)
	_, err := v.Validate(ctx, validator.EmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: ecID,
		Code:                "MNO345",
	})

	if err != nil {
		t.Fatalf("Validate()のエラー = %v、期待値 = nil", err)
	}
}
