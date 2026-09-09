package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// setupAuthTest はテスト用のAuthミドルウェアをセットアップする
func setupAuthTest(t *testing.T) (*Auth, *testutil.SessionBuilder, string) {
	t.Helper()

	_, tx := testutil.SetupTx(t)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()

	sessionBuilder := testutil.NewSessionBuilder(t, tx).
		WithUserID(userID)
	token := "test-auth-token-12345"
	sessionBuilder.WithToken(token).Build()

	// リポジトリを作成
	queries := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	// 設定を作成
	cfg := &config.Config{
		CookieDomain:    "localhost",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// セッションマネージャーを作成
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)

	// Authミドルウェアを作成
	auth := NewAuth(sessionMgr)

	return auth, sessionBuilder, token
}

func TestRequireAuth_認証済みの場合(t *testing.T) {
	t.Parallel()

	auth, _, token := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// コンテキストからユーザーを取得できることを確認
		user := UserFromContext(r.Context())
		if user == nil {
			t.Error("コンテキストにユーザーが設定されていない")
		}

		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: token,
	})
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if !handlerCalled {
		t.Error("次のハンドラーが呼び出されなかった")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRequireAuth_未認証の場合(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成（クッキーなし）
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if handlerCalled {
		t.Error("次のハンドラーが呼び出されるべきではない")
	}
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in?back=%2Fprotected" {
		t.Errorf("リダイレクト先が不正: got %s, want /sign_in?back=%%2Fprotected", location)
	}
}

func TestRequireAuth_無効なトークンの場合(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成（無効なトークン）
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "invalid-token",
	})
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if handlerCalled {
		t.Error("次のハンドラーが呼び出されるべきではない")
	}
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/sign_in?back=%2Fprotected" {
		t.Errorf("リダイレクト先が不正: got %s, want /sign_in?back=%%2Fprotected", location)
	}
}

// Verifies when the redirect to sign-in carries the original destination. Only GET and HEAD carry
// back; methods with side effects do not.
//
// [Ja] サインインへのリダイレクトに元の宛先を引き継ぐ条件を検証する。
// GET / HEAD だけが back を持ち、副作用のあるメソッドは持たない。
func TestRequireAuth_未認証時のbackパラメータ(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	pageNewTarget := "/s/example/topics/1/pages/new?title=%E3%83%A1%E3%83%A2&body=a+b"

	tests := []struct {
		name         string
		method       string
		target       string
		wantLocation string
	}{
		{
			name:         "GETはパスとクエリをbackに載せる",
			method:       http.MethodGet,
			target:       pageNewTarget,
			wantLocation: "/sign_in?back=" + url.QueryEscape(pageNewTarget),
		},
		{
			name:         "HEADもbackを載せる",
			method:       http.MethodHead,
			target:       "/protected",
			wantLocation: "/sign_in?back=%2Fprotected",
		},
		{
			name:         "POSTはbackを付けない",
			method:       http.MethodPost,
			target:       "/protected",
			wantLocation: "/sign_in",
		},
		{
			name:         "DELETEはbackを付けない",
			method:       http.MethodDelete,
			target:       "/protected",
			wantLocation: "/sign_in",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			rr := httptest.NewRecorder()

			auth.RequireAuth(nextHandler).ServeHTTP(rr, req)

			if rr.Code != http.StatusFound {
				t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusFound)
			}
			if location := rr.Header().Get("Location"); location != tt.wantLocation {
				t.Errorf("リダイレクト先が不正: got %s, want %s", location, tt.wantLocation)
			}
		})
	}
}

func TestRequireNoAuth_認証済みの場合(t *testing.T) {
	t.Parallel()

	auth, _, token := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: token,
	})
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireNoAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if handlerCalled {
		t.Error("次のハンドラーが呼び出されるべきではない")
	}
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusFound)
	}
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("リダイレクト先が不正: got %s, want /", location)
	}
}

func TestRequireNoAuth_未認証の場合(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成（クッキーなし）
	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.RequireNoAuth(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if !handlerCalled {
		t.Error("次のハンドラーが呼び出されなかった")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSetUser_認証済みの場合(t *testing.T) {
	t.Parallel()

	auth, _, token := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	var userFromCtx *model.User
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		userFromCtx = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: token,
	})
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.SetUser(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if !handlerCalled {
		t.Error("次のハンドラーが呼び出されなかった")
	}
	if userFromCtx == nil {
		t.Error("コンテキストにユーザーが設定されていない")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestSetUser_未認証の場合(t *testing.T) {
	t.Parallel()

	auth, _, _ := setupAuthTest(t)

	// テスト用のハンドラー
	handlerCalled := false
	var userFromCtx *model.User
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		userFromCtx = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	// リクエストを作成（クッキーなし）
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// ミドルウェアを実行
	auth.SetUser(nextHandler).ServeHTTP(rr, req)

	// アサーション
	if !handlerCalled {
		t.Error("次のハンドラーが呼び出されなかった")
	}
	if userFromCtx != nil {
		t.Error("未認証なのにコンテキストにユーザーが設定されている")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコードが不正: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestUserFromContext_ユーザーが設定されていない場合(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	user := UserFromContext(ctx)

	if user != nil {
		t.Error("ユーザーが設定されていないのにnilではない")
	}
}
