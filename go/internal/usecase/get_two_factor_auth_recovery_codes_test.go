package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetTwoFactorAuthRecoveryCodesUsecase_Execute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		secret      string
		enabled     bool
		wantEnabled bool
	}{
		{name: "有効なら返す", secret: twoFactorAuthTestSecret, enabled: true, wantEnabled: true},
		{name: "登録の途中なら返さない", secret: twoFactorAuthTestSecret},
		{name: "設定が無ければ返さない", secret: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			userID := twoFactorAuthUser(t, tx, "recovery_codes_2fa_uc", tc.secret, tc.enabled)
			if tc.enabled {
				if _, err := tx.ExecContext(ctx, "UPDATE user_two_factor_auths SET recovery_codes = '{abcd1234,efgh5678}' WHERE user_id = $1", string(userID)); err != nil {
					t.Fatalf("リカバリーコードの設定に失敗: %v", err)
				}
			}

			uc := NewGetTwoFactorAuthRecoveryCodesUsecase(repository.NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx)))
			output, err := uc.Execute(ctx, GetTwoFactorAuthRecoveryCodesInput{UserID: userID})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.Enabled != tc.wantEnabled {
				t.Errorf("Enabled = %v、期待値 = %v", output.Enabled, tc.wantEnabled)
			}
			if tc.wantEnabled && (len(output.RecoveryCodes) != 2 || output.RecoveryCodes[0] != "abcd1234") {
				t.Errorf("RecoveryCodes = %v、期待値 = [abcd1234 efgh5678]", output.RecoveryCodes)
			}
			if !tc.wantEnabled && len(output.RecoveryCodes) != 0 {
				t.Errorf("RecoveryCodes = %v、空を期待", output.RecoveryCodes)
			}
		})
	}
}
