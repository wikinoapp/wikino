package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestConsumeRecoveryCodeUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("リカバリーコードが正常に消費される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"abc12345", "def67890", "ghi11111"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("consume-valid@example.com").
			WithAtname("consumevalid").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		// リカバリーコードを消費
		err := uc.Execute(context.Background(), ConsumeRecoveryCodeInput{
			UserID:       userID,
			RecoveryCode: "abc12345",
			CurrentCodes: recoveryCodes,
		})
		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}

		// リカバリーコードが消費されていることを確認
		twoFactorAuth, err := userTwoFactorAuthRepo.FindEnabledByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID() error = %v", err)
		}
		if twoFactorAuth == nil {
			t.Fatal("FindEnabledByUserID() returned nil")
		}
		if len(twoFactorAuth.RecoveryCodes) != 2 {
			t.Errorf("RecoveryCodes length = %d, want 2", len(twoFactorAuth.RecoveryCodes))
		}
		// abc12345 が削除されていることを確認
		for _, code := range twoFactorAuth.RecoveryCodes {
			if code == "abc12345" {
				t.Error("RecoveryCodes still contains 'abc12345' after consumption")
			}
		}
	})

	t.Run("残りのリカバリーコードは保持される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)
		uc := NewConsumeRecoveryCodeUsecase(userTwoFactorAuthRepo)

		secret := "JBSWY3DPEHPK3PXP"
		recoveryCodes := []string{"code1111", "code2222", "code3333"}
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("consume-preserve@example.com").
			WithAtname("consumepreserve").
			BuildWithTwoFactorAuthAndRecoveryCodes(secret, true, recoveryCodes)

		// 最初のコードを消費
		err := uc.Execute(context.Background(), ConsumeRecoveryCodeInput{
			UserID:       userID,
			RecoveryCode: "code1111",
			CurrentCodes: recoveryCodes,
		})
		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}

		// 残りのコードが保持されていることを確認
		twoFactorAuth, err := userTwoFactorAuthRepo.FindEnabledByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID() error = %v", err)
		}
		if twoFactorAuth == nil {
			t.Fatal("FindEnabledByUserID() returned nil")
		}

		// code2222 と code3333 が残っていることを確認
		found2222 := false
		found3333 := false
		for _, code := range twoFactorAuth.RecoveryCodes {
			if code == "code2222" {
				found2222 = true
			}
			if code == "code3333" {
				found3333 = true
			}
		}
		if !found2222 {
			t.Error("RecoveryCodes does not contain 'code2222'")
		}
		if !found3333 {
			t.Error("RecoveryCodes does not contain 'code3333'")
		}
	})
}
