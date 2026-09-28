package model_test

import (
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestOAuthApplication_AcceptsRedirectURI(t *testing.T) {
	t.Parallel()

	app := &model.OAuthApplication{
		RedirectURIs: []string{
			"https://example.com/callback",
			"http://127.0.0.1/callback",
			"http://[::1]:8080/callback?app=cli",
		},
	}

	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{name: "登録した文字列と完全に一致する", uri: "https://example.com/callback", want: true},
		{name: "末尾のスラッシュが違えば一致しない", uri: "https://example.com/callback/", want: false},
		{name: "ホストの大文字小文字が違えば一致しない", uri: "https://EXAMPLE.com/callback", want: false},
		{name: "HTTPSのURIはポート番号を無視しない", uri: "https://example.com:8443/callback", want: false},
		{name: "ループバックはポート番号を除いて一致すればよい", uri: "http://127.0.0.1:53123/callback", want: true},
		{name: "ループバックのIPv6もポート番号を除いて一致すればよい", uri: "http://[::1]:53123/callback?app=cli", want: true},
		{name: "ループバックでもパスが違えば一致しない", uri: "http://127.0.0.1:53123/other", want: false},
		{name: "ループバックでもクエリが違えば一致しない", uri: "http://[::1]:53123/callback", want: false},
		{name: "ループバックでも登録と違うアドレスは一致しない", uri: "http://[::1]:53123/callback", want: false},
		{name: "localhostはループバックとして扱わない", uri: "http://localhost:53123/callback", want: false},
		{name: "ループバックでもフラグメントを含めば一致しない", uri: "http://127.0.0.1:53123/callback#x", want: false},
		{name: "ループバックでもユーザー情報を含めば一致しない", uri: "http://user@127.0.0.1:53123/callback", want: false},
		{name: "空文字は一致しない", uri: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := app.AcceptsRedirectURI(tt.uri); got != tt.want {
				t.Errorf("AcceptsRedirectURI(%q) = %v、期待値 = %v", tt.uri, got, tt.want)
			}
		})
	}
}

func TestOAuthAuthorizationError(t *testing.T) {
	t.Parallel()

	t.Run("リダイレクトURIを持つエラーはクライアントへ戻せる", func(t *testing.T) {
		t.Parallel()

		err := &model.OAuthAuthorizationError{Code: model.OAuthAuthorizationErrorInvalidScope, RedirectURI: "https://example.com/callback"}
		if !err.IsRedirectable() {
			t.Error("IsRedirectable() = false、期待値 = true")
		}
	})

	t.Run("リダイレクトURIを持たないエラーはクライアントへ戻さない", func(t *testing.T) {
		t.Parallel()

		err := &model.OAuthAuthorizationError{Code: model.OAuthAuthorizationErrorInvalidRequest, UserMsg: "不正"}
		if err.IsRedirectable() {
			t.Error("IsRedirectable() = true、期待値 = false")
		}
	})

	t.Run("包まれたエラーから取り出せる", func(t *testing.T) {
		t.Parallel()

		err := &model.OAuthAuthorizationError{Code: model.OAuthAuthorizationErrorAccessDenied, RedirectURI: "https://example.com/callback"}
		if got := model.AsOAuthAuthorizationError(fmt.Errorf("包む: %w", err)); got != err {
			t.Errorf("AsOAuthAuthorizationError() = %v、期待値 = %v", got, err)
		}
		if got := model.AsOAuthAuthorizationError(fmt.Errorf("別のエラー")); got != nil {
			t.Errorf("AsOAuthAuthorizationError() = %v、期待値 = nil", got)
		}
	})
}

func TestParseSpaceAPIResourceURL(t *testing.T) {
	t.Parallel()

	const appURL = "https://example.com"

	if got := model.SpaceAPIResourceURL(appURL, "seed-wiki"); got != "https://example.com/api/v1/spaces/seed-wiki" {
		t.Errorf("SpaceAPIResourceURL() = %q", got)
	}

	tests := []struct {
		name     string
		resource string
		want     model.SpaceIdentifier
		wantOK   bool
	}{
		{name: "スペースのAPIのURLから識別子を取り出す", resource: "https://example.com/api/v1/spaces/seed-wiki", want: "seed-wiki", wantOK: true},
		{name: "別のオリジンは受け付けない", resource: "https://evil.example/api/v1/spaces/seed-wiki", wantOK: false},
		{name: "HTTPは受け付けない", resource: "http://example.com/api/v1/spaces/seed-wiki", wantOK: false},
		{name: "識別子が無ければ受け付けない", resource: "https://example.com/api/v1/spaces/", wantOK: false},
		{name: "末尾のスラッシュは受け付けない", resource: "https://example.com/api/v1/spaces/seed-wiki/", wantOK: false},
		{name: "スペース配下のパスは受け付けない", resource: "https://example.com/api/v1/spaces/seed-wiki/pages", wantOK: false},
		{name: "クエリは受け付けない", resource: "https://example.com/api/v1/spaces/seed-wiki?x=1", wantOK: false},
		{name: "フラグメントは受け付けない", resource: "https://example.com/api/v1/spaces/seed-wiki#x", wantOK: false},
		{name: "APIのURLでなければ受け付けない", resource: "https://example.com/s/seed-wiki", wantOK: false},
		{name: "空文字は受け付けない", resource: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := model.ParseSpaceAPIResourceURL(appURL, tt.resource)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("ParseSpaceAPIResourceURL(%q) = (%q, %v)、期待値 = (%q, %v)", tt.resource, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
