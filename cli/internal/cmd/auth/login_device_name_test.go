package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestLoginSendsDeviceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		hostname func() (string, error)
		want     string
	}{
		{name: "省略するとホスト名を送る", want: "test-host"},
		{name: "--device-nameの値を前後の空白を除いて送る", args: []string{"--device-name", " work-laptop "}, want: "work-laptop"},
		{name: "有効なUTF-8の置換文字を送る", args: []string{"--device-name", "my\ufffdlaptop"}, want: "my\ufffdlaptop"},
		{
			name:     "ホスト名は100文字に切り詰める",
			hostname: func() (string, error) { return strings.Repeat("h", 120), nil },
			want:     strings.Repeat("h", 100),
		},
		{
			name:     "マルチバイトのホスト名も100文字に切り詰める",
			hostname: func() (string, error) { return strings.Repeat("あ", 120), nil },
			want:     strings.Repeat("あ", 100),
		},
		{
			name:     "ホスト名の有効なUTF-8の置換文字は送る",
			hostname: func() (string, error) { return "\ufffd" + strings.Repeat("h", 110), nil },
			want:     "\ufffd" + strings.Repeat("h", 99),
		},
		{
			name:     "短い不正UTF-8のホスト名を送らない",
			hostname: func() (string, error) { return "host\xff", nil },
		},
		{
			name:     "100文字を超える不正UTF-8のホスト名を送らない",
			hostname: func() (string, error) { return "\xff" + strings.Repeat("h", 110), nil },
		},
		{
			name:     "切り詰める範囲外に不正UTF-8があるホスト名も送らない",
			hostname: func() (string, error) { return strings.Repeat("h", 110) + "\xff", nil },
		},
		{
			name:     "ホスト名を得られなければ送らない",
			hostname: func() (string, error) { return "", errors.New("ホスト名を得られない") },
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			e.hostname = tt.hostname
			e.loginWithBrowser("team", tt.args...)

			request := e.oauth.authorizations[0]
			if got := request.Get("device_name"); got != tt.want || request.Has("device_name") != (tt.want != "") {
				t.Errorf("認可要求のdevice_name = %q (送ったか = %v)、期待値 = %q", got, request.Has("device_name"), tt.want)
			}
		})
	}
}

func TestLoginWithDeviceSendsDeviceName(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.oauth.deviceResults = []string{""}
	if err := e.run("", "auth", "login", "team", "--device", "--device-name", "ssh-box"); err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}

	if got := e.oauth.deviceRequests[0].Get("device_name"); got != "ssh-box" {
		t.Errorf("デバイス認可の要求のdevice_name = %q、期待値 = %q", got, "ssh-box")
	}
}

func TestLoginInvalidDeviceNameIsUsageError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{name: "空", args: []string{"--device-name", " "}, want: "100文字までの制御文字を含まない端末名"},
		{name: "100文字を超える", args: []string{"--device-name", strings.Repeat("a", 101)}, want: "100文字までの制御文字を含まない端末名"},
		{name: "制御文字を含む", args: []string{"--device-name", "a\tb"}, want: "100文字までの制御文字を含まない端末名"},
		{name: "不正UTF-8を含む", args: []string{"--device-name", "laptop\xff"}, want: "100文字までの制御文字を含まない端末名"},
		{name: "途中で切れたUTF-8を含む", args: []string{"--device-name", "laptop\xe3\x81"}, want: "100文字までの制御文字を含まない端末名"},
		{name: "デバイス認可でも不正UTF-8を含む", args: []string{"--device", "--device-name", "laptop\xff"}, want: "100文字までの制御文字を含まない端末名"},
		{name: "--with-tokenと併用した", stdin: aliceToken + "\n", args: []string{"--with-token", "--device-name", "laptop"}, want: "`--with-token` と併用できません"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newTestEnv(t)
			err := e.run(tt.stdin, append([]string{"auth", "login", "team"}, tt.args...)...)
			cliErr := assertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			if !strings.Contains(cliErr.Message, tt.want) {
				t.Errorf("メッセージ = %q、%qを含むことを期待", cliErr.Message, tt.want)
			}
			if len(e.requests) != 0 || len(e.oauth.authorizations) != 0 || len(e.keyring.items) != 0 {
				t.Error("使い方の誤りで認可を要求したか、資格情報を保存した")
			}
		})
	}
}

func TestReloginRevokesReplacedToken(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.loginWithBrowser("team")
	e.loginWithBrowser("team")

	// 新しいトークンを保存した後に、置き換えた古いリフレッシュトークンだけを失効させる。
	if len(e.oauth.revoked) != 1 || e.oauth.revoked[0] != "wikino_oauth_refresh-1" {
		t.Errorf("失効させたトークン = %v、期待値 = [wikino_oauth_refresh-1]", e.oauth.revoked)
	}
	if secret := e.savedSecret(t); secret["refresh_token"] != "wikino_oauth_refresh-2" {
		t.Errorf("保存したトークン = %v", secret)
	}
	e.assertNoSecret()
}

func TestReloginRevokeFailureStillSucceeds(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.loginWithBrowser("team")
	e.oauth.revokeStatus = 503
	e.loginWithBrowser("team")

	if !strings.Contains(e.stderr.String(), "置き換えた以前のトークンをサーバーで失効させられませんでした") {
		t.Errorf("標準エラーに警告を期待したが %q だった", e.stderr.String())
	}
	if secret := e.savedSecret(t); secret["refresh_token"] != "wikino_oauth_refresh-2" {
		t.Errorf("保存したトークン = %v", secret)
	}
	e.assertNoSecret()
}

func TestReloginWithPersonalAccessTokenDoesNotRevoke(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	if err := e.run(aliceToken+"\n", "auth", "login", "team", "--with-token"); err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}
	e.loginWithBrowser("team")

	if len(e.oauth.revoked) != 0 {
		t.Errorf("個人用アクセストークンの置き換えで失効を要求した: %v", e.oauth.revoked)
	}
}

func TestReloginWithPersonalAccessTokenRevokesReplacedOAuthToken(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.loginWithBrowser("team")
	if err := e.run(aliceToken+"\n", "auth", "login", "team", "--with-token"); err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}

	// 置き換えたOAuthのトークンは、新しいログインの方法によらず失効させる。
	if len(e.oauth.revoked) != 1 || e.oauth.revoked[0] != "wikino_oauth_refresh-1" {
		t.Errorf("失効させたトークン = %v、期待値 = [wikino_oauth_refresh-1]", e.oauth.revoked)
	}
}

func TestLoginAsAnotherMemberDoesNotRevoke(t *testing.T) {
	t.Parallel()

	e := newTestEnv(t)
	e.loginWithBrowser("team")
	if err := e.run(botToken+"\n", "auth", "login", "team", "--with-token"); err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}

	// 別のメンバーの資格情報は置き換えないため、そのリフレッシュトークンも失効させない。
	if len(e.oauth.revoked) != 0 {
		t.Errorf("別のメンバーのログインで失効を要求した: %v", e.oauth.revoked)
	}
	if len(e.keyring.items) != 2 {
		t.Errorf("キーチェーンの項目の数 = %d、期待値 = 2 (両方のメンバーの資格情報)", len(e.keyring.items))
	}
}
