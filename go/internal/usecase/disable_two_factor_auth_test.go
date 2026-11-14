package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newDisableTwoFactorAuthUsecaseForTest(tx *sql.Tx) (*DisableTwoFactorAuthUsecase, *repository.UserTwoFactorAuthRepository) {
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewUserTwoFactorAuthRepository(q)
	return NewDisableTwoFactorAuthUsecase(
		validator.NewSettingsTwoFactorAuthDeleteValidator(repository.NewUserPasswordRepository(q)),
		repo,
	), repo
}

func TestDisableTwoFactorAuthUsecase_Execute_無効にするとログインでコードを求めない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	userID := twoFactorAuthUser(t, tx, "disable_2fa_uc_ok", twoFactorAuthTestSecret, true)

	uc, repo := newDisableTwoFactorAuthUsecaseForTest(tx)
	if err := uc.Execute(ctx, DisableTwoFactorAuthInput{UserID: userID, Password: twoFactorAuthTestPassword}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	twoFactorAuth, err := repo.FindByUserID(ctx, userID)
	if err != nil || twoFactorAuth == nil {
		t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", twoFactorAuth, err)
	}
	if twoFactorAuth.Enabled || twoFactorAuth.EnabledAt != nil || len(twoFactorAuth.RecoveryCodes) != 0 {
		t.Errorf("設定 = %+v、無効でリカバリーコードの無い設定を期待", twoFactorAuth)
	}

	// Go版のログインは、有効な二要素認証の設定があるときだけコードを求める
	q := testutil.QueriesWithTx(tx)
	signInValidator := validator.NewSignInCreateValidator(repository.NewUserRepository(q), repository.NewUserPasswordRepository(q), repo)
	output, err := signInValidator.Validate(ctx, validator.SignInCreateValidatorInput{
		Email:    "disable_2fa_uc_ok@example.com",
		Password: twoFactorAuthTestPassword,
	})
	if err != nil {
		t.Fatalf("ログインの検証のエラー = %v", err)
	}
	if output.UserTwoFactorAuth != nil {
		t.Errorf("二要素認証の設定 = %+v、ログインでコードを求めないことを期待", output.UserTwoFactorAuth)
	}
}

func TestDisableTwoFactorAuthUsecase_Execute_パスワードが違うと無効にしない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	userID := twoFactorAuthUser(t, tx, "disable_2fa_uc_wrong", twoFactorAuthTestSecret, true)

	uc, repo := newDisableTwoFactorAuthUsecaseForTest(tx)
	err := uc.Execute(ctx, DisableTwoFactorAuthInput{UserID: userID, Password: "wrong-password"})
	if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("password") {
		t.Fatalf("passwordのValidationErrorを期待したが、%vだった", err)
	}

	twoFactorAuth, err := repo.FindByUserID(ctx, userID)
	if err != nil || twoFactorAuth == nil || !twoFactorAuth.Enabled {
		t.Errorf("設定 = %+v、有効なままを期待 (エラー = %v)", twoFactorAuth, err)
	}
}

func TestDisableTwoFactorAuthUsecase_Execute_有効でなければ競合にする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	userID := twoFactorAuthUser(t, tx, "disable_2fa_uc_off", twoFactorAuthTestSecret, false)

	uc, _ := newDisableTwoFactorAuthUsecaseForTest(tx)
	err := uc.Execute(ctx, DisableTwoFactorAuthInput{UserID: userID, Password: twoFactorAuthTestPassword})
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeConflict {
		t.Fatalf("ConflictのAppErrorを期待したが、%vだった", err)
	}
}
