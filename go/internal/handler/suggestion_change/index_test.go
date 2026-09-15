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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "差分テスト提案") {
		t.Error("レスポンスに編集提案のタイトルが含まれていない")
	}
	for _, want := range []string{
		`<title>変更内容: 差分テスト提案 - 編集提案 #1 | 公開トピック | Public Space</title>`,
		`<meta property="og:title" content="変更内容: 差分テスト提案 - 編集提案 #1 | 公開トピック | Public Space">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	// パンくずヘッダーはレイアウトが描画するため、<main> の外に出る (#mainへのスキップ
	// リンクが飛ばせる必要があるため)。この画面の本文幅max-w-3xlも維持する。
	if !strings.Contains(body, `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("共通のパンくずヘッダーがmax-w-3xlのコンテンツ幅を保っていない")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("共通のパンくずヘッダー (位置%d) が <main> (位置%d) より前にない", header, main)
	}

	// /homeは認証必須のため、この公開画面が経路の起点として提示してはいけない。リンクは未ログインの
	// 訪問者をログイン画面へ送ってしまう。
	if strings.Contains(body, `href="/home"`) {
		t.Error("未ログインの公開画面のパンくずが認証必須の /homeを指している")
	}

	// 見た目の経路はトピックの編集提案一覧を通り、編集提案詳細へ同画面が自身に付けるのと同じ名前で
	// リンクし、この変更差分画面を示すローカライズ済みの非リンクな現在項目で締める。同じラベルはページ
	// タイトルや本文にも出るため、パンくず内に絞って検証する。
	breadcrumbStart := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if breadcrumbStart == -1 {
		t.Fatal("レスポンスにパンくずのナビゲーションが含まれていない")
	}
	breadcrumbEnd := strings.Index(body[breadcrumbStart:], "</nav>")
	if breadcrumbEnd == -1 {
		t.Fatal("パンくずのナビゲーションに閉じタグが無い")
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
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/sc-pub-space/suggestions/1/changes"`) {
		t.Error("現在の変更内容のパンくずの項目がリンクになっている")
	}

	// インデックス対象の公開画面は、保存済みの識別子から組み立てた自己参照の絶対URLを正規URL
	// として宣言する。空のhrefだとリクエストされたURLに解決されてしまう。
	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/s/sc-pub-space/suggestions/1/changes">`,
		`<meta property="og:url" content="https://localhost/s/sc-pub-space/suggestions/1/changes">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	// インデックス対象の画面のため、見た目のパンくずと同じ項目列から作ったBreadcrumbList JSON-LD
	// で現在の変更差分画面までの経路を公開する。未ログインの閲覧者は公開スペースから始まるので、機械可読な
	// 複製にも /homeが出てはならない。
	for _, want := range []string{
		`<script type="application/ld+json">`,
		`"@type":"BreadcrumbList"`,
		`"position":1,"name":"Public Space","item":"https://localhost/s/sc-pub-space"`,
		`"position":2,"name":"公開トピック","item":"https://localhost/s/sc-pub-space/topics/1"`,
		// 編集提案は所属する一覧の下に置かれ、詳細画面が自身に付けるのと同じ名前を持つ。これにより
		// 両画面がこのURLを同じように表す。
		`"position":3,"name":"編集提案","item":"https://localhost/s/sc-pub-space/topics/1/suggestions"`,
		`"position":4,"name":"差分テスト提案","item":"https://localhost/s/sc-pub-space/suggestions/1"`,
		`"position":5,"name":"変更内容"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if strings.Contains(body, `https://localhost/home`) {
		t.Error("未ログインの公開画面の構造化データが認証必須の /homeを指している")
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}
