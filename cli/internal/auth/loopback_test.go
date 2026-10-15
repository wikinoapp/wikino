package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

// visitは、ブラウザの代わりにリダイレクトURIへ認可の応答を届け、表示されたページを返す。
func visit(t *testing.T, redirectURI string, query url.Values) string {
	t.Helper()

	resp, err := http.Get(redirectURI + "?" + query.Encode())
	if err != nil {
		t.Errorf("リダイレクトURIに届かない: %v", err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	return string(body)
}

func loopbackOptions(open func(string)) LoopbackOptions {
	return LoopbackOptions{
		Space:  "team",
		Scopes: []string{"page:read", "draft_page:read"},
		Open: func(authorizationURL string) error {
			open(authorizationURL)
			return nil
		},
		SuccessPage: "ログインしました",
		FailurePage: "ログインできませんでした",
		Lang:        "ja",
		Timeout:     5 * time.Second,
	}
}

func TestLoginWithLoopback(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	client := &OAuthClient{Host: testIssuer, HTTPClient: server.client()}
	metadata, err := client.FetchMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var page string
	var request url.Values
	token, err := client.LoginWithLoopback(context.Background(), metadata, loopbackOptions(func(authorizationURL string) {
		redirectURI, query := server.authorize(t, authorizationURL)
		request = server.authorizations[query.Get("state")]
		page = visit(t, redirectURI, query)
	}))
	if err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}
	if token.AccessToken != "wikino_oauth_access_1" || token.RefreshToken != "wikino_oauth_refresh_1" {
		t.Errorf("トークン = %+v", token)
	}
	if !strings.Contains(page, "ログインしました") {
		t.Errorf("ブラウザに表示したページ = %q", page)
	}

	redirect, err := url.Parse(request.Get("redirect_uri"))
	if err != nil || redirect.Scheme != "http" || redirect.Hostname() != "127.0.0.1" || redirect.Port() == "" || redirect.Path != "/callback" {
		t.Errorf("リダイレクトURI = %q", request.Get("redirect_uri"))
	}
	if request.Get("resource") != "https://wikino.app/api/v1/spaces/team" || request.Get("scope") != "page:read draft_page:read" ||
		request.Get("code_challenge_method") != "S256" || request.Get("client_id") != ClientID {
		t.Errorf("認可要求 = %v", request)
	}
	exchange := server.forms[0]
	if exchange.Get("grant_type") != "authorization_code" || exchange.Get("redirect_uri") != request.Get("redirect_uri") || exchange.Get("code_verifier") == "" {
		t.Errorf("交換の要求 = %v", exchange)
	}
}

func TestLoginWithLoopbackRejectsResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(query url.Values)
		wantErr func(error) bool
	}{
		{
			name:    "stateが一致しない応答を拒否する",
			modify:  func(q url.Values) { q.Set("state", "other-state") },
			wantErr: func(err error) bool { return errors.Is(err, ErrStateMismatch) },
		},
		{
			name:    "issが発行者と一致しない応答を拒否する",
			modify:  func(q url.Values) { q.Set("iss", "https://evil.example") },
			wantErr: func(err error) bool { return errors.Is(err, ErrIssuerMismatch) },
		},
		{
			name:    "issを載せると宣言した認可サーバーのissの無い応答を拒否する",
			modify:  func(q url.Values) { q.Del("iss") },
			wantErr: func(err error) bool { return errors.Is(err, ErrIssuerMismatch) },
		},
		{
			name: "拒否された応答をエラーコードで返す",
			modify: func(q url.Values) {
				q.Del("code")
				q.Set("error", "access_denied")
			},
			wantErr: func(err error) bool {
				var oauthErr *OAuthError
				return errors.As(err, &oauthErr) && oauthErr.Code == "access_denied"
			},
		},
		{
			name:    "stateが一致しなければエラーの応答も信用しない",
			modify:  func(q url.Values) { q.Set("state", "other-state"); q.Set("error", "access_denied") },
			wantErr: func(err error) bool { return errors.Is(err, ErrStateMismatch) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newFakeAuthServer()
			client := &OAuthClient{Host: testIssuer, HTTPClient: server.client()}
			metadata, err := client.FetchMetadata(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			var page string
			_, err = client.LoginWithLoopback(context.Background(), metadata, loopbackOptions(func(authorizationURL string) {
				redirectURI, query := server.authorize(t, authorizationURL)
				tt.modify(query)
				page = visit(t, redirectURI, query)
			}))
			if !tt.wantErr(err) {
				t.Errorf("エラー = %v", err)
			}
			if !strings.Contains(page, "ログインできませんでした") {
				t.Errorf("ブラウザに表示したページ = %q", page)
			}
			if len(server.forms) != 0 {
				t.Error("拒否した応答の認可コードを交換した")
			}
		})
	}
}

func TestLoginWithLoopbackTimeout(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	client := &OAuthClient{Host: testIssuer, HTTPClient: server.client()}
	metadata, err := client.FetchMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	opts := loopbackOptions(func(string) {})
	opts.Timeout = 50 * time.Millisecond
	if _, err := client.LoginWithLoopback(context.Background(), metadata, opts); !errors.Is(err, ErrLoopbackTimeout) {
		t.Errorf("エラー = %v、期待値 = ErrLoopbackTimeout", err)
	}
}

func TestLoginWithLoopbackBrowserUnavailable(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	client := &OAuthClient{Host: testIssuer, HTTPClient: server.client()}
	metadata, err := client.FetchMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	opts := loopbackOptions(func(string) {})
	opts.Open = func(string) error { return errors.New("xdg-open: not found") }
	// 呼び出し側がデバイス認可に切り替えられるよう、許可を待たずに見分けられるエラーを返す。
	if _, err := client.LoginWithLoopback(context.Background(), metadata, opts); !errors.Is(err, ErrLoopbackUnavailable) {
		t.Errorf("エラー = %v、期待値 = ErrLoopbackUnavailable", err)
	}
	if len(server.forms) != 0 {
		t.Error("ブラウザを開けなかったのにトークンを要求した")
	}
}

func TestLoginWithLoopbackIgnoresOtherPaths(t *testing.T) {
	t.Parallel()

	server := newFakeAuthServer()
	client := &OAuthClient{Host: testIssuer, HTTPClient: server.client()}
	metadata, err := client.FetchMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.LoginWithLoopback(context.Background(), metadata, loopbackOptions(func(authorizationURL string) {
		redirectURI, query := server.authorize(t, authorizationURL)
		// ブラウザが先にファビコンを要求しても、ログインの結果にしない。
		resp, err := http.Get(strings.TrimSuffix(redirectURI, "/callback") + "/favicon.ico")
		if err != nil {
			t.Errorf("ループバックに届かない: %v", err)
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("ファビコンの応答 = %d", resp.StatusCode)
		}
		visit(t, redirectURI, query)
	}))
	if err != nil {
		t.Fatalf("ログインに失敗した: %v", err)
	}
}

func TestCallbackPageLanguage(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
		for _, success := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/成功=%t", lang, success), func(t *testing.T) {
				t.Parallel()
				loc := i18n.NewLocalizer(lang)
				h := &callbackHandler{
					metadata: &Metadata{Issuer: testIssuer},
					state:    "expected-state",
					opts:     LoopbackOptions{Lang: lang, SuccessPage: loc.T("auth_login_page_success"), FailurePage: loc.T("auth_login_page_failure")},
					results:  make(chan callbackResult, 1),
				}
				state := "expected-state"
				messageID := "auth_login_page_success"
				if !success {
					state = "wrong-state"
					messageID = "auth_login_page_failure"
				}
				req := httptest.NewRequest(http.MethodGet, "/callback?code=code&state="+state, nil)
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
				body := w.Body.String()
				if !strings.Contains(body, `<html lang="`+lang+`">`) || !strings.Contains(body, loc.T(messageID)) {
					t.Errorf("結果画面 = %q、期待する言語 = %q、文言 = %q", body, lang, loc.T(messageID))
				}
			})
		}
	}
}
