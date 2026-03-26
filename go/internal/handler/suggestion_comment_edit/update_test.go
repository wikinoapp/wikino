package suggestion_comment_edit_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdate_未ログインでリダイレクトされる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	form := url.Values{}
	form.Set("body", "更新後のコメント")

	req := newRequest(t, http.MethodPatch, "/s/test/suggestions/1/comments/1", map[string]string{
		"space_identifier":  "test",
		"suggestion_number": "1",
		"comment_number":    "1",
	}, form)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
}

func TestUpdate_本文が空の場合バリデーションエラーで422が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("cupdate-empty@example.com").
		WithAtname("cupdateempty").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("cupdate-empty-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0).
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
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()
	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("元のコメント").
		Build()

	handler := setupHandler(t, db, queries)

	form := url.Values{}
	form.Set("body", "")
	form.Set("csrf_token", "test-csrf-token")

	req := newRequest(t, http.MethodPatch, "/s/cupdate-empty-sp/suggestions/1/comments/1", map[string]string{
		"space_identifier":  "cupdate-empty-sp",
		"suggestion_number": "1",
		"comment_number":    "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "cupdateempty"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestUpdate_正常にコメントが更新されリダイレクトされる(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("cupdate-ok@example.com").
		WithAtname("cupdateok").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("cupdate-ok-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		Build()
	suggestionID := testutil.NewSuggestionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()
	testutil.NewSuggestionCommentBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("元のコメント").
		Build()

	handler := setupHandler(t, db, queries)

	form := url.Values{}
	form.Set("body", "更新後のコメント")
	form.Set("csrf_token", "test-csrf-token")

	req := newRequest(t, http.MethodPatch, "/s/cupdate-ok-sp/suggestions/1/comments/1", map[string]string{
		"space_identifier":  "cupdate-ok-sp",
		"suggestion_number": "1",
		"comment_number":    "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "cupdateok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/cupdate-ok-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}
