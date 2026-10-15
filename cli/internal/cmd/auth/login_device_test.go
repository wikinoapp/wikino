package auth

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/output"
)

// assertDeviceLoginは、デバイス認可でteamにログインし、確認用のURLとコードを標準エラーに出したことを確かめる。
func (e *testEnv) assertDeviceLogin(t *testing.T) {
	t.Helper()

	var got credentialResult
	e.decode(&got)
	if got.Space != "team" || got.Member.Atname != "alice" || strings.Join(got.Scopes, " ") != "page:read page:write draft_page:read" {
		t.Errorf("出力 = %+v", got)
	}
	if secret := e.savedSecret(t); secret["access_token"] != "wikino_oauth_access-1" || secret["refresh_token"] != "wikino_oauth_refresh-1" {
		t.Errorf("保存したトークン = %v", secret)
	}

	request := e.oauth.deviceRequests[0]
	if request.Get("client_id") != "wikino-cli" || request.Get("scope") != "page:read page:write draft_page:read" ||
		request.Get("resource") != "https://wikino.app/api/v1/spaces/team" {
		t.Errorf("デバイス認可の要求 = %v", request)
	}

	stderr := e.stderr.String()
	for _, want := range []string{"BCDF-GHJK", "https://wikino.app/oauth/device\n", "https://wikino.app/oauth/device?user_code=BCDF-GHJK", "10分以内"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("標準エラーに %q が出ていない: %q", want, stderr)
		}
	}
	if len(e.oauth.authorizations) != 0 || strings.Contains(stderr, "/oauth/authorize?") {
		t.Error("デバイス認可なのに、ループバックの認可要求のURLを示した")
	}
	e.assertNoSecret()
}

func TestLoginWithDeviceFlag(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.oauth.deviceResults = []string{"authorization_pending", "slow_down", ""}
	e.browser = func(string) error {
		t.Error("`--device` なのにブラウザを開いた")
		return nil
	}
	if err := e.run("", "auth", "login", "team", "--device"); err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}

	e.assertDeviceLogin(t)
	// 明示したときは、切り替えたことを知らせない。
	if strings.Contains(e.stderr.String(), "デバイス認可でログインします") {
		t.Errorf("切り替えの案内を出した: %q", e.stderr.String())
	}
	if want := []time.Duration{5 * time.Second, 5 * time.Second, 10 * time.Second}; !slices.Equal(e.waits, want) {
		t.Errorf("待った時間 = %v、期待値 = %v", e.waits, want)
	}
}

func TestLoginSwitchesToDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		vars map[string]string
		// browserFailsは、ブラウザの起動に失敗させるか。falseなら、ループバックを試みずに切り替えることを確かめるため、
		// ブラウザを開いたらテストを失敗させる。
		browserFails bool
	}{
		{
			name: "SSHで接続している",
			vars: map[string]string{"SSH_CONNECTION": "198.51.100.1 50000 192.0.2.1 22"},
		},
		{
			name:         "ブラウザを起動できない",
			browserFails: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			for k, v := range tt.vars {
				e.vars[k] = v
			}
			e.browser = func(string) error {
				if tt.browserFails {
					return errors.New("xdg-open: not found")
				}
				// 許可を待ち続けないよう、失敗を記録したうえでブラウザの起動に失敗させる。
				t.Error("ループバックを試みずに切り替えるはずが、ブラウザを開いた")
				return errors.New("ブラウザを開いた")
			}
			e.oauth.deviceResults = []string{""}
			if err := e.run("", "auth", "login", "team"); err != nil {
				t.Fatalf("ログインに失敗した: %v", err)
			}

			e.assertDeviceLogin(t)
			if !strings.Contains(e.stderr.String(), "デバイス認可でログインします") {
				t.Errorf("切り替えの案内が出ていない: %q", e.stderr.String())
			}
		})
	}
}

func TestLoginWithDeviceFailureSavesNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		results      []string
		wantCode     string
		wantExitCode int
		wantMessage  string
	}{
		{
			name:         "拒否された",
			results:      []string{"authorization_pending", "access_denied"},
			wantCode:     output.CodeAuthorizationDenied,
			wantExitCode: output.ExitAuth,
			wantMessage:  "許可されませんでした",
		},
		{
			name:         "期限が切れた",
			results:      []string{"authorization_pending", "expired_token"},
			wantCode:     output.CodeLoginTimeout,
			wantExitCode: output.ExitError,
			wantMessage:  "デバイス認可のコードの期限が切れました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			e.oauth.deviceResults = tt.results
			err := e.run("", "auth", "login", "team", "--device")

			cliErr := assertCLIError(t, err, tt.wantCode, tt.wantExitCode)
			if !strings.Contains(cliErr.Message, tt.wantMessage) || !strings.Contains(cliErr.Hint, "wikino auth login team") {
				t.Errorf("メッセージ = %q、ヒント = %q", cliErr.Message, cliErr.Hint)
			}
			if e.oauth.devicePolls != len(tt.results) {
				t.Errorf("ポーリングの回数 = %d、期待値 = %d", e.oauth.devicePolls, len(tt.results))
			}
			if len(e.oauth.issued) != 0 || len(e.keyring.items) != 0 || len(e.config().Credentials) != 0 {
				t.Error("失敗したログインでトークンを発行・保存した")
			}
			e.assertNoSecret()
		})
	}
}

func TestLoginWithDeviceAndTokenIsUsageError(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	err := e.run(aliceToken+"\n", "auth", "login", "team", "--with-token", "--device")

	cliErr := assertCLIError(t, err, output.CodeUsage, output.ExitUsage)
	if !strings.Contains(cliErr.Message, "`--device` は `--with-token` と併用できません") {
		t.Errorf("メッセージ = %q", cliErr.Message)
	}
	if len(e.requests) != 0 || len(e.keyring.items) != 0 {
		t.Error("使い方の誤りなのに要求を送るか、保存した")
	}
}
