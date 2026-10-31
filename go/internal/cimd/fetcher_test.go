package cimd

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// testClientIDはテストで取得するCIMDのクライアントID
const testClientID = "https://claude.ai/oauth/claude-code-client-metadata"

// roundTripFuncは、テストで通信の経路を差し替えるhttp.RoundTripper。URLは本物のまま、
// 応答だけを返す
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// newTestFetcherは、各要求にrespondの応答を返すFetcherと、要求の回数を数えるカウンターを返す
func newTestFetcher(respond func(*http.Request) (*http.Response, error)) (*Fetcher, *atomic.Int32) {
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return respond(req)
	})}
	return NewFetcher(client), &calls
}

// jsonResponseは、statusとヘッダーheaderで本文bodyを返す応答を作る
func jsonResponse(status int, header http.Header, body string) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	header.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// validDocumentは、testClientIDの検証を通る文書
const validDocument = `{
  "client_id": "https://claude.ai/oauth/claude-code-client-metadata",
  "client_name": " Claude Code ",
  "redirect_uris": ["http://localhost/callback", "http://127.0.0.1/callback"],
  "token_endpoint_auth_method": "none"
}`

func TestFetcher_Fetch(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	fetcher, calls := newTestFetcher(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() != testClientID {
			t.Errorf("要求 = %s %s、期待値 = GET %s", req.Method, req.URL, testClientID)
		}
		if got := req.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q、期待値 = application/json", got)
		}
		return jsonResponse(http.StatusOK, http.Header{"Cache-Control": {"public, max-age=3600"}}, validDocument), nil
	})

	doc, err := fetcher.Fetch(context.Background(), testClientID, now)
	if err != nil {
		t.Fatalf("Fetch()のエラー = %v", err)
	}
	if doc.ClientID != testClientID {
		t.Errorf("ClientID = %q、期待値 = %q", doc.ClientID, testClientID)
	}
	if doc.ClientName != "Claude Code" {
		t.Errorf("ClientName = %q、期待値 = %q (前後の空白を除く)", doc.ClientName, "Claude Code")
	}
	if want := []string{"http://localhost/callback", "http://127.0.0.1/callback"}; !slices.Equal(doc.RedirectURIs, want) {
		t.Errorf("RedirectURIs = %v、期待値 = %v", doc.RedirectURIs, want)
	}
	if want := now.Add(time.Hour); !doc.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v、期待値 = %v", doc.ExpiresAt, want)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("要求の回数 = %d、期待値 = 1", got)
	}
}

func TestFetcher_Fetch_複数形の認証の方式にnoneがあれば受け付ける(t *testing.T) {
	t.Parallel()

	// ChatGPTの文書と同じく、単数形に別の方式を、複数形にnoneを載せる
	body := `{
	  "client_id": "https://claude.ai/oauth/claude-code-client-metadata",
	  "client_name": "ChatGPT",
	  "redirect_uris": ["https://chatgpt.com/connector/oauth/callback"],
	  "token_endpoint_auth_method": "private_key_jwt",
	  "token_endpoint_auth_methods_supported": ["none", "private_key_jwt"]
	}`
	fetcher, _ := newTestFetcher(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, nil, body), nil
	})

	if _, err := fetcher.Fetch(context.Background(), testClientID, time.Now()); err != nil {
		t.Fatalf("Fetch()のエラー = %v", err)
	}
}

func TestFetcher_Fetch_許可リストの外のクライアントIDは取得しない(t *testing.T) {
	t.Parallel()

	fetcher, calls := newTestFetcher(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, nil, validDocument), nil
	})

	_, err := fetcher.Fetch(context.Background(), "https://example.com/client.json", time.Now())
	if !errors.Is(err, ErrNotAllowed) {
		t.Errorf("エラー = %v、期待値 = ErrNotAllowed", err)
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("要求の回数 = %d、期待値 = 0", got)
	}
}

func TestFetcher_Fetch_リダイレクトをたどらない(t *testing.T) {
	t.Parallel()

	fetcher, calls := newTestFetcher(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusFound, http.Header{"Location": {"https://claude.ai/other"}}, ""), nil
	})

	if _, err := fetcher.Fetch(context.Background(), testClientID, time.Now()); err == nil {
		t.Error("エラー = nil、リダイレクトの応答をエラーにすることを期待")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("要求の回数 = %d、期待値 = 1 (リダイレクト先を取得しない)", got)
	}
}

func TestFetcher_Fetch_受け付けない応答(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "200以外のステータス", status: http.StatusNotFound, body: validDocument},
		{name: "5KBを超える文書", status: http.StatusOK, body: validDocument + strings.Repeat(" ", maxDocumentBytes)},
		{name: "JSONでない文書", status: http.StatusOK, body: "<html></html>"},
		{name: "client_idが取得したURLと違う", status: http.StatusOK, body: `{"client_id":"https://claude.ai/other","client_name":"x","redirect_uris":["https://claude.ai/cb"],"token_endpoint_auth_method":"none"}`},
		{name: "client_nameが無い", status: http.StatusOK, body: `{"client_id":"` + testClientID + `","redirect_uris":["https://claude.ai/cb"],"token_endpoint_auth_method":"none"}`},
		{name: "client_nameが空白だけ", status: http.StatusOK, body: `{"client_id":"` + testClientID + `","client_name":"  ","redirect_uris":["https://claude.ai/cb"],"token_endpoint_auth_method":"none"}`},
		{name: "redirect_urisが無い", status: http.StatusOK, body: `{"client_id":"` + testClientID + `","client_name":"x","token_endpoint_auth_method":"none"}`},
		{name: "redirect_urisに空文字がある", status: http.StatusOK, body: `{"client_id":"` + testClientID + `","client_name":"x","redirect_uris":[""],"token_endpoint_auth_method":"none"}`},
		{name: "redirect_urisが文字列の配列でない", status: http.StatusOK, body: `{"client_id":"` + testClientID + `","client_name":"x","redirect_uris":"https://claude.ai/cb","token_endpoint_auth_method":"none"}`},
		{name: "認証の方式にnoneが無い", status: http.StatusOK, body: `{"client_id":"` + testClientID + `","client_name":"x","redirect_uris":["https://claude.ai/cb"],"token_endpoint_auth_method":"private_key_jwt"}`},
		{name: "認証の方式を宣言していない", status: http.StatusOK, body: `{"client_id":"` + testClientID + `","client_name":"x","redirect_uris":["https://claude.ai/cb"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fetcher, _ := newTestFetcher(func(*http.Request) (*http.Response, error) {
				return jsonResponse(tt.status, nil, tt.body), nil
			})
			doc, err := fetcher.Fetch(context.Background(), testClientID, time.Now())
			if err == nil {
				t.Errorf("Fetch() = %+v、エラーを期待", doc)
			}
		})
	}
}

func TestFetcher_Fetch_ちょうど5KBの文書は受け付ける(t *testing.T) {
	t.Parallel()

	body := validDocument + strings.Repeat(" ", maxDocumentBytes-len(validDocument))
	fetcher, _ := newTestFetcher(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, nil, body), nil
	})

	if _, err := fetcher.Fetch(context.Background(), testClientID, time.Now()); err != nil {
		t.Fatalf("Fetch()のエラー = %v", err)
	}
}

func TestFetcher_Fetch_タイムアウトしたら打ち切る(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		fetcher, _ := newTestFetcher(func(req *http.Request) (*http.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		})

		now := time.Now()
		_, err := fetcher.Fetch(context.Background(), testClientID, now)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("エラー = %v、期待値 = context.DeadlineExceeded", err)
		}
		if elapsed := time.Since(now); elapsed != 5*time.Second {
			t.Errorf("タイムアウトまでの時間 = %v、期待値 = 5s", elapsed)
		}
	})
}

func TestCacheLifetime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cacheControl string
		want         time.Duration
	}{
		{cacheControl: "max-age=3600", want: time.Hour},
		{cacheControl: "public, max-age=7200", want: 2 * time.Hour},
		{cacheControl: "max-age=60", want: minCacheLifetime},
		{cacheControl: "max-age=604800", want: maxCacheLifetime},
		{cacheControl: "max-age=99999999999999999", want: maxCacheLifetime},
		{cacheControl: "max-age=3600, no-cache", want: minCacheLifetime},
		{cacheControl: "no-store", want: minCacheLifetime},
		{cacheControl: "max-age=abc", want: minCacheLifetime},
		{cacheControl: "", want: minCacheLifetime},
	}
	for _, tt := range tests {
		t.Run(tt.cacheControl, func(t *testing.T) {
			t.Parallel()

			if got := cacheLifetime(tt.cacheControl); got != tt.want {
				t.Errorf("cacheLifetime(%q) = %v、期待値 = %v", tt.cacheControl, got, tt.want)
			}
		})
	}
}
