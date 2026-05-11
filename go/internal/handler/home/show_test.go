package home_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/home"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestShow_Empty(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("home-empty@example.com").
		WithAtname("homeempty").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo)

	handler := home.NewHandler(cfg, getHomeShowUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/home", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "homeempty"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "ホーム") {
		t.Error("heading not found in response")
	}
	if !strings.Contains(body, "参加中のスペース") {
		t.Error("joined spaces heading not found in response")
	}
	if !strings.Contains(body, "まずはスペースを作成しましょう") {
		t.Error("empty state description not found in response")
	}
	if !strings.Contains(body, "/spaces/new") {
		t.Error("new space link not found in response")
	}
}

func TestShow_WithSpaces(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("home-spaces@example.com").
		WithAtname("homespaces").
		Build()
	firstSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("home-space-1").
		WithName("ホームスペース1").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(firstSpaceID).
		WithUserID(userID).
		Build()
	secondSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("home-space-2").
		WithName("ホームスペース2").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(secondSpaceID).
		WithUserID(userID).
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo)

	handler := home.NewHandler(cfg, getHomeShowUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/home", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "homespaces"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "ホームスペース1") {
		t.Error("first space name not found in response")
	}
	if !strings.Contains(body, "ホームスペース2") {
		t.Error("second space name not found in response")
	}
	if !strings.Contains(body, "/s/home-space-1") {
		t.Error("first space link not found in response")
	}
	if !strings.Contains(body, "/s/home-space-2") {
		t.Error("second space link not found in response")
	}
	if strings.Contains(body, "まずはスペースを作成しましょう") {
		t.Error("empty state description should not be shown when spaces exist")
	}
}
