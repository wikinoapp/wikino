package suggestion_close_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	suggestionclosehandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_close"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
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

// setupHandler はテスト用の編集提案クローズハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries, db *sql.DB) *suggestionclosehandler.Handler {
	t.Helper()

	flashMgr := session.NewFlashManager("localhost", false, false)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)

	closeSuggestionUC := usecase.NewCloseSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo)

	return suggestionclosehandler.NewHandler(
		flashMgr,
		closeSuggestionUC,
	)
}

func TestCreate_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()
	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/test/suggestions/1/close", map[string]string{
		"space_identifier":  "test",
		"suggestion_number": "1",
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

func TestCreate_存在しない編集提案で404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("close-nosugg@example.com").
		WithAtname("closenosugg").
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/nonexistent/suggestions/999/close", map[string]string{
		"space_identifier":  "nonexistent",
		"suggestion_number": "999",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "closenosugg"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_スペースメンバーでないユーザーは403が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("close-owner@example.com").
		WithAtname("closeowner").
		Build()
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("close-nonmember@example.com").
		WithAtname("closenonmember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("close-forbid-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/close-forbid-sp/suggestions/1/close", map[string]string{
		"space_identifier":  "close-forbid-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: nonMemberID, Atname: "closenonmember"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCreate_権限のない一般メンバーは403が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("close-ownerX@example.com").
		WithAtname("closeownerX").
		Build()
	memberID := testutil.NewUserBuilder(t, tx).
		WithEmail("close-member@example.com").
		WithAtname("closemember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("close-member-sp").
		Build()
	ownerSmID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	memberSmID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberID).
		WithRole(int32(model.SpaceMemberRoleMember)).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(memberSmID).
		WithRole(int32(model.TopicMemberRoleMember)).
		Build()
	// 提案はオーナーが作成（一般メンバーは作成者ではない）
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(ownerSmID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/close-member-sp/suggestions/1/close", map[string]string{
		"space_identifier":  "close-member-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: memberID, Atname: "closemember"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusForbidden)
	}
}

func TestCreate_作成者はクローズできる(t *testing.T) {
	t.Parallel()

	// UseCaseが独自トランザクションを管理するためDB直接書き込みを使用
	db := testutil.GetTestDB()
	queries := query.New(db)

	ownerID := testutil.NewUserBuilderDB(t, db).
		WithEmail("close-creator-owner@example.com").
		WithAtname("closecreatorowner").
		Build()
	creatorID := testutil.NewUserBuilderDB(t, db).
		WithEmail("close-creator@example.com").
		WithAtname("closecreator").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("close-creator-sp").
		Build()
	testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	creatorSmID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(creatorID).
		WithRole(int32(model.SpaceMemberRoleMember)).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(creatorSmID).
		Build()
	testutil.NewSuggestionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(creatorSmID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/close-creator-sp/suggestions/1/close", map[string]string{
		"space_identifier":  "close-creator-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: creatorID, Atname: "closecreator"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/close-creator-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}

func TestCreate_スペースオーナーがクローズできる(t *testing.T) {
	t.Parallel()

	// UseCaseが独自トランザクションを管理するためDB直接書き込みを使用
	db := testutil.GetTestDB()
	queries := query.New(db)

	ownerID := testutil.NewUserBuilderDB(t, db).
		WithEmail("close-ok-owner@example.com").
		WithAtname("closeokowner").
		Build()
	creatorID := testutil.NewUserBuilderDB(t, db).
		WithEmail("close-ok-creator@example.com").
		WithAtname("closeokcreator").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("close-ok-sp").
		Build()
	testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	creatorSmID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(creatorID).
		WithRole(int32(model.SpaceMemberRoleMember)).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewSuggestionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(creatorSmID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/close-ok-sp/suggestions/1/close", map[string]string{
		"space_identifier":  "close-ok-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "closeokowner"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/close-ok-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}

func TestCreate_トピック管理者がクローズできる(t *testing.T) {
	t.Parallel()

	// UseCaseが独自トランザクションを管理するためDB直接書き込みを使用
	db := testutil.GetTestDB()
	queries := query.New(db)

	ownerID := testutil.NewUserBuilderDB(t, db).
		WithEmail("close-admin-owner@example.com").
		WithAtname("closeadminowner").
		Build()
	adminID := testutil.NewUserBuilderDB(t, db).
		WithEmail("close-admin@example.com").
		WithAtname("closeadmin").
		Build()
	creatorID := testutil.NewUserBuilderDB(t, db).
		WithEmail("close-admin-creator@example.com").
		WithAtname("closeadmincreator").
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("close-admin-sp").
		Build()
	testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	adminSmID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(adminID).
		WithRole(int32(model.SpaceMemberRoleMember)).
		Build()
	creatorSmID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(creatorID).
		WithRole(int32(model.SpaceMemberRoleMember)).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(adminSmID).
		Build()
	testutil.NewSuggestionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(creatorSmID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/close-admin-sp/suggestions/1/close", map[string]string{
		"space_identifier":  "close-admin-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: adminID, Atname: "closeadmin"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/close-admin-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}

func TestCreate_反映済みの編集提案はクローズできずエラーが返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("close-applied@example.com").
		WithAtname("closeapplied").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("close-applied-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusApplied).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/close-applied-sp/suggestions/1/close", map[string]string{
		"space_identifier":  "close-applied-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "closeapplied"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// 反映済みの場合はリダイレクトでエラーメッセージが表示される
	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/close-applied-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}

func TestCreate_クローズ済みの編集提案はべき等に成功する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	db := testutil.GetTestDB()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("close-idem@example.com").
		WithAtname("closeidem").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("close-idem-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusClosed).
		Build()

	handler := setupHandler(t, queries, db)

	form := url.Values{}
	form.Set("csrf_token", "test-csrf-token")

	req := newPostRequest(t, "/s/close-idem-sp/suggestions/1/close", map[string]string{
		"space_identifier":  "close-idem-sp",
		"suggestion_number": "1",
	}, form)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "closeidem"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}
	loc := rr.Header().Get("Location")
	if loc != "/s/close-idem-sp/suggestions/1" {
		t.Errorf("wrong redirect location: got %q", loc)
	}
}
