package topic_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	topichandler "github.com/wikinoapp/wikino/go/internal/handler/topic"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newShowRequest はchiのURLパラメータ付きGETリクエストを作成するヘルパーです
func newShowRequest(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// setupHandler はテスト用のトピックハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *topichandler.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	flashMgr := session.NewFlashManager("", false, true)
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)

	return topichandler.NewHandler(
		cfg,
		flashMgr,
		spaceRepo,
		spaceMemberRepo,
		topicRepo,
		topicMemberRepo,
		pageRepo,
		sidebarHelper,
	)
}

func TestShow_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/nonexistent/topics/1", map[string]string{
		"space_identifier": "nonexistent",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_不正なトピック番号で404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/test-space/topics/abc", map[string]string{
		"space_identifier": "test-space",
		"topic_number":     "abc",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_存在しないトピックで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-noexist").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-noexist/topics/999", map[string]string{
		"space_identifier": "ts-noexist",
		"topic_number":     "999",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_公開トピックを未ログインで閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-public").
		WithName("Public Space").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-public/topics/1", map[string]string{
		"space_identifier": "ts-public",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "公開トピック") {
		t.Error("response should contain topic name")
	}
}

func TestShow_非公開トピックを未ログインで閲覧すると404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-priv1").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("非公開トピック").
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-priv1/topics/1", map[string]string{
		"space_identifier": "ts-priv1",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_非公開トピックをスペースオーナーが閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-owner@example.com").
		WithAtname("tsowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-priv2").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		WithRole(0). // owner
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("オーナー閲覧可能").
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-priv2/topics/1", map[string]string{
		"space_identifier": "ts-priv2",
		"topic_number":     "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "tsowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "オーナー閲覧可能") {
		t.Error("response should contain topic name")
	}
}

func TestShow_非公開トピックをトピックメンバーが閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	memberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-member@example.com").
		WithAtname("tsmember").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-priv3").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberUserID).
		WithRole(1). // member (not owner)
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("メンバー閲覧可能").
		WithVisibility(1). // private
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		WithRole(0). // admin
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-priv3/topics/1", map[string]string{
		"space_identifier": "ts-priv3",
		"topic_number":     "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: memberUserID, Atname: "tsmember"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "メンバー閲覧可能") {
		t.Error("response should contain topic name")
	}
}

func TestShow_非公開トピックを非メンバーが閲覧すると404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-owner4@example.com").
		WithAtname("tsowner4").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-nonmember@example.com").
		WithAtname("tsnonmember").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-priv4").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		WithRole(0). // owner
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(nonMemberID).
		WithRole(1). // member
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-priv4/topics/1", map[string]string{
		"space_identifier": "ts-priv4",
		"topic_number":     "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "tsnonmember"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_正常系_ページ一覧が表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-pages").
		WithName("Pages Space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
		WithVisibility(0).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("最初のページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("ピン留めページ").
		WithLinkedPageIDs([]model.PageID{}).
		WithPinnedAt(time.Now()).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-pages/topics/1", map[string]string{
		"space_identifier": "ts-pages",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "テストトピック") {
		t.Error("response should contain topic name")
	}
	if !strings.Contains(body, "最初のページ") {
		t.Error("response should contain regular page title")
	}
	if !strings.Contains(body, "ピン留めページ") {
		t.Error("response should contain pinned page title")
	}
}
