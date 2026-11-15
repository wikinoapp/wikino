package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newRegenerateTwoFactorAuthRecoveryCodesUsecaseForTest(tx *sql.Tx) (*RegenerateTwoFactorAuthRecoveryCodesUsecase, *repository.UserTwoFactorAuthRepository) {
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewUserTwoFactorAuthRepository(q)
	return NewRegenerateTwoFactorAuthRecoveryCodesUsecase(
		validator.NewSettingsTwoFactorAuthRecoveryCodeCreateValidator(repository.NewUserPasswordRepository(q)),
		repo,
	), repo
}

func TestRegenerateTwoFactorAuthRecoveryCodesUsecase_Execute_新しいコードだけがログインで通る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	digest, err := auth.HashPassword(twoFactorAuthTestPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("regenerate_2fa_uc_ok@example.com").
		WithAtname("regenerate_2fa_uc_ok").
		BuildWithPassword(digest)
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO user_two_factor_auths (user_id, secret, enabled, enabled_at, recovery_codes, created_at, updated_at) VALUES ($1, $2, true, NOW(), '{oldcode1}', NOW(), NOW())",
		string(userID), twoFactorAuthTestSecret); err != nil {
		t.Fatalf("二要素認証の設定の作成に失敗: %v", err)
	}

	uc, repo := newRegenerateTwoFactorAuthRecoveryCodesUsecaseForTest(tx)
	if err := uc.Execute(ctx, RegenerateTwoFactorAuthRecoveryCodesInput{UserID: userID, Password: twoFactorAuthTestPassword}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	twoFactorAuth, err := repo.FindByUserID(ctx, userID)
	if err != nil || twoFactorAuth == nil {
		t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", twoFactorAuth, err)
	}
	if len(twoFactorAuth.RecoveryCodes) != auth.RecoveryCodeCount || !twoFactorAuth.Enabled || twoFactorAuth.Secret != twoFactorAuthTestSecret {
		t.Fatalf("設定 = %+v、秘密鍵を保ったまま新しいリカバリーコードを持つことを期待", twoFactorAuth)
	}

	// Go版のログインのリカバリーコードの検証で、新しいコードが通り、前のコードは通らない
	recoveryValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(repo)
	if _, err := recoveryValidator.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
		UserID:       userID,
		RecoveryCode: twoFactorAuth.RecoveryCodes[0],
	}); err != nil {
		t.Errorf("新しいリカバリーコードの検証のエラー = %v", err)
	}
	_, err = recoveryValidator.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
		UserID:       userID,
		RecoveryCode: "oldcode1",
	})
	if ve := model.AsValidationError(err); ve == nil {
		t.Errorf("前のリカバリーコードでValidationErrorを期待したが、%vだった", err)
	}
}

func TestRegenerateTwoFactorAuthRecoveryCodesUsecase_Execute_パスワードが違うと作り直さない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	digest, err := auth.HashPassword(twoFactorAuthTestPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("regenerate_2fa_uc_wrong@example.com").
		WithAtname("regenerate_2fa_uc_ng").
		BuildWithPassword(digest)
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO user_two_factor_auths (user_id, secret, enabled, enabled_at, recovery_codes, created_at, updated_at) VALUES ($1, $2, true, NOW(), '{keepcode}', NOW(), NOW())",
		string(userID), twoFactorAuthTestSecret); err != nil {
		t.Fatalf("二要素認証の設定の作成に失敗: %v", err)
	}

	uc, repo := newRegenerateTwoFactorAuthRecoveryCodesUsecaseForTest(tx)
	err = uc.Execute(ctx, RegenerateTwoFactorAuthRecoveryCodesInput{UserID: userID, Password: "wrong-password"})
	if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("password") {
		t.Fatalf("passwordのValidationErrorを期待したが、%vだった", err)
	}

	twoFactorAuth, err := repo.FindByUserID(ctx, userID)
	if err != nil || twoFactorAuth == nil || len(twoFactorAuth.RecoveryCodes) != 1 || twoFactorAuth.RecoveryCodes[0] != "keepcode" {
		t.Errorf("設定 = %+v、リカバリーコードが変わらないことを期待 (エラー = %v)", twoFactorAuth, err)
	}
}

func TestRegenerateTwoFactorAuthRecoveryCodesUsecase_Execute_有効でなければ競合にする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	userID := twoFactorAuthUser(t, tx, "regenerate_2fa_uc_off", twoFactorAuthTestSecret, false)

	uc, _ := newRegenerateTwoFactorAuthRecoveryCodesUsecaseForTest(tx)
	err := uc.Execute(ctx, RegenerateTwoFactorAuthRecoveryCodesInput{UserID: userID, Password: twoFactorAuthTestPassword})
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeConflict {
		t.Fatalf("ConflictのAppErrorを期待したが、%vだった", err)
	}
}
