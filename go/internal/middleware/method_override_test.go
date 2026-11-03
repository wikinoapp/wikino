package middleware

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMethodOverride(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		method         string
		overrideMethod string
		expectedMethod string
	}{
		{
			name:           "POSTリクエストでDELETEに上書き",
			method:         http.MethodPost,
			overrideMethod: "DELETE",
			expectedMethod: http.MethodDelete,
		},
		{
			name:           "POSTリクエストでPATCHに上書き",
			method:         http.MethodPost,
			overrideMethod: "PATCH",
			expectedMethod: http.MethodPatch,
		},
		{
			name:           "POSTリクエストでPUTに上書き",
			method:         http.MethodPost,
			overrideMethod: "PUT",
			expectedMethod: http.MethodPut,
		},
		{
			name:           "POSTリクエストで小文字のdeleteに上書き",
			method:         http.MethodPost,
			overrideMethod: "delete",
			expectedMethod: http.MethodDelete,
		},
		{
			name:           "POSTリクエストで小文字のpatchに上書き",
			method:         http.MethodPost,
			overrideMethod: "patch",
			expectedMethod: http.MethodPatch,
		},
		{
			name:           "_methodパラメータがない場合はそのまま",
			method:         http.MethodPost,
			overrideMethod: "",
			expectedMethod: http.MethodPost,
		},
		{
			name:           "GETリクエストは上書きされない",
			method:         http.MethodGet,
			overrideMethod: "DELETE",
			expectedMethod: http.MethodGet,
		},
		{
			name:           "サポートされていないメソッドは無視される",
			method:         http.MethodPost,
			overrideMethod: "OPTIONS",
			expectedMethod: http.MethodPost,
		},
		{
			name:           "HEADメソッドは上書きされない",
			method:         http.MethodPost,
			overrideMethod: "HEAD",
			expectedMethod: http.MethodPost,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var receivedMethod string
			handler := MethodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))

			// リクエストを作成
			var req *http.Request
			if tc.method == http.MethodPost && tc.overrideMethod != "" {
				// フォームデータとして_methodを送信
				form := url.Values{}
				form.Set("_method", tc.overrideMethod)
				req = httptest.NewRequest(tc.method, "/test", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else if tc.method == http.MethodGet && tc.overrideMethod != "" {
				// GETリクエストでも_methodパラメータをクエリに含める
				req = httptest.NewRequest(tc.method, "/test?_method="+tc.overrideMethod, nil)
			} else {
				req = httptest.NewRequest(tc.method, "/test", nil)
				if tc.method == http.MethodPost {
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if receivedMethod != tc.expectedMethod {
				t.Errorf("メソッド = %q、期待値 = %q", receivedMethod, tc.expectedMethod)
			}
		})
	}
}

func TestMethodOverride_WithOtherFormData(t *testing.T) {
	t.Parallel()

	var receivedMethod string
	var receivedEmail string
	handler := MethodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		// フォームデータが正常に読み取れることを確認
		receivedEmail = r.PostFormValue("email")
		w.WriteHeader(http.StatusOK)
	}))

	// 他のフォームデータと一緒に_methodを送信
	form := url.Values{}
	form.Set("_method", "DELETE")
	form.Set("email", "test@example.com")
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if receivedMethod != http.MethodDelete {
		t.Errorf("メソッド = %q、期待値 = %q", receivedMethod, http.MethodDelete)
	}

	if receivedEmail != "test@example.com" {
		t.Errorf("email = %q、期待値 = %q", receivedEmail, "test@example.com")
	}
}

// APIのパスには適用せず、フォームとして本文を読み込まない
func TestMethodOverride_SkipsAPIPath(t *testing.T) {
	t.Parallel()

	var receivedMethod string
	var receivedBody string
	handler := MethodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))

	form := url.Values{}
	form.Set("_method", "DELETE")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/example/pages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if receivedMethod != http.MethodPost {
		t.Errorf("メソッド = %q、期待値 = %q", receivedMethod, http.MethodPost)
	}
	if receivedBody != form.Encode() {
		t.Errorf("本文 = %q、期待値 = %q", receivedBody, form.Encode())
	}
}

// multipart/form-dataで送ったフォームでも_methodを読み、後段のミドルウェアとハンドラーが
// ほかの項目 (CSRFトークンなど) をr.FormValueで読めることを確かめる
func TestMethodOverride_MultipartForm(t *testing.T) {
	t.Parallel()

	var receivedMethod, receivedToken, receivedBody string
	handler := MethodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedToken = r.FormValue("csrf_token")
		receivedBody = r.FormValue("body")
		w.WriteHeader(http.StatusOK)
	}))

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for key, value := range map[string]string{"_method": "PATCH", "csrf_token": "token", "body": "本文"} {
		if err := mw.WriteField(key, value); err != nil {
			t.Fatalf("フォームの項目の書き込みに失敗: %v", err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("フォームの書き込みに失敗: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/test", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if receivedMethod != http.MethodPatch {
		t.Errorf("メソッド = %q、期待値 = %q", receivedMethod, http.MethodPatch)
	}
	if receivedToken != "token" {
		t.Errorf("csrf_token = %q、期待値 = %q", receivedToken, "token")
	}
	if receivedBody != "本文" {
		t.Errorf("body = %q、期待値 = %q", receivedBody, "本文")
	}
}
