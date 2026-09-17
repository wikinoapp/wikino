package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimit_WithinLimit(t *testing.T) {
	t.Parallel()

	var receivedBody []byte
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ボディの読み込みに失敗: %v", err)
			return
		}
		receivedBody = b
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("a", 1024) // 1 KiB
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if string(receivedBody) != body {
		t.Errorf("ボディの長さ = %d、期待値 = %d", len(receivedBody), len(body))
	}
}

func TestBodyLimit_ExactLimit(t *testing.T) {
	t.Parallel()

	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			t.Errorf("ボディの読み込みに失敗: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("a", DefaultMaxBodyBytes)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
}

func TestBodyLimit_OverLimitReturns413(t *testing.T) {
	t.Parallel()

	called := false
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("a", DefaultMaxBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rr.Body.String(), "Request Entity Too Large") {
		t.Errorf("レスポンスボディに期待する文字列が含まれない: body=%q", rr.Body.String())
	}
	if called {
		t.Error("上限超過時は下流ハンドラーを呼び出すべきでない")
	}
}

// TestBodyLimit_ContentLengthOverLimitRejectsEarlyはContent-Lengthが上限を超えている場合に
// ボディを読み込まずに413を返すことを検証する。下流ハンドラーに到達しないこと・
// リクエストボディがまったく読まれていないこと(カウンタで検証)を確認する。
func TestBodyLimit_ContentLengthOverLimitRejectsEarly(t *testing.T) {
	t.Parallel()

	called := false
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	// 実際のボディは短いがContent-Lengthを上限超過で申告する。早期拒否が効いていれば
	// ボディの中身は読まれずに413が返る。
	body := &readCounter{src: strings.NewReader("dummy")}
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = DefaultMaxBodyBytes + 1
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rr.Body.String(), "Request Entity Too Large") {
		t.Errorf("レスポンスボディに期待する文字列が含まれない: body=%q", rr.Body.String())
	}
	if called {
		t.Error("Content-Length超過時は下流ハンドラーを呼び出すべきでない")
	}
	if body.reads > 0 {
		t.Errorf("Content-Lengthによる早期拒否時はボディを読むべきでない: reads=%d", body.reads)
	}
}

// readCounterはボディの読み込み回数をカウントするio.Reader。
// Content-Lengthでの早期拒否時にボディが読まれていないことを検証するために使う。
type readCounter struct {
	src   io.Reader
	reads int
}

func (r *readCounter) Read(p []byte) (int, error) {
	r.reads++
	return r.src.Read(p)
}

func TestBodyLimit_GetRequestPassesThrough(t *testing.T) {
	t.Parallel()

	called := false
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("ハンドラーが呼ばれていない")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
}

// TestBodyLimit_DownstreamCanReadFormValueは先読み後に下流ハンドラーで
// r.ParseForm / r.FormValueが引き続き機能することを検証する。
func TestBodyLimit_DownstreamCanReadFormValue(t *testing.T) {
	t.Parallel()

	var receivedValue string
	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedValue = r.FormValue("key")
		w.WriteHeader(http.StatusOK)
	}))

	body := "key=hello"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if receivedValue != "hello" {
		t.Errorf("フォーム値 = %q、期待値 = %q", receivedValue, "hello")
	}
}
