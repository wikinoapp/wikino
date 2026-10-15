package auth

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// deviceClockは、デバイス認可のポーリングで待った時間を記録し、その分だけ時刻を進める。
type deviceClock struct {
	now   time.Time
	waits []time.Duration
}

func newDeviceClock() *deviceClock {
	return &deviceClock{now: time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)}
}

func (c *deviceClock) options(show func(*DeviceAuthorization)) DeviceOptions {
	if show == nil {
		show = func(*DeviceAuthorization) {}
	}
	return DeviceOptions{
		Space:  "team",
		Scopes: []string{"page:read", "draft_page:read"},
		Show:   show,
		Now:    func() time.Time { return c.now },
		Sleep: func(_ context.Context, d time.Duration) error {
			c.waits = append(c.waits, d)
			c.now = c.now.Add(d)
			return nil
		},
	}
}

// deviceLoginは、serverを認可サーバーにして、デバイス認可でログインする。
func deviceLogin(t *testing.T, server *fakeAuthServer, opts DeviceOptions) (*TokenResponse, error) {
	t.Helper()

	client := &OAuthClient{Host: testIssuer, HTTPClient: server.client()}
	metadata, err := client.FetchMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	return client.LoginWithDevice(context.Background(), metadata, opts)
}

func TestLoginWithDevice(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	server.deviceResults = []string{"authorization_pending", "slow_down", "authorization_pending", ""}
	clock := newDeviceClock()
	var shown []*DeviceAuthorization

	token, err := deviceLogin(t, server, clock.options(func(device *DeviceAuthorization) {
		// ポーリングを始める前に、確認用のURLとコードを示す。
		if len(server.forms) != 0 {
			t.Error("確認用のURLとコードを示す前にポーリングした")
		}
		shown = append(shown, device)
	}))
	if err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}
	if token.AccessToken != "wikino_oauth_access_1" || token.RefreshToken != "wikino_oauth_refresh_1" {
		t.Errorf("トークン = %+v", token)
	}
	if len(shown) != 1 || shown[0].UserCode != "BCDF-GHJK" || shown[0].VerificationURI != testIssuer+"/oauth/device" ||
		shown[0].VerificationURIComplete != testIssuer+"/oauth/device?user_code=BCDF-GHJK" {
		t.Errorf("示した内容 = %+v", shown)
	}

	request := server.deviceForms[0]
	if request.Get("client_id") != ClientID || request.Get("scope") != "page:read draft_page:read" ||
		request.Get("resource") != "https://wikino.app/api/v1/spaces/team" {
		t.Errorf("デバイス認可の要求 = %v", request)
	}
	if len(server.forms) != 4 {
		t.Fatalf("ポーリングの回数 = %d、期待値 = 4", len(server.forms))
	}
	for _, form := range server.forms {
		if form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" || form.Get("device_code") != "device-code-secret" ||
			form.Get("client_id") != ClientID || form.Get("resource") != "https://wikino.app/api/v1/spaces/team" {
			t.Errorf("トークンの要求 = %v", form)
		}
	}

	// `slow_down` の後は、以後のすべてのポーリングの間隔を5秒延ばす。
	want := []time.Duration{5 * time.Second, 5 * time.Second, 10 * time.Second, 10 * time.Second}
	if !slices.Equal(clock.waits, want) {
		t.Errorf("待った時間 = %v、期待値 = %v", clock.waits, want)
	}
}

func TestLoginWithDeviceDefaultInterval(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	delete(server.deviceAuthorization, "interval")
	server.deviceResults = []string{""}
	clock := newDeviceClock()

	if _, err := deviceLogin(t, server, clock.options(nil)); err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}
	if !slices.Equal(clock.waits, []time.Duration{5 * time.Second}) {
		t.Errorf("待った時間 = %v、期待値 = [5s]", clock.waits)
	}
}

func TestLoginWithDeviceStops(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		results []string
		check   func(t *testing.T, err error)
	}{
		{
			name:    "拒否されたらaccess_deniedを返す",
			results: []string{"authorization_pending", "access_denied"},
			check: func(t *testing.T, err error) {
				var oauthErr *OAuthError
				if !errors.As(err, &oauthErr) || oauthErr.Code != "access_denied" {
					t.Errorf("エラー = %v、期待値 = access_denied", err)
				}
			},
		},
		{
			name:    "サーバーが期限切れを返したら止める",
			results: []string{"authorization_pending", "expired_token"},
			check: func(t *testing.T, err error) {
				if !errors.Is(err, ErrDeviceCodeExpired) {
					t.Errorf("エラー = %v、期待値 = ErrDeviceCodeExpired", err)
				}
			},
		},
		{
			name:    "想定しないエラーコードで止める",
			results: []string{"invalid_grant"},
			check: func(t *testing.T, err error) {
				var oauthErr *OAuthError
				if !errors.As(err, &oauthErr) || oauthErr.Code != "invalid_grant" {
					t.Errorf("エラー = %v、期待値 = invalid_grant", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newFakeAuthServer()
			server.deviceResults = tt.results
			token, err := deviceLogin(t, server, newDeviceClock().options(nil))
			if token != nil {
				t.Errorf("トークンを返した: %+v", token)
			}
			tt.check(t, err)
			if len(server.forms) != len(tt.results) {
				t.Errorf("ポーリングの回数 = %d、期待値 = %d", len(server.forms), len(tt.results))
			}
		})
	}
}

func TestLoginWithDeviceExpiresLocally(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		expiresIn int64
		results   []string
		wantWaits []time.Duration
		wantPolls int
	}{
		{
			name:      "期限までの残り時間が間隔より短い",
			expiresIn: 12,
			wantWaits: []time.Duration{5 * time.Second, 5 * time.Second, 2 * time.Second},
			wantPolls: 2,
		},
		{
			name:      "次のポーリングの時刻が期限と一致する",
			expiresIn: 10,
			wantWaits: []time.Duration{5 * time.Second, 5 * time.Second},
			wantPolls: 1,
		},
		{
			name:      "slow_downで延ばした間隔の途中で期限に達する",
			expiresIn: 12,
			results:   []string{"slow_down"},
			wantWaits: []time.Duration{5 * time.Second, 7 * time.Second},
			wantPolls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newFakeAuthServer()
			server.deviceAuthorization["expires_in"] = tt.expiresIn
			server.deviceResults = tt.results
			clock := newDeviceClock()

			// 期限までに縮めた待ちの後は、期限切れの要求を送らずに止める。
			if _, err := deviceLogin(t, server, clock.options(nil)); !errors.Is(err, ErrDeviceCodeExpired) {
				t.Errorf("エラー = %v、期待値 = ErrDeviceCodeExpired", err)
			}
			if !slices.Equal(clock.waits, tt.wantWaits) {
				t.Errorf("待った時間 = %v、期待値 = %v", clock.waits, tt.wantWaits)
			}
			if len(server.forms) != tt.wantPolls {
				t.Errorf("ポーリングの回数 = %d、期待値 = %d", len(server.forms), tt.wantPolls)
			}
		})
	}
}

func TestLoginWithDeviceCanceled(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	opts := newDeviceClock().options(nil)
	opts.Sleep = func(context.Context, time.Duration) error { return context.Canceled }

	if _, err := deviceLogin(t, server, opts); !errors.Is(err, context.Canceled) {
		t.Errorf("エラー = %v、期待値 = context.Canceled", err)
	}
	if len(server.forms) != 0 {
		t.Error("中止した後にポーリングした")
	}
}

func TestLoginWithDeviceUnsupported(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	delete(server.metadata, "device_authorization_endpoint")

	if _, err := deviceLogin(t, server, newDeviceClock().options(nil)); !errors.Is(err, ErrDeviceAuthorizationUnsupported) {
		t.Errorf("エラー = %v、期待値 = ErrDeviceAuthorizationUnsupported", err)
	}
	if len(server.deviceForms) != 0 {
		t.Error("デバイス認可を要求した")
	}
}

func TestLoginWithDeviceRejectsResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		modify func(m map[string]any)
	}{
		{name: "確認用のURLが別のオリジン", modify: func(m map[string]any) { m["verification_uri"] = "https://evil.example/device" }},
		{name: "コード入力済みのURLが別のオリジン", modify: func(m map[string]any) {
			m["verification_uri_complete"] = "https://evil.example/device?user_code=BCDF-GHJK"
		}},
		{name: "ユーザーコードに制御文字を含む", modify: func(m map[string]any) { m["user_code"] = "BCDF\x1b[2J" }},
		{name: "デバイスコードが無い", modify: func(m map[string]any) { delete(m, "device_code") }},
		{name: "期限が無い", modify: func(m map[string]any) { delete(m, "expires_in") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newFakeAuthServer()
			tt.modify(server.deviceAuthorization)
			shown := false
			_, err := deviceLogin(t, server, newDeviceClock().options(func(*DeviceAuthorization) { shown = true }))
			if !errors.Is(err, ErrUnexpectedResponse) {
				t.Errorf("エラー = %v、期待値 = ErrUnexpectedResponse", err)
			}
			if shown || len(server.forms) != 0 {
				t.Error("不正な応答のURLやコードを示すか、ポーリングした")
			}
		})
	}
}

func TestPrefersDeviceAuthorization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		goos string
		env  map[string]string
		want bool
	}{
		{name: "Linuxでディスプレイがある", goos: "linux", env: map[string]string{"DISPLAY": ":0"}, want: false},
		{name: "LinuxでWaylandのディスプレイがある", goos: "linux", env: map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, want: false},
		{name: "Linuxでディスプレイが無い", goos: "linux", env: map[string]string{}, want: true},
		{name: "FreeBSDでディスプレイが無い", goos: "freebsd", env: map[string]string{}, want: true},
		{name: "macOSはディスプレイの環境変数を見ない", goos: "darwin", env: map[string]string{}, want: false},
		{name: "Windowsはディスプレイの環境変数を見ない", goos: "windows", env: map[string]string{}, want: false},
		{name: "SSHの接続中 (SSH_CONNECTION)", goos: "darwin", env: map[string]string{"SSH_CONNECTION": "198.51.100.1 50000 192.0.2.1 22"}, want: true},
		{name: "SSHの接続中 (SSH_CLIENT)", goos: "linux", env: map[string]string{"DISPLAY": ":0", "SSH_CLIENT": "198.51.100.1 50000 22"}, want: true},
		{name: "SSHの接続中 (SSH_TTY)", goos: "windows", env: map[string]string{"SSH_TTY": "/dev/pts/0"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := PrefersDeviceAuthorization(tt.goos, func(key string) string { return tt.env[key] }); got != tt.want {
				t.Errorf("PrefersDeviceAuthorization = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}
