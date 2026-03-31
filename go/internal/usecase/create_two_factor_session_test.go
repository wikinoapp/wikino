package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestCreateTwoFactorSessionUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("正常系: TOTPコード検証後にセッションを作成できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)

		secret := "JBSWY3DPEHPK3PXP"
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("2fa-session-success@example.com").
			WithAtname("twofasession").
			BuildWithTwoFactorAuth(secret, true)

		code, err := totp.GenerateCode(secret, time.Now())
		if err != nil {
			t.Fatalf("TOTPコード生成に失敗: %v", err)
		}

		twoFactorValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
		createSessionUC := NewCreateUserSessionUsecase(userSessionRepo)
		uc := NewCreateTwoFactorSessionUsecase(twoFactorValidator, createSessionUC)

		ctx := i18n.SetLocale(context.Background(), "ja")
		output, err := uc.Execute(ctx, CreateTwoFactorSessionInput{
			UserID:    userID,
			TOTPCode:  code,
			IPAddress: "192.168.1.1",
			UserAgent: "TestAgent",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("Execute() returned nil, want output")
		}
		if output.Token == "" {
			t.Error("Execute() returned empty token")
		}

		// セッションがDBに保存されていることを確認
		session, err := userSessionRepo.FindByToken(ctx, output.Token)
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
		}
		if session == nil {
			t.Fatal("session not found in DB")
		}
		if session.UserID != userID {
			t.Errorf("session.UserID = %v, want %v", session.UserID, userID)
		}
	})

	t.Run("異常系: 無効なTOTPコードの場合はValidationErrorを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)

		secret := "JBSWY3DPEHPK3PXP"
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("2fa-session-invalid@example.com").
			WithAtname("twofainvalid").
			BuildWithTwoFactorAuth(secret, true)

		twoFactorValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
		createSessionUC := NewCreateUserSessionUsecase(userSessionRepo)
		uc := NewCreateTwoFactorSessionUsecase(twoFactorValidator, createSessionUC)

		ctx := i18n.SetLocale(context.Background(), "ja")
		_, err := uc.Execute(ctx, CreateTwoFactorSessionInput{
			UserID:    userID,
			TOTPCode:  "000000",
			IPAddress: "192.168.1.1",
			UserAgent: "TestAgent",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("expected ValidationError, got %v", err)
		}
		if !ve.HasErrors() {
			t.Error("expected validation errors")
		}
	})

	t.Run("異常系: 空のTOTPコードの場合はValidationErrorを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)

		twoFactorValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
		createSessionUC := NewCreateUserSessionUsecase(userSessionRepo)
		uc := NewCreateTwoFactorSessionUsecase(twoFactorValidator, createSessionUC)

		ctx := i18n.SetLocale(context.Background(), "ja")
		_, err := uc.Execute(ctx, CreateTwoFactorSessionInput{
			UserID:    "dummy-user-id",
			TOTPCode:  "",
			IPAddress: "192.168.1.1",
			UserAgent: "TestAgent",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("expected ValidationError, got %v", err)
		}
		if !ve.HasFieldError("totp_code") {
			t.Error("expected field error for totp_code")
		}
	})

	t.Run("異常系: 2FAが有効でないユーザーの場合はAppErrorを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("2fa-session-notenabled@example.com").
			WithAtname("twofanotenabled").
			BuildWithTwoFactorAuth("JBSWY3DPEHPK3PXP", false)

		twoFactorValidator := validator.NewSignInTwoFactorCreateValidator(userTwoFactorAuthRepo)
		createSessionUC := NewCreateUserSessionUsecase(userSessionRepo)
		uc := NewCreateTwoFactorSessionUsecase(twoFactorValidator, createSessionUC)

		ctx := i18n.SetLocale(context.Background(), "ja")
		_, err := uc.Execute(ctx, CreateTwoFactorSessionInput{
			UserID:    userID,
			TOTPCode:  "123456",
			IPAddress: "192.168.1.1",
			UserAgent: "TestAgent",
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %v", err)
		}
		if ae.Code != model.AppErrCodeTwoFactorNotEnabled {
			t.Errorf("expected AppErrCodeTwoFactorNotEnabled, got %d", ae.Code)
		}
	})
}
