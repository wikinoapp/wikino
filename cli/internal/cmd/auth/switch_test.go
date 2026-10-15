package auth

import (
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestSwitch(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.login("team", botToken)
	e.login("solo", carolToken)

	if err := e.run("", "auth", "switch", "Solo"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	var got credentialResult
	e.decode(&got)
	if got.Space != "solo" || got.Member.Atname != "carol" || !got.Default {
		t.Errorf("出力 = %+v", got)
	}
	if cfg := e.config(); cfg.Default != (config.Default{Host: config.DefaultHost, Space: "solo"}) {
		t.Errorf("既定の組 = %+v", cfg.Default)
	}

	// --memberでスペースの既定のメンバーも切り替える。
	if err := e.run("", "auth", "switch", "team", "--member", "BOT"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	cfg := e.config()
	if cfg.Default.Space != "team" || cfg.Credentials[0].SpaceDefault || !cfg.Credentials[1].SpaceDefault {
		t.Errorf("設定 = %+v", cfg)
	}

	// 切り替えた既定は、指定の無いコマンドの解決に使われる。
	if err := e.run("", "auth", "logout"); err != nil {
		t.Fatal(err)
	}
	e.decode(&got)
	if got.Member.Atname != "bot" {
		t.Errorf("既定のメンバーで解決されなかった: %+v", got)
	}
}

func TestSwitchErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantCode     string
		wantExitCode int
		wantHint     string
	}{
		{name: "資格情報の無いスペース", args: []string{"auth", "switch", "unknown"}, wantCode: output.CodeNotLoggedIn, wantExitCode: output.ExitAuth},
		{name: "資格情報の無いメンバー", args: []string{"auth", "switch", "team", "--member", "carol"}, wantCode: output.CodeNotLoggedIn, wantExitCode: output.ExitAuth, wantHint: "alice, bot"},
		{name: "スペースが無い", args: []string{"auth", "switch"}, wantCode: output.CodeUsage, wantExitCode: output.ExitUsage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			e.login("team", aliceToken)
			e.login("team", botToken)

			cliErr := assertCLIError(t, e.run("", tt.args...), tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒントに %q を含むことを期待したが %q だった", tt.wantHint, cliErr.Hint)
			}
		})
	}
}
