package clientip_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/clientip"
)

func TestGetClientIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		headers  map[string]string
		remoteIP string
		wantIP   string
	}{
		{
			name:     "CF-Connecting-IPヘッダーがある場合",
			headers:  map[string]string{"CF-Connecting-IP": "203.0.113.1"},
			remoteIP: "10.0.0.1:12345",
			wantIP:   "203.0.113.1",
		},
		{
			name: "CF-Connecting-IPとX-Forwarded-For両方がある場合はCF優先",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.1",
				"X-Forwarded-For":  "192.168.1.1",
			},
			remoteIP: "10.0.0.1:12345",
			wantIP:   "203.0.113.1",
		},
		{
			name:     "X-Forwarded-Forヘッダーがある場合 (単一IP)",
			headers:  map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteIP: "10.0.0.1:12345",
			wantIP:   "192.168.1.1",
		},
		{
			name:     "X-Forwarded-Forヘッダーがある場合 (複数IP)",
			headers:  map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.2, 172.16.0.1"},
			remoteIP: "10.0.0.1:12345",
			wantIP:   "192.168.1.1",
		},
		{
			name:     "X-Forwarded-Forヘッダーがある場合 (空白付き)",
			headers:  map[string]string{"X-Forwarded-For": "  192.168.1.1  "},
			remoteIP: "10.0.0.1:12345",
			wantIP:   "192.168.1.1",
		},
		{
			name:     "X-Real-IPヘッダーがある場合",
			headers:  map[string]string{"X-Real-IP": "172.16.0.1"},
			remoteIP: "10.0.0.1:12345",
			wantIP:   "172.16.0.1",
		},
		{
			name: "X-Forwarded-ForとX-Real-IP両方がある場合はX-Forwarded-For優先",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
				"X-Real-IP":       "172.16.0.1",
			},
			remoteIP: "10.0.0.1:12345",
			wantIP:   "192.168.1.1",
		},
		{
			name:     "ヘッダーがない場合はRemoteAddrを使用 (ポート付き)",
			headers:  map[string]string{},
			remoteIP: "10.0.0.1:12345",
			wantIP:   "10.0.0.1",
		},
		{
			name:     "ヘッダーがない場合はRemoteAddrを使用 (ポートなし)",
			headers:  map[string]string{},
			remoteIP: "10.0.0.1",
			wantIP:   "10.0.0.1",
		},
		{
			name:     "IPv6アドレスの場合",
			headers:  map[string]string{"CF-Connecting-IP": "2001:db8::1"},
			remoteIP: "[::1]:12345",
			wantIP:   "2001:db8::1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteIP

			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			gotIP := clientip.GetClientIP(req)
			if gotIP != tc.wantIP {
				t.Errorf("GetClientIP() = %q、期待値 = %q", gotIP, tc.wantIP)
			}
		})
	}
}

func TestGetTrustedClientIP(t *testing.T) {
	t.Parallel()

	trusted := []netip.Prefix{
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.51.100.0/24"),
	}
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{name: "直接接続は偽装したヘッダーを無視する", remoteAddr: "203.0.113.10:1234", forwarded: "203.0.113.99", want: "203.0.113.10"},
		{name: "信頼済みプロキシが追加した送信元を使う", remoteAddr: "192.0.2.10:1234", forwarded: "203.0.113.99, 203.0.113.10", want: "203.0.113.10"},
		{name: "複数の信頼済みプロキシを右からたどる", remoteAddr: "192.0.2.10:1234", forwarded: "203.0.113.99, 203.0.113.10, 198.51.100.10", want: "203.0.113.10"},
		{name: "不正な転送ヘッダーは接続元に戻す", remoteAddr: "192.0.2.10:1234", forwarded: "invalid", want: "192.0.2.10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("POST", "/oauth/token", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set("CF-Connecting-IP", "203.0.113.99")
			req.Header.Set("X-Forwarded-For", tt.forwarded)
			if got := clientip.GetTrustedClientIP(req, trusted); got != tt.want {
				t.Errorf("GetTrustedClientIP() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}
