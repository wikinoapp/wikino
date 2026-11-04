package draft_page_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/draft_page"
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

// setupHandlerはテスト用のハンドラーを生成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *draft_page.Handler {
	t.Helper()

	db := testutil.GetTestDB()

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)

	attachmentRepo := repository.NewAttachmentRepository(queries)

	flashMgr := session.NewFlashManager("", false, true)

	return draft_page.NewHandler(
		flashMgr,
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
		usecase.NewAutoSaveDraftPageUsecase(
			db,
			spaceRepo,
			spaceMemberRepo,
			draftPageRepo,
			pageRepo,
			repository.NewPageEditorRepository(queries),
			topicRepo,
			topicMemberRepo,
			attachmentRepo,
			validator.NewDraftPageSaveValidator(),
		),
		usecase.NewDeleteDraftPageUsecase(
			db,
			spaceRepo,
			spaceMemberRepo,
			draftPageRepo,
			draftPageRevisionRepo,
			pageRepo,
			topicRepo,
			topicMemberRepo,
		),
		usecase.NewGetEditLinkDataUsecase(pageRepo, topicRepo),
	)
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

	formData := url.Values{
		"pages_edit_form[title]": {"Test"},
		"pages_edit_form[body]":  {"body"},
	}
	req := newRequestWithChiParams(t, "/s/my-space/pages/1/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	}, formData)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnauthorized)
	}

	// エラーメッセージを確認
	body := strings.TrimSpace(rr.Body.String())
	if body != "Unauthorized" {
		t.Errorf("エラーメッセージ = %v、期待値 = Unauthorized", body)
	}
}

func TestUpdate_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("invalidnum@example.com").
		WithAtname("invalidnum").
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[title]": {"Test"},
		"pages_edit_form[body]":  {"body"},
	}
	req := newRequestWithChiParams(t, "/s/my-space/pages/abc/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "abc",
	}, formData)
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
		WithEmail("spacenotfound@example.com").
		WithAtname("spacenotfound").
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[title]":             {"Test"},
		"pages_edit_form[draft_page_digest]": {""},
		"pages_edit_form[body]":              {"body"},
	}
	req := newRequestWithChiParams(t, "/s/nonexistent/pages/1/draft_page", map[string]string{
		"space_identifier": "nonexistent",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_NotSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpowner@example.com").
		WithAtname("dpowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpoutsider@example.com").
		WithAtname("dpoutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-private-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[title]":             {"Test"},
		"pages_edit_form[draft_page_digest]": {""},
		"pages_edit_form[body]":              {"body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-private-space/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-private-space",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: outsiderID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_PageNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dppagemissing@example.com").
		WithAtname("dppagemissing").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-page-missing").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[title]":             {"Test"},
		"pages_edit_form[draft_page_digest]": {""},
		"pages_edit_form[body]":              {"body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-page-missing/pages/999/draft_page", map[string]string{
		"space_identifier": "dp-page-missing",
		"page_number":      "999",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_PermissionDenied(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dppolicy@example.com").
		WithAtname("dppolicy").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-policy-space").
		Build()
	// page:writeを持たない閲覧者を作成
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(model.SpaceRoleViewer).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Restricted Topic").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Restricted Page").
		Build()

	handler := setupHandler(t, queries)

	formData := url.Values{
		"pages_edit_form[title]":             {"Test"},
		"pages_edit_form[draft_page_digest]": {""},
		"pages_edit_form[body]":              {"body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-policy-space/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-policy-space",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestUpdate_ResponseContentType(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	// 未認証リクエストでもContent-Typeがtext/plainであることを確認
	formData := url.Values{}
	req := newRequestWithChiParams(t, "/s/my-space/pages/1/draft_page", map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	}, formData)

	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Content-Type = %v、期待値 = text/plain", contentType)
	}
}

func TestUpdate_BodyTooLong(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dp-body-too-long@example.com").
		WithAtname("dpbodytoolong").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dp-body-too-long").
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

	formData := url.Values{
		"pages_edit_form[title]":             {"Test Page"},
		"pages_edit_form[draft_page_digest]": {""},
		"pages_edit_form[body]":              {strings.Repeat("a", model.PageBodyMaxLength+1)},
	}
	req := newRequestWithChiParams(t, "/s/dp-body-too-long/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-body-too-long",
		"page_number":      "1",
	}, formData)
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

// draftPageDigestFixtureは、版の照合を確かめるための下書きを持つメンバーとスペース
type draftPageDigestFixture struct {
	userID      model.UserID
	spaceID     model.SpaceID
	draftPageID model.DraftPageID
}

// setupDraftPageDigestFixtureは、UseCaseが自前でトランザクションを管理するため、データをテストDBへ直接コミットして作る
func setupDraftPageDigestFixture(t *testing.T, identifier string) draftPageDigestFixture {
	t.Helper()

	db := testutil.GetTestDB()
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

	return draftPageDigestFixture{userID: userID, spaceID: spaceID, draftPageID: draftPageID}
}

// currentDigestは下書きの今の版を返す
func (f draftPageDigestFixture) currentDigest(t *testing.T) string {
	t.Helper()

	draftPage, err := repository.NewDraftPageRepository(query.New(testutil.GetTestDB())).FindByID(t.Context(), f.draftPageID, f.spaceID)
	if err != nil || draftPage == nil {
		t.Fatalf("下書きの取得 = (%v, %v)、期待値 = 下書き", draftPage, err)
	}
	return draftPage.ContentDigest()
}

func TestUpdate_DraftPageDigest(t *testing.T) {
	t.Parallel()

	f := setupDraftPageDigestFixture(t, "dp-digest-match")
	formData := url.Values{
		"pages_edit_form[title]":             {"Draft Title"},
		"pages_edit_form[body]":              {"Saved body"},
		"pages_edit_form[draft_page_digest]": {f.currentDigest(t)},
	}
	req := newRequestWithChiParams(t, "/s/dp-digest-match/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-digest-match",
		"page_number":      "1",
	}, formData)
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: f.userID}))

	rr := httptest.NewRecorder()
	setupHandler(t, query.New(testutil.GetTestDB())).Update(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNoContent)
	}
	if got, want := rr.Header().Get("ETag"), `"`+f.currentDigest(t)+`"`; got != want {
		t.Errorf("ETag = %q、期待値 = 保存後の今の版 %q", got, want)
	}
}

// TestUpdate_DraftPageDigestMissingは、前提の版の分からない保存を、別の場所の変更を上書きしうるため断り、
// 編集画面が保存の失敗として表示できる422と文面を返すことを確かめる
func TestUpdate_DraftPageDigestMissing(t *testing.T) {
	t.Parallel()

	f := setupDraftPageDigestFixture(t, "dp-digest-missing")
	before := f.currentDigest(t)
	formData := url.Values{
		"pages_edit_form[title]": {"Draft Title"},
		"pages_edit_form[body]":  {"Saved body"},
	}
	req := newRequestWithChiParams(t, "/s/dp-digest-missing/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-digest-missing",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: f.userID})
	req = req.WithContext(i18n.SetLocale(ctx, i18n.LangJa))

	rr := httptest.NewRecorder()
	setupHandler(t, query.New(testutil.GetTestDB())).Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}
	want := "この画面は古くなっています。書いた内容を控えてから、ページを再読み込みしてください。"
	if got := strings.TrimSpace(rr.Body.String()); got != want {
		t.Errorf("レスポンスの本文 = %q、期待値 = %q", got, want)
	}
	if got := f.currentDigest(t); got != before {
		t.Errorf("断った保存の後の版 = %q、期待値 = 保存前の版 %q", got, before)
	}
}

func TestUpdate_DraftPageDigestConflict(t *testing.T) {
	t.Parallel()

	f := setupDraftPageDigestFixture(t, "dp-digest-conflict")
	staleDigest := f.currentDigest(t)
	if _, err := testutil.GetTestDB().ExecContext(t.Context(),
		"UPDATE draft_pages SET body = 'API body', modified_at = modified_at + interval '1 second' WHERE id = $1 AND space_id = $2",
		f.draftPageID, f.spaceID); err != nil {
		t.Fatalf("下書きの更新: %v", err)
	}
	currentDigest := f.currentDigest(t)

	formData := url.Values{
		"pages_edit_form[title]":             {"Draft Title"},
		"pages_edit_form[body]":              {"Saved body"},
		"pages_edit_form[draft_page_digest]": {staleDigest},
	}
	req := newRequestWithChiParams(t, "/s/dp-digest-conflict/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-digest-conflict",
		"page_number":      "1",
	}, formData)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: f.userID})
	req = req.WithContext(i18n.SetLocale(ctx, i18n.LangJa))

	rr := httptest.NewRecorder()
	setupHandler(t, query.New(testutil.GetTestDB())).Update(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusConflict)
	}
	if got, want := rr.Header().Get("ETag"), `"`+currentDigest+`"`; got != want {
		t.Errorf("ETag = %q、期待値 = 今の版 %q", got, want)
	}
	if got, want := strings.TrimSpace(rr.Body.String()), "下書きは、編集画面を開いたあとに別の場所で更新されています"; got != want {
		t.Errorf("レスポンスの本文 = %q、期待値 = %q", got, want)
	}
	if got := f.currentDigest(t); got != currentDigest {
		t.Errorf("競合した保存の後の版 = %q、期待値 = 保存前の版 %q", got, currentDigest)
	}
}

// TestUpdate_DeletedDraftPageDigestConflictは、下書きが削除された場合も、
// 下書きが無いことを示す空の版をETagで返し、古い内容で作り直さないことを確かめる
func TestUpdate_DeletedDraftPageDigestConflict(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	f := setupDraftPageDigestFixture(t, "dp-digest-deleted")
	staleDigest := f.currentDigest(t)
	draftPageRepo := repository.NewDraftPageRepository(query.New(db))
	if err := draftPageRepo.Delete(t.Context(), f.draftPageID, f.spaceID); err != nil {
		t.Fatalf("下書きの削除: %v", err)
	}

	formData := url.Values{
		"pages_edit_form[title]":             {"Draft Title"},
		"pages_edit_form[body]":              {"Saved body"},
		"pages_edit_form[draft_page_digest]": {staleDigest},
	}
	req := newRequestWithChiParams(t, "/s/dp-digest-deleted/pages/1/draft_page", map[string]string{
		"space_identifier": "dp-digest-deleted",
		"page_number":      "1",
	}, formData)
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: f.userID}))

	rr := httptest.NewRecorder()
	setupHandler(t, query.New(db)).Update(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusConflict)
	}
	if got := rr.Header().Get("ETag"); got != `""` {
		t.Errorf("ETag = %q、期待値 = 引用符で囲んだ空の版", got)
	}
	var count int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM draft_pages WHERE space_id = $1", f.spaceID).Scan(&count); err != nil {
		t.Fatalf("下書きの件数取得: %v", err)
	}
	if count != 0 {
		t.Errorf("削除後の下書きの数 = %d、期待値 = 0", count)
	}
}
