package storage_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/storage"
)

// helloMD5は "hello" のMD5をBase64にした値
const helloMD5 = "XUFAKrxLKna5cZ2REBfFkg=="

// serveFakeは、署名付きURL (サーバーの中の相対パス) へのリクエストをフェイクのストレージに渡す
func serveFake(fake *storage.FakeObjectStorage, method, target string, body string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	fake.ServeHTTP(rr, req)
	return rr
}

func TestFakeObjectStorage_ServeHTTP(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	putOptions := storage.PresignedPutOptions{ContentType: "image/png", ContentLength: 5, ContentMD5: helloMD5}

	t.Run("正常系: アップロードの署名付きURLで書き込み、取得の署名付きURLで読み出せる", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewServedFakeObjectStorage()
		putURL, err := fake.PresignedPutURL(ctx, "upload-key", 5*time.Minute, putOptions)
		if err != nil {
			t.Fatalf("PresignedPutURL()のエラー = %v", err)
		}
		if !strings.HasPrefix(putURL, storage.FakeObjectStoragePathPrefix+"upload-key?") {
			t.Fatalf("PresignedPutURL() = %q、期待値 = %sの下の相対パス", putURL, storage.FakeObjectStoragePathPrefix)
		}

		rr := serveFake(fake, http.MethodPut, putURL, "hello", map[string]string{"Content-Type": "image/png", "Content-MD5": helloMD5})
		if rr.Code != http.StatusOK {
			t.Fatalf("PUTのステータスコード = %d、期待値 = %d (本文 = %s)", rr.Code, http.StatusOK, rr.Body.String())
		}
		if exists, _ := fake.Exists(ctx, "upload-key"); !exists {
			t.Fatal("アップロードしたオブジェクトが無い")
		}

		getURL, err := fake.PresignedGetURL(ctx, "upload-key", 5*time.Minute, storage.PresignedGetOptions{
			ResponseContentType:        "application/octet-stream",
			ResponseContentDisposition: `attachment; filename="a.png"`,
		})
		if err != nil {
			t.Fatalf("PresignedGetURL()のエラー = %v", err)
		}
		rr = serveFake(fake, http.MethodGet, getURL, "", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("GETのステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
		}
		if got, _ := io.ReadAll(rr.Body); string(got) != "hello" {
			t.Errorf("取得した本文 = %q、期待値 = %q", got, "hello")
		}
		if got := rr.Header().Get("Content-Type"); got != "application/octet-stream" {
			t.Errorf("Content-Type = %q、期待値 = %q", got, "application/octet-stream")
		}
		if got := rr.Header().Get("Content-Disposition"); got != `attachment; filename="a.png"` {
			t.Errorf("Content-Disposition = %q、期待値 = %q", got, `attachment; filename="a.png"`)
		}
	})

	t.Run("正常系: 応答ヘッダーの指定が無ければ保持した種類で返す", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewServedFakeObjectStorage()
		fake.Put("stored-key", []byte("<svg/>"), "image/svg+xml")
		getURL, _ := fake.PresignedGetURL(ctx, "stored-key", time.Minute, storage.PresignedGetOptions{})
		rr := serveFake(fake, http.MethodGet, getURL, "", nil)
		if got := rr.Header().Get("Content-Type"); got != "image/svg+xml" {
			t.Errorf("Content-Type = %q、期待値 = %q", got, "image/svg+xml")
		}
	})

	for _, tc := range []struct {
		name     string
		body     string
		header   map[string]string
		wantCode int
	}{
		{name: "異常系: 種類が署名と違えば拒否する", body: "hello", header: map[string]string{"Content-Type": "image/jpeg", "Content-MD5": helloMD5}, wantCode: http.StatusForbidden},
		{name: "異常系: チェックサムのヘッダーが署名と違えば拒否する", body: "hello", header: map[string]string{"Content-Type": "image/png", "Content-MD5": "1B2M2Y8AsgTpgAmY7PhCfg=="}, wantCode: http.StatusForbidden},
		{name: "異常系: 大きさが署名と違えば拒否する", body: "hello!", header: map[string]string{"Content-Type": "image/png", "Content-MD5": helloMD5}, wantCode: http.StatusForbidden},
		{name: "異常系: 内容がチェックサムと違えば拒否する", body: "HELLO", header: map[string]string{"Content-Type": "image/png", "Content-MD5": helloMD5}, wantCode: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fake := storage.NewServedFakeObjectStorage()
			putURL, _ := fake.PresignedPutURL(ctx, "rejected-key", time.Minute, putOptions)
			rr := serveFake(fake, http.MethodPut, putURL, tc.body, tc.header)
			if rr.Code != tc.wantCode {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, tc.wantCode)
			}
			if exists, _ := fake.Exists(ctx, "rejected-key"); exists {
				t.Error("拒否したアップロードが書き込まれた")
			}
		})
	}

	t.Run("異常系: 本文サイズの制限を超えたら413で書き込まない", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewServedFakeObjectStorage()
		putURL, err := fake.PresignedPutURL(ctx, "too-large-key", time.Minute, putOptions)
		if err != nil {
			t.Fatalf("PresignedPutURL()のエラー = %v", err)
		}
		req := httptest.NewRequest(http.MethodPut, putURL, strings.NewReader("hello"))
		req.Header.Set("Content-Type", "image/png")
		req.Header.Set("Content-MD5", helloMD5)
		rr := httptest.NewRecorder()
		req.Body = http.MaxBytesReader(rr, req.Body, 4)
		fake.ServeHTTP(rr, req)

		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusRequestEntityTooLarge)
		}
		if exists, _ := fake.Exists(ctx, "too-large-key"); exists {
			t.Error("上限を超えたアップロードが書き込まれた")
		}
	})

	t.Run("異常系: 無いオブジェクトの取得は404", func(t *testing.T) {
		t.Parallel()

		rr := serveFake(storage.NewServedFakeObjectStorage(), http.MethodGet, storage.FakeObjectStoragePathPrefix+"missing", "", nil)
		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("異常系: 対応しないメソッドは405", func(t *testing.T) {
		t.Parallel()

		rr := serveFake(storage.NewServedFakeObjectStorage(), http.MethodDelete, storage.FakeObjectStoragePathPrefix+"key", "", nil)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusMethodNotAllowed)
		}
	})
}
