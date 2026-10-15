package auth

import (
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestLogout(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.login("team", botToken)

	if err := e.run("", "auth", "logout", "team", "--member", "bot"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	e.assertNoSecret()

	var got credentialResult
	e.decode(&got)
	if got.Space != "team" || got.Member.Atname != "bot" || got.Member.Name != nil {
		t.Errorf("出力 = %+v", got)
	}
	cfg := e.config()
	if len(cfg.Credentials) != 1 || cfg.Credentials[0].MemberAtname != "alice" || cfg.Default.Space != "team" {
		t.Errorf("既定でない資格情報を消した後の設定 = %+v", cfg)
	}
	if len(e.keyring.items) != 1 {
		t.Errorf("キーチェーンの項目の数 = %d、期待値 = 1", len(e.keyring.items))
	}
	if len(e.requests) != 2 {
		// ログインの2回だけで、ログアウトはサーバーに要求を送らない (個人アクセストークンは失効させない)。
		t.Errorf("要求の数 = %d、期待値 = 2", len(e.requests))
	}

	// 既定の組を消すと既定を外す。WIKINO_TOKENがあっても保存した資格情報を対象にする。
	e.vars["WIKINO_TOKEN"] = carolToken
	if err := e.run("", "auth", "logout"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	cfg = e.config()
	if len(cfg.Credentials) != 0 || cfg.Default != (config.Default{}) {
		t.Errorf("既定の組を消した後の設定 = %+v", cfg)
	}
	if len(e.keyring.items) != 0 {
		t.Errorf("キーチェーンに項目が残った: %v", e.keyring.items)
	}
}

func TestLogoutErrors(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.login("team", botToken)
	if err := e.run("", "auth", "switch", "team", "--member", "bot"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		args         []string
		wantCode     string
		wantExitCode int
	}{
		{name: "資格情報の無いスペース", args: []string{"auth", "logout", "solo"}, wantCode: output.CodeNotLoggedIn, wantExitCode: output.ExitAuth},
		{name: "資格情報の無いメンバー", args: []string{"auth", "logout", "team", "--member", "carol"}, wantCode: output.CodeNotLoggedIn, wantExitCode: output.ExitAuth},
		{name: "引数と--spaceが違う", args: []string{"auth", "logout", "team", "--space", "solo"}, wantCode: output.CodeUsage, wantExitCode: output.ExitUsage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.run("", tt.args...)
			_ = assertCLIError(t, err, tt.wantCode, tt.wantExitCode)
		})
	}

	if got := len(e.config().Credentials); got != 2 {
		t.Errorf("失敗したログアウトで資格情報が減った: %d", got)
	}
}
