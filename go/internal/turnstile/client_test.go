package turnstile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerify_Success(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content-Typeがapplication/jsonであることを確認
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", r.Header.Get("Content-Type"))
		}

		// 成功レスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": true,
			"challenge_ts": "2024-01-01T00:00:00Z",
			"hostname": "wikino.app"
		}`))
	}))
	defer mockServer.Close()

	// クライアントを作成（モックサーバーのURLを使用）
	client := NewClient("test-secret-key")
	// siteverifyURLを書き換えることができないため、httpClientのTransportを使ってリダイレクト
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "test-token")

	// アサーション
	if err != nil {
		t.Errorf("Verify() error = %v, want nil", err)
	}
	if !success {
		t.Errorf("Verify() success = %v, want true", success)
	}
}

func TestVerify_Failure(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 失敗レスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": false}`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "invalid-token")

	// アサーション（検証失敗の場合はエラーが返る）
	if err == nil {
		t.Error("Verify() error = nil, want error")
	}
	if success {
		t.Errorf("Verify() success = %v, want false", success)
	}
	if !strings.Contains(err.Error(), "turnstile検証に失敗しました") {
		t.Errorf("Verify() error = %v, want error containing 'turnstile検証に失敗しました'", err)
	}
}

func TestVerify_FailureWithErrorCodes(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// エラーコード付きの失敗レスポンスを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"success": false,
			"error-codes": ["invalid-input-response", "timeout-or-duplicate"]
		}`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "invalid-token")

	// アサーション
	if err == nil {
		t.Error("Verify() error = nil, want error")
	}
	if success {
		t.Errorf("Verify() success = %v, want false", success)
	}
	if !strings.Contains(err.Error(), "エラーコード") {
		t.Errorf("Verify() error = %v, want error containing 'エラーコード'", err)
	}
}

func TestVerify_EmptyToken(t *testing.T) {
	t.Parallel()

	// クライアントを作成
	client := NewClient("test-secret-key")

	// Verifyメソッドをテスト（空トークン）
	ctx := context.Background()
	success, err := client.Verify(ctx, "")

	// アサーション
	if err == nil {
		t.Error("Verify() error = nil, want error")
	}
	if success {
		t.Errorf("Verify() success = %v, want false", success)
	}
	if !strings.Contains(err.Error(), "トークンが空です") {
		t.Errorf("Verify() error = %v, want error containing 'トークンが空です'", err)
	}
}

func TestVerify_EmptySecretKey(t *testing.T) {
	t.Parallel()

	// SecretKeyが空のクライアントを作成（テスト環境用）
	client := NewClient("")

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "any-token")

	// アサーション（SecretKeyが空の場合は常に成功を返す）
	if err != nil {
		t.Errorf("Verify() error = %v, want nil", err)
	}
	if !success {
		t.Errorf("Verify() success = %v, want true", success)
	}
}

func TestVerify_Timeout(t *testing.T) {
	t.Parallel()

	// コンテキストのタイムアウトを使ってタイムアウトをテスト
	// （HTTPクライアントのタイムアウト10秒より短いコンテキストを使用）
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 2秒待機（コンテキストのタイムアウト1秒より長い）
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// 1秒でタイムアウトするコンテキストを使用
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	success, err := client.Verify(ctx, "test-token")

	// アサーション
	if err == nil {
		t.Error("Verify() error = nil, want timeout error")
	}
	if success {
		t.Errorf("Verify() success = %v, want false", success)
	}
}

func TestVerify_InvalidJSON(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 無効なJSONを返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": invalid json`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "test-token")

	// アサーション
	if err == nil {
		t.Error("Verify() error = nil, want JSON decode error")
	}
	if success {
		t.Errorf("Verify() success = %v, want false", success)
	}
	if !strings.Contains(err.Error(), "JSONデコードに失敗しました") {
		t.Errorf("Verify() error = %v, want error containing 'JSONデコードに失敗しました'", err)
	}
}

func TestVerify_NonOKStatusCode(t *testing.T) {
	t.Parallel()

	// モックHTTPサーバーを作成
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 500エラーを返す
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	}))
	defer mockServer.Close()

	// クライアントを作成
	client := NewClient("test-secret-key")
	client.httpClient.Transport = &mockTransport{
		target: mockServer.URL,
	}

	// Verifyメソッドをテスト
	ctx := context.Background()
	success, err := client.Verify(ctx, "test-token")

	// アサーション
	if err == nil {
		t.Error("Verify() error = nil, want HTTP error")
	}
	if success {
		t.Errorf("Verify() success = %v, want false", success)
	}
	if !strings.Contains(err.Error(), "siteverify APIがエラーを返しました") {
		t.Errorf("Verify() error = %v, want error containing 'siteverify APIがエラーを返しました'", err)
	}
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	client := NewClient("my-secret-key")

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.secretKey != "my-secret-key" {
		t.Errorf("client.secretKey = %v, want my-secret-key", client.secretKey)
	}
	if client.httpClient == nil {
		t.Error("client.httpClient is nil")
	}
	if client.httpClient.Timeout != requestTimeout {
		t.Errorf("client.httpClient.Timeout = %v, want %v", client.httpClient.Timeout, requestTimeout)
	}
}

// mockTransport はHTTPリクエストをモックサーバーにリダイレクトするTransport
type mockTransport struct {
	target string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// リクエストURLをモックサーバーのURLに書き換え
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.target, "http://")
	req.URL.Path = ""
	return http.DefaultTransport.RoundTrip(req)
}
