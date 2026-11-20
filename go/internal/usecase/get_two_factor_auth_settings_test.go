package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetTwoFactorAuthSettingsUsecase_Execute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		secret      string
		enabled     bool
		wantEnabled bool
	}{
		{name: "有効", secret: twoFactorAuthTestSecret, enabled: true, wantEnabled: true},
		{name: "登録の途中は無効として扱う", secret: twoFactorAuthTestSecret},
		{name: "設定が無ければ無効", secret: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			userID := twoFactorAuthUser(t, tx, "settings_2fa_uc", tc.secret, tc.enabled)

			uc := NewGetTwoFactorAuthSettingsUsecase(repository.NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx)))
			output, err := uc.Execute(ctx, GetTwoFactorAuthSettingsInput{UserID: userID})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.Enabled != tc.wantEnabled {
				t.Errorf("Enabled = %v、期待値 = %v", output.Enabled, tc.wantEnabled)
			}
		})
	}
}
