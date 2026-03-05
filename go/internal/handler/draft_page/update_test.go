package draft_page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/draft_page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *draft_page.Handler {
	t.Helper()

	db := testutil.GetTestDB()

	draftPageRepo := repository.NewDraftPageRepository(queries)

	return draft_page.NewHandler(
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewPageRepository(queries),
		repository.NewTopicRepository(queries),
		repository.NewTopicMemberRepository(queries),
		draftPageRepo,
		usecase.NewAutoSaveDraftPageUsecase(
			db,
			draftPageRepo,
			repository.NewPageRepository(queries),
			repository.NewPageEditorRepository(queries),
			repository.NewTopicRepository(queries),
			repository.NewAttachmentRepository(queries),
		),
		usecase.NewManualSaveDraftPageUsecase(
			db,
			draftPageRepo,
			repository.NewDraftPageRevisionRepository(queries),
		),
	)
}

// newRequestWithChiParams はchiのURLパラメータ付きPATCHリクエストを作成するヘルパーです
func newRequestWithChiParams(t *testing.T, path string, params map[string]string, formData url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestUpdate_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[topic_number]": {"1"},
		"pages_edit_form[title]":        {"Test"},
		"pages_edit_form[body]":         {"body"},
	}
	req := newRequestWithChiParams(t, "/s/my-space/pages/1/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	}, formData)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnauthorized)
	}

	// エラーメッセージを確認
	body := strings.TrimSpace(rr.Body.String())
	if body != "Unauthorized" {
		t.Errorf("wrong error message: got %v want Unauthorized", body)
	}
}

func TestUpdate_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("invalidnum@example.com").
		WithAtname("invalidnum").
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[topic_number]": {"1"},
		"pages_edit_form[title]":        {"Test"},
		"pages_edit_form[body]":         {"body"},
	}
	req := newRequestWithChiParams(t, "/s/my-space/pages/abc/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "abc",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spacenotfound@example.com").
		WithAtname("spacenotfound").
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[topic_number]": {"1"},
		"pages_edit_form[title]":        {"Test"},
		"pages_edit_form[body]":         {"body"},
	}
	req := newRequestWithChiParams(t, "/s/nonexistent/pages/1/draft_page", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_NotSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpowner@example.com").
		WithAtname("dpowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpoutsider@example.com").
		WithAtname("dpoutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-private-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[topic_number]": {"1"},
		"pages_edit_form[title]":        {"Test"},
		"pages_edit_form[body]":         {"body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-private-space/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-private-space",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: outsiderID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_PageNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dppagemissing@example.com").
		WithAtname("dppagemissing").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-page-missing").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[topic_number]": {"1"},
		"pages_edit_form[title]":        {"Test"},
		"pages_edit_form[body]":         {"body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-page-missing/pages/999/draft_page", map[string]string{
		"space_identifier": "dp-page-missing",
		"page_number":      "999",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_TopicPolicyDenied(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dppolicy@example.com").
		WithAtname("dppolicy").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-policy-space").
		Build()
	// メンバーロール（オーナーではない）で作成し、トピックメンバーには登録しない
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(1). // member（非オーナー）
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Restricted Topic").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Restricted Page").
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[topic_number]": {"1"},
		"pages_edit_form[title]":        {"Test"},
		"pages_edit_form[body]":         {"body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-policy-space/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-policy-space",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_InvalidTopicNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpinvalidtopic@example.com").
		WithAtname("dpinvalidtopic").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-invalid-topic").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
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

	handler := setupHandler(t, queries)

	// topic_numberが不正な値
	formData := url.Values{
		"pages_edit_form[topic_number]": {"abc"},
		"pages_edit_form[title]":        {"Test"},
		"pages_edit_form[body]":         {"body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-invalid-topic/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-invalid-topic",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_TopicNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dptopicnotfound@example.com").
		WithAtname("dptopicnotfound").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-topic-notfound").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
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

	handler := setupHandler(t, queries)

	// 存在しないtopic_numberを指定
	formData := url.Values{
		"pages_edit_form[topic_number]": {"999"},
		"pages_edit_form[title]":        {"Test"},
		"pages_edit_form[body]":         {"body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-topic-notfound/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-topic-notfound",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_ResponseContentType(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	// 未認証リクエストでもContent-Typeがtext/plainであることを確認
	formData := url.Values{}
	req := newRequestWithChiParams(t, "/s/my-space/pages/1/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	}, formData)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("wrong content type: got %v want text/plain", contentType)
	}
}
