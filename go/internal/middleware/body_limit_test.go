package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/model"
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
	req.Header.Set("Accept-Language", "ja")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rr.Body.String(), "一度に送れる大きさは8MBまでです") {
		t.Errorf("レスポンスボディに上限の案内が含まれない: body=%.200q", rr.Body.String())
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
	req.Header.Set("Accept-Language", "ja")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(rr.Body.String(), "一度に送れる大きさは8MBまでです") {
		t.Errorf("レスポンスボディに上限の案内が含まれない: body=%.200q", rr.Body.String())
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

// TestBodyLimit_FitsPageBodyMaxLengthは、上限の文字数のページ本文が、どの送り方でも
// 送るときの上限で弾かれないことを確かめる。弾かれると、本文の上限を伝える422ではなく
// 理由の分からない413になるため
func TestBodyLimit_FitsPageBodyMaxLength(t *testing.T) {
	t.Parallel()

	// JSONでは非BMP文字をASCIIエスケープするとサロゲートペア (12バイト) になる。
	// multipart (1字最大4バイト) はこれより小さい。タイトル (200字) やほかの項目の分として64KiBを見込む
	worst := model.PageBodyMaxLength*12 + 64<<10
	if DefaultMaxBodyBytes < worst {
		t.Errorf("DefaultMaxBodyBytes = %d、期待値 = %d以上", DefaultMaxBodyBytes, worst)
	}

	// GoのJSONエンコーダーがエスケープする文字だけで書いた本文も上限に収まることを確かめる
	encoded, err := json.Marshal(map[string]string{"body": strings.Repeat("<", model.PageBodyMaxLength)})
	if err != nil {
		t.Fatalf("JSONへの変換に失敗: %v", err)
	}
	if len(encoded) > DefaultMaxBodyBytes {
		t.Errorf("上限の文字数の本文のJSONのバイト数 = %d、期待値 = %d以下", len(encoded), DefaultMaxBodyBytes)
	}

	// すべての非ASCII文字をエスケープするJSONエンコーダーの送り方でも、本文を下流で読めることを確かめる
	escaped := `{"body":"` + strings.Repeat(`\ud83d\ude00`, model.PageBodyMaxLength) + `"}`
	tests := []struct {
		name          string
		contentLength int64
	}{
		{name: "サロゲートペアのJSONでContent-Lengthが既知", contentLength: int64(len(escaped))},
		{name: "サロゲートペアのJSONでContent-Lengthが不明", contentLength: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewBodyLimit(apierror.NewWriter("https://example.com"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload struct {
					Body string `json:"body"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("下流ハンドラーでJSONを読めない: %v", err)
				}
				if got := utf8.RuneCountInString(payload.Body); got != model.PageBodyMaxLength {
					t.Errorf("下流ハンドラーで読んだ本文の文字数 = %d、期待値 = %d", got, model.PageBodyMaxLength)
				}
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(escaped))
			req.ContentLength = tt.contentLength
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコード = %d、期待値 = %d (本文: %.200s)", rr.Code, http.StatusOK, rr.Body.String())
			}
		})
	}
}

func TestBodyLimit_OverLimitPageInEnglish(t *testing.T) {
	t.Parallel()

	handler := BodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("dummy"))
	req.ContentLength = DefaultMaxBodyBytes + 1
	req.Header.Set("Accept-Language", "en")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q、期待値 = text/html; charset=utf-8", got)
	}
	body := rr.Body.String()
	for _, want := range []string{`<html lang="en">`, "You can send up to 8MB at a time."} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %qが含まれていない", want)
		}
	}
}

// TestBodyLimit_APIUnreadableBodyは、APIのパスで本文を読めないときに、利用者の言語の説明で400を返すことを確かめる
func TestBodyLimit_APIUnreadableBody(t *testing.T) {
	t.Parallel()

	handler := NewBodyLimit(apierror.NewWriter("https://example.com"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", errReader{})
	req.Header.Set("Accept-Language", "ja")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "リクエスト本文を読み取れません") {
		t.Errorf("レスポンスに説明が含まれていない: %s", rr.Body.String())
	}
}

// errReaderは読み込みに必ず失敗するio.Reader
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("読み込みに失敗")
}
