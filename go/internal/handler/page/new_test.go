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

// newPageFixtureはページ新規作成の入口のテストで共有するフィクスチャ一式。
type newPageFixture struct {
	userID          model.UserID
	spaceID         model.SpaceID
	spaceIdentifier string
	spaceMemberID   model.SpaceMemberID
}

// setupNewPageFixtureはスペース・メンバー・トピック・トピックメンバーをテストDBへ直接
// コミットして作成する (ハンドラーの背後のUseCaseが自前でトランザクションを管理するため)。
// prefixとatnameはどちらもテストDBを共有する並行テスト間で識別子を一意に保つ。atnameを別に
// 受け取るのは、prefixでは超えてしまうvalidator.AtnameMaxLengthにも収める必要があるため。
// スコープの引数がnilの場合は各ビルダーの既定値を使う。
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

// requestNewPageは指定トピックの入口へ、指定のクエリ文字列でリクエストを送り、記録した
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

// findCreatedPageはフィクスチャのスペースで指定番号に作成されたページを返す。
func findCreatedPage(t *testing.T, db *sql.DB, f newPageFixture, number model.PageNumber) *model.Page {
	t.Helper()

	page, err := repository.NewPageRepository(query.New(db)).
		FindBySpaceAndNumber(context.Background(), f.spaceID, number)
	if err != nil {
		t.Fatalf("FindBySpaceAndNumber()のエラー = %v", err)
	}

	return page
}

// findCreatedDraftPageはフィクスチャのメンバーが作成したページの下書きを返す。下書きが
// 存在しない場合はnilを返す。
func findCreatedDraftPage(t *testing.T, db *sql.DB, f newPageFixture, pageID model.PageID) *model.DraftPage {
	t.Helper()

	draftPage, err := repository.NewDraftPageRepository(query.New(db)).
		FindByPageAndMember(context.Background(), pageID, f.spaceMemberID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByPageAndMember()のエラー = %v", err)
	}

	return draftPage
}

// assertRedirectedToEditPageは、作成されたページの編集画面への一時的なリダイレクトを検証する。
func assertRedirectedToEditPage(t *testing.T, rr *httptest.ResponseRecorder, spaceIdentifier string, pageNumber model.PageNumber) {
	t.Helper()

	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	want := fmt.Sprintf("/s/%s/pages/%d/edit", spaceIdentifier, pageNumber)
	if got := rr.Header().Get("Location"); got != want {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", got, want)
	}
}

// TestNew_NotLoggedInは、認証済みユーザーなしでハンドラーへ到達した場合のフォールバックを
// 固定する。RequireAuthミドルウェアのback URLの挙動は別のテストで検証する。
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("リダイレクト先 = %v、期待値 = /sign_in", got)
	}
}

// TestNew_WithoutPrefilledContentは入口の素の形を固定する。空ページだけを作って下書きは
// 残さないため、エディタは空ページそのものを開く。
func TestNew_WithoutPrefilledContent(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	f := setupNewPageFixture(t, db, "handler-new-blank", "hnblank", nil, nil)

	rr := requestNewPage(t, db, f, f.spaceIdentifier, "1", "")

	assertRedirectedToEditPage(t, rr, f.spaceIdentifier, 1)

	createdPage := findCreatedPage(t, db, f, 1)
	if createdPage == nil {
		t.Fatal("ページが作成されていない")
	}
	if createdPage.Title != nil {
		t.Errorf("Page.Title = %v、期待値 = nil", createdPage.Title)
	}

	if draftPage := findCreatedDraftPage(t, db, f, createdPage.ID); draftPage != nil {
		t.Errorf("下書きが存在している: %+v", draftPage)
	}
}

// TestNew_WithPrefilledContentは、パーセントエンコードされたクエリが改行やマルチバイト文字を
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
		t.Fatal("ページが作成されていない")
	}

	// 事前入力された値が入るのは下書きだけで、ページ本体は編集画面から公開されるまで空のまま。
	if createdPage.Title != nil {
		t.Errorf("Page.Title = %v、期待値 = nil", createdPage.Title)
	}
	if createdPage.Body != "" {
		t.Errorf("Page.Body = %q、期待値 = 空文字列", createdPage.Body)
	}

	draftPage := findCreatedDraftPage(t, db, f, createdPage.ID)
	if draftPage == nil {
		t.Fatal("下書きが存在しない")
	}
	if draftPage.Title == nil || *draftPage.Title != title {
		t.Errorf("DraftPage.Title = %v、期待値 = %q", draftPage.Title, title)
	}
	if draftPage.Body != body {
		t.Errorf("DraftPage.Body = %q、期待値 = %q", draftPage.Body, body)
	}
	if !strings.Contains(draftPage.BodyHTML, "引用された文章") {
		t.Errorf("DraftPage.BodyHTML = %q、期待値 = 引用したテキストを含む", draftPage.BodyHTML)
	}
}

// TestNew_NotFoundは入口が404で答える条件を固定する。解釈できないトピック番号と、閲覧者が
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
				t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
			}
		})
	}
}

// TestNew_WithoutPageWriteScopeは、ページを読むことしかできないメンバーが404を受け取り
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}

	if page := findCreatedPage(t, db, f, 1); page != nil {
		t.Errorf("ページが作成されている: %+v", page)
	}
}
