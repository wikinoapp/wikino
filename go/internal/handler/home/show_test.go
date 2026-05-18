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
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
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
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo, topicRepo)

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

	// The "joined topics" section is always rendered with its heading; with 0 topics
	// the empty-state message should appear instead of any topic card.
	// [Ja] 「参加中のトピック」セクションは常に見出しが描画される。0 件のときは
	// トピックカードの代わりに空状態メッセージが表示される。
	if !strings.Contains(body, "参加中のトピック") {
		t.Error("joined topics heading not found in response")
	}
	if !strings.Contains(body, "参加中のトピックは") {
		t.Error("no joined topics empty state not found in response")
	}
	// The "new topic" button has no global URL in Rails so the home page does not render it
	// when the user has no joined spaces (the target space cannot be derived).
	// [Ja] Rails には新規トピックのグローバル URL が無いため、参加中スペースが 0 件のときは
	// ホーム画面に「新規トピック」ボタンを描画しない。
	if strings.Contains(body, "新規トピック") {
		t.Error("new topic button should not be rendered when there are no joined spaces")
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
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo, topicRepo)

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

	// Verify the SpaceIcon (first-letter label and deterministic background color) is rendered for each space.
	// We assert the IconBackgroundColor() output directly since FNV-1a maps each identifier to one of the
	// 12 palette colors and the value would otherwise be brittle to read off by hand.
	//
	// [Ja] 各スペースカードに SpaceIcon (頭文字ラベルと決定論的な背景色) がレンダリングされていることを検証する。
	// FNV-1a で 12 色パレットから決まる背景色は手計算ではなく IconBackgroundColor() の戻り値で比較する。
	spaces := []struct {
		identifier string
		label      string
	}{
		{identifier: "home-space-1", label: "H"},
		{identifier: "home-space-2", label: "H"},
	}
	for _, s := range spaces {
		vm := viewmodel.Space{Identifier: viewmodel.SpaceIdentifier(s.identifier)}
		expectedBg := "background-color: " + vm.IconBackgroundColor()
		if !strings.Contains(body, expectedBg) {
			t.Errorf("space icon background-color for %q (%q) not found in response", s.identifier, expectedBg)
		}
		expectedLabel := ">" + s.label + "</div>"
		if !strings.Contains(body, expectedLabel) {
			t.Errorf("space icon label for %q (%q) not found in response", s.identifier, expectedLabel)
		}
	}

	// The "new topic" button is rendered with one of the joined space's identifiers when the
	// user has at least one joined space (Rails has no global new-topic page; topic creation
	// requires a space). The exact identifier depends on sm.joined_at ordering, so accept either.
	// [Ja] 参加中スペースが 1 件以上あるときは、いずれかのスペースの identifier を埋め込んだ
	// 「新規トピック」ボタンが描画される (Rails にはグローバルな新規トピック画面が無いため)。
	// 並び順は sm.joined_at DESC のため、どちらの identifier が先頭になるかは決定論的に確定しない。
	if !strings.Contains(body, "新規トピック") {
		t.Error("new topic button label not found in response")
	}
	hasFirstLink := strings.Contains(body, "/s/home-space-1/topics/new")
	hasSecondLink := strings.Contains(body, "/s/home-space-2/topics/new")
	if !hasFirstLink && !hasSecondLink {
		t.Error("new topic button link to one of the joined spaces not found in response")
	}
}

func TestShow_WithJoinedTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("home-topics@example.com").
		WithAtname("hometopics").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("home-topics-space").
		WithName("ホームトピックスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(7).
		WithName("ホームトピックA").
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
		WithTitle("公開ページ1").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("公開ページ2").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	sidebarHelper := sidebar.NewHelper(topicRepo, draftPageRepo)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo, topicRepo)

	handler := home.NewHandler(cfg, getHomeShowUC, sidebarHelper)

	req := httptest.NewRequest(http.MethodGet, "/home", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "hometopics"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "参加中のトピック") {
		t.Error("joined topics heading not found in response")
	}
	if !strings.Contains(body, "ホームトピックA") {
		t.Error("topic name not found in response")
	}
	if !strings.Contains(body, "/s/home-topics-space/topics/7") {
		t.Error("topic link not found in response")
	}
	if !strings.Contains(body, "2 ページ") {
		t.Error("published pages count not found in response")
	}
	if strings.Contains(body, "参加中のトピックは") {
		t.Error("topics empty state should not be shown when topics exist")
	}
}
