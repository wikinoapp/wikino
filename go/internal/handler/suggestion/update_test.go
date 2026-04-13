package suggestion_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdate_バリデーションエラーで422が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-val@example.com").
		WithAtname("updateval").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("update-val-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithTitle("バリデーションテスト").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	form := url.Values{}
	form.Set("title", "") // 空のタイトル
	form.Set("body", "テスト本文")

	req := newSuggestionRequest(t, http.MethodPatch, "/s/update-val-sp/suggestions/1", map[string]string{
		"space_identifier":  "update-val-sp",
		"suggestion_number": "1",
	}, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "updateval"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestUpdate_正常に更新してリダイレクトされる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-ok@example.com").
		WithAtname("updateok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("update-ok-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
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
		WithTitle("更新前タイトル").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	form := url.Values{}
	form.Set("title", "更新後タイトル")
	form.Set("body", "更新後本文")

	req := newSuggestionRequest(t, http.MethodPatch, "/s/update-ok-sp/suggestions/1", map[string]string{
		"space_identifier":  "update-ok-sp",
		"suggestion_number": "1",
	}, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "updateok"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}

	location := rr.Header().Get("Location")
	expectedLocation := "/s/update-ok-sp/suggestions/1"
	if location != expectedLocation {
		t.Errorf("Location = %q, want %q", location, expectedLocation)
	}
}
