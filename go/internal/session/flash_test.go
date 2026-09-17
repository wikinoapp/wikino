package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/session"
)

func TestFlashManager_SetSuccess(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("successタイプのフラッシュメッセージが設定されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		fm.SetSuccess(rr, "ログインしました")

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Cookieの数 = %d、期待値 = 1", len(cookies))
		}

		cookie := cookies[0]
		if cookie.Name != session.FlashCookieName {
			t.Errorf("Cookie名 = %s、期待値 = %s", cookie.Name, session.FlashCookieName)
		}

		// JSONフォーマットでsuccessタイプとメッセージが含まれていること
		if cookie.Value == "" {
			t.Error("Cookie値が空です")
		}

		// GetFlashで内容を検証
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)

		rr2 := httptest.NewRecorder()
		flash := fm.GetFlash(rr2, req)

		if flash == nil {
			t.Fatal("フラッシュメッセージがnilです")
		}

		if flash.Type != session.FlashSuccess {
			t.Errorf("タイプ = %s、期待値 = %s", flash.Type, session.FlashSuccess)
		}

		if flash.Message != "ログインしました" {
			t.Errorf("メッセージ = %s、期待値 = ログインしました", flash.Message)
		}
	})
}

func TestFlashManager_SetError(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("errorタイプのフラッシュメッセージが設定されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		fm.SetError(rr, "エラーが発生しました")

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Cookieの数 = %d、期待値 = 1", len(cookies))
		}

		cookie := cookies[0]

		// GetFlashで内容を検証
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)

		rr2 := httptest.NewRecorder()
		flash := fm.GetFlash(rr2, req)

		if flash == nil {
			t.Fatal("フラッシュメッセージがnilです")
		}

		if flash.Type != session.FlashError {
			t.Errorf("タイプ = %s、期待値 = %s", flash.Type, session.FlashError)
		}

		if flash.Message != "エラーが発生しました" {
			t.Errorf("メッセージ = %s、期待値 = エラーが発生しました", flash.Message)
		}
	})
}

func TestFlashManager_SetWarning(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("warningタイプのフラッシュメッセージが設定されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		fm.SetWarning(rr, "注意が必要です")

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Cookieの数 = %d、期待値 = 1", len(cookies))
		}

		cookie := cookies[0]

		// GetFlashで内容を検証
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)

		rr2 := httptest.NewRecorder()
		flash := fm.GetFlash(rr2, req)

		if flash == nil {
			t.Fatal("フラッシュメッセージがnilです")
		}

		if flash.Type != session.FlashWarning {
			t.Errorf("タイプ = %s、期待値 = %s", flash.Type, session.FlashWarning)
		}

		if flash.Message != "注意が必要です" {
			t.Errorf("メッセージ = %s、期待値 = 注意が必要です", flash.Message)
		}
	})
}

func TestFlashManager_SetInfo(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("infoタイプのフラッシュメッセージが設定されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		fm.SetInfo(rr, "お知らせがあります")

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Cookieの数 = %d、期待値 = 1", len(cookies))
		}

		cookie := cookies[0]

		// GetFlashで内容を検証
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)

		rr2 := httptest.NewRecorder()
		flash := fm.GetFlash(rr2, req)

		if flash == nil {
			t.Fatal("フラッシュメッセージがnilです")
		}

		if flash.Type != session.FlashInfo {
			t.Errorf("タイプ = %s、期待値 = %s", flash.Type, session.FlashInfo)
		}

		if flash.Message != "お知らせがあります" {
			t.Errorf("メッセージ = %s、期待値 = お知らせがあります", flash.Message)
		}
	})
}

func TestFlashManager_GetFlash(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("Cookieがない場合はnilを返すこと", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		flash := fm.GetFlash(rr, req)

		if flash != nil {
			t.Errorf("フラッシュメッセージがnilではありません: %+v", flash)
		}
	})

	t.Run("フラッシュ取得後にCookieが削除されること", func(t *testing.T) {
		t.Parallel()

		// フラッシュを設定
		rr := httptest.NewRecorder()
		fm.SetSuccess(rr, "テストメッセージ")
		cookie := rr.Result().Cookies()[0]

		// フラッシュを取得
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(cookie)

		rr2 := httptest.NewRecorder()
		flash := fm.GetFlash(rr2, req)

		if flash == nil {
			t.Fatal("フラッシュメッセージがnilです")
		}

		// 削除用のCookieがセットされていること
		deleteCookies := rr2.Result().Cookies()
		if len(deleteCookies) != 1 {
			t.Fatalf("削除用Cookieの数 = %d、期待値 = 1", len(deleteCookies))
		}

		deleteCookie := deleteCookies[0]
		if deleteCookie.MaxAge != -1 {
			t.Errorf("MaxAge = %d、期待値 = -1", deleteCookie.MaxAge)
		}
	})

	t.Run("不正なJSON形式の場合はnilを返してCookieを削除すること", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  session.FlashCookieName,
			Value: "invalid-json",
		})

		rr := httptest.NewRecorder()
		flash := fm.GetFlash(rr, req)

		if flash != nil {
			t.Errorf("フラッシュメッセージがnilではありません: %+v", flash)
		}

		// 削除用のCookieがセットされていること
		deleteCookies := rr.Result().Cookies()
		if len(deleteCookies) != 1 {
			t.Fatalf("削除用Cookieの数 = %d、期待値 = 1", len(deleteCookies))
		}

		deleteCookie := deleteCookies[0]
		if deleteCookie.MaxAge != -1 {
			t.Errorf("MaxAge = %d、期待値 = -1", deleteCookie.MaxAge)
		}
	})
}

func TestFlashManager_CookieAttributes(t *testing.T) {
	t.Parallel()

	t.Run("HttpOnlyがfalseであること", func(t *testing.T) {
		t.Parallel()

		fm := session.NewFlashManager(".example.com", true, true)
		rr := httptest.NewRecorder()
		fm.SetSuccess(rr, "テスト")

		cookie := rr.Result().Cookies()[0]

		// フラッシュCookieはJavaScriptからアクセス可能にするためHttpOnly=false
		if cookie.HttpOnly {
			t.Error("HttpOnlyフラグがtrueです (falseであるべき)")
		}
	})

	t.Run("SameSiteがLaxであること", func(t *testing.T) {
		t.Parallel()

		fm := session.NewFlashManager(".example.com", true, true)
		rr := httptest.NewRecorder()
		fm.SetSuccess(rr, "テスト")

		cookie := rr.Result().Cookies()[0]

		if cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("SameSite = %v、期待値 = %v", cookie.SameSite, http.SameSiteLaxMode)
		}
	})
}
