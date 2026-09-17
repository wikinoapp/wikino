package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGenerateSecureToken(t *testing.T) {
	t.Parallel()

	t.Run("トークンが32文字のBase64 URL-safeエンコード文字列であること", func(t *testing.T) {
		t.Parallel()

		token, err := session.GenerateSecureToken()
		if err != nil {
			t.Fatalf("トークン生成に失敗: %v", err)
		}

		// 24バイト -> Base64エンコードで32文字
		if len(token) != 32 {
			t.Errorf("トークンの長さ = %d、期待値 = 32", len(token))
		}
	})

	t.Run("生成されるトークンが毎回異なること", func(t *testing.T) {
		t.Parallel()

		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			token, err := session.GenerateSecureToken()
			if err != nil {
				t.Fatalf("トークン生成に失敗: %v", err)
			}
			if tokens[token] {
				t.Errorf("重複するトークンが生成された: %s", token)
			}
			tokens[token] = true
		}
	})
}

func TestManager_GetCurrentUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)

	cfg := &config.Config{
		CookieDomain:    "localhost",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	manager := session.NewManager(userRepo, userSessionRepo, cfg)

	t.Run("有効なセッションがある場合、ユーザーを返すこと", func(t *testing.T) {
		// ユーザーを作成
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("session-test@example.com").
			WithAtname("sessiontestuser").
			Build()

		// セッションを作成
		token, err := session.GenerateSecureToken()
		if err != nil {
			t.Fatalf("トークン生成に失敗: %v", err)
		}

		_, err = userSessionRepo.Create(context.Background(), repository.CreateInput{
			UserID:     userID,
			Token:      token,
			IPAddress:  "127.0.0.1",
			UserAgent:  "TestAgent",
			SignedInAt: time.Now(),
		})
		if err != nil {
			t.Fatalf("セッション作成に失敗: %v", err)
		}

		// リクエストにCookieを設定
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  session.CookieName,
			Value: token,
		})

		user, err := manager.GetCurrentUser(context.Background(), req)
		if err != nil {
			t.Fatalf("GetCurrentUserに失敗: %v", err)
		}

		if user == nil {
			t.Fatal("ユーザーがnilでした")
		}

		if user.ID != userID {
			t.Errorf("ユーザーID = %s、期待値 = %s", user.ID, userID)
		}
	})

	t.Run("Cookieがない場合、nilを返すこと", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		user, err := manager.GetCurrentUser(context.Background(), req)
		if err != nil {
			t.Fatalf("GetCurrentUserに失敗: %v", err)
		}

		if user != nil {
			t.Errorf("ユーザーがnilではありませんでした: %+v", user)
		}
	})

	t.Run("無効なトークンの場合、nilを返すこと", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  session.CookieName,
			Value: "invalid_token",
		})

		user, err := manager.GetCurrentUser(context.Background(), req)
		if err != nil {
			t.Fatalf("GetCurrentUserに失敗: %v", err)
		}

		if user != nil {
			t.Errorf("ユーザーがnilではありませんでした: %+v", user)
		}
	})
}

func TestManager_SetSessionCookie(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   true,
		SessionHTTPOnly: true,
	}

	manager := session.NewManager(nil, nil, cfg)

	t.Run("正しいCookie属性が設定されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		token := "test_token_12345"

		manager.SetSessionCookie(rr, token)

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Cookieの数 = %d、期待値 = 1", len(cookies))
		}

		cookie := cookies[0]

		if cookie.Name != session.CookieName {
			t.Errorf("Cookie名 = %s、期待値 = %s", cookie.Name, session.CookieName)
		}

		if cookie.Value != token {
			t.Errorf("Cookie値 = %s、期待値 = %s", cookie.Value, token)
		}

		// GoのhttpパッケージはDomainの先頭のドットを自動的に削除する
		if cookie.Domain != "example.com" {
			t.Errorf("Domain = %s、期待値 = example.com", cookie.Domain)
		}

		if !cookie.Secure {
			t.Error("Secureフラグがfalseです")
		}

		if !cookie.HttpOnly {
			t.Error("HttpOnlyフラグがfalseです")
		}

		if cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("SameSite = %v、期待値 = %v", cookie.SameSite, http.SameSiteLaxMode)
		}

		// 10年分のMaxAge (秒) を確認
		expectedMaxAge := 10 * 365 * 24 * 60 * 60
		if cookie.MaxAge != expectedMaxAge {
			t.Errorf("MaxAge = %d、期待値 = %d", cookie.MaxAge, expectedMaxAge)
		}
	})
}

func TestManager_DeleteSessionCookie(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		CookieDomain:    ".example.com",
		SessionSecure:   true,
		SessionHTTPOnly: true,
	}

	manager := session.NewManager(nil, nil, cfg)

	t.Run("MaxAge=-1のCookieが設定されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()

		manager.DeleteSessionCookie(rr)

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Cookieの数 = %d、期待値 = 1", len(cookies))
		}

		cookie := cookies[0]

		if cookie.Name != session.CookieName {
			t.Errorf("Cookie名 = %s、期待値 = %s", cookie.Name, session.CookieName)
		}

		if cookie.Value != "" {
			t.Errorf("Cookie値 = %s、期待値 = 空", cookie.Value)
		}

		if cookie.MaxAge != -1 {
			t.Errorf("MaxAge = %d、期待値 = -1", cookie.MaxAge)
		}
	})
}
