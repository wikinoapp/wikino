package page_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newPageFixture is the fixture set shared by the page creation entry point tests.
//
// [Ja] newPageFixture はページ新規作成の入口のテストで共有するフィクスチャ一式。
type newPageFixture struct {
	userID          model.UserID
	spaceID         model.SpaceID
	spaceIdentifier string
	spaceMemberID   model.SpaceMemberID
}

// setupNewPageFixture creates a space, member, topic and topic member committed directly to the
// test DB, because the UseCase behind the handler manages its own transaction. prefix and atname
// both keep identifiers unique across parallel tests sharing the test DB; atname is passed
// separately because it must also stay within validator.AtnameMaxLength, which the prefixes
// exceed. Nil scope arguments use each builder's defaults.
//
// [Ja] setupNewPageFixture はスペース・メンバー・トピック・トピックメンバーをテスト DB へ直接
// コミットして作成する (ハンドラーの背後の UseCase が自前でトランザクションを管理するため)。
// prefix と atname はどちらもテスト DB を共有する並行テスト間で識別子を一意に保つ。atname を別に
// 受け取るのは、prefix では超えてしまう validator.AtnameMaxLength にも収める必要があるため。
// スコープの引数が nil の場合は各ビルダーの既定値を使う。
func setupNewPageFixture(
	t *testing.T,
	db *sql.DB,
	prefix string,
	atname string,
	spaceMemberScopes []model.Scope,
	topicMemberScopes []model.Scope,
) newPageFixture {
	t.Helper()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail(prefix + "@example.com").
		WithAtname(atname).
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(prefix + "-space").
		Build()

	spaceMemberBuilder := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if spaceMemberScopes != nil {
		spaceMemberBuilder = spaceMemberBuilder.WithScopes(spaceMemberScopes)
	}
	spaceMemberID := spaceMemberBuilder.Build()

	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	topicMemberBuilder := testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID)
	if topicMemberScopes != nil {
		topicMemberBuilder = topicMemberBuilder.WithScopes(topicMemberScopes)
	}
	topicMemberBuilder.Build()

	return newPageFixture{
		userID:          userID,
		spaceID:         spaceID,
		spaceIdentifier: prefix + "-space",
		spaceMemberID:   spaceMemberID,
	}
}

// requestNewPage sends a request to the entry point of the given topic with the given query
// string, and returns the recorded response.
//
// [Ja] requestNewPage は指定トピックの入口へ、指定のクエリ文字列でリクエストを送り、記録した
// レスポンスを返す。
func requestNewPage(t *testing.T, db *sql.DB, f newPageFixture, spaceIdentifier string, topicNumber string, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()

	path := fmt.Sprintf("/s/%s/topics/%s/pages/new", spaceIdentifier, topicNumber)
	if rawQuery != "" {
		path += "?" + rawQuery
	}

	req := newRequestWithChiParams(t, http.MethodGet, path, map[string]string{
		"space_identifier": spaceIdentifier,
		"topic_number":     topicNumber,
	})

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: f.userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	setupHandlerWithDB(t, db, query.New(db)).New(rr, req)

	return rr
}

// findCreatedPage returns the page created at the given number in the fixture's space.
//
// [Ja] findCreatedPage はフィクスチャのスペースで指定番号に作成されたページを返す。
func findCreatedPage(t *testing.T, db *sql.DB, f newPageFixture, number model.PageNumber) *model.Page {
	t.Helper()

	page, err := repository.NewPageRepository(query.New(db)).
		FindBySpaceAndNumber(context.Background(), f.spaceID, number)
	if err != nil {
		t.Fatalf("FindBySpaceAndNumber() error = %v", err)
	}

	return page
}

// findCreatedDraftPage returns the fixture member's draft for a page, or nil when none exists.
//
// [Ja] findCreatedDraftPage はフィクスチャのメンバーが作成したページの下書きを返す。下書きが
// 存在しない場合は nil を返す。
func findCreatedDraftPage(t *testing.T, db *sql.DB, f newPageFixture, pageID model.PageID) *model.DraftPage {
	t.Helper()

	draftPage, err := repository.NewDraftPageRepository(query.New(db)).
		FindByPageAndMember(context.Background(), pageID, f.spaceMemberID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByPageAndMember() error = %v", err)
	}

	return draftPage
}

// assertRedirectedToEditPage checks the temporary redirect to the edit screen of the created page.
//
// [Ja] assertRedirectedToEditPage は、作成されたページの編集画面への一時的なリダイレクトを検証する。
func assertRedirectedToEditPage(t *testing.T, rr *httptest.ResponseRecorder, spaceIdentifier string, pageNumber model.PageNumber) {
	t.Helper()

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	want := fmt.Sprintf("/s/%s/pages/%d/edit", spaceIdentifier, pageNumber)
	if got := rr.Header().Get("Location"); got != want {
		t.Errorf("wrong redirect location: got %v want %v", got, want)
	}
}

// TestNew_NotLoggedIn pins the handler-level fallback for requests that reach it without an
// authenticated user. The RequireAuth middleware's back-URL behavior is tested separately.
//
// [Ja] TestNew_NotLoggedIn は、認証済みユーザーなしでハンドラーへ到達した場合のフォールバックを
// 固定する。RequireAuth ミドルウェアの back URL の挙動は別のテストで検証する。
func TestNew_NotLoggedIn(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	req := newRequestWithChiParams(t, http.MethodGet, "/s/my-space/topics/1/pages/new", map[string]string{
		"space_identifier": "my-space",
		"topic_number":     "1",
	})
	req = req.WithContext(i18n.SetLocale(req.Context(), i18n.LangJa))

	rr := httptest.NewRecorder()
	setupHandlerWithDB(t, db, query.New(db)).New(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", got)
	}
}

// TestNew_WithoutPrefilledContent pins the entry point's plain form: it creates an empty page and
// leaves no draft behind, so the editor opens on the blank page itself.
//
// [Ja] TestNew_WithoutPrefilledContent は入口の素の形を固定する。空ページだけを作って下書きは
// 残さないため、エディタは空ページそのものを開く。
func TestNew_WithoutPrefilledContent(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	f := setupNewPageFixture(t, db, "handler-new-blank", "hnblank", nil, nil)

	rr := requestNewPage(t, db, f, f.spaceIdentifier, "1", "")

	assertRedirectedToEditPage(t, rr, f.spaceIdentifier, 1)

	createdPage := findCreatedPage(t, db, f, 1)
	if createdPage == nil {
		t.Fatal("page should have been created")
	}
	if createdPage.Title != nil {
		t.Errorf("Page.Title = %v, want nil", createdPage.Title)
	}

	if draftPage := findCreatedDraftPage(t, db, f, createdPage.ID); draftPage != nil {
		t.Errorf("draft page should not exist, got %+v", draftPage)
	}
}

// TestNew_WithPrefilledContent checks that the percent-encoded query reaches the draft as typed,
// newlines and multibyte characters included, which is what a bookmarklet sends.
//
// [Ja] TestNew_WithPrefilledContent は、パーセントエンコードされたクエリが改行やマルチバイト文字を
// 含めてそのまま下書きに届くことを検証する。ブックマークレットが送るのはこの形である。
func TestNew_WithPrefilledContent(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	f := setupNewPageFixture(t, db, "handler-new-prefill", "hnprefill", nil, nil)

	title := "記事タイトル - example.com"
	body := "https://example.com/article\n\n> 引用された文章"

	queryParams := url.Values{}
	queryParams.Set("title", title)
	queryParams.Set("body", body)

	rr := requestNewPage(t, db, f, f.spaceIdentifier, "1", queryParams.Encode())

	assertRedirectedToEditPage(t, rr, f.spaceIdentifier, 1)

	createdPage := findCreatedPage(t, db, f, 1)
	if createdPage == nil {
		t.Fatal("page should have been created")
	}

	// The prefilled values live in the draft alone; the page itself stays empty until it is
	// published from the edit screen.
	//
	// [Ja] 事前入力された値が入るのは下書きだけで、ページ本体は編集画面から公開されるまで空のまま。
	if createdPage.Title != nil {
		t.Errorf("Page.Title = %v, want nil", createdPage.Title)
	}
	if createdPage.Body != "" {
		t.Errorf("Page.Body = %q, want empty string", createdPage.Body)
	}

	draftPage := findCreatedDraftPage(t, db, f, createdPage.ID)
	if draftPage == nil {
		t.Fatal("draft page should exist")
	}
	if draftPage.Title == nil || *draftPage.Title != title {
		t.Errorf("DraftPage.Title = %v, want %q", draftPage.Title, title)
	}
	if draftPage.Body != body {
		t.Errorf("DraftPage.Body = %q, want %q", draftPage.Body, body)
	}
	if !strings.Contains(draftPage.BodyHTML, "引用された文章") {
		t.Errorf("DraftPage.BodyHTML = %q, want it to contain the quoted text", draftPage.BodyHTML)
	}
}

// TestNew_NotFound pins the conditions the entry point answers with a 404: an unparsable topic
// number, and a space or topic the visitor cannot reach.
//
// [Ja] TestNew_NotFound は入口が 404 で答える条件を固定する。解釈できないトピック番号と、閲覧者が
// 到達できないスペース・トピックである。
func TestNew_NotFound(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	f := setupNewPageFixture(t, db, "handler-new-notfound", "hnnf", nil, nil)

	testCases := []struct {
		name            string
		spaceIdentifier string
		topicNumber     string
	}{
		{
			name:            "トピック番号が数字でない",
			spaceIdentifier: f.spaceIdentifier,
			topicNumber:     "abc",
		},
		{
			name:            "存在しないスペース",
			spaceIdentifier: "handler-new-notfound-missing-space",
			topicNumber:     "1",
		},
		{
			name:            "存在しないトピック",
			spaceIdentifier: f.spaceIdentifier,
			topicNumber:     "999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rr := requestNewPage(t, db, f, tc.spaceIdentifier, tc.topicNumber, "")

			if rr.Code != http.StatusNotFound {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
			}
		})
	}
}

// TestNew_WithoutPageWriteScope checks that a member who may only read pages gets a 404 and leaves
// no page behind, so the entry point cannot be used to write into a space one only reads.
//
// [Ja] TestNew_WithoutPageWriteScope は、ページを読むことしかできないメンバーが 404 を受け取り
// ページも残らないことを検証する。読むだけのスペースへ入口から書き込めてはならないため。
func TestNew_WithoutPageWriteScope(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	f := setupNewPageFixture(
		t,
		db,
		"handler-new-readonly",
		"hnread",
		[]model.Scope{model.ScopePageRead},
		[]model.Scope{model.ScopePageRead},
	)

	rr := requestNewPage(t, db, f, f.spaceIdentifier, "1", "title=%E3%83%86%E3%82%B9%E3%83%88")

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}

	if page := findCreatedPage(t, db, f, 1); page != nil {
		t.Errorf("page should not have been created, got %+v", page)
	}
}
