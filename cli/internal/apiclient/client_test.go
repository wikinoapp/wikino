package apiclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
)

const testHost = "https://example.com"

// fakeTransportは、メソッド・ホスト・パスで登録した応答を返し、受け取った要求を記録する。
// URLを書き換えずにトランスポートを差し替え、組み立てた要求をそのまま検証するために使う。
type fakeTransport struct {
	mu        sync.Mutex
	responses map[string]func(req *http.Request) *http.Response
	requests  []*http.Request
	bodies    []string
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{responses: map[string]func(*http.Request) *http.Response{}}
}

func (f *fakeTransport) handle(method, rawURL string, fn func(req *http.Request) *http.Response) {
	f.responses[method+" "+rawURL] = fn
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body string
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}

	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.bodies = append(f.bodies, body)
	f.mu.Unlock()

	// クエリは登録のキーに含めず、パスまでで照合する。
	key := req.Method + " " + req.URL.Scheme + "://" + req.URL.Host + req.URL.EscapedPath()
	fn, ok := f.responses[key]
	if !ok {
		return nil, errors.New("登録されていない要求です: " + key)
	}

	return fn(req), nil
}

func (f *fakeTransport) client() *http.Client {
	return &http.Client{Transport: f}
}

func jsonResponse(status int, contentType, body string, headers ...string) *http.Response {
	h := http.Header{}
	h.Set("Content-Type", contentType)
	for i := 0; i+1 < len(headers); i += 2 {
		h.Set(headers[i], headers[i+1])
	}

	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

const pageJSON = `{"id":"0b3a5c1e-6f43-4f6e-9a39-0f6f1b0f1a01","number":12,"topic_number":3,"title":"議事録","body":"本文","status":"published","modified_at":"2026-10-01T09:00:00Z","url":"https://example.com/s/my-space/pages/12"}`

func newTestClient(t *testing.T, ft *fakeTransport, opts Options) *Client {
	t.Helper()

	opts.Host = testHost
	opts.HTTPClient = ft.client()
	c, err := New(opts)
	if err != nil {
		t.Fatalf("クライアントを作れません: %v", err)
	}

	return c
}

func TestClient_Headers(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, testHost+"/api/v1/spaces/my-space/pages/12", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "application/json", pageJSON, "ETag", `"v1"`)
	})
	c := newTestClient(t, ft, Options{
		Token:          StaticToken("secret-token"),
		UserAgent:      "wikino-cli/1.0.0 (linux; amd64)",
		AcceptLanguage: "ja",
	})

	resp, err := c.GetPageWithResponse(context.Background(), "my-space", 12, nil)
	if err != nil {
		t.Fatalf("要求に失敗しました: %v", err)
	}

	req := ft.requests[0]
	if got := req.Header.Get("Authorization"); got != "Bearer secret-token" {
		t.Errorf("Authorizationが違います: %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != "wikino-cli/1.0.0 (linux; amd64)" {
		t.Errorf("User-Agentが違います: %q", got)
	}
	if got := req.Header.Get("Accept-Language"); got != "ja" {
		t.Errorf("Accept-Languageが違います: %q", got)
	}
	if resp.JSON200 == nil || resp.JSON200.Number != 12 {
		t.Fatalf("応答のページを読めません: %+v", resp.JSON200)
	}
	if resp.Headers200 == nil || resp.Headers200.ETag != `"v1"` {
		t.Errorf("応答のETagを読めません: %+v", resp.Headers200)
	}
}

func TestClient_NoToken(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, testHost+"/api/v1/spaces/my-space/pages/12", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "application/json", pageJSON, "ETag", `"v1"`)
	})
	c := newTestClient(t, ft, Options{})

	if _, err := c.GetPageWithResponse(context.Background(), "my-space", 12, nil); err != nil {
		t.Fatalf("要求に失敗しました: %v", err)
	}
	if got := ft.requests[0].Header.Get("Authorization"); got != "" {
		t.Errorf("トークンが無いのにAuthorizationが付いています: %q", got)
	}
}

func TestClient_IfMatch(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodPatch, testHost+"/api/v1/spaces/my-space/pages/12", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "application/json", pageJSON, "ETag", `"v2"`)
	})
	c := newTestClient(t, ft, Options{Token: StaticToken("secret-token")})

	etag := `"v1"`
	title := "議事録"
	resp, err := c.UpdatePageWithApplicationMergePatchPlusJSONBodyWithResponse(context.Background(), "my-space", 12,
		&apigen.UpdatePageParams{IfMatch: &etag}, apigen.PagePatch{Title: &title})
	if err != nil {
		t.Fatalf("要求に失敗しました: %v", err)
	}

	if got := ft.requests[0].Header.Get("If-Match"); got != `"v1"` {
		t.Errorf("If-Matchが違います: %q", got)
	}
	if got := ft.bodies[0]; got != `{"title":"議事録"}` {
		t.Errorf("本文が違います: %s", got)
	}
	if resp.Headers200 == nil || resp.Headers200.ETag != `"v2"` {
		t.Errorf("更新後のETagを読めません: %+v", resp.Headers200)
	}
}

func TestClient_ProblemDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		resp    func() *http.Response
		wantErr Error
	}{
		{
			name: "412はProblem Detailsの項目を持つ",
			resp: func() *http.Response {
				return jsonResponse(http.StatusPreconditionFailed, "application/problem+json",
					`{"type":"https://example.com/api/problems/precondition-failed","title":"前提条件を満たしません","status":412,"detail":"ページはその後に更新されています。"}`)
			},
			wantErr: Error{
				StatusCode: 412,
				Type:       "https://example.com/api/problems/precondition-failed",
				Title:      "前提条件を満たしません",
				Detail:     "ページはその後に更新されています。",
			},
		},
		{
			name: "422は項目ごとの問題を持つ",
			resp: func() *http.Response {
				return jsonResponse(http.StatusUnprocessableEntity, "application/problem+json",
					`{"type":"about:blank","title":"入力が正しくありません","status":422,"detail":null,"errors":[{"detail":"タイトルが重複しています","pointer":"#/title","parameter":null}]}`)
			},
			wantErr: Error{
				StatusCode:  422,
				Type:        "about:blank",
				Title:       "入力が正しくありません",
				FieldErrors: []FieldError{{Detail: "タイトルが重複しています", Pointer: "#/title"}},
			},
		},
		{
			name: "403はWWW-Authenticateから要るスコープを読む",
			resp: func() *http.Response {
				return jsonResponse(http.StatusForbidden, "application/problem+json",
					`{"type":"about:blank","title":"Forbidden","status":403,"detail":null}`,
					"WWW-Authenticate", `Bearer error="insufficient_scope", scope="page:write topic:read", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/my-space"`)
			},
			wantErr: Error{
				StatusCode:     403,
				Type:           "about:blank",
				Title:          "Forbidden",
				RequiredScopes: []string{"page:write", "topic:read"},
			},
		},
		{
			name: "429はRetry-Afterを読む",
			resp: func() *http.Response {
				return jsonResponse(http.StatusTooManyRequests, "application/problem+json",
					`{"type":"about:blank","title":"Too Many Requests","status":429,"detail":null}`,
					"Retry-After", "120")
			},
			wantErr: Error{
				StatusCode:    429,
				Type:          "about:blank",
				Title:         "Too Many Requests",
				RetryAfter:    120 * time.Second,
				HasRetryAfter: true,
			},
		},
		{
			name: "Problem Detailsでない応答はステータスだけを持つ",
			resp: func() *http.Response {
				return jsonResponse(http.StatusBadGateway, "text/html", "<html>Bad Gateway</html>")
			},
			wantErr: Error{StatusCode: 502},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ft := newFakeTransport()
			ft.handle(http.MethodGet, testHost+"/api/v1/spaces/my-space/pages/12", func(*http.Request) *http.Response {
				return tt.resp()
			})
			c := newTestClient(t, ft, Options{Token: StaticToken("secret-token")})

			_, err := c.GetPageWithResponse(context.Background(), "my-space", 12, nil)

			var apiErr *Error
			if !errors.As(err, &apiErr) {
				t.Fatalf("*Errorが返りません: %v", err)
			}
			assertError(t, apiErr, &tt.wantErr)
		})
	}
}

func assertError(t *testing.T, got, want *Error) {
	t.Helper()

	if got.StatusCode != want.StatusCode || got.Type != want.Type || got.Title != want.Title || got.Detail != want.Detail ||
		got.RetryAfter != want.RetryAfter || got.HasRetryAfter != want.HasRetryAfter {
		t.Errorf("エラーが違います:\n実際  %+v\n期待 %+v", got, want)
	}
	if strings.Join(got.RequiredScopes, " ") != strings.Join(want.RequiredScopes, " ") {
		t.Errorf("要るスコープが違います: 実際 %v, 期待 %v", got.RequiredScopes, want.RequiredScopes)
	}
	if len(got.FieldErrors) != len(want.FieldErrors) {
		t.Fatalf("項目ごとの問題の数が違います: 実際 %+v, 期待 %+v", got.FieldErrors, want.FieldErrors)
	}
	for i := range want.FieldErrors {
		if got.FieldErrors[i] != want.FieldErrors[i] {
			t.Errorf("%d番目の項目の問題が違います: 実際 %+v, 期待 %+v", i, got.FieldErrors[i], want.FieldErrors[i])
		}
	}
}

func TestClient_BeforeWrite(t *testing.T) {
	t.Parallel()

	errStop := errors.New("送らない")

	ft := newFakeTransport()
	ft.handle(http.MethodGet, testHost+"/api/v1/spaces/my-space/pages/12", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "application/json", pageJSON, "ETag", `"v1"`)
	})
	ft.handle(http.MethodPost, testHost+"/api/v1/spaces/my-space/wikilink_previews", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "application/json", `{"links":[]}`)
	})

	var intercepted []string
	c := newTestClient(t, ft, Options{
		Token: StaticToken("secret-token"),
		BeforeWrite: func(req *http.Request) error {
			intercepted = append(intercepted, req.Method+" "+req.URL.Path)
			if req.Header.Get("Authorization") == "" {
				t.Error("フックが受け取る要求にAuthorizationが付いていません")
			}
			return errStop
		},
	})
	ctx := context.Background()

	// 読み取り専用のoperationは、POSTでもフックを通さずに送る。
	if _, err := c.GetPageWithResponse(ctx, "my-space", 12, nil); err != nil {
		t.Fatalf("読み取りの要求に失敗しました: %v", err)
	}
	if _, err := c.CreateWikilinkPreviewWithResponse(ctx, "my-space", apigen.NewWikilinkPreview{TopicNumber: 3, Body: "[[a]]"}); err != nil {
		t.Fatalf("読み取り専用のPOSTの要求に失敗しました: %v", err)
	}

	etag := `"v1"`
	_, err := c.DeleteDraftPageWithResponse(ctx, "my-space", 12, &apigen.DeleteDraftPageParams{IfMatch: &etag})
	if !errors.Is(err, errStop) {
		t.Fatalf("フックのエラーが返りません: %v", err)
	}

	if len(intercepted) != 1 || intercepted[0] != "DELETE /api/v1/spaces/my-space/pages/12/draft_page" {
		t.Errorf("フックを通った要求が違います: %v", intercepted)
	}
	if len(ft.requests) != 2 {
		t.Errorf("状態を変える要求が送られています: %d件送りました", len(ft.requests))
	}
}

type failingToken struct{}

func (failingToken) Token(context.Context) (string, error) {
	return "", errors.New("トークンを更新できません")
}

func TestClient_TokenError(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	c := newTestClient(t, ft, Options{Token: failingToken{}})

	_, err := c.GetPageWithResponse(context.Background(), "my-space", 12, nil)
	if err == nil || !strings.Contains(err.Error(), "トークンを更新できません") {
		t.Fatalf("トークンのエラーが返りません: %v", err)
	}
	if len(ft.requests) != 0 {
		t.Error("トークンを得られないのに要求を送っています")
	}
}

func TestNew_InvalidHost(t *testing.T) {
	t.Parallel()

	_, err := New(Options{Host: "http://example.com"})

	var inputErr *InputError
	if !errors.As(err, &inputErr) || inputErr.Reason != ReasonInsecureScheme {
		t.Fatalf("HTTPの接続先を受け付けています: %v", err)
	}
}

func TestCheckRedirect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{name: "HTTPSへは辿る", target: "https://example.com/api/v1/x"},
		{name: "ループバックのHTTPへは辿る", target: "http://127.0.0.1:3000/api/v1/x"},
		{name: "同じホストのHTTPへは辿らない", target: "http://example.com/api/v1/x", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(http.MethodGet, tt.target, nil)
			if err != nil {
				t.Fatal(err)
			}

			err = checkRedirect(req, nil)
			if errors.Is(err, ErrInsecureRedirect) != tt.wantErr || (!tt.wantErr && err != nil) {
				t.Errorf("リダイレクトの判定が違います: err=%v, 期待するエラー=%t", err, tt.wantErr)
			}
		})
	}
}

func TestClient_OpenAPIDescription(t *testing.T) {
	t.Parallel()
	spec, err := os.ReadFile("../../../go/api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ft := newFakeTransport()
	ft.handle(http.MethodGet, testHost+"/api/v1/openapi.yaml", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "application/yaml", string(spec))
	})
	c := newTestClient(t, ft, Options{})
	resp, err := c.GetOpenAPIDescriptionWithResponse(context.Background())
	if err != nil {
		t.Fatalf("OpenAPI記述を取得できません: %v", err)
	}
	if resp.YAML200 == nil || *resp.YAML200 != string(spec) || string(resp.Body) != string(spec) {
		t.Error("OpenAPI記述の生本文が保存されていません")
	}
}

func TestClient_Do(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodPatch, testHost+"/api/v1/spaces/my-space/pages/12", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "application/json", pageJSON)
	})
	c := newTestClient(t, ft, Options{
		Token:          StaticToken("secret-token"),
		UserAgent:      "wikino-cli/1.0.0 (linux; amd64)",
		AcceptLanguage: "ja",
	})

	header := http.Header{}
	header.Set("If-Match", `"v1"`)
	header.Set("Accept-Language", "en")
	resp, err := c.Do(context.Background(), http.MethodPatch, "/spaces/my-space/pages/12?x=1", strings.NewReader(`{"title":"a"}`), header)
	if err != nil {
		t.Fatalf("要求に失敗しました: %v", err)
	}
	_ = resp.Body.Close()

	req := ft.requests[0]
	if got := req.URL.String(); got != testHost+"/api/v1/spaces/my-space/pages/12?x=1" {
		t.Errorf("URLが違います: %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer secret-token" {
		t.Errorf("Authorizationが違います: %q", got)
	}
	if got := req.Header.Get("If-Match"); got != `"v1"` {
		t.Errorf("If-Matchが違います: %q", got)
	}
	// 呼び出し側が指定したAccept-Languageは上書きしない。
	if got := req.Header.Get("Accept-Language"); got != "en" {
		t.Errorf("Accept-Languageが違います: %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != "wikino-cli/1.0.0 (linux; amd64)" {
		t.Errorf("User-Agentが違います: %q", got)
	}
	if got := ft.bodies[0]; got != `{"title":"a"}` {
		t.Errorf("本文が違います: %q", got)
	}
}

func TestClient_DoErrors(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, testHost+"/api/v1/spaces/my-space/pages/99", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusNotFound, "application/problem+json", `{"type":"about:blank","title":"Not Found","status":404,"detail":"ページが見つかりません。"}`)
	})
	c := newTestClient(t, ft, Options{})

	_, err := c.Do(context.Background(), http.MethodGet, "/spaces/my-space/pages/99", nil, nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound || apiErr.Detail != "ページが見つかりません。" {
		t.Errorf("404の*Errorが返りません: %v", err)
	}

	// 基底の外を指すパスは送らない。
	_, err = c.Do(context.Background(), http.MethodGet, "/../oauth/token", nil, nil)
	var inputErr *InputError
	if !errors.As(err, &inputErr) || inputErr.Reason != ReasonDotSegment {
		t.Errorf("*InputErrorが返りません: %v", err)
	}
	if len(ft.requests) != 1 {
		t.Errorf("送った要求の数が違います: %d", len(ft.requests))
	}
}

func TestClient_DoBeforeWrite(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, testHost+"/api/v1/spaces/my-space/pages/12", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "application/json", pageJSON)
	})
	errStop := errors.New("止めた")
	var intercepted []string
	c := newTestClient(t, ft, Options{BeforeWrite: func(req *http.Request) error {
		intercepted = append(intercepted, req.Method+" "+req.URL.Path)
		return errStop
	}})
	ctx := context.Background()

	resp, err := c.Do(ctx, http.MethodGet, "/spaces/my-space/pages/12", nil, nil)
	if err != nil {
		t.Fatalf("読み取りの要求に失敗しました: %v", err)
	}
	_ = resp.Body.Close()

	// どのoperationにも当たらない要求も、状態を変えうるものとしてフックを通す。
	for _, path := range []string{"/spaces/my-space/pages/12/draft_page", "/spaces/my-space/unknown"} {
		if _, err := c.Do(ctx, http.MethodDelete, path, nil, nil); !errors.Is(err, errStop) {
			t.Errorf("%sがフックを通りません: %v", path, err)
		}
	}
	if len(intercepted) != 2 || len(ft.requests) != 1 {
		t.Errorf("フックを通った要求 = %v、送った要求の数 = %d", intercepted, len(ft.requests))
	}
}
