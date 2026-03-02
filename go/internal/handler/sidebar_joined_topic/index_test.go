package sidebar_joined_topic_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/sidebar_joined_topic"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func setupHandler(t *testing.T, queries *query.Queries) *sidebar_joined_topic.Handler {
	t.Helper()

	return sidebar_joined_topic.NewHandler(
		repository.NewTopicRepository(queries),
	)
}

func TestIndex_未ログイン時に空フラグメントのSSEレスポンスが返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/sidebar/joined_topics", nil)
	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("wrong content type: got %v, want text/event-stream", contentType)
	}
}

func TestIndex_ログイン済みでトピック未参加時にSSEレスポンスが返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sidebar-empty@example.com").
		WithAtname("sidebarempty").
		Build()

	handler := setupHandler(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/sidebar/joined_topics", nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("wrong content type: got %v, want text/event-stream", contentType)
	}
}

func TestIndex_ログイン済みで参加中トピックがSSEレスポンスに含まれる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sidebar-topics@example.com").
		WithAtname("sidebartopics").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sidebar-topics").
		WithName("Sidebar Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Test Topic").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		WithRole(0).
		Build()

	handler := setupHandler(t, queries)

	req := httptest.NewRequest(http.MethodGet, "/sidebar/joined_topics", nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("wrong content type: got %v, want text/event-stream", contentType)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "Test Topic") {
		t.Error("response should contain topic name 'Test Topic'")
	}

	if !strings.Contains(body, "Sidebar Space") {
		t.Error("response should contain space name 'Sidebar Space'")
	}
}
