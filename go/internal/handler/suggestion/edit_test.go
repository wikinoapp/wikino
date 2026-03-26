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

func TestEdit_未ログインでリダイレクトされる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/test/suggestions/1/edit", map[string]string{
		"space_identifier":  "test",
		"suggestion_number": "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
}

func TestEdit_存在しない編集提案で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("edit-404@example.com").
		WithAtname("edit404").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/nonexistent/suggestions/999/edit", map[string]string{
		"space_identifier":  "nonexistent",
		"suggestion_number": "999",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "edit404"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestEdit_編集権限がある場合にフォームが表示される(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("edit-ok@example.com").
		WithAtname("editok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("edit-ok-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0). // owner
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("編集テスト提案").
		WithBody("テスト本文").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/edit-ok-sp/suggestions/1/edit", map[string]string{
		"space_identifier":  "edit-ok-sp",
		"suggestion_number": "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "editok"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Edit(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "編集テスト提案") {
		t.Error("response should contain suggestion title")
	}
	if !strings.Contains(body, "テスト本文") {
		t.Error("response should contain suggestion body")
	}
}
