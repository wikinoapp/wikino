package page_trash_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/page_trash"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func addChiParams(t *testing.T, req *http.Request, params map[string]string) *http.Request {
	t.Helper()

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func setupHandler(t *testing.T, queries *query.Queries) *page_trash.Handler {
	t.Helper()

	flashMgr := session.NewFlashManager("", false, true)
	trashPageUC := usecase.NewTrashPageUsecase(
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewPageRepository(queries),
		repository.NewTopicRepository(queries),
		repository.NewTopicMemberRepository(queries),
	)

	return page_trash.NewHandler(flashMgr, trashPageUC)
}

// newRequest builds a POST request that moves a page to the trash as the specified user. An empty
// userID creates an unauthenticated request.
//
// [Ja] newRequest は指定ユーザーとしてページをゴミ箱へ移動する POST リクエストを組み立てる。
// userID が空文字の場合は未ログインのリクエストになる。
func newRequest(t *testing.T, spaceIdentifier string, pageNumber int32, userID model.UserID) *http.Request {
	t.Helper()

	path := fmt.Sprintf("/s/%s/pages/%d/trash", spaceIdentifier, pageNumber)
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req = addChiParams(t, req, map[string]string{
		"space_identifier": spaceIdentifier,
		"page_number":      fmt.Sprintf("%d", pageNumber),
	})

	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "tester"})
	}

	return req.WithContext(ctx)
}

func TestCreate(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(queries)
	h := setupHandler(t, queries)

	trashMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("pt-trash@example.com").
		WithAtname("pttrash").
		Build()
	writerMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("pt-writer@example.com").
		WithAtname("ptwriter").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pt-space").
		WithName("PT Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(trashMemberID).
		WithScopes([]model.Scope{model.ScopePageTrash}).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(writerMemberID).
		WithScopes([]model.Scope{model.ScopePageWrite}).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(7).
		WithName("General").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	newPage := func(number model.PageNumber, title string) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(number).
			WithTitle(title).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
	}
	newPage(1, "Trashable Page")
	newPage(2, "Writer Page")
	newPage(3, "Guest Page")

	findPage := func(t *testing.T, number model.PageNumber) *model.Page {
		t.Helper()

		page, err := pageRepo.FindBySpaceAndNumber(context.Background(), spaceID, number)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if page == nil {
			t.Fatal("FindBySpaceAndNumber() returned nil, want page")
		}
		return page
	}

	t.Run("正常系: page:trash を持つメンバーはトピック画面へリダイレクトされる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, "pt-space", 1, trashMemberID))

		if rr.Code != http.StatusSeeOther {
			t.Errorf("status code = %v, want %v", rr.Code, http.StatusSeeOther)
		}
		// The page is gone from the screen the request came from, so the user lands on its topic.
		//
		// [Ja] リクエスト元の画面からページは消えるため、属していたトピックへ着地する。
		if location := rr.Header().Get("Location"); location != "/s/pt-space/topics/7" {
			t.Errorf("Location = %v, want /s/pt-space/topics/7", location)
		}
		// The page disappears from the topic the user lands on, so the flash is the only
		// confirmation that the operation went through.
		//
		// [Ja] 着地先のトピックからページは消えるため、操作が通ったことを利用者に伝えるのは
		// フラッシュメッセージだけになる。
		hasFlashCookie := false
		for _, c := range rr.Result().Cookies() {
			if c.Name == session.FlashCookieName && c.Value != "" {
				hasFlashCookie = true
				break
			}
		}
		if !hasFlashCookie {
			t.Errorf("成功時はフラッシュ Cookie (%s) がセットされるべき", session.FlashCookieName)
		}
		if page := findPage(t, 1); page.TrashedAt == nil {
			t.Error("page.TrashedAt = nil, want a stamped time")
		}
	})

	t.Run("異常系: page:trash を持たないメンバーは404になる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, "pt-space", 2, writerMemberID))

		if rr.Code != http.StatusNotFound {
			t.Errorf("status code = %v, want %v", rr.Code, http.StatusNotFound)
		}
		if page := findPage(t, 2); page.TrashedAt != nil {
			t.Errorf("page.TrashedAt = %v, want nil (権限が無いので更新されないべき)", page.TrashedAt)
		}
	})

	t.Run("異常系: 存在しないページは404になる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, "pt-space", 999, trashMemberID))

		if rr.Code != http.StatusNotFound {
			t.Errorf("status code = %v, want %v", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("異常系: 未ログインではログイン画面へリダイレクトされる", func(t *testing.T) {
		rr := httptest.NewRecorder()
		h.Create(rr, newRequest(t, "pt-space", 3, ""))

		if rr.Code != http.StatusFound {
			t.Errorf("status code = %v, want %v", rr.Code, http.StatusFound)
		}
		if location := rr.Header().Get("Location"); location != "/sign_in" {
			t.Errorf("Location = %v, want /sign_in", location)
		}
		if page := findPage(t, 3); page.TrashedAt != nil {
			t.Errorf("page.TrashedAt = %v, want nil (未ログインでは更新されないべき)", page.TrashedAt)
		}
	})

	t.Run("異常系: ページ番号が数値でない場合は404になる", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/s/pt-space/pages/abc/trash", nil)
		req = addChiParams(t, req, map[string]string{
			"space_identifier": "pt-space",
			"page_number":      "abc",
		})
		ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: trashMemberID, Atname: "tester"})

		rr := httptest.NewRecorder()
		h.Create(rr, req.WithContext(ctx))

		if rr.Code != http.StatusNotFound {
			t.Errorf("status code = %v, want %v", rr.Code, http.StatusNotFound)
		}
	})
}
