package main

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // テストでアップロードする内容のContent-MD5を作る
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/welcome"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestWelcomePageDoesNotQueryFeatureFlags(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	counter := testutil.NewQueryCounter(tx)
	queries := counter.Queries()
	cfg := &config.Config{Env: "test", Domain: "example.test"}
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	r := newHTTPRouter(cfg, apierror.NewWriter(cfg.AppURL()), flashMgr)

	sessionMgr := session.NewManager(repository.NewUserRepository(queries), repository.NewUserSessionRepository(queries), cfg)
	authMiddleware := middleware.NewAuth(sessionMgr)
	welcomeHandler := welcome.NewHandler(cfg, flashMgr)
	r.With(authMiddleware.SetUser, middleware.SentryUserContext, middleware.TimeZone).Get("/", welcomeHandler.Show)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `<meta name="color-scheme" content="light dark">`) {
		t.Error("トップページに配色の宣言がない")
	}
	if got := counter.CountContaining("feature_flags"); got != 0 {
		t.Errorf("フィーチャーフラグの照会回数 = %d、期待値 = 0", got)
	}
}

// TestHTTPRouter_NotFoundは、ルートの無いパスと、ルートはあるが対応していないメソッドに、404ページを返すことを確かめる。
// 公開Web APIのパスは、ルートのルーターの下でもProblem Detailsの404・405のまま変わらない
func TestHTTPRouter_NotFound(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Env: "test", Domain: "example.test"}
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)
	r := newHTTPRouter(cfg, apierror.NewWriter(cfg.AppURL()), flashMgr)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/example", ok)
	r.Get("/value/{value}", ok)
	r.Post("/example/submit", ok)
	r.Delete("/example/session", ok)
	auth := middleware.NewAuth(session.NewManager(nil, nil, cfg))
	r.With(auth.RequireAuth).Route("/namespace", func(r chi.Router) {
		r.Get("/", ok)
	})
	mountTestAPIRouter(t, r, newTestAPIServer(), testAPITokenAuthenticator{}, exceededAPIRateLimiter{})

	const csrfToken = "test_csrf_token_12345678901234567890123456789012"
	tests := []struct {
		name        string
		method      string
		target      string
		form        string
		withCSRF    bool
		wantStatus  int
		wantProblem bool
	}{
		{name: "ルートのあるパスは200", method: http.MethodGet, target: "/example", wantStatus: http.StatusOK},
		{name: "エスケープされたパスもchiと同じ方法で照合する", method: http.MethodGet, target: "/value/a%2Fb", wantStatus: http.StatusOK},
		{name: "ルートの無いパスは404ページ", method: http.MethodGet, target: "/unknown", wantStatus: http.StatusNotFound},
		{name: "末尾にスラッシュを付けたパスは404ページ", method: http.MethodGet, target: "/example/", wantStatus: http.StatusNotFound},
		{name: "トークンの無い未登録のPOSTは404ページ", method: http.MethodPost, target: "/unknown", wantStatus: http.StatusNotFound},
		{name: "GETだけのルートへのトークンの無いPOSTは404ページ", method: http.MethodPost, target: "/example", wantStatus: http.StatusNotFound},
		{name: "POSTだけのルートへのGETは404ページ", method: http.MethodGet, target: "/example/submit", wantStatus: http.StatusNotFound},
		{name: "DELETEだけのルートへのGETは404ページ", method: http.MethodGet, target: "/example/session", wantStatus: http.StatusNotFound},
		{name: "未ログインでもサブルーターの未登録パスは404ページ", method: http.MethodGet, target: "/namespace/unknown", wantStatus: http.StatusNotFound},
		{name: "未ログインでもサブルーターの未対応メソッドは404ページ", method: http.MethodDelete, target: "/namespace", wantStatus: http.StatusNotFound},
		{name: "末尾スラッシュ付きのサブルーターの未対応メソッドも404ページ", method: http.MethodHead, target: "/namespace/", wantStatus: http.StatusNotFound},
		{name: "認証必須の登録済みルートはサインインに転送", method: http.MethodGet, target: "/namespace", wantStatus: http.StatusFound},
		{name: "末尾スラッシュ付きの登録済みルートもサインインに転送", method: http.MethodGet, target: "/namespace/", wantStatus: http.StatusFound},
		{name: "メソッド変換で一致するルートにはCSRFを検証する", method: http.MethodPost, target: "/example/session", form: "_method=DELETE", wantStatus: http.StatusForbidden},
		{name: "メソッド変換とCSRF検証を通るルートは200", method: http.MethodPost, target: "/example/session", form: "_method=DELETE", withCSRF: true, wantStatus: http.StatusOK},
		{name: "APIのルートの無いパスはProblem Detailsの404", method: http.MethodGet, target: "/api/v1/unknown", wantStatus: http.StatusNotFound, wantProblem: true},
		{name: "APIのルートへの別のメソッドはProblem Detailsの405", method: http.MethodPost, target: "/api/v1/openapi.yaml", wantStatus: http.StatusMethodNotAllowed, wantProblem: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, strings.NewReader(tt.form))
			if tt.form != "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if tt.withCSRF {
				req.Header.Set("X-CSRF-Token", csrfToken)
				req.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: csrfToken})
			}
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if tt.wantProblem {
				assertProblem(t, rr, tt.wantStatus)
			} else if rr.Code != tt.wantStatus {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
			if !tt.wantProblem && tt.wantStatus == http.StatusNotFound && !strings.Contains(rr.Body.String(), "<title>ページが見つかりませんでした") {
				t.Errorf("本文 = %q、404ページを期待", rr.Body.String())
			}
			for _, cookie := range rr.Result().Cookies() {
				if cookie.Name == "device_token" {
					t.Errorf("device_tokenのSet-Cookieが付いている: %v", cookie)
				}
			}
		})
	}
}

// TestNewObjectStorage_TestE2EStorageは、フェイクのストレージとその送受信を受けるハンドラーを、e2eのサーバーでだけ組み立てることを確かめる
func TestNewObjectStorage_TestE2EStorage(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		env      string
		wantFake bool
	}{
		{env: "test_e2e", wantFake: true},
		{env: "test", wantFake: false},
		{env: "dev", wantFake: false},
		{env: "prod", wantFake: false},
	} {
		t.Run(tc.env, func(t *testing.T) {
			t.Parallel()

			// バケットの設定を空にし、e2eのサーバー以外ではストレージを組み立てないようにする
			objectStorage, handler := newObjectStorage(&config.Config{Env: tc.env})
			if got := handler != nil; got != tc.wantFake {
				t.Errorf("送受信を受けるハンドラーがある = %v、期待値 = %v", got, tc.wantFake)
			}
			_, isFake := objectStorage.(*storage.FakeObjectStorage)
			if isFake != tc.wantFake {
				t.Errorf("フェイクのストレージを使う = %v、期待値 = %v", isFake, tc.wantFake)
			}
		})
	}
}

// TestRequestBodyLimitは、e2eのフェイクへのPUTだけが種類ごとの上限まで通り、ほかの経路では8 MiBの制限が保たれることを確かめる
func TestRequestBodyLimit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		env           string
		method        string
		path          string
		contentType   string
		byteSize      int
		unknownLength bool
		wantStatus    int
	}{
		{name: "e2eのテキストは9 MiBでも送受信できる", env: "test_e2e", method: http.MethodPut, contentType: "text/plain", byteSize: 9 << 20, wantStatus: http.StatusOK},
		{name: "e2eの画像は上限の10 MiBまで送受信できる", env: "test_e2e", method: http.MethodPut, contentType: "image/png", byteSize: 10 << 20, wantStatus: http.StatusOK},
		{name: "大きさ不明の画像も10 MiBを超えれば413", env: "test_e2e", method: http.MethodPut, contentType: "image/png", byteSize: (10 << 20) + 1, unknownLength: true, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "通常のテスト環境は8 MiBのまま", env: "test", method: http.MethodPut, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "開発環境は8 MiBのまま", env: "dev", method: http.MethodPut, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "本番環境は8 MiBのまま", env: "prod", method: http.MethodPut, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "e2eでも通常の経路は8 MiBのまま", env: "test_e2e", method: http.MethodPost, path: "/regular", wantStatus: http.StatusRequestEntityTooLarge},
		{name: "e2eでもフェイクへのPOSTは8 MiBのまま", env: "test_e2e", method: http.MethodPost, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "e2eでも似た名前の経路は8 MiBのまま", env: "test_e2e", method: http.MethodPut, path: "/_test/storage-other/key", wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// 大きな本文を扱うため、ケース同士は並列にしない
			cfg := &config.Config{Env: tc.env}
			_, handler := newObjectStorage(cfg)
			r := chi.NewRouter()
			r.Use(newRequestBodyLimit(cfg, nil))
			r.Use(middleware.NewCSRF(cfg).Middleware)
			mountTestE2EStorage(r, handler)
			r.Post("/regular", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

			byteSize := tc.byteSize
			if byteSize == 0 {
				byteSize = 9 << 20
			}
			contentType := tc.contentType
			if contentType == "" {
				contentType = "text/plain"
			}
			body := bytes.Repeat([]byte("x"), byteSize)
			digest := md5.Sum(body) //nolint:gosec // アップロードのContent-MD5を作る
			checksum := base64.StdEncoding.EncodeToString(digest[:])
			// ルートを登録しない環境でも同じURLで試し、通常の本文制限が先に働くことを確かめる
			signer := storage.NewServedFakeObjectStorage()
			putURL, err := signer.PresignedPutURL(context.Background(), "body-limit-key", time.Minute, storage.PresignedPutOptions{
				ContentType: contentType, ContentLength: int64(byteSize), ContentMD5: checksum,
			})
			if err != nil {
				t.Fatalf("アップロードURLの生成に失敗: %v", err)
			}
			if tc.path != "" {
				putURL = tc.path
			}
			req := httptest.NewRequest(tc.method, putURL, bytes.NewReader(body))
			req.Header.Set("Content-Type", contentType)
			req.Header.Set("Content-MD5", checksum)
			if tc.unknownLength {
				req.ContentLength = -1
			}
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, tc.wantStatus)
			}
			if tc.wantStatus != http.StatusOK {
				return
			}

			getURL, err := signer.PresignedGetURL(context.Background(), "body-limit-key", time.Minute, storage.PresignedGetOptions{})
			if err != nil {
				t.Fatalf("取得URLの生成に失敗: %v", err)
			}
			getResponse := httptest.NewRecorder()
			r.ServeHTTP(getResponse, httptest.NewRequest(http.MethodGet, getURL, nil))
			if getResponse.Code != http.StatusOK || !bytes.Equal(getResponse.Body.Bytes(), body) {
				t.Errorf("取得結果 = (%d, %dバイト)、期待値 = (200, アップロードした%dバイトと同じ内容)", getResponse.Code, getResponse.Body.Len(), len(body))
			}
		})
	}
}

// TestMountTestE2EStorageは、フェイクのストレージの送受信を受けるルートが、e2eのサーバーでだけ登録されることを確かめる
func TestMountTestE2EStorage(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		env        string
		wantStatus int
	}{
		{env: "test_e2e", wantStatus: http.StatusOK},
		{env: "test", wantStatus: http.StatusNotFound},
		{env: "dev", wantStatus: http.StatusNotFound},
		{env: "prod", wantStatus: http.StatusNotFound},
	} {
		t.Run(tc.env, func(t *testing.T) {
			t.Parallel()

			objectStorage, handler := newObjectStorage(&config.Config{Env: tc.env})
			r := chi.NewRouter()
			mountTestE2EStorage(r, handler)

			target := storage.FakeObjectStoragePathPrefix + "key"
			if fake, ok := objectStorage.(*storage.FakeObjectStorage); ok {
				fake.Put("key", []byte("hello"), "text/plain")
			}
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))

			if rr.Code != tc.wantStatus {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusOK && !strings.Contains(rr.Body.String(), "hello") {
				t.Errorf("本文 = %q、期待値 = %q", rr.Body.String(), "hello")
			}
		})
	}
}
