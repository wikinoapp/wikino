package page_location_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/page_location"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page_location.Handler {
	t.Helper()

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	getPageLocationsUC := usecase.NewGetPageLocationsUsecase(spaceRepo, spaceMemberRepo, pageRepo)

	return page_location.NewHandler(
		getPageLocationsUC,
	)
}

// newRequestWithChiParams はchiのURLパラメータ付きリクエストを作成するヘルパーです
func newRequestWithChiParams(t *testing.T, method, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

type pageLocationsResponse struct {
	PageLocations []pageLocationItem `json:"page_locations"`
}

type pageLocationItem struct {
	Key string `json:"key"`
}

func TestIndex(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("search-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Hello World").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Hello Go").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Goodbye World").
		Build()

	handler := setupHandler(t, queries)

	// 「Hello」で検索
	req := newRequestWithChiParams(t, http.MethodGet, "/s/search-space/page_locations?q=Hello", map[string]string{
		"space_identifier": "search-space",
	})
	req.URL.RawQuery = "q=Hello"
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Content-Typeを確認
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("wrong content type: got %v want application/json", contentType)
	}

	// JSONレスポンスをパース
	var resp pageLocationsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 「Hello」を含むページが2件返ること
	if len(resp.PageLocations) != 2 {
		t.Errorf("wrong number of results: got %v want 2", len(resp.PageLocations))
	}

	// キーの形式を確認（トピック名/ページタイトル）
	for _, item := range resp.PageLocations {
		if item.Key != "General/Hello World" && item.Key != "General/Hello Go" {
			t.Errorf("unexpected key: %v", item.Key)
		}
	}
}

func TestIndex_MultipleWords(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("multi@example.com").
		WithAtname("multiuser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("multi-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("Dev").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Hello World").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Hello Go").
		Build()

	handler := setupHandler(t, queries)

	// 「Hello World」で検索（両方の単語を含むページのみ）
	req := newRequestWithChiParams(t, http.MethodGet, "/s/multi-space/page_locations?q=Hello+World", map[string]string{
		"space_identifier": "multi-space",
	})
	req.URL.RawQuery = "q=Hello+World"
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var resp pageLocationsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 両方の単語を含む「Hello World」のみが返ること
	if len(resp.PageLocations) != 1 {
		t.Errorf("wrong number of results: got %v want 1", len(resp.PageLocations))
	}

	if len(resp.PageLocations) > 0 && resp.PageLocations[0].Key != "Dev/Hello World" {
		t.Errorf("wrong key: got %v want Dev/Hello World", resp.PageLocations[0].Key)
	}
}

func TestIndex_EmptyQuery(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("empty@example.com").
		WithAtname("emptyuser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("empty-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Some Page").
		Build()

	handler := setupHandler(t, queries)

	// 空のクエリで検索
	req := newRequestWithChiParams(t, http.MethodGet, "/s/empty-space/page_locations", map[string]string{
		"space_identifier": "empty-space",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var resp pageLocationsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 空クエリでも公開済みページが返ること
	if len(resp.PageLocations) != 1 {
		t.Errorf("wrong number of results: got %v want 1", len(resp.PageLocations))
	}
}

func TestIndex_ExcludesUnpublishedPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("unpub@example.com").
		WithAtname("unpubuser").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("unpub-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	// 公開済みページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Published Page").
		Build()
	// 非公開ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Unpublished Page").
		WithUnpublished().
		Build()
	// 廃棄済みページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Discarded Page").
		WithDiscarded().
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/unpub-space/page_locations?q=Page", map[string]string{
		"space_identifier": "unpub-space",
	})
	req.URL.RawQuery = "q=Page"
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var resp pageLocationsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// 公開済みの1件のみが返ること
	if len(resp.PageLocations) != 1 {
		t.Errorf("wrong number of results: got %v want 1", len(resp.PageLocations))
	}

	if len(resp.PageLocations) > 0 && resp.PageLocations[0].Key != "General/Published Page" {
		t.Errorf("wrong key: got %v want General/Published Page", resp.PageLocations[0].Key)
	}
}

func TestIndex_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/some-space/page_locations", map[string]string{
		"space_identifier": "some-space",
	})

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_NotSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("plowner@example.com").
		WithAtname("plowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("ploutsider@example.com").
		WithAtname("ploutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("private-pl-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/private-pl-space/page_locations", map[string]string{
		"space_identifier": "private-pl-space",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: outsiderID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("plnotfound@example.com").
		WithAtname("plnotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/nonexistent/page_locations", map[string]string{
		"space_identifier": "nonexistent",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}
