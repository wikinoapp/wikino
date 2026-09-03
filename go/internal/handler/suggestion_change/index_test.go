package suggestion_change_test

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
	suggestionchangehandler "github.com/wikinoapp/wikino/go/internal/handler/suggestion_change"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func newRequest(t *testing.T, method string, path string, params map[string]string, body io.Reader) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, body)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func setupHandler(t *testing.T, db *sql.DB, queries *query.Queries) *suggestionchangehandler.Handler {
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
	suggestionCommentRepo := repository.NewSuggestionCommentRepository(queries)
	userRepo := repository.NewUserRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	pageRevisionRepo := repository.NewPageRevisionRepository(queries)

	getSuggestionDetailUC := usecase.NewGetSuggestionDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, suggestionRepo, suggestionPageRepo, suggestionCommentRepo, pageRepo, userRepo)
	getSuggestionDiffUC := usecase.NewGetSuggestionDiffUsecase(pageRevisionRepo)

	return suggestionchangehandler.NewHandler(
		cfg,
		getSuggestionDetailUC,
		getSuggestionDiffUC,
	)
}

func TestIndex_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newRequest(t, http.MethodGet, "/s/nonexistent/suggestions/1/changes", map[string]string{
		"space_identifier":  "nonexistent",
		"suggestion_number": "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_不正な提案番号で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newRequest(t, http.MethodGet, "/s/test-space/suggestions/abc/changes", map[string]string{
		"space_identifier":  "test-space",
		"suggestion_number": "abc",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestIndex_公開トピックの差分を未ログインで閲覧できる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-pub@example.com").
		WithAtname("scpub").
		WithName("提案者").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-pub-space").
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

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Original").
		WithBody("元の本文").
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("差分テスト提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()
	testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		WithTitle("提案タイトル").
		WithBody("提案本文").
		WithBodyHTML("<p>提案本文</p>").
		Build()

	handler := setupHandler(t, db, queries)

	req := newRequest(t, http.MethodGet, "/s/sc-pub-space/suggestions/1/changes", map[string]string{
		"space_identifier":  "sc-pub-space",
		"suggestion_number": "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "差分テスト提案") {
		t.Error("response should contain suggestion title")
	}
	for _, want := range []string{
		`<title>変更内容: 差分テスト提案 - 編集提案 #1 | 公開トピック | Public Space</title>`,
		`<meta property="og:title" content="変更内容: 差分テスト提案 - 編集提案 #1 | 公開トピック | Public Space">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
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

	// The visible trail runs through the topic's suggestion list, links to the suggestion detail under
	// the same name that screen gives itself, and ends with a localized, non-linked current item for
	// this changes screen. Scope the assertions to the breadcrumb because the same labels also appear
	// in the page title and body.
	//
	// [Ja] 見た目の経路はトピックの編集提案一覧を通り、編集提案詳細へ同画面が自身に付けるのと同じ名前で
	// リンクし、この変更差分画面を示すローカライズ済みの非リンクな現在項目で締める。同じラベルはページ
	// タイトルや本文にも出るため、パンくず内に絞って検証する。
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
		`href="/s/sc-pub-space/topics/1/suggestions"`,
		`href="/s/sc-pub-space/suggestions/1"`,
		"差分テスト提案",
		`aria-current="page"`,
		"変更内容",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("breadcrumb does not contain %q", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/sc-pub-space/suggestions/1/changes"`) {
		t.Error("current changes breadcrumb item must not be a link")
	}

	// An indexable public screen declares an absolute self-referencing canonical URL built from the
	// stored identifier. An empty href would resolve to whatever URL was requested instead.
	//
	// [Ja] インデックス対象の公開画面は、保存済みの識別子から組み立てた自己参照の絶対 URL を正規 URL
	// として宣言する。空の href だとリクエストされた URL に解決されてしまう。
	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/s/sc-pub-space/suggestions/1/changes">`,
		`<meta property="og:url" content="https://localhost/s/sc-pub-space/suggestions/1/changes">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}

	// Being indexable, the screen publishes its full trail through the current changes screen as
	// BreadcrumbList JSON-LD built from the same items as the visible breadcrumb. A signed-out viewer
	// starts at the public space, so /home must not appear in the machine-readable copy either.
	//
	// [Ja] インデックス対象の画面のため、見た目のパンくずと同じ項目列から作った BreadcrumbList JSON-LD
	// で現在の変更差分画面までの経路を公開する。未ログインの閲覧者は公開スペースから始まるので、機械可読な
	// 複製にも /home が出てはならない。
	for _, want := range []string{
		`<script type="application/ld+json">`,
		`"@type":"BreadcrumbList"`,
		`"position":1,"name":"Public Space","item":"https://localhost/s/sc-pub-space"`,
		`"position":2,"name":"公開トピック","item":"https://localhost/s/sc-pub-space/topics/1"`,
		// The suggestion sits under the list it belongs to, and carries the same name the detail
		// screen gives itself, so both screens describe this URL the same way.
		//
		// [Ja] 編集提案は所属する一覧の下に置かれ、詳細画面が自身に付けるのと同じ名前を持つ。これにより
		// 両画面がこの URL を同じように表す。
		`"position":3,"name":"編集提案","item":"https://localhost/s/sc-pub-space/topics/1/suggestions"`,
		`"position":4,"name":"差分テスト提案","item":"https://localhost/s/sc-pub-space/suggestions/1"`,
		`"position":5,"name":"変更内容"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}
	if strings.Contains(body, `https://localhost/home`) {
		t.Error("未ログインの公開画面の構造化データが認証必須の /home を指している")
	}
}

func TestIndex_非公開トピックを未ログインで閲覧すると404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("sc-priv@example.com").
		WithAtname("scpriv").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sc-priv-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(1). // private
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("非公開提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	req := newRequest(t, http.MethodGet, "/s/sc-priv-space/suggestions/1/changes", map[string]string{
		"space_identifier":  "sc-priv-space",
		"suggestion_number": "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Index(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}
