package apiclient

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const testDownloadURL = "https://storage.example.com/attachments/key?X-Amz-Signature=secret-signature"

func TestClient_Download(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, "https://storage.example.com/attachments/key", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "image/png", "PNGDATA")
	})
	c := newTestClient(t, ft, Options{
		Token:          StaticToken("secret-token"),
		UserAgent:      "wikino-cli/1.0.0 (linux; amd64)",
		AcceptLanguage: "ja",
	})

	resp, err := c.Download(context.Background(), testDownloadURL)
	if err != nil {
		t.Fatalf("取得に失敗しました: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "PNGDATA" {
		t.Errorf("本文が違います: %q", body)
	}

	req := ft.requests[0]
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("署名付きURLへの要求にAuthorizationが付いています: %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != "wikino-cli/1.0.0 (linux; amd64)" {
		t.Errorf("User-Agentが違います: %q", got)
	}
	if got := req.URL.Query().Get("X-Amz-Signature"); got != "secret-signature" {
		t.Errorf("署名のクエリが送られていません: %q", got)
	}
}

func TestClient_DownloadRejectsInsecureURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{name: "ループバックでないHTTP", url: "http://storage.example.com/key"},
		{name: "HTTP以外のスキーム", url: "file:///etc/passwd"},
		{name: "ホストの無いURL", url: "/attachments/key"},
		{name: "利用者の情報を含むURL", url: "https://user:pass@storage.example.com/key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ft := newFakeTransport()
			c := newTestClient(t, ft, Options{})

			_, err := c.Download(context.Background(), tt.url)
			if !errors.Is(err, ErrInsecureDownloadURL) {
				t.Errorf("ErrInsecureDownloadURLを期待しましたが %v でした", err)
			}
			if len(ft.requests) != 0 {
				t.Errorf("要求を送っています: %d件", len(ft.requests))
			}
		})
	}
}

func TestClient_DownloadLoopback(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, "http://127.0.0.1:9000/key", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "text/plain", "ok")
	})
	c := newTestClient(t, ft, Options{})

	resp, err := c.Download(context.Background(), "http://127.0.0.1:9000/key")
	if err != nil {
		t.Fatalf("ループバックのHTTPからの取得に失敗しました: %v", err)
	}
	_ = resp.Body.Close()
}

func TestClient_DownloadError(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, "https://storage.example.com/attachments/key", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusForbidden, "application/xml", "<Error><Code>AccessDenied</Code></Error>")
	})
	c := newTestClient(t, ft, Options{})

	_, err := c.Download(context.Background(), testDownloadURL)
	var downloadErr *DownloadError
	if !errors.As(err, &downloadErr) || downloadErr.StatusCode != http.StatusForbidden {
		t.Fatalf("403の *DownloadError を期待しましたが %v でした", err)
	}
	if strings.Contains(err.Error(), "secret-signature") {
		t.Errorf("エラーに署名が含まれています: %v", err)
	}
}

func TestClient_DownloadNetworkErrorRedactsSignature(t *testing.T) {
	t.Parallel()

	// 登録していないURLへの要求は、トランスポートが通信の失敗を返す。
	ft := newFakeTransport()
	c := newTestClient(t, ft, Options{})

	_, err := c.Download(context.Background(), testDownloadURL+"#fragment")
	if err == nil {
		t.Fatal("通信の失敗を期待しましたが成功しました")
	}
	if strings.Contains(err.Error(), "secret-signature") || strings.Contains(err.Error(), "fragment") {
		t.Errorf("エラーに署名付きURLのクエリが含まれています: %v", err)
	}
	var networkErr *DownloadNetworkError
	if !errors.As(err, &networkErr) {
		t.Fatalf("通信の失敗の型が違います: %T", err)
	}
}

func TestClient_DownloadRedirect(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name         string
		location     string
		wantError    bool
		wantInsecure bool
	}{
		{name: "正常なリダイレクト", location: "https://storage.example.com/final?X-Amz-Signature=secret-signature"},
		{name: "署名を含む不正なLocation", location: "https://storage.example.com/%zz?X-Amz-Signature=secret-signature", wantError: true},
		{name: "ループバックでないHTTPへのリダイレクト", location: "http://storage.example.com/final?X-Amz-Signature=secret-signature", wantError: true, wantInsecure: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ft := newFakeTransport()
			ft.handle(http.MethodGet, "https://storage.example.com/attachments/key", func(*http.Request) *http.Response {
				return jsonResponse(http.StatusFound, "text/plain", "", "Location", tt.location)
			})
			ft.handle(http.MethodGet, "https://storage.example.com/final", func(*http.Request) *http.Response {
				return jsonResponse(http.StatusOK, "text/plain", "downloaded")
			})
			c, err := New(Options{Host: testHost, HTTPClient: NewHTTPClient(ft)})
			if err != nil {
				t.Fatal(err)
			}
			resp, err := c.Download(context.Background(), testDownloadURL)
			if tt.wantError {
				var networkErr *DownloadNetworkError
				if !errors.As(err, &networkErr) {
					t.Fatalf("通信の失敗を期待しました: %v", err)
				}
				if strings.Contains(err.Error(), "secret-signature") {
					t.Fatalf("署名が表示されます: %v", err)
				}
				if tt.wantInsecure {
					if !errors.Is(err, ErrInsecureRedirect) {
						t.Errorf("ErrInsecureRedirectを期待しました: %v", err)
					}
					if len(ft.requests) != 1 {
						t.Errorf("リダイレクト先へ要求を送っています: %d件", len(ft.requests))
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			body, err := io.ReadAll(resp.Body)
			if err != nil || string(body) != "downloaded" {
				t.Fatalf("取得した本文 = %q、エラー = %v", body, err)
			}
		})
	}
}

// downloadErrorDoerは、署名付きURLを含む通信エラーを返す。
type downloadErrorDoer struct{ err error }

func (d downloadErrorDoer) Do(*http.Request) (*http.Response, error) {
	return nil, &url.Error{Op: "Get", URL: testDownloadURL, Err: d.err}
}

func TestClient_DownloadNetworkErrorCause(t *testing.T) {
	t.Parallel()
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, &net.DNSError{Err: "timeout", IsTimeout: true}} {
		c, err := New(Options{Host: testHost, HTTPClient: downloadErrorDoer{err: cause}})
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Download(context.Background(), testDownloadURL)
		if !errors.Is(err, cause) {
			t.Fatalf("原因が保持されていません: %v", err)
		}
		var urlErr *url.Error
		if !errors.As(err, &urlErr) {
			t.Fatal("通信の原因の型が失われています")
		}
		if strings.Contains(err.Error(), "secret-signature") {
			t.Fatalf("署名が表示されます: %v", err)
		}
		if timeout, ok := cause.(net.Error); ok && timeout.Timeout() {
			var netErr net.Error
			if !errors.As(err, &netErr) || !netErr.Timeout() {
				t.Fatal("タイムアウトを判定できません")
			}
		}
	}
}

// closeRecordingBodyは、成功以外の応答の本文が閉じられたかを記録する。
type closeRecordingBody struct {
	io.Reader
	closed bool
}

func (b *closeRecordingBody) Close() error { b.closed = true; return nil }

func TestClient_DownloadRejectsNonSuccess(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusFound, http.StatusNotModified, http.StatusForbidden} {
		ft := newFakeTransport()
		body := &closeRecordingBody{Reader: strings.NewReader("not an attachment")}
		ft.handle(http.MethodGet, "https://storage.example.com/attachments/key", func(*http.Request) *http.Response {
			return &http.Response{StatusCode: status, Header: http.Header{}, Body: body}
		})
		c := newTestClient(t, ft, Options{})
		_, err := c.Download(context.Background(), testDownloadURL)
		var downloadErr *DownloadError
		if !errors.As(err, &downloadErr) || downloadErr.StatusCode != status {
			t.Fatalf("ステータス %d の失敗を期待しました: %v", status, err)
		}
		if !body.closed {
			t.Fatalf("ステータス %d の本文が閉じられていません", status)
		}
	}
}

// failingReaderは、本文の読み取りの途中で通信が切れたことを再現する。
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestClient_DownloadBodyReadError(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, "https://storage.example.com/attachments/key", func(*http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(io.MultiReader(strings.NewReader("PNG"), failingReader{})),
		}
	})
	c := newTestClient(t, ft, Options{})

	resp, err := c.Download(context.Background(), testDownloadURL)
	if err != nil {
		t.Fatalf("応答の受け取りに失敗しました: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if string(body) != "PNG" {
		t.Errorf("失敗までに読んだ本文が違います: %q", body)
	}
	var networkErr *DownloadNetworkError
	if !errors.As(err, &networkErr) {
		t.Fatalf("本文の読み取りの失敗を *DownloadNetworkError で期待しましたが %v でした", err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("原因が保持されていません: %v", err)
	}
}

func TestClient_DownloadBodyEOF(t *testing.T) {
	t.Parallel()

	ft := newFakeTransport()
	ft.handle(http.MethodGet, "https://storage.example.com/attachments/key", func(*http.Request) *http.Response {
		return jsonResponse(http.StatusOK, "image/png", "PNGDATA")
	})
	c := newTestClient(t, ft, Options{})

	resp, err := c.Download(context.Background(), testDownloadURL)
	if err != nil {
		t.Fatalf("取得に失敗しました: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 読み終えたことはio.EOFのまま返し、io.Copyが失敗として扱わないようにする。
	var buf strings.Builder
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		t.Fatalf("読み終えたのに失敗しました: %v", err)
	}
	if buf.String() != "PNGDATA" {
		t.Errorf("本文が違います: %q", buf.String())
	}
}
