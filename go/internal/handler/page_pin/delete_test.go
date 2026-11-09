package page_pin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestDelete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	h := setupHandler(t, queries)
	pinnedAt := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	f := setupFixture(t, tx, "pph-delete", &pinnedAt)

	t.Run("正常系: page_pin:deleteを持つメンバーはピンを外し、ページ自身へリダイレクトされる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Delete(rr, newRequest(t, http.MethodDelete, "pph-delete-space", "1", f.editorID))

		if rr.Code != http.StatusSeeOther {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusSeeOther)
		}
		if location := rr.Header().Get("Location"); location != "/s/pph-delete-space/pages/1" {
			t.Errorf("Location = %v、期待値 = /s/pph-delete-space/pages/1", location)
		}
		if !hasFlashCookie(rr) {
			t.Errorf("成功時はフラッシュCookie (%s) がセットされるべき", session.FlashCookieName)
		}
		if page := findPage(t, queries, f.spaceID, 1); page.PinnedAt != nil {
			t.Errorf("page.PinnedAt = %v、期待値 = nil", page.PinnedAt)
		}
	})

	t.Run("異常系: page_pin:deleteを持たない閲覧者は404になる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Delete(rr, newRequest(t, http.MethodDelete, "pph-delete-space", "2", f.viewerID))

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
		}
		if page := findPage(t, queries, f.spaceID, 2); page.PinnedAt == nil {
			t.Error("page.PinnedAt = nil、期待値 = 元の日時 (権限が無いので更新されない)")
		}
	})

	t.Run("異常系: 未ログインではログイン画面へリダイレクトされる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Delete(rr, newRequest(t, http.MethodDelete, "pph-delete-space", "3", ""))

		if rr.Code != http.StatusFound {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
		}
		if location := rr.Header().Get("Location"); location != "/sign_in" {
			t.Errorf("Location = %v、期待値 = /sign_in", location)
		}
		if page := findPage(t, queries, f.spaceID, 3); page.PinnedAt == nil {
			t.Error("page.PinnedAt = nil、期待値 = 元の日時 (未ログインでは更新されない)")
		}
	})

	t.Run("異常系: 存在しないページは404になる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Delete(rr, newRequest(t, http.MethodDelete, "pph-delete-space", "999", f.editorID))

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
		}
	})
}
