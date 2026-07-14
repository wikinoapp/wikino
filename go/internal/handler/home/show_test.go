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
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)

	handler := home.NewHandler(cfg, getHomeShowUC)

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
	if !strings.Contains(body, "Wikinoへようこそ") {
		t.Error("welcome empty state message not found in response")
	}
	// home_welcome_description_html embeds `<br class="md:hidden"/>` between the two halves,
	// so assert each half independently.
	// [Ja] home_welcome_description_html は前後 2 文の間に <br class="md:hidden"/> を挟むため、
	// 半分ずつ独立して検証する。
	if !strings.Contains(body, "まずはスペースを作成して") || !strings.Contains(body, "ページを書き始めましょう") {
		t.Error("welcome empty state description not found in response")
	}
	if !strings.Contains(body, "/spaces/new") {
		t.Error("new space link not found in response")
	}

	// When the user has no spaces, no topics, and no drafts, the home content collapses
	// into a single welcome empty state, so the per-section heading `home_joined_spaces_heading`
	// (= "参加中のスペース") must not render.
	//
	// [Ja] スペース / トピック / 下書きがすべて 0 件のとき、ホーム本体は 1 つのウェルカム空状態に
	// 統合されるため、セクション見出し `home_joined_spaces_heading` (= "参加中のスペース") は
	// 描画されない。
	if strings.Contains(body, "参加中のスペース") {
		t.Error("joined spaces section heading should not be rendered when everything is empty")
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
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)

	handler := home.NewHandler(cfg, getHomeShowUC)

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
	if strings.Contains(body, "Wikinoへようこそ") {
		t.Error("welcome empty state should not be shown when spaces exist")
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
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)

	handler := home.NewHandler(cfg, getHomeShowUC)

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
	if strings.Contains(body, "参加中のトピックは") {
		t.Error("topics empty state should not be shown when topics exist")
	}
}

func TestShow_WithDraftPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("home-drafts@example.com").
		WithAtname("homedrafts").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("home-drafts-space").
		WithName("ホーム下書きスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("ホーム下書きトピック").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(11).
		WithTitle("公開ページ").
		Build()
	draftTitle := "下書きタイトル"
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle(draftTitle).
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)

	handler := home.NewHandler(cfg, getHomeShowUC)

	req := httptest.NewRequest(http.MethodGet, "/home", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "homedrafts"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// The home content's draft section renders its heading when drafts exist. It is the only place
	// the draft list appears now that the sidebar is gone.
	// [Ja] 下書きがあるとき、ホーム本体の下書きセクションが見出しを描画する。サイドバー廃止後は
	// 下書き一覧が現れる唯一の場所。
	if !strings.Contains(body, "下書きのページ") {
		t.Error("draft pages heading not found in home content")
	}
	if !strings.Contains(body, "下書きタイトル") {
		t.Error("draft page title not found in response")
	}
	// Space name, separator, visibility icon and topic name are rendered as separate elements,
	// so assert each piece is present independently.
	// [Ja] スペース名・区切り・公開範囲アイコン・トピック名は個別の要素として描画されるため、
	// それぞれが含まれることを別々に検証する
	if !strings.Contains(body, "ホーム下書きスペース") {
		t.Error("draft page space name not found in response")
	}
	if !strings.Contains(body, "ホーム下書きトピック") {
		t.Error("draft page topic name not found in response")
	}
	// Link to the page editor for this draft (page number 11).
	// [Ja] 下書きカードはページ編集 (PageEditPath) へのリンクを描画する
	if !strings.Contains(body, "/s/home-drafts-space/pages/11/edit") {
		t.Error("draft page edit link not found in response")
	}
	// The "View all" link in the home draft pages section heading points to /drafts.
	// The link is rendered only when at least one draft exists; the 0-draft case is
	// covered by TestShow_DraftPagesEmpty.
	//
	// [Ja] ホームの下書きセクション見出しの「全て見る」リンクは /drafts に張られる。
	// リンクは下書きが 1 件以上あるときだけ描画される (0 件時の挙動は TestShow_DraftPagesEmpty で検証)。
	if !strings.Contains(body, `href="/drafts"`) {
		t.Error(`"View all" link to /drafts not found in response`)
	}
}

func TestShow_DraftPagesEmpty(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("home-drafts-empty@example.com").
		WithAtname("homedraftsempty").
		Build()

	// The unified welcome empty state only renders when spaces / topics / drafts are all empty.
	// To exercise the per-section drafts empty state we give the user at least one joined space,
	// which forces the home content into the three-section layout.
	//
	// [Ja] 統合ウェルカム空状態は スペース / トピック / 下書き が全て 0 件のときのみ表示される。
	// 下書きセクション固有の空状態を検証するため、ここでは参加スペースを 1 件だけ用意し、
	// ホーム本体を 3 セクション構成に分岐させる。
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("home-drafts-empty-space").
		WithName("空のスペース").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	getHomeShowUC := usecase.NewGetHomeShowUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)

	handler := home.NewHandler(cfg, getHomeShowUC)

	req := httptest.NewRequest(http.MethodGet, "/home", nil)
	ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "homedraftsempty"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// With at least one joined space, the home content stays in the three-section layout
	// and the drafts section renders its own empty state when there are 0 drafts.
	// [Ja] 参加スペースが 1 件以上あるため、ホーム本体は 3 セクション構成のまま描画され、
	// 下書きが 0 件のときは下書きセクション固有の空状態が表示される。
	if !strings.Contains(body, "下書きのページは") {
		t.Error("no draft pages empty state not found in response")
	}

	// When there are 0 drafts, the home content's "View all" link to /drafts must not be rendered.
	// [Ja] 下書き 0 件のときは、ホーム本体の /drafts への「全て見る」リンクは描画されない。
	if strings.Contains(body, `href="/drafts"`) {
		t.Error(`unexpected "View all" link to /drafts found when draft count is 0`)
	}
}
