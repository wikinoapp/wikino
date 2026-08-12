package suggestion_test

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	suggestionhandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// newSuggestionRequest はchiのURLパラメータ付きHTTPリクエストを作成するヘルパーです
func newSuggestionRequest(t *testing.T, method string, path string, params map[string]string, body io.Reader) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, body)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// setupHandler はテスト用の編集提案ハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, db *sql.DB, queries *query.Queries) *suggestionhandler.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	suggestionPageRevisionRepo := repository.NewSuggestionPageRevisionRepository(queries)
	userRepo := repository.NewUserRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	pageRevisionRepo := repository.NewPageRevisionRepository(queries)
	flashMgr := session.NewFlashManager("localhost", false, false)

	suggestionCommentRepo := repository.NewSuggestionCommentRepository(queries)

	getSuggestionListUC := usecase.NewGetSuggestionListUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, userRepo)
	getSuggestionDetailUC := usecase.NewGetSuggestionDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo)
	getSuggestionEditUC := usecase.NewGetSuggestionEditUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, userRepo)
	getSuggestionNewUC := usecase.NewGetSuggestionNewUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, draftPageRepo)
	suggestionCreateValidator := validator.NewSuggestionCreateValidator(draftPageRepo, pageRepo)
	createSuggestionUC := usecase.NewCreateSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionPageRevisionRepo, draftPageRepo, pageRevisionRepo, suggestionCreateValidator)
	suggestionUpdateValidator := validator.NewSuggestionUpdateValidator()
	updateSuggestionUC := usecase.NewUpdateSuggestionUsecase(db, spaceRepo, spaceMemberRepo, topicMemberRepo, suggestionRepo, suggestionUpdateValidator)

	return suggestionhandler.NewHandler(
		cfg,
		flashMgr,
		getSuggestionListUC,
		getSuggestionDetailUC,
		getSuggestionEditUC,
		getSuggestionNewUC,
		createSuggestionUC,
		updateSuggestionUC,
	)
}

func TestIndex_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/nonexistent/topics/1/suggestions", map[string]string{
		"space_identifier": "nonexistent",
		"topic_number":     "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_不正なトピック番号で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/test-space/topics/abc/suggestions", map[string]string{
		"space_identifier": "test-space",
		"topic_number":     "abc",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_存在しないトピックで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-noexist").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/si-noexist/topics/999/suggestions", map[string]string{
		"space_identifier": "si-noexist",
		"topic_number":     "999",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_公開トピックの編集提案一覧を未ログインで閲覧できる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("si-pub@example.com").
		WithAtname("sipub").
		WithName("提案者").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-pub-space").
		WithName("Public Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/si-pub-space/topics/1/suggestions", map[string]string{
		"space_identifier": "si-pub-space",
		"topic_number":     "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "テスト提案") {
		t.Error("response should contain suggestion title")
	}
	if !strings.Contains(body, "提案者") {
		t.Error("response should contain creator name")
	}
	if strings.Contains(body, "/s/si-pub-space/topics/1/suggestions/new") {
		t.Error("新規編集提案ボタンが未ログインユーザーに表示されてはならない")
	}

	// The breadcrumb header comes from the layout, so it renders outside <main> (the #main skip
	// link has to bypass it) and keeps this screen's max-w-3xl content width.
	//
	// [Ja] パンくずヘッダーはレイアウトが描画するため、<main> の外に出る (#main へのスキップ
	// リンクが飛ばせる必要があるため)。この画面の本文幅 max-w-3xl も維持する。
	if !strings.Contains(body, `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("shared breadcrumb header should keep the max-w-3xl content width")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("shared breadcrumb header (index %d) must precede <main> (index %d)", header, main)
	}

	// /home is behind authentication, so this public screen must not offer it as the trail's root:
	// the link would send a signed-out visitor to the sign-in screen.
	//
	// [Ja] /home は認証必須のため、この公開画面が経路の起点として提示してはいけない。リンクは未ログインの
	// 訪問者をログイン画面へ送ってしまう。
	if strings.Contains(body, `href="/home"`) {
		t.Error("未ログインの公開画面のパンくずが認証必須の /home を指している")
	}

	// An indexable public screen declares an absolute self-referencing canonical URL built from the
	// stored identifier. An empty href would resolve to whatever URL was requested instead.
	//
	// [Ja] インデックス対象の公開画面は、保存済みの識別子から組み立てた自己参照の絶対 URL を正規 URL
	// として宣言する。空の href だとリクエストされた URL に解決されてしまう。
	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/s/si-pub-space/topics/1/suggestions">`,
		`<meta property="og:url" content="https://localhost/s/si-pub-space/topics/1/suggestions">`,
		`<title>編集提案 | 公開トピック | Public Space</title>`,
		`<meta property="og:title" content="編集提案 | 公開トピック | Public Space">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}

	// The list is the current page, so the visible trail runs through the topic and ends with the
	// list itself as a non-linked current item. Scope the assertions to the breadcrumb because the
	// same labels also appear in the heading and metadata.
	//
	// [Ja] 一覧は現在地のため、見た目の経路はトピックを通り、一覧自身の非リンクな現在項目で締める。
	// 同じラベルは見出しとメタ情報にも出るため、パンくず内に絞って検証する。
	breadcrumbStart := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if breadcrumbStart == -1 {
		t.Fatal("response should contain the breadcrumb navigation")
	}
	breadcrumbEnd := strings.Index(body[breadcrumbStart:], "</nav>")
	if breadcrumbEnd == -1 {
		t.Fatal("breadcrumb navigation should have a closing tag")
	}
	breadcrumb := body[breadcrumbStart : breadcrumbStart+breadcrumbEnd]
	for _, want := range []string{
		`href="/s/si-pub-space"`,
		`href="/s/si-pub-space/topics/1"`,
		`aria-current="page"`,
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("breadcrumb does not contain %q", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/si-pub-space/topics/1/suggestions"`) {
		t.Error("current suggestion list breadcrumb item must not be a link")
	}

	// Being indexable, the list publishes the same trail as BreadcrumbList JSON-LD. The screens under
	// a suggestion declare this URL at the same position, so one URL keeps one place in the
	// hierarchy.
	//
	// [Ja] インデックス対象のため、一覧は同じ経路を BreadcrumbList JSON-LD としても公開する。編集提案
	// 配下の画面はこの URL を同じ位置に宣言するため、1 つの URL が階層上で 1 つの位置を持つ。
	for _, want := range []string{
		`<script type="application/ld+json">`,
		`"@type":"BreadcrumbList"`,
		`"position":1,"name":"Public Space","item":"https://localhost/s/si-pub-space"`,
		`"position":2,"name":"公開トピック","item":"https://localhost/s/si-pub-space/topics/1"`,
		`"position":3,"name":"編集提案"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}
	if strings.Contains(body, `https://localhost/home`) {
		t.Error("未ログインの公開画面の構造化データが認証必須の /home を指している")
	}
}

// Each status tab lists a different set of suggestions, so the closed tab declares itself rather
// than the open tab as its canonical address and identifies its state in the page title.
//
// [Ja] ステータスタブごとに載っている編集提案が異なるため、クローズタブはオープンタブではなく自分
// 自身を正規アドレスとして宣言し、ページタイトルでも状態を識別できるようにする。
func TestIndex_ClosedTabMetadataIdentifiesTabState(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-tab-canonical").
		WithName("Tab Canonical Space").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/si-tab-canonical/topics/1/suggestions?tab=closed", map[string]string{
		"space_identifier": "si-tab-canonical",
		"topic_number":     "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/s/si-tab-canonical/topics/1/suggestions?tab=closed">`,
		`<title>クローズした編集提案 | 公開トピック | Tab Canonical Space</title>`,
		`<meta property="og:title" content="クローズした編集提案 | 公開トピック | Tab Canonical Space">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}
	if strings.Contains(body, `<title>編集提案 | 公開トピック | Tab Canonical Space</title>`) {
		t.Error("closed tab must not reuse the open tab title")
	}
}

func TestIndex_非公開トピックを未ログインで閲覧すると404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-priv1").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/si-priv1/topics/1/suggestions", map[string]string{
		"space_identifier": "si-priv1",
		"topic_number":     "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_クローズタブで反映済みとクローズの提案が表示される(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("si-closed@example.com").
		WithAtname("siclosed").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-closed-sp").
		WithName("Closed Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("クローズテスト").
		WithVisibility(0).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("オープン提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("クローズ提案").
		WithStatus(model.SuggestionStatusClosed).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/si-closed-sp/topics/1/suggestions?tab=closed", map[string]string{
		"space_identifier": "si-closed-sp",
		"topic_number":     "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "クローズ提案") {
		t.Error("response should contain closed suggestion title")
	}
	if strings.Contains(body, "オープン提案") {
		t.Error("response should not contain open suggestion title on closed tab")
	}
}

func TestIndex_非公開トピックをスペースオーナーが閲覧できる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("si-owner@example.com").
		WithAtname("siowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-priv2").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("オーナー閲覧").
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/si-priv2/topics/1/suggestions", map[string]string{
		"space_identifier": "si-priv2",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "siowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "/s/si-priv2/topics/1/suggestions/new") {
		t.Error("新規編集提案ボタンがスペースメンバーに表示されていない")
	}
}

func TestIndex_他スペースのログインユーザーには新規編集提案ボタンが表示されない(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// スペース B（対象スペース、公開トピック）
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-outsider-space").
		WithName("Target Space").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()

	// スペース A 所属のユーザー（スペース B には所属しない）
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("si-outsider@example.com").
		WithAtname("sioutsider").
		Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("si-other-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithUserID(outsiderID).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/si-outsider-space/topics/1/suggestions", map[string]string{
		"space_identifier": "si-outsider-space",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: outsiderID, Atname: "sioutsider"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if strings.Contains(body, "/s/si-outsider-space/topics/1/suggestions/new") {
		t.Error("新規編集提案ボタンが非メンバーに表示されてはならない")
	}
}
