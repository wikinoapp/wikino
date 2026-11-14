package usecase

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

const (
	twoFactorAuthTestPassword = "password123"
	twoFactorAuthTestSecret   = "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
)

// twoFactorAuthUserは、パスワードと、secretの二要素認証の設定 (enabledで有効か) を持つユーザーを作る。
// secretが空なら二要素認証の設定を作らない。
func twoFactorAuthUser(t *testing.T, tx *sql.Tx, atname, secret string, enabled bool) model.UserID {
	t.Helper()

	digest, err := auth.HashPassword(twoFactorAuthTestPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	builder := testutil.NewUserBuilder(t, tx).WithEmail(atname + "@example.com").WithAtname(atname)
	if secret == "" {
		return builder.BuildWithPassword(digest)
	}
	return builder.BuildWithPasswordAndTwoFactorAuth(digest, secret, enabled)
}

func currentTOTPCode(t *testing.T, secret string) string {
	t.Helper()

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("コードの生成に失敗: %v", err)
	}
	return code
}

func newEnableTwoFactorAuthUsecaseForTest(tx *sql.Tx) (*EnableTwoFactorAuthUsecase, *repository.UserTwoFactorAuthRepository) {
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewUserTwoFactorAuthRepository(q)
	return NewEnableTwoFactorAuthUsecase(
		validator.NewSettingsTwoFactorAuthCreateValidator(repository.NewUserPasswordRepository(q)),
		repo,
	), repo
}

func TestEnableTwoFactorAuthUsecase_Execute_有効にしてログインでコードが通る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	userID := twoFactorAuthUser(t, tx, "enable_2fa_uc_ok", "", false)

	// 登録の画面を開いたときと同じく、新しい秘密鍵で登録の途中の設定を作る
	q := testutil.QueriesWithTx(tx)
	prepared, err := NewPrepareTwoFactorAuthUsecase(repository.NewUserTwoFactorAuthRepository(q)).Execute(ctx, PrepareTwoFactorAuthInput{UserID: userID, Email: "enable_2fa_uc_ok@example.com"})
	if err != nil || prepared.Setup == nil {
		t.Fatalf("登録の準備に失敗: 結果 = %+v、エラー = %v", prepared, err)
	}

	uc, repo := newEnableTwoFactorAuthUsecaseForTest(tx)
	if err := uc.Execute(ctx, EnableTwoFactorAuthInput{
		UserID:   userID,
		Password: twoFactorAuthTestPassword,
		// 貼り付けで混じった空白は取り除く
		TOTPCode: " " + currentTOTPCode(t, prepared.Setup.Secret) + " ",
	}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	twoFactorAuth, err := repo.FindByUserID(ctx, userID)
	if err != nil || twoFactorAuth == nil {
		t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", twoFactorAuth, err)
	}
	if !twoFactorAuth.Enabled || twoFactorAuth.EnabledAt == nil {
		t.Errorf("設定 = %+v、有効になっていることを期待", twoFactorAuth)
	}
	if twoFactorAuth.Secret != prepared.Setup.Secret {
		t.Errorf("秘密鍵 = %q、登録の画面で出した秘密鍵 = %q", twoFactorAuth.Secret, prepared.Setup.Secret)
	}
	format := regexp.MustCompile(`^[a-z0-9]{8}$`)
	if len(twoFactorAuth.RecoveryCodes) != auth.RecoveryCodeCount {
		t.Errorf("リカバリーコードの個数 = %d、期待値 = %d", len(twoFactorAuth.RecoveryCodes), auth.RecoveryCodeCount)
	}
	for _, code := range twoFactorAuth.RecoveryCodes {
		if !format.MatchString(code) {
			t.Errorf("リカバリーコード = %q、8文字の小文字英数字を期待", code)
		}
	}

	// Go版のログインの二要素認証で、同じ秘密鍵のコードが通る
	signInValidator := validator.NewSignInTwoFactorCreateValidator(repo)
	if err := signInValidator.Validate(ctx, validator.SignInTwoFactorCreateValidatorInput{
		UserID:   userID,
		TOTPCode: currentTOTPCode(t, prepared.Setup.Secret),
	}); err != nil {
		t.Errorf("ログインの二要素認証のエラー = %v", err)
	}
}

func TestEnableTwoFactorAuthUsecase_Execute_Rails版が作った登録の途中の設定から有効にする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	// Rails版の登録の画面は、秘密鍵を持つenabled = falseの行を作る
	userID := twoFactorAuthUser(t, tx, "enable_2fa_uc_rails", twoFactorAuthTestSecret, false)

	uc, repo := newEnableTwoFactorAuthUsecaseForTest(tx)
	if err := uc.Execute(ctx, EnableTwoFactorAuthInput{
		UserID:   userID,
		Password: twoFactorAuthTestPassword,
		TOTPCode: currentTOTPCode(t, twoFactorAuthTestSecret),
	}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	twoFactorAuth, err := repo.FindByUserID(ctx, userID)
	if err != nil || twoFactorAuth == nil {
		t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", twoFactorAuth, err)
	}
	if !twoFactorAuth.Enabled || twoFactorAuth.Secret != twoFactorAuthTestSecret {
		t.Errorf("設定 = %+v、Rails版の秘密鍵のまま有効になっていることを期待", twoFactorAuth)
	}
}

func TestEnableTwoFactorAuthUsecase_Execute_誤りでは有効にせず秘密鍵を保つ(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		password  string
		totpCode  func(t *testing.T) string
		wantField string
	}{
		{
			name:      "コードが違う",
			password:  twoFactorAuthTestPassword,
			totpCode:  func(t *testing.T) string { return wrongTOTPCode(t) },
			wantField: "totp_code",
		},
		{
			name:      "パスワードが違う",
			password:  "wrong-password",
			totpCode:  func(t *testing.T) string { return currentTOTPCode(t, twoFactorAuthTestSecret) },
			wantField: "password",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			userID := twoFactorAuthUser(t, tx, "enable_2fa_uc_wrong", twoFactorAuthTestSecret, false)

			uc, repo := newEnableTwoFactorAuthUsecaseForTest(tx)
			err := uc.Execute(ctx, EnableTwoFactorAuthInput{
				UserID:   userID,
				Password: tc.password,
				TOTPCode: tc.totpCode(t),
			})
			ve := model.AsValidationError(err)
			if ve == nil || !ve.HasFieldError(tc.wantField) {
				t.Fatalf("%sのValidationErrorを期待したが、%vだった", tc.wantField, err)
			}

			twoFactorAuth, err := repo.FindByUserID(ctx, userID)
			if err != nil || twoFactorAuth == nil {
				t.Fatalf("設定の取得に失敗: 設定 = %v、エラー = %v", twoFactorAuth, err)
			}
			if twoFactorAuth.Enabled {
				t.Error("誤りがあったのに有効になった")
			}
			if twoFactorAuth.Secret != twoFactorAuthTestSecret {
				t.Errorf("秘密鍵 = %q、変わらないことを期待", twoFactorAuth.Secret)
			}
		})
	}
}

// wrongTOTPCodeは、今の正しいコードと異なる6桁の数字を返す
func wrongTOTPCode(t *testing.T) string {
	t.Helper()

	// 許容幅の中のどのタイムステップのコードとも重ならない値にする
	now := time.Now()
	for _, c := range []string{"000000", "111111", "222222", "333333"} {
		if !auth.ValidateTOTPCode(c, twoFactorAuthTestSecret, now) {
			return c
		}
	}
	t.Fatal("誤りのコードを用意できなかった")
	return ""
}

func TestEnableTwoFactorAuthUsecase_Execute_登録の途中の設定が無ければ競合にする(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		secret  string
		enabled bool
	}{
		{name: "既に有効", secret: twoFactorAuthTestSecret, enabled: true},
		{name: "設定が無い", secret: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			userID := twoFactorAuthUser(t, tx, "enable_2fa_uc_conflict", tc.secret, tc.enabled)

			uc, _ := newEnableTwoFactorAuthUsecaseForTest(tx)
			err := uc.Execute(ctx, EnableTwoFactorAuthInput{
				UserID:   userID,
				Password: twoFactorAuthTestPassword,
				TOTPCode: currentTOTPCode(t, twoFactorAuthTestSecret),
			})
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != model.AppErrCodeConflict {
				t.Fatalf("ConflictのAppErrorを期待したが、%vだった", err)
			}
		})
	}
}
