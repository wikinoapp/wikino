package draft_page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newDeleteRequestWithChiParams は chi の URL パラメータ付き DELETE リクエストを作成するヘルパーです
func newDeleteRequestWithChiParams(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, path, strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestDelete_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newDeleteRequestWithChiParams(t, "/s/my-space/pages/1/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("wrong redirect: got %q want /sign_in", loc)
	}
}

func TestDelete_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("delete-invalidnum@example.com").
		WithAtname("deleteinvalidnum").
		Build()

	handler := setupHandler(t, queries)

	req := newDeleteRequestWithChiParams(t, "/s/my-space/pages/abc/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "abc",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestDelete_DraftNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("delete-notfound@example.com").
		WithAtname("deletenotfound").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("delete-notfound-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	// 下書きは作成しない

	handler := setupHandler(t, queries)

	req := newDeleteRequestWithChiParams(t, "/s/delete-notfound-space/pages/1/draft_page", map[string]string{
		"space_identifier": "delete-notfound-space",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	// 削除 UseCase 内でトランザクションを開始するため、ここではトランザクション分離せず
	// 共有 DB を直接使う。テストデータはユニーク識別子で衝突を避ける。
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-success@example.com").
		WithAtname("deletesuccess").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("delete-success-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("draft body").
		Build()

	queries := query.New(db)
	handler := setupHandler(t, queries)

	req := newDeleteRequestWithChiParams(t, "/s/delete-success-space/pages/1/draft_page", map[string]string{
		"space_identifier": "delete-success-space",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	if loc := rr.Header().Get("Location"); loc != "/drafts" {
		t.Errorf("wrong redirect: got %q want /drafts", loc)
	}

	// 成功時はフラッシュ Cookie がセットされていることを確認
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

	// DB から削除されていることを確認
	draftPageRepo := repository.NewDraftPageRepository(queries)
	got, err := draftPageRepo.FindByID(context.Background(), draftPageID, spaceID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if got != nil {
		t.Error("削除後は下書きが取得できないべき")
	}
}

func TestDelete_PermissionDenied(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("delete-noperm@example.com").
		WithAtname("deletenoperm").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("delete-noperm-space").
		Build()
	// draft_page:delete を持たないメンバー (draft_page:write のみ)
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes([]model.Scope{model.ScopeDraftPageWrite}).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("draft body").
		Build()

	handler := setupHandler(t, queries)

	req := newDeleteRequestWithChiParams(t, "/s/delete-noperm-space/pages/1/draft_page", map[string]string{
		"space_identifier": "delete-noperm-space",
		"page_number":      "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Delete(rr, req)

	// Forbidden は 404 に変換される (リソース存在の漏洩を防ぐ既存パターン)
	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}
