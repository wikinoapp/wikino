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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// ホームにパンくずは無いが、ナビバーのために共有ヘッダーは描画される。<main> の外
	// (#mainへのスキップリンクが飛ばせる必要があるため)・この画面の本文幅max-w-3xlで出る。
	// ここではバーがヘッダーの唯一の中身になるため、ヘッダーはバーと一緒に切り替わり、ブレーク
	// ポイント未満で中身の無いbannerランドマークが残らない。
	if !strings.Contains(body, `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("共通のヘッダーがmax-w-3xlのコンテンツ幅を保っていない")
	}
	if !strings.Contains(body, `<header class="pt-4 hidden md:block">`) {
		t.Error("共通のヘッダーが持っているナビゲーションバーと一緒に切り替わっていない")
	}
	if strings.Contains(body, `aria-label="パンくずリスト"`) {
		t.Error("ホームにパンくずのランドマークが描画されている")
	}
	if header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`); header == -1 || main == -1 || header > main {
		t.Errorf("共通のヘッダー (位置%d) が <main> (位置%d) より前にない", header, main)
	}

	if !strings.Contains(body, "ホーム") {
		t.Error("レスポンスに見出しが見つからない")
	}
	if !strings.Contains(body, "Wikinoへようこそ") {
		t.Error("レスポンスにウェルカムの空状態のメッセージが見つからない")
	}
	// home_welcome_description_htmlは前後2文の間に <br class="md:hidden"/> を挟むため、
	// 半分ずつ独立して検証する。
	if !strings.Contains(body, "まずはスペースを作成して") || !strings.Contains(body, "ページを書き始めましょう") {
		t.Error("レスポンスにウェルカムの空状態の説明が見つからない")
	}
	if !strings.Contains(body, "/spaces/new") {
		t.Error("レスポンスに新規スペースのリンクが見つからない")
	}

	// スペース / トピック / 下書きがすべて0件のとき、ホーム本体は1つのウェルカム空状態に
	// 統合されるため、セクション見出し `home_joined_spaces_heading` (= "参加中のスペース") は
	// 描画されない。
	if strings.Contains(body, "参加中のスペース") {
		t.Error("すべてが空なのに参加中のスペースのセクション見出しが描画されている")
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "ホームスペース1") {
		t.Error("レスポンスに1つ目のスペース名が見つからない")
	}
	if !strings.Contains(body, "ホームスペース2") {
		t.Error("レスポンスに2つ目のスペース名が見つからない")
	}
	if !strings.Contains(body, "/s/home-space-1") {
		t.Error("レスポンスに1つ目のスペースのリンクが見つからない")
	}
	if !strings.Contains(body, "/s/home-space-2") {
		t.Error("レスポンスに2つ目のスペースのリンクが見つからない")
	}
	if strings.Contains(body, "Wikinoへようこそ") {
		t.Error("スペースがあるのにウェルカムの空状態が表示されている")
	}

	// 各スペースカードにSpaceIcon (頭文字ラベルと決定論的な背景色) がレンダリングされていることを検証する。
	// FNV-1aで12色パレットから決まる背景色は手計算ではなくIconBackgroundColor() の戻り値で比較する。
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
			t.Errorf("レスポンスに%q (%q) のスペースアイコンのbackground-colorが見つからない", s.identifier, expectedBg)
		}
		expectedLabel := ">" + s.label + "</div>"
		if !strings.Contains(body, expectedLabel) {
			t.Errorf("レスポンスに%q (%q) のスペースアイコンのラベルが見つからない", s.identifier, expectedLabel)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "参加中のトピック") {
		t.Error("レスポンスに参加中のトピックの見出しが見つからない")
	}
	if !strings.Contains(body, "ホームトピックA") {
		t.Error("レスポンスにトピック名が見つからない")
	}
	if !strings.Contains(body, "/s/home-topics-space/topics/7") {
		t.Error("レスポンスにトピックのリンクが見つからない")
	}
	if strings.Contains(body, "参加中のトピックは") {
		t.Error("トピックがあるのにトピックの空状態が表示されている")
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 下書きがあるとき、ホーム本体の下書きセクションが見出しを描画する。サイドバー廃止後は
	// 下書き一覧が現れる唯一の場所。
	if !strings.Contains(body, "下書きのページ") {
		t.Error("ホームのコンテンツに下書きの見出しが見つからない")
	}
	if !strings.Contains(body, "下書きタイトル") {
		t.Error("レスポンスに下書きのタイトルが見つからない")
	}
	// スペース名・区切り・公開範囲アイコン・トピック名は個別の要素として描画されるため、
	// それぞれが含まれることを別々に検証する
	if !strings.Contains(body, "ホーム下書きスペース") {
		t.Error("レスポンスに下書きのスペース名が見つからない")
	}
	if !strings.Contains(body, "ホーム下書きトピック") {
		t.Error("レスポンスに下書きのトピック名が見つからない")
	}
	// 下書きカードはページ編集 (PageEditPath) へのリンクを描画する
	if !strings.Contains(body, "/s/home-drafts-space/pages/11/edit") {
		t.Error("レスポンスに下書きの編集リンクが見つからない")
	}
	// ホームの下書きセクション見出しの「全て見る」リンクは /draftsに張られる。
	// リンクは下書きが1件以上あるときだけ描画される (0件時の挙動はTestShow_DraftPagesEmptyで検証)。
	if !strings.Contains(body, `href="/drafts"`) {
		t.Error(`レスポンスに /draftsへの「すべて表示」リンクが見つからない`)
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

	// 統合ウェルカム空状態は スペース / トピック / 下書き が全て0件のときのみ表示される。
	// 下書きセクション固有の空状態を検証するため、ここでは参加スペースを1件だけ用意し、
	// ホーム本体を3セクション構成に分岐させる。
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	// 参加スペースが1件以上あるため、ホーム本体は3セクション構成のまま描画され、
	// 下書きが0件のときは下書きセクション固有の空状態が表示される。
	if !strings.Contains(body, "下書きのページは") {
		t.Error("レスポンスに下書きが無いときの空状態が見つからない")
	}

	// 下書き0件のときは、ホーム本体の /draftsへの「全て見る」リンクは描画されない。
	if strings.Contains(body, `href="/drafts"`) {
		t.Error(`下書きが0件なのに /draftsへの「すべて表示」リンクがある`)
	}
}
