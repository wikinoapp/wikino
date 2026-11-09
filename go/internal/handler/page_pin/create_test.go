package page_pin_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/page_pin"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func setupHandler(t *testing.T, queries *query.Queries) *page_pin.Handler {
	t.Helper()

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)

	return page_pin.NewHandler(
		session.NewFlashManager("", false, true),
		usecase.NewPinPageUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo),
		usecase.NewUnpinPageUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo),
	)
}

// newRequestは指定ユーザーとしてピン留めのパスへ送るリクエストを組み立てる。
// userIDが空文字の場合は未ログインのリクエストになる。
func newRequest(t *testing.T, method, spaceIdentifier, pageNumber string, userID model.UserID) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, fmt.Sprintf("/s/%s/pages/%s/pin", spaceIdentifier, pageNumber), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", spaceIdentifier)
	rctx.URLParams.Add("page_number", pageNumber)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "tester"})
	}

	return req.WithContext(ctx)
}

// pinFixtureはハンドラーのテストで使うスペースとメンバー。
type pinFixture struct {
	spaceID  model.SpaceID
	editorID model.UserID
	viewerID model.UserID
}

// setupFixtureは、prefixで識別子を分けたスペースに、公開トピックのページ1〜3を作る。
// pinnedAtがnilでなければ、すべてのページにピンを打った状態で作る。
func setupFixture(t *testing.T, tx *sql.Tx, prefix string, pinnedAt *time.Time) pinFixture {
	t.Helper()

	f := pinFixture{
		editorID: testutil.NewUserBuilder(t, tx).
			WithEmail(prefix + "-editor@example.com").
			WithAtname(prefix + "editor").
			Build(),
		viewerID: testutil.NewUserBuilder(t, tx).
			WithEmail(prefix + "-viewer@example.com").
			WithAtname(prefix + "viewer").
			Build(),
	}

	f.spaceID = testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(prefix + "-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithUserID(f.editorID).
		WithRole(model.SpaceRoleEditor).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithUserID(f.viewerID).
		WithRole(model.SpaceRoleViewer).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithNumber(1).
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	for number := model.PageNumber(1); number <= 3; number++ {
		b := testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithNumber(number).
			WithTitle(fmt.Sprintf("Page %d", number)).
			WithLinkedPageIDs([]model.PageID{})
		if pinnedAt != nil {
			b = b.WithPinnedAt(*pinnedAt)
		}
		b.Build()
	}

	return f
}

func findPage(t *testing.T, queries *query.Queries, spaceID model.SpaceID, number model.PageNumber) *model.Page {
	t.Helper()

	page, err := repository.NewPageRepository(queries).FindBySpaceAndNumber(context.Background(), spaceID, number)
	if err != nil {
		t.Fatalf("FindBySpaceAndNumber()のエラー = %v", err)
	}
	if page == nil {
		t.Fatal("FindBySpaceAndNumber()がnilを返した、期待値 = ページ")
	}
	return page
}

func hasFlashCookie(rr *httptest.ResponseRecorder) bool {
	for _, c := range rr.Result().Cookies() {
		if c.Name == session.FlashCookieName && c.Value != "" {
			return true
		}
	}
	return false
}

func TestCreate(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	h := setupHandler(t, queries)
	f := setupFixture(t, tx, "pph-create", nil)

	t.Run("正常系: page_pin:writeを持つメンバーはピンを打ち、ページ自身へリダイレクトされる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, http.MethodPost, "pph-create-space", "1", f.editorID))

		if rr.Code != http.StatusSeeOther {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusSeeOther)
		}
		if location := rr.Header().Get("Location"); location != "/s/pph-create-space/pages/1" {
			t.Errorf("Location = %v、期待値 = /s/pph-create-space/pages/1", location)
		}
		if !hasFlashCookie(rr) {
			t.Errorf("成功時はフラッシュCookie (%s) がセットされるべき", session.FlashCookieName)
		}
		if page := findPage(t, queries, f.spaceID, 1); page.PinnedAt == nil {
			t.Error("page.PinnedAt = nil、期待値 = ピンを打った日時")
		}
	})

	t.Run("異常系: page_pin:writeを持たない閲覧者は404になる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, http.MethodPost, "pph-create-space", "2", f.viewerID))

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
		}
		if page := findPage(t, queries, f.spaceID, 2); page.PinnedAt != nil {
			t.Errorf("page.PinnedAt = %v、期待値 = nil (権限が無いので更新されない)", page.PinnedAt)
		}
	})

	t.Run("異常系: 未ログインではログイン画面へリダイレクトされる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, http.MethodPost, "pph-create-space", "3", ""))

		if rr.Code != http.StatusFound {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
		}
		if location := rr.Header().Get("Location"); location != "/sign_in" {
			t.Errorf("Location = %v、期待値 = /sign_in", location)
		}
		if page := findPage(t, queries, f.spaceID, 3); page.PinnedAt != nil {
			t.Errorf("page.PinnedAt = %v、期待値 = nil (未ログインでは更新されない)", page.PinnedAt)
		}
	})

	t.Run("異常系: 存在しないページは404になる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, http.MethodPost, "pph-create-space", "999", f.editorID))

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("異常系: ページ番号が数値でない場合は404になる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, http.MethodPost, "pph-create-space", "abc", f.editorID))

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
		}
	})
}
