package auth

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestLoginWithToken(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)

	if err := e.run(aliceToken+"\n", "auth", "login", "team", "--with-token"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	e.assertNoSecret()

	var got credentialResult
	e.decode(&got)
	if got.Host != config.DefaultHost || got.Space != "team" || got.Member.Atname != "alice" ||
		got.Member.Name == nil || *got.Member.Name != "Alice" || got.TokenStore != "keyring" || !got.Default || !got.SpaceDefault {
		t.Errorf("出力 = %+v", got)
	}

	if len(e.requests) != 1 {
		t.Fatalf("要求の数 = %d、期待値 = 1", len(e.requests))
	}
	req := e.requests[0]
	if req.Method != "GET" || req.URL.Host != "wikino.app" || req.URL.Path != "/api/v1/spaces/team/members/me" {
		t.Errorf("要求 = %s %s", req.Method, req.URL)
	}
	if req.Header.Get("Authorization") != "Bearer "+aliceToken {
		t.Error("標準入力のトークンで要求していない")
	}

	cfg := e.config()
	if cfg.Default != (config.Default{Host: config.DefaultHost, Space: "team"}) || len(cfg.Credentials) != 1 {
		t.Fatalf("設定 = %+v", cfg)
	}
	if cred := cfg.Credentials[0]; cred.MemberID != testOwners[aliceToken].memberID || cred.Store != "keyring" {
		t.Errorf("資格情報 = %+v", cred)
	}
	if len(e.keyring.items) != 1 {
		t.Fatalf("キーチェーンの項目の数 = %d、期待値 = 1", len(e.keyring.items))
	}
	for _, v := range e.keyring.items {
		if !strings.Contains(v, aliceToken) {
			t.Errorf("キーチェーンにトークンが保存されていない: %q", v)
		}
	}
}

func TestLoginFallsBackToFile(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.keyring.err = errors.New("dbus: no session bus")

	if err := e.run(aliceToken, "auth", "login", "team", "--with-token"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	e.assertNoSecret()

	var got credentialResult
	e.decode(&got)
	if got.TokenStore != "file" {
		t.Errorf("保存先 = %q、期待値 = file", got.TokenStore)
	}
	if !strings.Contains(e.stderr.String(), "キーチェーンを使えない") {
		t.Errorf("標準エラーに警告を期待したが %q だった", e.stderr.String())
	}
	if e.config().Credentials[0].Store != "file" {
		t.Error("設定に保存先が記録されていない")
	}
}

func TestLoginAddsMembersAndReplaces(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.login("TEAM", botToken)
	e.login("solo", carolToken)

	cfg := e.config()
	if cfg.Default != (config.Default{Host: config.DefaultHost, Space: "team"}) {
		t.Errorf("2回目以降のログインで既定が変わった: %+v", cfg.Default)
	}
	if len(cfg.Credentials) != 3 || cfg.Credentials[1].SpaceDefault || !cfg.Credentials[2].SpaceDefault {
		t.Fatalf("資格情報 = %+v", cfg.Credentials)
	}

	// 同じメンバーで再びログインすると、保存先が変わっても古いトークンを残さず置き換える。
	e.vars["WIKINO_TOKEN_STORE"] = "file"
	e.login("team", aliceToken)
	cfg = e.config()
	if len(cfg.Credentials) != 3 || cfg.Credentials[0].Store != "file" || !cfg.Credentials[0].SpaceDefault {
		t.Errorf("置き換えた資格情報 = %+v", cfg.Credentials[0])
	}
	for _, v := range e.keyring.items {
		if strings.Contains(v, aliceToken) {
			t.Error("古い保存先のキーチェーンにトークンが残った")
		}
	}
}

func TestLoginValidationFailureSavesNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		space        string
		token        string
		wantCode     string
		wantExitCode int
		wantHint     string
	}{
		{name: "無効なトークン (401)", space: "team", token: "wikino_pat_invalid", wantCode: output.CodeUnauthorized, wantExitCode: output.ExitAuth, wantHint: "スペース設定で発行し直せます"},
		{name: "別のスペースのトークン (404)", space: "solo", token: aliceToken, wantCode: output.CodeNotFound, wantExitCode: output.ExitNotFound, wantHint: "トークンを発行したスペースの識別子"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			err := e.run(tt.token, "auth", "login", tt.space, "--with-token")

			cliErr := assertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒントに %q を含むことを期待したが %q だった", tt.wantHint, cliErr.Hint)
			}
			if strings.Contains(cliErr.Message+cliErr.Hint, tt.token) {
				t.Error("エラーにトークンが含まれる")
			}
			if len(e.keyring.items) != 0 {
				t.Error("検証に失敗したのにキーチェーンに保存した")
			}
			if _, err := os.Stat(e.vars["XDG_CONFIG_HOME"] + "/wikino"); !errors.Is(err, os.ErrNotExist) {
				t.Error("検証に失敗したのに設定のディレクトリを作った")
			}
		})
	}
}

func TestLoginUsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		stdin string
		args  []string
	}{
		{name: "--with-tokenと--scopesを併用した", stdin: aliceToken, args: []string{"auth", "login", "team", "--with-token", "--scopes", "page:read"}},
		{name: "--memberを指定した", stdin: aliceToken, args: []string{"auth", "login", "team", "--with-token", "--member", "alice"}},
		{name: "スペースが無い", stdin: aliceToken, args: []string{"auth", "login", "--with-token"}},
		{name: "引数と--spaceが違う", stdin: aliceToken, args: []string{"auth", "login", "team", "--space", "solo", "--with-token"}},
		{name: "引数が多い", stdin: aliceToken, args: []string{"auth", "login", "team", "solo", "--with-token"}},
		{name: "標準入力が空", stdin: "\n", args: []string{"auth", "login", "team", "--with-token"}},
		{name: "トークンが複数行", stdin: aliceToken + "\n" + botToken, args: []string{"auth", "login", "team", "--with-token"}},
		{name: "スペースの識別子にパスの文字を含む", stdin: aliceToken, args: []string{"auth", "login", "team/../x", "--with-token"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			err := e.run(tt.stdin, tt.args...)

			_ = assertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if len(e.requests) != 0 {
				t.Error("使い方の誤りなのに要求を送った")
			}
			e.assertNoSecret()
		})
	}
}
