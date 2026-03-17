package suggestion_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newPostRequest はchiのURLパラメータ付きPOSTリクエストを作成するヘルパーです
func newPostRequest(t *testing.T, path string, params map[string]string, form url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestCreate_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	form := url.Values{}
	form.Set("title", "テスト提案")

	req := newPostRequest(t, "/s/test/topics/1/suggestions", map[string]string{
		"space_identifier": "test",
		"topic_number":     "1",
	}, form)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("wrong redirect location: got %q want %q", loc, "/sign_in")
	}
}

func TestCreate_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("create-nosp@example.com").
		WithAtname("createnosp").
		Build()

	handler := setupHandler(t, db, queries)

	form := url.Values{}
	form.Set("title", "テスト提案")

	req := newPostRequest(t, "/s/nonexistent/topics/1/suggestions", map[string]string{
		"space_identifier": "nonexistent",
		"topic_number":     "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "createnosp"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_タイトル未入力でバリデーションエラーが返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("create-notitle@example.com").
		WithAtname("createnotitle").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("create-notitle-sp").
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

	form := url.Values{}
	form.Set("title", "")
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/create-notitle-sp/topics/1/suggestions", map[string]string{
		"space_identifier": "create-notitle-sp",
		"topic_number":     "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "createnotitle"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestCreate_下書きページ未選択でバリデーションエラーが返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("create-nodraft@example.com").
		WithAtname("createnodraft").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("create-nodraft-sp").
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

	form := url.Values{}
	form.Set("title", "テスト提案")
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/create-nodraft-sp/topics/1/suggestions", map[string]string{
		"space_identifier": "create-nodraft-sp",
		"topic_number":     "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "createnodraft"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestCreate_正常に編集提案が作成されリダイレクトされる(t *testing.T) {
	t.Parallel()

	// usecaseが独自トランザクションを管理するためDB直接書き込みを使用
	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-ok@example.com").
		WithAtname("createok").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("create-ok-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Create OK Topic").
		Build()

	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("テストページ").
		Build()
	testutil.NewPageRevisionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("下書きタイトル").
		Build()

	handler := setupHandler(t, db, queries)

	form := url.Values{}
	form.Set("title", "テスト編集提案")
	form.Set("body", "提案の説明")
	form.Set("csrf_token", "test-csrf-token")
	form.Add("draft_page_ids", string(draftPageID))

	req := newPostRequest(t, "/s/create-ok-sp/topics/1/suggestions", map[string]string{
		"space_identifier": "create-ok-sp",
		"topic_number":     "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "createok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}

	loc := rr.Header().Get("Location")
	if !strings.HasPrefix(loc, "/s/create-ok-sp/topics/1/suggestions/") {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}
