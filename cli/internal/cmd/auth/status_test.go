package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

type statusOutput struct {
	Items []struct {
		credentialResult
		Token *struct {
			Kind   string   `json:"kind"`
			Scopes []string `json:"scopes"`
		} `json:"token"`
		Error *struct {
			Code   string `json:"code"`
			Status int    `json:"status"`
		} `json:"error"`
	} `json:"items"`
}

func TestStatus(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.login("solo", carolToken)

	if err := e.run("", "auth", "status"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	e.assertNoSecret()

	var got statusOutput
	e.decode(&got)
	if len(got.Items) != 2 {
		t.Fatalf("項目の数 = %d、期待値 = 2", len(got.Items))
	}
	alice, carol := got.Items[0], got.Items[1]
	if alice.Space != "team" || alice.Member.Atname != "alice" || !alice.Default || alice.TokenStore != "keyring" {
		t.Errorf("1件目 = %+v", alice.credentialResult)
	}
	if alice.Token == nil || alice.Token.Kind != "personal_access_token" || len(alice.Token.Scopes) != 2 || alice.Error != nil {
		t.Errorf("1件目のトークン = %+v、エラー = %+v", alice.Token, alice.Error)
	}
	if carol.Space != "solo" || carol.Default || !carol.SpaceDefault {
		t.Errorf("2件目 = %+v", carol.credentialResult)
	}
}

func TestStatusFilters(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.login("team", botToken)
	e.login("solo", carolToken)

	if err := e.run("", "auth", "status", "--space", "TEAM", "--member", "Bot"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got statusOutput
	e.decode(&got)
	if len(got.Items) != 1 || got.Items[0].Member.Atname != "bot" {
		t.Errorf("絞り込んだ項目 = %+v", got.Items)
	}
}

func TestStatusReportsFailedCredential(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.login("solo", carolToken)
	// サーバーでトークンを削除した状態にする。
	delete(e.owners, carolToken)

	err := e.run("", "auth", "status")

	// 失敗した項目があれば、全件を出したうえで終了コードで失敗を知らせ、標準エラーには何も出さない。
	if output.ExitCodeOf(err) != output.ExitAuth {
		t.Errorf("終了コード = %d、期待値 = %d", output.ExitCodeOf(err), output.ExitAuth)
	}
	var buf strings.Builder
	if werr := output.WriteError(&buf, err, true); werr != nil || buf.Len() != 0 {
		t.Errorf("標準エラーに出すものが無いことを期待したが %q だった", buf.String())
	}

	var got statusOutput
	e.decode(&got)
	if len(got.Items) != 2 || got.Items[0].Error != nil {
		t.Fatalf("項目 = %+v", got.Items)
	}
	if failed := got.Items[1]; failed.Error == nil || failed.Error.Code != output.CodeUnauthorized || failed.Error.Status != 401 || failed.Token != nil {
		t.Errorf("失敗した項目 = %+v", failed)
	}
}

func TestStatusUpdatesAtname(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)

	owner := e.owners[aliceToken]
	owner.atname = "alice-renamed"
	e.owners[aliceToken] = owner

	if err := e.run("", "auth", "status"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if got := e.config().Credentials[0].MemberAtname; got != "alice-renamed" {
		t.Errorf("保存したアットネーム = %q、期待値 = alice-renamed", got)
	}

	// 更新したアットネームで `--member` を照合できる。
	if err := e.run("", "auth", "switch", "team", "--member", "alice-renamed"); err != nil {
		t.Errorf("更新したアットネームで照合できない: %v", err)
	}
}

func TestStatusWithEnvToken(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.vars["WIKINO_TOKEN"] = botToken
	e.vars["WIKINO_SPACE"] = "team"

	if err := e.run("", "auth", "status"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	e.assertNoSecret()

	var got statusOutput
	e.decode(&got)
	if len(got.Items) != 1 {
		t.Fatalf("項目の数 = %d、期待値 = 1", len(got.Items))
	}
	if item := got.Items[0]; item.TokenStore != "env" || item.Member.Atname != "bot" || item.Token == nil || item.Default {
		t.Errorf("項目 = %+v", item)
	}
}

func TestStatusErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		vars         map[string]string
		args         []string
		wantCode     string
		wantExitCode int
	}{
		{name: "ログインしていない", args: []string{"auth", "status"}, wantCode: output.CodeNotLoggedIn, wantExitCode: output.ExitAuth},
		{name: "WIKINO_TOKENと--memberの併用", vars: map[string]string{"WIKINO_TOKEN": botToken, "WIKINO_SPACE": "team"}, args: []string{"auth", "status", "--member", "bot"}, wantCode: output.CodeUsage, wantExitCode: output.ExitUsage},
		{name: "WIKINO_TOKENでスペースの指定が無い", vars: map[string]string{"WIKINO_TOKEN": botToken}, args: []string{"auth", "status"}, wantCode: output.CodeUsage, wantExitCode: output.ExitUsage},
		{name: "引数を渡した", args: []string{"auth", "status", "team"}, wantCode: output.CodeUsage, wantExitCode: output.ExitUsage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			for k, v := range tt.vars {
				e.vars[k] = v
			}

			err := e.run("", tt.args...)

			cliErr := assertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if strings.Contains(cliErr.Message+cliErr.Hint, botToken) {
				t.Error("エラーにトークンが含まれる")
			}
			if len(e.requests) != 0 {
				t.Error("要求を送った")
			}
		})
	}
}

func TestStatusHuman(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)

	if err := e.runTTY("", "auth", "status"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	for _, want := range []string{"接続先:", "https://wikino.app", "メンバー:", "キーチェーン", "@alice (Alice)", "既定の組", "スコープ:", "page:read draft_page:read"} {
		if !strings.Contains(e.stdout.String(), want) {
			t.Errorf("表示に %q を含むことを期待したが、次の内容だった:\n%s", want, e.stdout.String())
		}
	}
}

func TestStatusAtnameUpdatePreservesConcurrentChanges(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.login("team", aliceToken)
	e.login("solo", carolToken)
	f := e.factory()
	dir, err := f.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	old := e.config().Credentials[0]
	// APIから名前を取得している間に、同じ資格情報がログアウトされて別のログインと既定変更があった状態。
	err = config.WithLocked(context.Background(), dir, func(cfg *config.Config) error {
		cfg.RemoveCredential(&old)
		cfg.AddCredential(config.Credential{Host: config.DefaultHost, Space: "team", MemberID: "new-member", MemberAtname: "new", Store: "file"})
		cfg.SetDefault(&cfg.Credentials[0])
		return cfg.Save(dir)
	})
	if err != nil {
		t.Fatal(err)
	}
	before := e.config()
	if err := updateAtnames(context.Background(), dir, []atnameUpdate{{credential: old, atname: "alice-renamed"}}); err != nil {
		t.Fatal(err)
	}
	after := e.config()
	if after.Default != before.Default || len(after.Credentials) != 2 {
		t.Fatalf("並行した更新が失われた: %+v", after)
	}
	for i := range before.Credentials {
		if after.Credentials[i] != before.Credentials[i] {
			t.Fatalf("資格情報が上書きされた: %+v", after.Credentials)
		}
	}
}

func TestStatusAtnameUpdatePreservesNewerName(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.login("team", aliceToken)
	f := e.factory()
	dir, err := f.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	old := e.config().Credentials[0]
	err = config.WithLocked(context.Background(), dir, func(cfg *config.Config) error {
		cfg.Credentials[0].MemberAtname = "newer-name"
		return cfg.Save(dir)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := updateAtnames(context.Background(), dir, []atnameUpdate{{credential: old, atname: "older-api-name"}}); err != nil {
		t.Fatal(err)
	}
	if e.config().Credentials[0].MemberAtname != "newer-name" {
		t.Fatal("古いAPI結果で新しい名前を上書きした")
	}
}
