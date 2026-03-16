package user_session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/user_session"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("test@example.com").
		WithAtname("testuser").
		Build()

	// テストセッションを作成
	sessionToken := testutil.NewSessionBuilder(t, tx).
		WithUserID(userID).
		BuildAndGetToken()

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリとUseCaseを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	deleteUserSessionUC := usecase.NewDeleteUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	handler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		deleteUserSessionUC,
	)

	// セッションCookieを設定したリクエストを作成
	req := httptest.NewRequest(http.MethodDelete, "/user_session", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: sessionToken,
	})

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	// リダイレクトを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("wrong redirect location: got %v want /", location)
	}

	// セッションCookieが削除されているか確認（MaxAge=-1）
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Error("session cookie not found in response")
	}
	if sessionCookie != nil && sessionCookie.MaxAge != -1 {
		t.Errorf("session cookie MaxAge should be -1, got %d", sessionCookie.MaxAge)
	}

	// フラッシュCookieが設定されているか確認
	var flashCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.FlashCookieName {
			flashCookie = c
			break
		}
	}
	if flashCookie == nil {
		t.Error("flash cookie not set")
	}

	// データベースからセッションが削除されているか確認
	savedSession, err := userSessionRepo.FindByToken(context.Background(), sessionToken)
	if err != nil {
		t.Fatalf("セッション取得でエラー: %v", err)
	}
	if savedSession != nil {
		t.Error("session should be deleted from database")
	}
}

func TestDelete_NoSession(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリとUseCaseを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	deleteUserSessionUC := usecase.NewDeleteUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	handler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		deleteUserSessionUC,
	)

	// セッションCookieなしでリクエスト
	req := httptest.NewRequest(http.MethodDelete, "/user_session", nil)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	// リダイレクトを検証（エラーにならずにリダイレクトされる）
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("wrong redirect location: got %v want /", location)
	}

	// フラッシュCookieが設定されているか確認
	cookies := rr.Result().Cookies()
	var flashCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.FlashCookieName {
			flashCookie = c
			break
		}
	}
	if flashCookie == nil {
		t.Error("flash cookie not set")
	}
}

func TestDelete_InvalidSessionToken(t *testing.T) {
	t.Parallel()

	// テスト用DBをセットアップ
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリとUseCaseを初期化
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	deleteUserSessionUC := usecase.NewDeleteUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ハンドラーを初期化
	handler := user_session.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		deleteUserSessionUC,
	)

	// 存在しないセッショントークンでリクエスト
	req := httptest.NewRequest(http.MethodDelete, "/user_session", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.CookieName,
		Value: "invalid-token",
	})

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	// リダイレクトを検証（エラーにならずにリダイレクトされる）
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("wrong redirect location: got %v want /", location)
	}

	// セッションCookieが削除されているか確認
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == session.CookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Error("session cookie not found in response")
	}
	if sessionCookie != nil && sessionCookie.MaxAge != -1 {
		t.Errorf("session cookie MaxAge should be -1, got %d", sessionCookie.MaxAge)
	}
}
