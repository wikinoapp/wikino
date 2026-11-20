package usecase

import (
	"context"
	"net/url"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPrepareTwoFactorAuthUsecase_Execute_開くたびに新しい秘密鍵で登録の途中の設定を作る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	userID := twoFactorAuthUser(t, tx, "prepare_2fa_uc_new", "", false)
	repo := repository.NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))
	uc := NewPrepareTwoFactorAuthUsecase(repo)

	first, err := uc.Execute(ctx, PrepareTwoFactorAuthInput{UserID: userID, Email: "prepare_2fa_uc_new@example.com"})
	if err != nil || first.Setup == nil {
		t.Fatalf("Execute()の結果 = %+v、エラー = %v", first, err)
	}
	u, err := url.Parse(first.Setup.OTPAuthURL)
	if err != nil || u.Query().Get("secret") != first.Setup.Secret || u.Path != "/Wikino:prepare_2fa_uc_new@example.com" {
		t.Errorf("otpauth URI = %q、秘密鍵とメールアドレスを含むことを期待", first.Setup.OTPAuthURL)
	}

	second, err := uc.Execute(ctx, PrepareTwoFactorAuthInput{UserID: userID, Email: "prepare_2fa_uc_new@example.com"})
	if err != nil || second.Setup == nil {
		t.Fatalf("2回目のExecute()の結果 = %+v、エラー = %v", second, err)
	}
	if second.Setup.Secret == first.Setup.Secret {
		t.Error("開き直しても秘密鍵が変わらない")
	}

	twoFactorAuth, err := repo.FindByUserID(ctx, userID)
	if err != nil || twoFactorAuth == nil {
		t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", twoFactorAuth, err)
	}
	if twoFactorAuth.Enabled || twoFactorAuth.Secret != second.Setup.Secret {
		t.Errorf("設定 = %+v、最後に出した秘密鍵の無効な設定を期待", twoFactorAuth)
	}
}

func TestPrepareTwoFactorAuthUsecase_Execute_有効なら秘密鍵を差し替えない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	userID := twoFactorAuthUser(t, tx, "prepare_2fa_uc_enabled", twoFactorAuthTestSecret, true)
	repo := repository.NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx))

	output, err := NewPrepareTwoFactorAuthUsecase(repo).Execute(ctx, PrepareTwoFactorAuthInput{UserID: userID, Email: "prepare_2fa_uc_enabled@example.com"})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output.Setup != nil {
		t.Errorf("Setup = %+v、nilを期待", output.Setup)
	}

	twoFactorAuth, err := repo.FindByUserID(ctx, userID)
	if err != nil || twoFactorAuth == nil {
		t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", twoFactorAuth, err)
	}
	if !twoFactorAuth.Enabled || twoFactorAuth.Secret != twoFactorAuthTestSecret {
		t.Errorf("設定 = %+v、有効な設定がそのまま残ることを期待", twoFactorAuth)
	}
}
