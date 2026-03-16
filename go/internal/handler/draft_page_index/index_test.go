package draft_page_index_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/draft_page_index"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestIndex_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	flashMgr := session.NewFlashManager("", false, true)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, flashMgr, getDraftPagesUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	location := rr.Header().Get("Location")
	if location != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", location)
	}
}

func TestIndex_Empty(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-empty@example.com").
		WithAtname("dpiempty").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	flashMgr := session.NewFlashManager("", false, true)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, flashMgr, getDraftPagesUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	req.Header.Set("Accept-Language", "ja")
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "dpiempty"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "下書き") {
		t.Error("heading not found in response")
	}

	if !strings.Contains(body, "下書きはありません") {
		t.Error("empty message not found in response")
	}
}

func TestIndex_WithDrafts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-drafts@example.com").
		WithAtname("dpidrafts").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dpi-drafts-space").
		WithName("テストスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
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
		WithBody("下書き本文").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	flashMgr := session.NewFlashManager("", false, true)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, flashMgr, getDraftPagesUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	req.Header.Set("Accept-Language", "ja")
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "dpidrafts"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "テストスペース") {
		t.Error("space name not found in response")
	}

	if !strings.Contains(body, "テストトピック") {
		t.Error("topic name not found in response")
	}

	if !strings.Contains(body, "下書きタイトル") {
		t.Error("draft page title not found in response")
	}

	if !strings.Contains(body, "/s/dpi-drafts-space/pages/1/edit") {
		t.Error("edit page link not found in response")
	}
}
