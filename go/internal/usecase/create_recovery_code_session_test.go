package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestCreateRecoveryCodeSessionUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("正常系: リカバリーコード検証・消費後にセッションを作成できる", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"code1111", "code2222", "code3333"}
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("recovery-session-success@example.com").
			WithAtname("recoverysession").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		recoveryValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
		uc := NewCreateRecoveryCodeSessionUsecase(db, recoveryValidator, userTwoFactorAuthRepo, userSessionRepo)

		ctx := i18n.SetLocale(context.Background(), "ja")
		output, err := uc.Execute(ctx, CreateRecoveryCodeSessionInput{
			UserID:       userID,
			RecoveryCode: "code1111",
			IPAddress:    "192.168.1.1",
			UserAgent:    "TestAgent",
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

		// リカバリーコードが消費されていることを確認
		twoFactorAuth, err := userTwoFactorAuthRepo.FindEnabledByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID() error = %v", err)
		}
		if twoFactorAuth == nil {
			t.Fatal("FindEnabledByUserID() returned nil")
		}
		if len(twoFactorAuth.RecoveryCodes) != 2 {
			t.Errorf("RecoveryCodes length = %d, want 2", len(twoFactorAuth.RecoveryCodes))
		}
		for _, code := range twoFactorAuth.RecoveryCodes {
			if code == "code1111" {
				t.Error("RecoveryCodes still contains 'code1111' after consumption")
			}
		}
	})

	t.Run("異常系: 無効なリカバリーコードの場合はValidationErrorを返す", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"real1111", "real2222"}
		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("recovery-session-invalid@example.com").
			WithAtname("recoveryinvalid").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		recoveryValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
		uc := NewCreateRecoveryCodeSessionUsecase(db, recoveryValidator, userTwoFactorAuthRepo, userSessionRepo)

		ctx := i18n.SetLocale(context.Background(), "ja")
		_, err := uc.Execute(ctx, CreateRecoveryCodeSessionInput{
			UserID:       userID,
			RecoveryCode: "wrongcod",
			IPAddress:    "192.168.1.1",
			UserAgent:    "TestAgent",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("expected ValidationError, got %v", err)
		}
		if !ve.HasErrors() {
			t.Error("expected validation errors")
		}

		// リカバリーコードが消費されていないことを確認
		twoFactorAuth, err := userTwoFactorAuthRepo.FindEnabledByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID() error = %v", err)
		}
		if len(twoFactorAuth.RecoveryCodes) != 2 {
			t.Errorf("RecoveryCodes should not be consumed, length = %d, want 2", len(twoFactorAuth.RecoveryCodes))
		}
	})

	t.Run("異常系: 空のリカバリーコードの場合はValidationErrorを返す", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)

		recoveryValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
		uc := NewCreateRecoveryCodeSessionUsecase(db, recoveryValidator, userTwoFactorAuthRepo, userSessionRepo)

		ctx := i18n.SetLocale(context.Background(), "ja")
		_, err := uc.Execute(ctx, CreateRecoveryCodeSessionInput{
			UserID:       "dummy-user-id",
			RecoveryCode: "",
			IPAddress:    "192.168.1.1",
			UserAgent:    "TestAgent",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("expected ValidationError, got %v", err)
		}
		if !ve.HasFieldError("recovery_code") {
			t.Error("expected field error for recovery_code")
		}
	})

	t.Run("異常系: 2FAが有効でないユーザーの場合はAppErrorを返す", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)

		userID := testutil.NewUserBuilderDB(t, db).
			WithEmail("recovery-session-notenabled@example.com").
			WithAtname("recoverynotenabled").
			BuildWithTwoFactorAuthAndRecoveryCodes("JBSWY3DPEHPK3PXP", false, []string{"code1111"})

		recoveryValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(userTwoFactorAuthRepo)
		uc := NewCreateRecoveryCodeSessionUsecase(db, recoveryValidator, userTwoFactorAuthRepo, userSessionRepo)

		ctx := i18n.SetLocale(context.Background(), "ja")
		_, err := uc.Execute(ctx, CreateRecoveryCodeSessionInput{
			UserID:       userID,
			RecoveryCode: "code1111",
			IPAddress:    "192.168.1.1",
			UserAgent:    "TestAgent",
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
