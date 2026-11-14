package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUserTwoFactorAuthRepository_FindByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserTwoFactorAuthRepository(q)

	// テストユーザーと二要素認証設定を作成 (無効)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("repo-2fa@example.com").
		WithAtname("repo-twofauser").
		BuildWithTwoFactorAuth("JBSWY3DPEHPK3PXP", false)

	t.Run("存在するユーザーの二要素認証設定を取得できる", func(t *testing.T) {
		twoFA, err := repo.FindByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindByUserID()のエラー = %v", err)
		}
		if twoFA == nil {
			t.Fatal("FindByUserID()がnilを返した、期待値 = twoFA")
		}
		if twoFA.UserID != userID {
			t.Errorf("twoFA.UserID = %v、期待値 = %v", twoFA.UserID, userID)
		}
		if twoFA.Secret != "JBSWY3DPEHPK3PXP" {
			t.Errorf("twoFA.Secret = %v、期待値 = JBSWY3DPEHPK3PXP", twoFA.Secret)
		}
		if twoFA.Enabled != false {
			t.Errorf("twoFA.Enabled = %v、期待値 = false", twoFA.Enabled)
		}
	})

	t.Run("存在しないユーザーIDはnilを返す", func(t *testing.T) {
		twoFA, err := repo.FindByUserID(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindByUserID()のエラー = %v", err)
		}
		if twoFA != nil {
			t.Errorf("FindByUserID() = %v、期待値 = nil", twoFA)
		}
	})
}

func TestUserTwoFactorAuthRepository_FindEnabledByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserTwoFactorAuthRepository(q)

	// 有効な二要素認証設定を持つユーザーを作成
	enabledUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("repo-2fa-enabled@example.com").
		WithAtname("repo-twofaenableduser").
		BuildWithTwoFactorAuth("ENABLEDSECRET123", true)

	// 無効な二要素認証設定を持つユーザーを作成
	disabledUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("repo-2fa-disabled@example.com").
		WithAtname("repo-twofadisableduser").
		BuildWithTwoFactorAuth("DISABLEDSECRET12", false)

	t.Run("有効な二要素認証設定を取得できる", func(t *testing.T) {
		twoFA, err := repo.FindEnabledByUserID(context.Background(), enabledUserID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID()のエラー = %v", err)
		}
		if twoFA == nil {
			t.Fatal("FindEnabledByUserID()がnilを返した、期待値 = twoFA")
		}
		if twoFA.Enabled != true {
			t.Errorf("twoFA.Enabled = %v、期待値 = true", twoFA.Enabled)
		}
		if twoFA.EnabledAt == nil {
			t.Error("twoFA.EnabledAtがnil")
		}
	})

	t.Run("無効な二要素認証設定はnilを返す", func(t *testing.T) {
		twoFA, err := repo.FindEnabledByUserID(context.Background(), disabledUserID)
		if err != nil {
			t.Fatalf("FindEnabledByUserID()のエラー = %v", err)
		}
		if twoFA != nil {
			t.Errorf("FindEnabledByUserID() = %v、期待値 = nil", twoFA)
		}
	})
}

func TestUserTwoFactorAuthRepository_UpdateRecoveryCodes(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserTwoFactorAuthRepository(q)

	// テストユーザーと二要素認証設定を作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("repo-2fa-recovery@example.com").
		WithAtname("repo-twofarecoveryuser").
		BuildWithTwoFactorAuth("RECOVERYSECRET12", true)

	t.Run("リカバリーコードを更新できる", func(t *testing.T) {
		newCodes := []string{"CODE1", "CODE2", "CODE3", "CODE4"}
		err := repo.UpdateRecoveryCodes(context.Background(), userID, newCodes)
		if err != nil {
			t.Fatalf("UpdateRecoveryCodes()のエラー = %v", err)
		}

		// 更新後の値を確認
		twoFA, err := repo.FindByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindByUserID()のエラー = %v", err)
		}
		if twoFA == nil {
			t.Fatal("FindByUserID()がnilを返した、期待値 = twoFA")
		}
		if len(twoFA.RecoveryCodes) != 4 {
			t.Errorf("len(twoFA.RecoveryCodes) = %d、期待値 = 4", len(twoFA.RecoveryCodes))
		}
	})
}

func TestUserTwoFactorAuthRepository_UpsertPending(t *testing.T) {
	t.Parallel()

	t.Run("設定の無いユーザーには登録の途中の設定を作る", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-upsert-new@example.com").
			WithAtname("repo_2fa_upsert_new").
			Build()

		pending, err := repo.UpsertPending(context.Background(), userID, "JBSWY3DPEHPK3PXP")
		if err != nil {
			t.Fatalf("UpsertPending()のエラー = %v", err)
		}
		if pending == nil || pending.Secret != "JBSWY3DPEHPK3PXP" || pending.Enabled || pending.EnabledAt != nil || len(pending.RecoveryCodes) != 0 {
			t.Errorf("設定 = %+v、新しい秘密鍵の無効な設定を期待", pending)
		}
	})

	t.Run("登録の途中の設定は秘密鍵を差し替えてリカバリーコードを空にする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-upsert-pending@example.com").
			WithAtname("repo_2fa_upsert_pend").
			BuildWithTwoFactorAuthAndRecoveryCodes("OLDSECRETOLDSECR", false, []string{"abcd1234"})

		pending, err := repo.UpsertPending(context.Background(), userID, "NEWSECRETNEWSECR")
		if err != nil {
			t.Fatalf("UpsertPending()のエラー = %v", err)
		}
		if pending == nil || pending.Secret != "NEWSECRETNEWSECR" || len(pending.RecoveryCodes) != 0 {
			t.Errorf("設定 = %+v、新しい秘密鍵でリカバリーコードの無い設定を期待", pending)
		}
	})

	t.Run("有効な設定は書き換えずにnilを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-upsert-enabled@example.com").
			WithAtname("repo_2fa_upsert_enab").
			BuildWithTwoFactorAuthAndRecoveryCodes("ENABLEDSECRETABC", true, []string{"abcd1234"})

		pending, err := repo.UpsertPending(context.Background(), userID, "NEWSECRETNEWSECR")
		if err != nil {
			t.Fatalf("UpsertPending()のエラー = %v", err)
		}
		if pending != nil {
			t.Errorf("設定 = %+v、nilを期待", pending)
		}

		current, err := repo.FindByUserID(context.Background(), userID)
		if err != nil || current == nil {
			t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", current, err)
		}
		if current.Secret != "ENABLEDSECRETABC" || !current.Enabled || len(current.RecoveryCodes) != 1 {
			t.Errorf("設定 = %+v、有効な設定がそのまま残ることを期待", current)
		}
	})
}

func TestUserTwoFactorAuthRepository_Enable(t *testing.T) {
	t.Parallel()

	t.Run("照合した秘密鍵の登録の途中の設定を有効にする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-enable@example.com").
			WithAtname("repo_2fa_enable").
			BuildWithTwoFactorAuth("PENDINGSECRETABC", false)

		codes := []string{"abcd1234", "efgh5678"}
		enabled, err := repo.Enable(context.Background(), userID, "PENDINGSECRETABC", codes)
		if err != nil {
			t.Fatalf("Enable()のエラー = %v", err)
		}
		if !enabled {
			t.Fatal("Enable() = false、trueを期待")
		}

		current, err := repo.FindByUserID(context.Background(), userID)
		if err != nil || current == nil {
			t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", current, err)
		}
		if !current.Enabled || current.EnabledAt == nil || len(current.RecoveryCodes) != 2 || current.RecoveryCodes[0] != "abcd1234" {
			t.Errorf("設定 = %+v、有効でリカバリーコードを持つ設定を期待", current)
		}
	})

	t.Run("秘密鍵が差し替わっていたら有効にしない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-enable-replaced@example.com").
			WithAtname("repo_2fa_enable_repl").
			BuildWithTwoFactorAuth("REPLACEDSECRETAB", false)

		enabled, err := repo.Enable(context.Background(), userID, "PENDINGSECRETABC", []string{"abcd1234"})
		if err != nil {
			t.Fatalf("Enable()のエラー = %v", err)
		}
		if enabled {
			t.Error("Enable() = true、falseを期待")
		}
	})

	t.Run("既に有効な設定は書き換えない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-enable-enabled@example.com").
			WithAtname("repo_2fa_enable_enab").
			BuildWithTwoFactorAuthAndRecoveryCodes("ENABLEDSECRETABC", true, []string{"keep1234"})

		enabled, err := repo.Enable(context.Background(), userID, "ENABLEDSECRETABC", []string{"abcd1234"})
		if err != nil {
			t.Fatalf("Enable()のエラー = %v", err)
		}
		if enabled {
			t.Error("Enable() = true、falseを期待")
		}

		current, err := repo.FindByUserID(context.Background(), userID)
		if err != nil || current == nil {
			t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", current, err)
		}
		if len(current.RecoveryCodes) != 1 || current.RecoveryCodes[0] != "keep1234" {
			t.Errorf("リカバリーコード = %v、書き換わらないことを期待", current.RecoveryCodes)
		}
	})
}

func TestUserTwoFactorAuthRepository_Disable(t *testing.T) {
	t.Parallel()

	t.Run("有効な設定を無効にしてリカバリーコードを空にする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-disable@example.com").
			WithAtname("repo_2fa_disable").
			BuildWithTwoFactorAuthAndRecoveryCodes("ENABLEDSECRETABC", true, []string{"abcd1234"})

		disabled, err := repo.Disable(context.Background(), userID)
		if err != nil {
			t.Fatalf("Disable()のエラー = %v", err)
		}
		if !disabled {
			t.Fatal("Disable() = false、trueを期待")
		}

		current, err := repo.FindByUserID(context.Background(), userID)
		if err != nil || current == nil {
			t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", current, err)
		}
		if current.Enabled || current.EnabledAt != nil || len(current.RecoveryCodes) != 0 {
			t.Errorf("設定 = %+v、無効でリカバリーコードの無い設定を期待", current)
		}
	})

	t.Run("無効な設定は書き換えずにfalseを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-disable-off@example.com").
			WithAtname("repo_2fa_disable_off").
			BuildWithTwoFactorAuth("PENDINGSECRETABC", false)

		disabled, err := repo.Disable(context.Background(), userID)
		if err != nil {
			t.Fatalf("Disable()のエラー = %v", err)
		}
		if disabled {
			t.Error("Disable() = true、falseを期待")
		}
	})
}

func TestUserTwoFactorAuthRepository_RegenerateRecoveryCodes(t *testing.T) {
	t.Parallel()

	t.Run("有効な設定のリカバリーコードを置き換える", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-regenerate@example.com").
			WithAtname("repo_2fa_regenerate").
			BuildWithTwoFactorAuthAndRecoveryCodes("ENABLEDSECRETABC", true, []string{"old12345"})

		regenerated, err := repo.RegenerateRecoveryCodes(context.Background(), userID, []string{"new12345", "new67890"})
		if err != nil {
			t.Fatalf("RegenerateRecoveryCodes()のエラー = %v", err)
		}
		if !regenerated {
			t.Fatal("RegenerateRecoveryCodes() = false、trueを期待")
		}

		current, err := repo.FindByUserID(context.Background(), userID)
		if err != nil || current == nil {
			t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", current, err)
		}
		if len(current.RecoveryCodes) != 2 || current.RecoveryCodes[0] != "new12345" || !current.Enabled {
			t.Errorf("設定 = %+v、新しいリカバリーコードの有効な設定を期待", current)
		}
	})

	t.Run("無効な設定は書き換えずにfalseを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("repo-2fa-regenerate-off@example.com").
			WithAtname("repo_2fa_regen_off").
			BuildWithTwoFactorAuth("PENDINGSECRETABC", false)

		regenerated, err := repo.RegenerateRecoveryCodes(context.Background(), userID, []string{"new12345"})
		if err != nil {
			t.Fatalf("RegenerateRecoveryCodes()のエラー = %v", err)
		}
		if regenerated {
			t.Error("RegenerateRecoveryCodes() = true、falseを期待")
		}

		current, err := repo.FindByUserID(context.Background(), userID)
		if err != nil || current == nil {
			t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", current, err)
		}
		if len(current.RecoveryCodes) != 0 {
			t.Errorf("リカバリーコード = %v、書き換わらないことを期待", current.RecoveryCodes)
		}
	})
}
