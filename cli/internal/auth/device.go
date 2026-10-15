package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

// deviceCodeGrantTypeは、デバイスコードをトークンに交換するグラントの種類 (RFC 8628 §3.4)。
const deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// defaultDeviceIntervalは、デバイス認可の応答に `interval` が無いときのポーリングの間隔 (RFC 8628 §3.2)。
const defaultDeviceInterval = 5 * time.Second

// slowDownIncrementは、`slow_down` を受けたときに延ばすポーリングの間隔 (RFC 8628 §3.5)。
// 延ばした間隔は、以後のすべてのポーリングに使う。
const slowDownIncrement = 5 * time.Second

// ErrDeviceAuthorizationUnsupportedは、認可サーバーのメタデータにデバイス認可のエンドポイントが無いことを表す。
var ErrDeviceAuthorizationUnsupported = errors.New("認可サーバーがデバイス認可に対応していません")

// ErrDeviceCodeExpiredは、利用者が許可する前にデバイス認可の要求の期限が切れたことを表す。
var ErrDeviceCodeExpired = errors.New("デバイス認可の要求の期限が切れました")

// DeviceAuthorizationは、デバイス認可の応答 (RFC 8628 §3.2)。
type DeviceAuthorization struct {
	// DeviceCodeはトークンの要求に使う秘密の値で、利用者に示さない。
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

// DeviceOptionsは、デバイス認可によるログインの設定。
type DeviceOptions struct {
	// Spaceはログインするスペースの識別子。
	Space string
	// Scopesは要求するスコープ。
	Scopes []string
	// Showは、確認用のURLとユーザーコードを利用者に示す。
	Show func(device *DeviceAuthorization)
	// Nowは現在の時刻を返す。nilならtime.Now。
	Now func() time.Time
	// Sleepは、ctxが終わらない限りdだけ待つ。nilなら実際に待つ。ポーリングの間隔をテストで差し替える。
	Sleep func(ctx context.Context, d time.Duration) error
}

// LoginWithDeviceは、デバイス認可 (RFC 8628) でトークンを得る。
// 確認用のURLとユーザーコードを利用者に示し、別の端末のブラウザで許可されるまで、
// `interval` と `slow_down` に従ってトークンエンドポイントをポーリングする。
// 拒否されたときは `access_denied` のOAuthErrorを、期限が切れたときはErrDeviceCodeExpiredを返す。
func (c *OAuthClient) LoginWithDevice(ctx context.Context, metadata *Metadata, opts DeviceOptions) (*TokenResponse, error) {
	if metadata.DeviceAuthorizationEndpoint == "" {
		return nil, ErrDeviceAuthorizationUnsupported
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	sleep := opts.Sleep
	if sleep == nil {
		sleep = sleepContext
	}
	resource := ResourceURL(c.Host, opts.Space)

	device, err := c.requestDeviceAuthorization(ctx, metadata, opts.Scopes, resource)
	if err != nil {
		return nil, err
	}
	deadline := now().Add(time.Duration(device.ExpiresIn) * time.Second)
	interval := time.Duration(device.Interval) * time.Second
	if interval <= 0 {
		interval = defaultDeviceInterval
	}
	opts.Show(device)

	form := url.Values{
		"grant_type":  {deviceCodeGrantType},
		"device_code": {device.DeviceCode},
		"client_id":   {ClientID},
		"resource":    {resource},
	}
	for {
		remaining := deadline.Sub(now())
		if remaining <= 0 {
			return nil, ErrDeviceCodeExpired
		}
		if err := sleep(ctx, min(interval, remaining)); err != nil {
			return nil, err
		}
		// 期限までに縮めた待ちや処理の遅延で期限に達していたら、要求を送らない。
		if !now().Before(deadline) {
			return nil, ErrDeviceCodeExpired
		}

		token, err := c.requestToken(ctx, metadata, form)
		var oauthErr *OAuthError
		if !errors.As(err, &oauthErr) {
			return token, err
		}
		switch oauthErr.Code {
		case "authorization_pending":
		case "slow_down":
			interval += slowDownIncrement
		case "expired_token":
			return nil, ErrDeviceCodeExpired
		default:
			return nil, err
		}
	}
}

// requestDeviceAuthorizationは、デバイス認可を要求し (RFC 8628 §3.1)、応答を検証する。
// 確認用のURLは利用者に開かせるため、接続先と同じオリジンにあることを確かめ、
// ユーザーコードは端末に出すため、制御文字を含まないことを確かめる。
func (c *OAuthClient) requestDeviceAuthorization(ctx context.Context, metadata *Metadata, scopes []string, resource string) (*DeviceAuthorization, error) {
	resp, err := c.postForm(ctx, metadata.DeviceAuthorizationEndpoint, url.Values{
		"client_id": {ClientID},
		"scope":     {strings.Join(scopes, " ")},
		"resource":  {resource},
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, oauthResponseError(resp)
	}

	var device DeviceAuthorization
	if err := decodeJSON(resp, &device); err != nil {
		return nil, err
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.ExpiresIn <= 0 || device.Interval < 0 {
		return nil, fmt.Errorf("%w: デバイス認可の応答に必要な項目がありません", ErrUnexpectedResponse)
	}
	if strings.ContainsFunc(device.UserCode, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) {
		return nil, fmt.Errorf("%w: ユーザーコードの形式が不正です", ErrUnexpectedResponse)
	}
	if !sameOrigin(c.Host, device.VerificationURI) {
		return nil, fmt.Errorf("%w: 確認用のURL %q が接続先と別のオリジンにあります", ErrUnexpectedResponse, device.VerificationURI)
	}
	if device.VerificationURIComplete != "" && !sameOrigin(c.Host, device.VerificationURIComplete) {
		return nil, fmt.Errorf("%w: 確認用のURL %q が接続先と別のオリジンにあります", ErrUnexpectedResponse, device.VerificationURIComplete)
	}

	return &device, nil
}

// sleepContextは、dが過ぎるかctxが終わるまで待つ。
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
