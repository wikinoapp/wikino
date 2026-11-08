package draft_page_revision_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/draft_page_revision"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// newTestHandlerは依存するユースケース一式をqueriesに結線したハンドラーを生成します。
func newTestHandler(db *sql.DB, queries *query.Queries) *draft_page_revision.Handler {
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	attachmentRepo := repository.NewAttachmentRepository(queries)

	flashMgr := session.NewFlashManager("", false, false)

	return draft_page_revision.NewHandler(
		flashMgr,
		usecase.NewManualSaveDraftPageUsecase(
			db,
			spaceRepo,
			spaceMemberRepo,
			draftPageRepo,
			draftPageRevisionRepo,
			pageRepo,
			repository.NewPageEditorRepository(queries),
			topicRepo,
			topicMemberRepo,
			attachmentRepo,
			validator.NewDraftPageSaveValidator(),
		),
		usecase.NewGetDraftPageRevisionDiffUsecase(
			spaceRepo,
			spaceMemberRepo,
			pageRepo,
			topicRepo,
			topicMemberRepo,
			draftPageRepo,
			draftPageRevisionRepo,
		),
		usecase.NewGetPageDetailUsecase(
			spaceRepo,
			spaceMemberRepo,
			pageRepo,
			draftPageRepo,
			draftPageRevisionRepo,
			topicRepo,
			topicMemberRepo,
			repository.NewSuggestionPageRepository(queries),
			repository.NewSuggestionRepository(queries),
		),
	)
}

// setupHandlerはテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *draft_page_revision.Handler {
	t.Helper()

	return newTestHandler(testutil.GetTestDB(), queries)
}

// newRequestWithChiParamsはchiのURLパラメータ付きPATCHリクエストを作成するヘルパーです
func newRequestWithChiParams(t *testing.T, path string, params map[string]string, formData url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestUpdate_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, "/s/my-space/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	}, url.Values{})

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestUpdate_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-invalidnum@example.com").
		WithAtname("dprinvalidnum").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, "/s/my-space/pages/abc/draft_page_revision", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "abc",
	}, url.Values{})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_SpaceNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-spacenotfound@example.com").
		WithAtname("dprspacenotfound").
		Build()

	handler := setupHandler(t, queries)

	req := newRequestWithChiParams(t, "/s/nonexistent/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	}, url.Values{"draft_page_digest": {""}})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	// UseCaseが独自トランザクションを管理するため、DB直接書き込みを使用
	db := testutil.GetTestDB()
	q := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("dpr-update-success@example.com").
		WithAtname("dprupdatesuccess").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("dpr-update-success-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		Build()

	handler := newTestHandler(db, q)

	formData := url.Values{}
	formData.Set("draft_page_digest", draftPageDigest(t, db, draftPageID, spaceID))
	formData.Set("title", "Draft Title")
	formData.Set("body", "Draft body")
	req := newRequestWithChiParams(t, "/s/dpr-update-success-space/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-update-success-space",
		"page_number":      "1",
	}, formData)

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusSeeOther)
	}

	location := rr.Header().Get("Location")
	if location != "/drafts" {
		t.Errorf("リダイレクト先 = %v、期待値 = /drafts", location)
	}
}

func TestUpdate_WithoutDraftPage(t *testing.T) {
	t.Parallel()

	// DraftPageが存在しない場合でも、find_or_createにより成功する
	db := testutil.GetTestDB()
	q := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("dpr-nodraft@example.com").
		WithAtname("dprnodraft").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("dpr-nodraft-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	handler := newTestHandler(db, q)

	formData := url.Values{}
	formData.Set("draft_page_digest", "")
	formData.Set("title", "新規下書き")
	formData.Set("body", "新規下書き本文")
	req := newRequestWithChiParams(t, "/s/dpr-nodraft-space/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-nodraft-space",
		"page_number":      "1",
	}, formData)

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusSeeOther)
	}

	location := rr.Header().Get("Location")
	if location != "/drafts" {
		t.Errorf("リダイレクト先 = %v、期待値 = /drafts", location)
	}
}

func TestUpdate_RedirectToSuggestionNew(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("dpr-suggest-redir@example.com").
		WithAtname("dprsuggestredir").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("dpr-suggest-redir-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		Build()

	handler := newTestHandler(db, q)

	formData := url.Values{}
	formData.Set("draft_page_digest", draftPageDigest(t, db, draftPageID, spaceID))
	formData.Set("title", "Draft Title")
	formData.Set("body", "Draft body")
	req := newRequestWithChiParams(t, "/s/dpr-suggest-redir-sp/pages/1/draft_page_revision?redirect_to=suggestion_new", map[string]string{
		"space_identifier": "dpr-suggest-redir-sp",
		"page_number":      "1",
	}, formData)

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusSeeOther)
	}

	location := rr.Header().Get("Location")
	expectedLocation := fmt.Sprintf("/s/dpr-suggest-redir-sp/topics/1/suggestions/new?draft_page_ids=%s", string(draftPageID))
	if location != expectedLocation {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, expectedLocation)
	}
}

// TestUpdate_HTMXRedirectToSuggestionNewは、htmxで送った「下書き保存して編集提案を作成」が
// 保存に成功したときに、編集提案の作成画面への移動をHX-Redirectで伝えることを検証する。
func TestUpdate_HTMXRedirectToSuggestionNew(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("dpr-hx-suggest-redir@example.com").
		WithAtname("dprhxsuggestredir").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("dpr-hx-suggest-redir-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		Build()

	handler := newTestHandler(db, q)

	formData := url.Values{}
	formData.Set("draft_page_digest", draftPageDigest(t, db, draftPageID, spaceID))
	formData.Set("title", "Draft Title")
	formData.Set("body", "Draft body")
	req := newRequestWithChiParams(t, "/s/dpr-hx-suggest-redir-sp/pages/1/draft_page_revision?redirect_to=suggestion_new", map[string]string{
		"space_identifier": "dpr-hx-suggest-redir-sp",
		"page_number":      "1",
	}, formData)

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	// fetchはリダイレクトを追ってしまうため、移動先はHX-Redirectで伝える
	if rr.Code != http.StatusNoContent {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNoContent)
	}

	location := rr.Header().Get("HX-Redirect")
	expectedLocation := fmt.Sprintf("/s/dpr-hx-suggest-redir-sp/topics/1/suggestions/new?draft_page_ids=%s", string(draftPageID))
	if location != expectedLocation {
		t.Errorf("HX-Redirect = %v、期待値 = %v", location, expectedLocation)
	}
}

// TestUpdate_HTMXRequestはhtmxからの手動保存が画面遷移なしで応答することを検証する。
// リダイレクトの代わりに、保存時刻表示と編集履歴カラム2箇所を更新するOOBスワップ
// フラグメントが返る。
func TestUpdate_HTMXRequest(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("dpr-htmx@example.com").
		WithAtname("dprhtmx").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("dpr-htmx-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		Build()

	handler := newTestHandler(db, q)

	formData := url.Values{}
	formData.Set("draft_page_digest", draftPageDigest(t, db, draftPageID, spaceID))
	formData.Set("title", "Saved Title")
	formData.Set("body", "Saved body")
	req := newRequestWithChiParams(t, "/s/dpr-htmx-space/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-htmx-space",
		"page_number":      "1",
	}, formData)
	req.Header.Set("HX-Request", "true")

	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}
	if location := rr.Header().Get("Location"); location != "" {
		t.Errorf("htmxのリクエストがリダイレクトされている: Location = %v", location)
	}

	body := rr.Body.String()
	expectedChecks := []string{
		`id="page-draft-saved-at" hx-swap-oob="outerHTML"`,
		`id="page-revision-list" hx-swap-oob="innerHTML"`,
		`id="page-revision-list-drawer" hx-swap-oob="innerHTML"`,
		// この保存で下書きの最初のリビジョンが作成されるため、更新後の履歴にv1が表示される。
		"v1",
	}
	for _, expected := range expectedChecks {
		if !strings.Contains(body, expected) {
			t.Errorf("レスポンスに%qが含まれていない", expected)
		}
	}
}

func TestUpdate_BodyTooLong(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-body-too-long@example.com").
		WithAtname("dprbodytoolong").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dpr-body-too-long").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
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
		WithTitle("Test Page").
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{}
	formData.Set("draft_page_digest", "")
	formData.Set("title", "Test Page")
	formData.Set("body", strings.Repeat("a", model.PageBodyMaxLength+1))
	req := newRequestWithChiParams(t, "/s/dpr-body-too-long/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-body-too-long",
		"page_number":      "1",
	}, formData)
	req.Header.Set("HX-Request", "true")
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
	want := "本文は500,000文字以内で入力してください (現在500,001文字)"
	if got := strings.TrimSpace(rr.Body.String()); got != want {
		t.Errorf("レスポンスの本文 = %q、期待値 = %q", got, want)
	}
}

// setupDraftPageDigestFixtureは、版の照合を確かめるための下書きを持つメンバーを作り、ユーザーと下書きのIDを返す。
// UseCaseが自前でトランザクションを管理するため、データをテストDBへ直接コミットして作る
func setupDraftPageDigestFixture(t *testing.T, db *sql.DB, identifier string) (model.UserID, model.SpaceID, model.DraftPageID) {
	t.Helper()

	atname := strings.ReplaceAll(identifier, "-", "")
	userID := testutil.NewUserBuilderDB(t, db).WithEmail(identifier + "@example.com").WithAtname(atname).Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("General").Build()
	testutil.NewTopicMemberBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithSpaceMemberID(spaceMemberID).Build()
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("Test Page").Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).WithPageID(pageID).WithSpaceMemberID(spaceMemberID).WithTopicID(topicID).
		WithTitle("Draft Title").WithBody("Draft body").Build()
	return userID, spaceID, draftPageID
}

// draftPageDigestは下書きの今の版を返す
func draftPageDigest(t *testing.T, db *sql.DB, id model.DraftPageID, spaceID model.SpaceID) string {
	t.Helper()

	draftPage, err := repository.NewDraftPageRepository(query.New(db)).FindByID(t.Context(), id, spaceID)
	if err != nil || draftPage == nil {
		t.Fatalf("下書きの取得 = (%v, %v)、期待値 = 下書き", draftPage, err)
	}
	return draftPage.ContentDigest()
}

func TestUpdate_DraftPageDigestMatch(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID, spaceID, draftPageID := setupDraftPageDigestFixture(t, db, "dpr-digest-match")

	formData := url.Values{}
	formData.Set("title", "Draft Title")
	formData.Set("body", "Saved body")
	formData.Set("draft_page_digest", draftPageDigest(t, db, draftPageID, spaceID))
	req := newRequestWithChiParams(t, "/s/dpr-digest-match/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-digest-match",
		"page_number":      "1",
	}, formData)
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: userID}))

	rr := httptest.NewRecorder()
	newTestHandler(db, query.New(db)).Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}
	if got, want := rr.Header().Get("ETag"), `"`+draftPageDigest(t, db, draftPageID, spaceID)+`"`; got != want {
		t.Errorf("ETag = %q、期待値 = 保存後の今の版 %q", got, want)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-transform" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-transform")
	}
}

func TestUpdate_DraftPageDigestConflict(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID, spaceID, draftPageID := setupDraftPageDigestFixture(t, db, "dpr-digest-conflict")
	staleDigest := draftPageDigest(t, db, draftPageID, spaceID)
	if _, err := db.ExecContext(t.Context(),
		"UPDATE draft_pages SET body = 'API body', modified_at = modified_at + interval '1 second' WHERE id = $1 AND space_id = $2",
		draftPageID, spaceID); err != nil {
		t.Fatalf("下書きの更新: %v", err)
	}
	currentDigest := draftPageDigest(t, db, draftPageID, spaceID)

	formData := url.Values{}
	formData.Set("title", "Draft Title")
	formData.Set("body", "Saved body")
	formData.Set("draft_page_digest", staleDigest)
	req := newRequestWithChiParams(t, "/s/dpr-digest-conflict/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-digest-conflict",
		"page_number":      "1",
	}, formData)
	req.Header.Set("HX-Request", "true")
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(i18n.SetLocale(ctx, i18n.LangJa))

	rr := httptest.NewRecorder()
	newTestHandler(db, query.New(db)).Update(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusConflict)
	}
	if got, want := rr.Header().Get("ETag"), `"`+currentDigest+`"`; got != want {
		t.Errorf("ETag = %q、期待値 = 今の版 %q", got, want)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-transform" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-transform")
	}
	if got, want := strings.TrimSpace(rr.Body.String()), "下書きは、編集画面を開いたあとに別の場所で更新されています"; got != want {
		t.Errorf("レスポンスの本文 = %q、期待値 = %q", got, want)
	}
	if got := draftPageDigest(t, db, draftPageID, spaceID); got != currentDigest {
		t.Errorf("競合した保存の後の版 = %q、期待値 = 保存前の版 %q", got, currentDigest)
	}
}

// TestUpdate_DeletedDraftPageDigestConflictは、下書きが削除された場合も、
// 下書きが無いことを示す空の版をETagで返し、古い内容で作り直さないことを確かめる
func TestUpdate_DeletedDraftPageDigestConflict(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID, spaceID, draftPageID := setupDraftPageDigestFixture(t, db, "dpr-digest-deleted")
	staleDigest := draftPageDigest(t, db, draftPageID, spaceID)
	draftPageRepo := repository.NewDraftPageRepository(query.New(db))
	if err := draftPageRepo.Delete(t.Context(), draftPageID, spaceID); err != nil {
		t.Fatalf("下書きの削除: %v", err)
	}

	formData := url.Values{
		"title":             {"Draft Title"},
		"body":              {"Saved body"},
		"draft_page_digest": {staleDigest},
	}
	req := newRequestWithChiParams(t, "/s/dpr-digest-deleted/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-digest-deleted",
		"page_number":      "1",
	}, formData)
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: userID}))

	rr := httptest.NewRecorder()
	newTestHandler(db, query.New(db)).Update(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusConflict)
	}
	if got := rr.Header().Get("ETag"); got != `""` {
		t.Errorf("ETag = %q、期待値 = 引用符で囲んだ空の版", got)
	}
	var count int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM draft_pages WHERE space_id = $1", spaceID).Scan(&count); err != nil {
		t.Fatalf("下書きの件数取得: %v", err)
	}
	if count != 0 {
		t.Errorf("削除後の下書きの数 = %d、期待値 = 0", count)
	}
}

// TestUpdate_DraftPageDigestMissingは、前提の版の分からない手動保存を、別の場所の変更を上書きしうるため断り、
// 編集画面が保存の失敗として表示できる422と文面を返すことを確かめる
func TestUpdate_DraftPageDigestMissing(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID, spaceID, draftPageID := setupDraftPageDigestFixture(t, db, "dpr-digest-missing")
	before := draftPageDigest(t, db, draftPageID, spaceID)

	formData := url.Values{}
	formData.Set("title", "Draft Title")
	formData.Set("body", "Saved body")
	req := newRequestWithChiParams(t, "/s/dpr-digest-missing/pages/1/draft_page_revision", map[string]string{
		"space_identifier": "dpr-digest-missing",
		"page_number":      "1",
	}, formData)
	req.Header.Set("HX-Request", "true")
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(i18n.SetLocale(ctx, i18n.LangJa))

	rr := httptest.NewRecorder()
	newTestHandler(db, query.New(db)).Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
	want := "この画面は古くなっています。書いた内容を控えてから、ページを再読み込みしてください。"
	if got := strings.TrimSpace(rr.Body.String()); got != want {
		t.Errorf("レスポンスの本文 = %q、期待値 = %q", got, want)
	}
	if got := draftPageDigest(t, db, draftPageID, spaceID); got != before {
		t.Errorf("断った保存の後の版 = %q、期待値 = 保存前の版 %q", got, before)
	}
}
