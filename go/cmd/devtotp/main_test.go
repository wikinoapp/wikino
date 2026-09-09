package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.SetupTestMain(m))
}

func TestEnsureNotProduction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{name: "開発環境では実行できる", env: "dev", wantErr: false},
		{name: "テスト環境では実行できる", env: "test", wantErr: false},
		{name: "本番環境では実行できない", env: "prod", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureNotProduction(&config.Config{Env: tt.env})

			if tt.wantErr {
				if err == nil {
					t.Error("エラーが返ることを期待したがnilだった")
				}
				return
			}
			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
			}
		})
	}
}

func TestCodeForUser(t *testing.T) {
	t.Parallel()

	t.Run("正常系: 検証側が受け付けるコードを生成する", func(t *testing.T) {
		t.Parallel()
		db, tx := testutil.SetupTx(t)

		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Wikino",
			AccountName: "devtotp-enabled@example.com",
		})
		if err != nil {
			t.Fatalf("シークレット生成に失敗: %v", err)
		}

		testutil.NewUserBuilder(t, tx).
			WithEmail("devtotp-enabled@example.com").
			WithAtname("devtotpenabled").
			BuildWithTwoFactorAuth(key.Secret(), true)

		queries := query.New(db).WithTx(tx)

		code, err := codeForUser(context.Background(), queries, "devtotp-enabled@example.com", time.Now())
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}

		if !totp.Validate(code, key.Secret()) {
			t.Errorf("生成したコード %q が検証を通らなかった", code)
		}
	})

	t.Run("異常系: ユーザーが存在しない", func(t *testing.T) {
		t.Parallel()
		db, tx := testutil.SetupTx(t)

		queries := query.New(db).WithTx(tx)

		if _, err := codeForUser(context.Background(), queries, "devtotp-missing@example.com", time.Now()); err == nil {
			t.Error("エラーが返ることを期待したがnilだった")
		}
	})

	t.Run("異常系: 二要素認証が有効になっていない", func(t *testing.T) {
		t.Parallel()
		db, tx := testutil.SetupTx(t)

		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Wikino",
			AccountName: "devtotp-disabled@example.com",
		})
		if err != nil {
			t.Fatalf("シークレット生成に失敗: %v", err)
		}

		testutil.NewUserBuilder(t, tx).
			WithEmail("devtotp-disabled@example.com").
			WithAtname("devtotpdisabled").
			BuildWithTwoFactorAuth(key.Secret(), false)

		queries := query.New(db).WithTx(tx)

		if _, err := codeForUser(context.Background(), queries, "devtotp-disabled@example.com", time.Now()); err == nil {
			t.Error("エラーが返ることを期待したがnilだった")
		}
	})
}
