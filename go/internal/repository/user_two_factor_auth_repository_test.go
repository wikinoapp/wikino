package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUserTwoFactorAuthRepository_FindByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserTwoFactorAuthRepository(q)

	// テストユーザーと二要素認証設定を作成（無効）
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("2fa@example.com").
		WithAtname("twofauser").
		BuildWithTwoFactorAuth("JBSWY3DPEHPK3PXP", false)

	t.Run("存在するユーザーの二要素認証設定を取得できる", func(t *testing.T) {
		twoFA, err := repo.FindByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}
		if twoFA == nil {
			t.Fatal("FindByUserID() returned nil, want twoFA")
		}
		if twoFA.UserID != userID {
			t.Errorf("twoFA.UserID = %v, want %v", twoFA.UserID, userID)
		}
		if twoFA.Secret != "JBSWY3DPEHPK3PXP" {
			t.Errorf("twoFA.Secret = %v, want JBSWY3DPEHPK3PXP", twoFA.Secret)
		}
		if twoFA.Enabled != false {
			t.Errorf("twoFA.Enabled = %v, want false", twoFA.Enabled)
		}
	})

	t.Run("存在しないユーザーIDはnilを返す", func(t *testing.T) {
		twoFA, err := repo.FindByUserID(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}
		if twoFA != nil {
			t.Errorf("FindByUserID() = %v, want nil", twoFA)
		}
	})
}

func TestUserTwoFactorAuthRepository_FindEnabledByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserTwoFactorAuthRepository(q)

	// 有効な二要素認証設定を持つユーザーを作成
	enabledUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("2fa-enabled@example.com").
		WithAtname("twofaenableduser").
		BuildWithTwoFactorAuth("ENABLEDSECRET123", true)

	// 無効な二要素認証設定を持つユーザーを作成
	disabledUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("2fa-disabled@example.com").
		WithAtname("twofadisableduser").
		BuildWithTwoFactorAuth("DISABLEDSECRET12", false)

	t.Run("有効な二要素認証設定を取得できる", func(t *testing.T) {
		twoFA, err := repo.FindEnabledByUserID(context.Background(), enabledUserID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID() error = %v", err)
		}
		if twoFA == nil {
			t.Fatal("FindEnabledByUserID() returned nil, want twoFA")
		}
		if twoFA.Enabled != true {
			t.Errorf("twoFA.Enabled = %v, want true", twoFA.Enabled)
		}
		if twoFA.EnabledAt == nil {
			t.Error("twoFA.EnabledAt should not be nil")
		}
	})

	t.Run("無効な二要素認証設定はnilを返す", func(t *testing.T) {
		twoFA, err := repo.FindEnabledByUserID(context.Background(), disabledUserID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID() error = %v", err)
		}
		if twoFA != nil {
			t.Errorf("FindEnabledByUserID() = %v, want nil", twoFA)
		}
	})
}

func TestUserTwoFactorAuthRepository_UpdateRecoveryCodes(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserTwoFactorAuthRepository(q)

	// テストユーザーと二要素認証設定を作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("2fa-recovery@example.com").
		WithAtname("twofarecoveryuser").
		BuildWithTwoFactorAuth("RECOVERYSECRET12", true)

	t.Run("リカバリーコードを更新できる", func(t *testing.T) {
		newCodes := []string{"CODE1", "CODE2", "CODE3", "CODE4"}
		err := repo.UpdateRecoveryCodes(context.Background(), userID, newCodes)
		if err != nil {
			t.Fatalf("UpdateRecoveryCodes() error = %v", err)
		}

		// 更新後の値を確認
		twoFA, err := repo.FindByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindByUserID() error = %v", err)
		}
		if twoFA == nil {
			t.Fatal("FindByUserID() returned nil, want twoFA")
		}
		if len(twoFA.RecoveryCodes) != 4 {
			t.Errorf("len(twoFA.RecoveryCodes) = %d, want 4", len(twoFA.RecoveryCodes))
		}
	})
}
