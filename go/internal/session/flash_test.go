package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/session"
)

func TestFlashManager_SetNotice(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("noticeタイプのフラッシュメッセージが設定されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		fm.SetNotice(rr, "ログインしました")

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Cookieの数が不正: got %d, want 1", len(cookies))
		}

		cookie := cookies[0]
		if cookie.Name != session.FlashCookieName {
			t.Errorf("Cookie名が不正: got %s, want %s", cookie.Name, session.FlashCookieName)
		}

		// JSONフォーマットでnoticeタイプとメッセージが含まれていること
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

		if flash.Type != session.FlashNotice {
			t.Errorf("タイプが不正: got %s, want %s", flash.Type, session.FlashNotice)
		}

		if flash.Message != "ログインしました" {
			t.Errorf("メッセージが不正: got %s, want ログインしました", flash.Message)
		}
	})
}

func TestFlashManager_SetAlert(t *testing.T) {
	t.Parallel()

	fm := session.NewFlashManager(".example.com", true, true)

	t.Run("alertタイプのフラッシュメッセージが設定されること", func(t *testing.T) {
		t.Parallel()

		rr := httptest.NewRecorder()
		fm.SetAlert(rr, "エラーが発生しました")

		cookies := rr.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("Cookieの数が不正: got %d, want 1", len(cookies))
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

		if flash.Type != session.FlashAlert {
			t.Errorf("タイプが不正: got %s, want %s", flash.Type, session.FlashAlert)
		}

		if flash.Message != "エラーが発生しました" {
			t.Errorf("メッセージが不正: got %s, want エラーが発生しました", flash.Message)
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
		fm.SetNotice(rr, "テストメッセージ")
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
			t.Fatalf("削除用Cookieの数が不正: got %d, want 1", len(deleteCookies))
		}

		deleteCookie := deleteCookies[0]
		if deleteCookie.MaxAge != -1 {
			t.Errorf("MaxAgeが不正: got %d, want -1", deleteCookie.MaxAge)
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
			t.Fatalf("削除用Cookieの数が不正: got %d, want 1", len(deleteCookies))
		}

		deleteCookie := deleteCookies[0]
		if deleteCookie.MaxAge != -1 {
			t.Errorf("MaxAgeが不正: got %d, want -1", deleteCookie.MaxAge)
		}
	})
}

func TestFlashManager_CookieAttributes(t *testing.T) {
	t.Parallel()

	t.Run("HttpOnlyがfalseであること", func(t *testing.T) {
		t.Parallel()

		fm := session.NewFlashManager(".example.com", true, true)
		rr := httptest.NewRecorder()
		fm.SetNotice(rr, "テスト")

		cookie := rr.Result().Cookies()[0]

		// フラッシュCookieはJavaScriptからアクセス可能にするためHttpOnly=false
		if cookie.HttpOnly {
			t.Error("HttpOnlyフラグがtrueです（falseであるべき）")
		}
	})

	t.Run("SameSiteがLaxであること", func(t *testing.T) {
		t.Parallel()

		fm := session.NewFlashManager(".example.com", true, true)
		rr := httptest.NewRecorder()
		fm.SetNotice(rr, "テスト")

		cookie := rr.Result().Cookies()[0]

		if cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("SameSiteが不正: got %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
		}
	})
}
