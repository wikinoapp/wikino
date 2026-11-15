package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPendingTwoFactorAuthUsecase_Execute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		secret    string
		enabled   bool
		wantSetup bool
	}{
		{name: "登録の途中の設定は秘密鍵を返す", secret: twoFactorAuthTestSecret, wantSetup: true},
		{name: "有効な設定は返さない", secret: twoFactorAuthTestSecret, enabled: true},
		{name: "設定が無ければ返さない", secret: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			userID := twoFactorAuthUser(t, tx, "pending_2fa_uc", tc.secret, tc.enabled)

			uc := NewGetPendingTwoFactorAuthUsecase(repository.NewUserTwoFactorAuthRepository(testutil.QueriesWithTx(tx)))
			output, err := uc.Execute(ctx, GetPendingTwoFactorAuthInput{UserID: userID, Email: "pending_2fa_uc@example.com"})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if !tc.wantSetup {
				if output.Setup != nil {
					t.Errorf("Setup = %+v、nilを期待", output.Setup)
				}
				return
			}
			if output.Setup == nil || output.Setup.Secret != twoFactorAuthTestSecret || output.Setup.OTPAuthURL == "" {
				t.Errorf("Setup = %+v、登録の途中の秘密鍵を期待", output.Setup)
			}
		})
	}
}
