package draft_page_index_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/draft_page_index"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

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
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, getDraftPagesUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpiempty"})
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
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, getDraftPagesUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpidrafts"})
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

// Regression test verifying that a suggestion button is shown per topic group
// on the draft list screen and links to the correct creation screen path.
//
// [Ja] 下書き一覧画面でトピックグループごとに編集提案ボタンが表示され、
// 正しい作成画面のパスにリンクすることを検証する回帰テスト。
func TestIndex_編集提案ボタンがトピックグループに表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpi-sugg-btn@example.com").
		WithAtname("dpisuggbtn").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dpi-sugg-btn").
		WithName("提案ボタンスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("提案ボタントピック").
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
		WithTitle("対象ページ").
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("下書き").
		WithBody("本文").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getDraftPagesUC := usecase.NewGetDraftPagesUsecase(draftPageRepo)

	handler := draft_page_index.NewHandler(cfg, getDraftPagesUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/drafts", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "dpisuggbtn"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "新規編集提案") {
		t.Error("suggestion button not found in response")
	}
	if !strings.Contains(body, "/s/dpi-sugg-btn/topics/2/suggestions/new") {
		t.Error("suggestion new path not found in response")
	}
}
