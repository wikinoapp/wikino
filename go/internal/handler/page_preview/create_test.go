package page_preview_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/page_preview"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler はテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *page_preview.Handler {
	t.Helper()

	getPagePreviewUC := usecase.NewGetPagePreviewUsecase(
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewPageRepository(queries),
		repository.NewTopicRepository(queries),
		repository.NewTopicMemberRepository(queries),
		repository.NewAttachmentRepository(queries),
	)

	return page_preview.NewHandler(getPagePreviewUC)
}

// newPreviewRequest はプレビュー用の POST リクエストを作成するヘルパーです
func newPreviewRequest(t *testing.T, spaceIdentifier, pageNumber, title, body string) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("title", title)
	form.Set("body", body)

	req := httptest.NewRequest(http.MethodPost, "/s/"+spaceIdentifier+"/pages/"+pageNumber+"/preview", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", spaceIdentifier)
	rctx.URLParams.Add("page_number", pageNumber)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	return req
}

func TestCreate(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("preview-space").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(0).
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
		WithTitle("Editing Page").
		Build()

	handler := setupHandler(t, queries)

	req := newPreviewRequest(t, "preview-space", "1", "Preview Title", "# Hello\n\n**bold**")
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Preview Title") {
		t.Error("preview title not found in response")
	}
	if !strings.Contains(body, "<h1") {
		t.Error("rendered heading not found in response")
	}
	if !strings.Contains(body, "<strong>bold</strong>") {
		t.Error("rendered bold text not found in response")
	}
	if !strings.Contains(body, "wikino-markdown") {
		t.Error("markdown wrapper not found in response")
	}
	// The preview is swapped in by htmx, and web/markdown-table.ts reads this attribute off the
	// swapped container to name the scrollable region it wraps a table in.
	//
	// [Ja] プレビューは htmx が差し込み、web/markdown-table.ts はスワップされたコンテナから
	// この属性を読んで、テーブルを包むスクロール領域に名前を付ける。
	if !strings.Contains(body, `data-markdown-table-label="スクロールできる表"`) {
		t.Error("markdown table label not found in response")
	}
}

func TestCreate_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newPreviewRequest(t, "preview-space", "1", "Title", "body")

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestCreate_NotSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("preview-owner@example.com").
		WithAtname("previewowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("preview-outsider@example.com").
		WithAtname("previewoutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("preview-private-space").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(0).
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
		WithTitle("Editing Page").
		Build()

	handler := setupHandler(t, queries)

	req := newPreviewRequest(t, "preview-private-space", "1", "Title", "secret")
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: outsiderID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("preview-invalid@example.com").
		WithAtname("previewinvalid").
		Build()

	handler := setupHandler(t, queries)

	req := newPreviewRequest(t, "preview-space", "abc", "Title", "body")
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}
