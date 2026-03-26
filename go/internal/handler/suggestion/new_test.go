package suggestion_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestNew_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/test/topics/1/suggestions/new", map[string]string{
		"space_identifier": "test",
		"topic_number":     "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("wrong redirect location: got %q want %q", loc, "/sign_in")
	}
}

func TestNew_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-nosp@example.com").
		WithAtname("newnosp").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/nonexistent/topics/1/suggestions/new", map[string]string{
		"space_identifier": "nonexistent",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newnosp"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_不正なトピック番号で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-badnum@example.com").
		WithAtname("newbadnum").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/test/topics/abc/suggestions/new", map[string]string{
		"space_identifier": "test",
		"topic_number":     "abc",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newbadnum"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_スペースメンバーでない場合404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-nomember@example.com").
		WithAtname("newnomember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("new-nomember-sp").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/new-nomember-sp/topics/1/suggestions/new", map[string]string{
		"space_identifier": "new-nomember-sp",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newnomember"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_スペースメンバーで正常にフォームが表示される(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-ok@example.com").
		WithAtname("newok").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("new-ok-sp").
		WithName("New OK Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("New OK Topic").
		WithVisibility(0).
		Build()

	// 下書きページを作成
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("テストページ").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("下書きタイトル").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/new-ok-sp/topics/1/suggestions/new", map[string]string{
		"space_identifier": "new-ok-sp",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "下書きタイトル") {
		t.Error("response should contain draft page title")
	}
	if !strings.Contains(body, "csrf_token") {
		t.Error("response should contain CSRF token")
	}
}

func TestNew_下書きページがない場合でもフォームが表示される(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-nodraft@example.com").
		WithAtname("newnodraft").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("new-nodraft-sp").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/new-nodraft-sp/topics/1/suggestions/new", map[string]string{
		"space_identifier": "new-nodraft-sp",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newnodraft"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
}
