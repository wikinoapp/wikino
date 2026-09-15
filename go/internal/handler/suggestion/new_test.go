package suggestion_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestNew_未ログインでサインインにリダイレクトされる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/test/topics/1/suggestions/new", map[string]string{
		"space_identifier": "test",
		"topic_number":     "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("リダイレクト先 = %q、期待値 = %q", loc, "/sign_in")
	}
}

func TestNew_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-nosp@example.com").
		WithAtname("newnosp").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/nonexistent/topics/1/suggestions/new", map[string]string{
		"space_identifier": "nonexistent",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newnosp"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_不正なトピック番号で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-badnum@example.com").
		WithAtname("newbadnum").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/test/topics/abc/suggestions/new", map[string]string{
		"space_identifier": "test",
		"topic_number":     "abc",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newbadnum"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_スペースメンバーでない場合404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-nomember@example.com").
		WithAtname("newnomember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("new-nomember-sp").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/new-nomember-sp/topics/1/suggestions/new", map[string]string{
		"space_identifier": "new-nomember-sp",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newnomember"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestNew_スペースメンバーで正常にフォームが表示される(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-ok@example.com").
		WithAtname("newok").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("new-ok-sp").
		WithName("New OK Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("New OK Topic").
		WithVisibility(0).
		Build()

	// 下書きページを作成
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
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/new-ok-sp/topics/1/suggestions/new", map[string]string{
		"space_identifier": "new-ok-sp",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newok"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "下書きタイトル") {
		t.Error("レスポンスに下書きのタイトルが含まれていない")
	}
	if !strings.Contains(body, "csrf_token") {
		t.Error("レスポンスにCSRFトークンが含まれていない")
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
}

func TestNew_下書きページがない場合でもフォームが表示される(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-nodraft@example.com").
		WithAtname("newnodraft").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("new-nodraft-sp").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/new-nodraft-sp/topics/1/suggestions/new", map[string]string{
		"space_identifier": "new-nodraft-sp",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newnodraft"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}
}

// 経路はこの画面自身で終わるため、末尾の項目はトピックへのリンクではなくaria-currentを持つ
// ラベルになる。同じラベルは見出しとページタイトルにも出るため、パンくず内に絞って検証する。
func TestNew_パンくずが現在地の項目で終わる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("new-crumb@example.com").
		WithAtname("newcrumb").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("new-crumb-sp").
		WithName("New Crumb Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("New Crumb Topic").
		WithVisibility(0).
		Build()

	req := newSuggestionRequest(t, http.MethodGet, "/s/new-crumb-sp/topics/1/suggestions/new", map[string]string{
		"space_identifier": "new-crumb-sp",
		"topic_number":     "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: "newcrumb"})
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setupHandler(t, db, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	breadcrumb := suggestionFormBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/s/new-crumb-sp"`,
		`href="/s/new-crumb-sp/topics/1"`,
		`aria-current="page"`,
		"新規編集提案",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/new-crumb-sp/topics/1/suggestions/new"`) {
		t.Error("現在の編集提案作成のパンくずの項目がリンクになっている")
	}
}

// suggestionFormBreadcrumbはパンくずのナビゲーション部分だけのマークアップを返す。経路に
// ついての検証が、画面の他の場所に出た同じ文字列で満たされてしまうのを防ぐ。
func suggestionFormBreadcrumb(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if start == -1 {
		t.Fatal("レスポンスにパンくずのナビゲーションが含まれていない")
	}
	endOffset := strings.Index(body[start:], "</nav>")
	if endOffset == -1 {
		t.Fatal("パンくずのナビゲーションに閉じタグが無い")
	}

	return body[start : start+endOffset]
}
