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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if loc := rr.Header().Get("Location"); loc != "/sign_in" {
		t.Errorf("wrong redirect location: got %q want %q", loc, "/sign_in")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "下書きタイトル") {
		t.Error("response should contain draft page title")
	}
	if !strings.Contains(body, "csrf_token") {
		t.Error("response should contain CSRF token")
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
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
}

// The trail ends with the screen itself, so the last item must be a plain label carrying
// aria-current rather than a link back to the topic. Scope the assertions to the breadcrumb because
// the same label also appears in the heading and the page title.
//
// [Ja] 経路はこの画面自身で終わるため、末尾の項目はトピックへのリンクではなく aria-current を持つ
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
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	breadcrumb := suggestionFormBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/s/new-crumb-sp"`,
		`href="/s/new-crumb-sp/topics/1"`,
		`aria-current="page"`,
		"新規編集提案",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("breadcrumb does not contain %q", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/new-crumb-sp/topics/1/suggestions/new"`) {
		t.Error("current suggestion creation breadcrumb item must not be a link")
	}
}

// suggestionFormBreadcrumb returns the markup of the breadcrumb navigation alone, so that an
// assertion about the trail is not satisfied by the same text appearing elsewhere on the screen.
//
// [Ja] suggestionFormBreadcrumb はパンくずのナビゲーション部分だけのマークアップを返す。経路に
// ついての検証が、画面の他の場所に出た同じ文字列で満たされてしまうのを防ぐ。
func suggestionFormBreadcrumb(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if start == -1 {
		t.Fatal("response does not contain the breadcrumb navigation")
	}
	endOffset := strings.Index(body[start:], "</nav>")
	if endOffset == -1 {
		t.Fatal("breadcrumb navigation does not have a closing tag")
	}

	return body[start : start+endOffset]
}
