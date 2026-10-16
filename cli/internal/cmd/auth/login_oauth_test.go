package auth

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestLoginWithBrowser(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	var page string
	e.browser = func(authorizationURL string) error {
		page = e.approve(authorizationURL, nil)
		return nil
	}
	e.loginWithBrowser("team")
	if !strings.Contains(page, `<html lang="ja">`) || !strings.Contains(page, "ターミナルでログインの結果を確認してください") {
		t.Errorf("日本語の結果画面を期待したが %q だった", page)
	}
	e.assertNoSecret()

	var got credentialResult
	e.decode(&got)
	if got.Space != "team" || got.Member.Atname != "alice" || got.TokenStore != "keyring" || !got.Default ||
		strings.Join(got.Scopes, " ") != "page:read page:write draft_page:read" {
		t.Errorf("出力 = %+v", got)
	}

	// 既定では、認可サーバーが扱うすべてのスコープを要求し、スペースのAPIを `resource` にする。
	request := e.oauth.authorizations[0]
	if request.Get("scope") != "page:read page:write draft_page:read" || request.Get("resource") != "https://wikino.app/api/v1/spaces/team" ||
		request.Get("client_id") != "wikino-cli" || request.Get("code_challenge_method") != "S256" {
		t.Errorf("認可要求 = %v", request)
	}
	if !strings.Contains(e.stderr.String(), "https://wikino.app/oauth/authorize?") {
		t.Errorf("標準エラーに認可要求のURLが出ていない: %q", e.stderr.String())
	}

	// 得たアクセストークンで持ち主のメンバーを確かめてから、リフレッシュトークンと期限も保存する。
	last := e.requests[len(e.requests)-1]
	if last.URL.Path != "/api/v1/spaces/team/members/me" || last.Header.Get("Authorization") != "Bearer wikino_oauth_access-1" {
		t.Errorf("最後の要求 = %s %s", last.Method, last.URL)
	}
	secret := e.savedSecret(t)
	if secret["access_token"] != "wikino_oauth_access-1" || secret["refresh_token"] != "wikino_oauth_refresh-1" ||
		secret["expires_at"] != "2026-10-14T10:00:00Z" {
		t.Errorf("保存したトークン = %v", secret)
	}
}

func TestLoginWithBrowserScopes(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.loginWithBrowser("team", "--scopes", "page:read,draft_page:read", "--scopes", "page:read")

	if got := e.oauth.authorizations[0].Get("scope"); got != "page:read draft_page:read" {
		t.Errorf("要求したスコープ = %q", got)
	}
}

func TestLoginWithBrowserEmptyScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "等号で空文字を指定した", args: []string{"--scopes="}},
		{name: "空文字を指定した", args: []string{"--scopes", ""}},
		{name: "空白だけを指定した", args: []string{"--scopes", " \t "}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newTestEnv(t)
			browserOpened := false
			e.browser = func(authorizationURL string) error {
				browserOpened = true
				// 誤って認可へ進む回帰でも、応答を返して待ち時間を発生させない。
				e.approve(authorizationURL, nil)
				return nil
			}
			err := e.run("", append([]string{"auth", "login", "team"}, tt.args...)...)
			cliErr := assertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, "スコープを1つ以上指定してください") {
				t.Errorf("メッセージ = %q", cliErr.Message)
			}
			if browserOpened || len(e.oauth.authorizations) != 0 {
				t.Error("空のスコープ指定でブラウザを開いた")
			}
			if len(e.oauth.issued) != 0 || len(e.keyring.items) != 0 || len(e.config().Credentials) != 0 {
				t.Error("空のスコープ指定でトークンを発行・保存した")
			}
		})
	}
}

func TestLoginWithBrowserUnknownScope(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	err := e.run("", "auth", "login", "team", "--scopes", "page:read,page:delete")

	cliErr := assertCLIError(t, err, output.CodeUsage, output.ExitUsage)
	if !strings.Contains(cliErr.Message, "page:delete") || !strings.Contains(cliErr.Message, "draft_page:read") {
		t.Errorf("メッセージ = %q", cliErr.Message)
	}
	if len(e.oauth.authorizations) != 0 {
		t.Error("未知のスコープなのにブラウザを開いた")
	}
}

func TestLoginWithBrowserFailureSavesNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		modify       func(url.Values)
		wantCode     string
		wantExitCode int
	}{
		{
			name:         "認可画面で拒否した",
			modify:       func(q url.Values) { q.Del("code"); q.Set("error", "access_denied") },
			wantCode:     output.CodeAuthorizationDenied,
			wantExitCode: output.ExitAuth,
		},
		{
			name:         "stateが一致しない",
			modify:       func(q url.Values) { q.Set("state", "forged") },
			wantCode:     output.CodeOAuthError,
			wantExitCode: output.ExitAuth,
		},
		{
			name:         "issが一致しない",
			modify:       func(q url.Values) { q.Set("iss", "https://evil.example") },
			wantCode:     output.CodeOAuthError,
			wantExitCode: output.ExitAuth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			e.browser = func(authorizationURL string) error {
				e.approve(authorizationURL, tt.modify)
				return nil
			}
			err := e.run("", "auth", "login", "team")

			cliErr := assertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Hint, "wikino auth login team") {
				t.Errorf("ヒント = %q", cliErr.Hint)
			}
			if len(e.oauth.issued) != 0 || len(e.keyring.items) != 0 {
				t.Error("失敗したログインでトークンを発行・保存した")
			}
			e.assertNoSecret()
		})
	}
}

func TestStatusRefreshesExpiredToken(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.loginWithBrowser("team")

	// アクセストークンの期限 (1時間) が過ぎた後に、APIを呼ぶ前に更新する。
	e.now = e.now.Add(2 * time.Hour)
	if err := e.run("", "auth", "status"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	e.assertNoSecret()

	var refreshed bool
	for _, req := range e.requests {
		if req.URL.Path == "/api/v1/spaces/team/members/me" && req.Header.Get("Authorization") == "Bearer wikino_oauth_access-2" {
			refreshed = true
		}
	}
	if !refreshed {
		t.Error("更新したアクセストークンでAPIを呼んでいない")
	}
	if secret := e.savedSecret(t); secret["refresh_token"] != "wikino_oauth_refresh-2" {
		t.Errorf("更新したトークンを保存していない: %v", secret)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.loginWithBrowser("team")

	if err := e.run("", "auth", "logout", "team"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if len(e.oauth.revoked) != 1 || e.oauth.revoked[0] != "wikino_oauth_refresh-1" {
		t.Errorf("失効させたトークン = %v", e.oauth.revoked)
	}
	if len(e.keyring.items) != 0 || len(e.config().Credentials) != 0 {
		t.Error("資格情報が残った")
	}
	e.assertNoSecret()
}

func TestLogoutRevokeFailureStillRemovesCredentials(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.loginWithBrowser("team")
	e.oauth.revokeStatus = 503

	if err := e.run("", "auth", "logout", "team"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if !strings.Contains(e.stderr.String(), "連携を解除してください") {
		t.Errorf("標準エラーに警告を期待したが %q だった", e.stderr.String())
	}
	if len(e.keyring.items) != 0 || len(e.config().Credentials) != 0 {
		t.Error("資格情報が残った")
	}
	e.assertNoSecret()
}

func TestLogoutPersonalAccessTokenDoesNotRevoke(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.login("team", aliceToken)

	if err := e.run("", "auth", "logout", "team"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	for _, req := range e.requests {
		if strings.HasPrefix(req.URL.Path, "/oauth/") || strings.HasPrefix(req.URL.Path, "/.well-known/") {
			t.Errorf("個人用アクセストークンのログアウトで認可サーバーに要求した: %s", req.URL.Path)
		}
	}
}

func TestLogoutWarnsWhenTokenCannotBeRead(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.loginWithBrowser("team")
	e.keyring.getErr = errors.New("キーチェーンの読み取りに失敗しました")

	if err := e.run("", "auth", "logout", "team"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if len(e.oauth.revoked) != 0 {
		t.Errorf("失効要求の数 = %d、期待値 = 0", len(e.oauth.revoked))
	}
	if !strings.Contains(e.stderr.String(), "連携を解除してください") {
		t.Errorf("標準エラーに未失効の警告を期待したが %q だった", e.stderr.String())
	}
	if len(e.keyring.items) != 0 || len(e.config().Credentials) != 0 {
		t.Error("資格情報が残った")
	}
	e.assertNoSecret()
}

func TestLogoutDoesNotWarnRemovedWhenDeleteFails(t *testing.T) {
	t.Parallel()
	e := newTestEnv(t)
	e.loginWithBrowser("team")
	// キーチェーンのデーモンが応答しないときなど、読み取りも削除も失敗する。
	e.keyring.getErr = errors.New("キーチェーンの読み取りに失敗しました")
	e.keyring.deleteErr = errors.New("キーチェーンから削除できません")

	err := e.run("", "auth", "logout", "team")
	_ = assertCLIError(t, err, output.CodeUnexpected, output.ExitError)
	// 資格情報を消せていないため、「手元の資格情報は消しました」とは伝えない。
	if strings.Contains(e.stderr.String(), "連携を解除してください") {
		t.Errorf("削除に失敗したのに未失効の警告を出した: %q", e.stderr.String())
	}
	if len(e.config().Credentials) != 1 {
		t.Error("削除を再試行できるよう、設定に資格情報を残すことを期待した")
	}
}

func TestLoginWithBrowserRevokesTokenWhenVerificationFails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		lang        string
		received    string
		checkResult string
		loggedIn    string
	}{
		{name: "日本語", lang: i18n.LangJa, received: "認可の応答を受け取りました", checkResult: "ターミナルでログインの結果を確認してください", loggedIn: "ログインしました"},
		{name: "英語", lang: i18n.LangEn, received: "The authorization response was received", checkResult: "check the login result in the terminal", loggedIn: "You are logged in"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newTestEnv(t)
			e.vars["WIKINO_LANG"] = tt.lang
			var page string
			e.browser = func(authorizationURL string) error {
				page = e.approve(authorizationURL, nil)
				return nil
			}
			// 許可で得たトークンの持ち主はteamのメンバーのため、otherのAPIでは404になる。
			err := e.run("", "auth", "login", "other")

			_ = assertCLIError(t, err, output.CodeNotFound, output.ExitNotFound)
			if !strings.Contains(page, `<html lang="`+tt.lang+`">`) || !strings.Contains(page, tt.received) || !strings.Contains(page, tt.checkResult) || strings.Contains(page, tt.loggedIn) {
				t.Errorf("結果画面は応答の受領とターミナルでの結果確認を案内することを期待したが %q だった", page)
			}
			// 保存しないトークンを失効させ、使えない連携をサーバーに残さない。
			if len(e.oauth.revoked) != 1 || e.oauth.revoked[0] != "wikino_oauth_refresh-1" {
				t.Errorf("失効したトークン = %v、期待値 = [wikino_oauth_refresh-1]", e.oauth.revoked)
			}
			if len(e.keyring.items) != 0 || len(e.config().Credentials) != 0 {
				t.Error("失敗したログインで資格情報を保存した")
			}
			e.assertNoSecret()
		})
	}
}
