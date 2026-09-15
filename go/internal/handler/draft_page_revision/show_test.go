package draft_page_revision_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newGetRequestWithChiParamsはchiのURLパラメータ付きGETリクエストを作成するヘルパー。
func newGetRequestWithChiParams(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// showFixtureはShowハンドラーのテストで共有するフィクスチャ一式。
type showFixture struct {
	userID        model.UserID
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
	topicID       model.TopicID
	pageID        model.PageID
	revision1     *model.DraftPageRevision
	revision2     *model.DraftPageRevision
}

// setupShowFixtureはスペース・メンバー・トピック・ページ・下書きとリビジョン2件
// (v1 → v2) を作成する。prefixはテストDBを共有する並行テスト間で識別子を一意に保つ。
func setupShowFixture(t *testing.T, tx *sql.Tx, queries *query.Queries, prefix string) showFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(prefix + "@example.com").
		WithAtname(strings.ReplaceAll(prefix, "-", "")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(prefix + "-space").
		WithName("Show Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(0). // 公開
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Show Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	revisionRepo := repository.NewDraftPageRevisionRepository(queries)
	revision1, err := revisionRepo.Create(context.Background(), repository.CreateDraftPageRevisionInput{
		DraftPageID:   draftPageID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         "Old Title",
		Body:          "line one\n",
		BodyHTML:      "<p>line one</p>",
	})
	if err != nil {
		t.Fatalf("Create() (revision1) のエラー = %v", err)
	}
	revision2, err := revisionRepo.Create(context.Background(), repository.CreateDraftPageRevisionInput{
		DraftPageID:   draftPageID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         "New Title",
		Body:          "line one\nline two\n",
		BodyHTML:      "<p>line one</p><p>line two</p>",
	})
	if err != nil {
		t.Fatalf("Create() (revision2) のエラー = %v", err)
	}

	return showFixture{
		userID:        userID,
		spaceID:       spaceID,
		spaceMemberID: spaceMemberID,
		topicID:       topicID,
		pageID:        pageID,
		revision1:     revision1,
		revision2:     revision2,
	}
}

// newShowRequestはフィクスチャとリビジョンIDに対するShowリクエストを作成する。
func newShowRequest(t *testing.T, spaceIdentifier string, revisionID string, userID model.UserID) *http.Request {
	t.Helper()

	req := newGetRequestWithChiParams(t, "/s/"+spaceIdentifier+"/pages/1/draft_page_revisions/"+revisionID, map[string]string{
		"space_identifier":       spaceIdentifier,
		"page_number":            "1",
		"draft_page_revision_id": revisionID,
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	return req.WithContext(ctx)
}

func TestShow_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newGetRequestWithChiParams(t, "/s/my-space/pages/1/draft_page_revisions/00000000-0000-0000-0000-000000000000", map[string]string{
		"space_identifier":       "my-space",
		"page_number":            "1",
		"draft_page_revision_id": "00000000-0000-0000-0000-000000000000",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestShow_InvalidPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-show-invalidnum@example.com").
		WithAtname("dprshowinvalidnum").
		Build()

	handler := setupHandler(t, queries)

	req := newGetRequestWithChiParams(t, "/s/my-space/pages/abc/draft_page_revisions/00000000-0000-0000-0000-000000000000", map[string]string{
		"space_identifier":       "my-space",
		"page_number":            "abc",
		"draft_page_revision_id": "00000000-0000-0000-0000-000000000000",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	fixture := setupShowFixture(t, tx, queries, "dpr-show-success")
	handler := setupHandler(t, queries)

	req := newShowRequest(t, "dpr-show-success-space", string(fixture.revision2.ID), fixture.userID)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %v、期待値 = %v (本文: %s)", rr.Code, http.StatusOK, rr.Body.String())
	}

	body := rr.Body.String()
	// フラグメントにはタイトル変更 (新旧) と、追加された本文行のみが差分として表示されること。
	for _, want := range []string{"Old Title", "New Title", "line two"} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	// revision2は最新リビジョン (= 現在) のため、現在の状態への復元はno-opであり、インライン
	// 復元フォームは表示されない。
	restoreAction := `action="/s/dpr-show-success-space/pages/1/draft_page_revisions/` + string(fixture.revision2.ID) + `/restore"`
	if strings.Contains(body, restoreAction) {
		t.Errorf("レスポンスに復元フォームの送信先%q (現在のリビジョン) が含まれている", restoreAction)
	}
}

func TestShow_OldestRevisionShowsFullAddition(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	fixture := setupShowFixture(t, tx, queries, "dpr-show-oldest")
	handler := setupHandler(t, queries)

	req := newShowRequest(t, "dpr-show-oldest-space", string(fixture.revision1.ID), fixture.userID)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %v、期待値 = %v (本文: %s)", rr.Code, http.StatusOK, rr.Body.String())
	}

	body := rr.Body.String()
	// 最古のリビジョンは空の内容と比較され、本文全体が追加として表示されること。
	if !strings.Contains(body, "line one") {
		t.Errorf("レスポンスに%qが含まれていない", "line one")
	}
	if strings.Contains(body, "line two") {
		t.Errorf("レスポンスに%q (v2で追加) が含まれている", "line two")
	}

	// revision1は最新リビジョンではないため、その復元URLへPOSTするインライン復元フォームが
	// 表示されること (隠れるのは現在のリビジョンのみ)。
	restoreAction := `action="/s/dpr-show-oldest-space/pages/1/draft_page_revisions/` + string(fixture.revision1.ID) + `/restore"`
	if !strings.Contains(body, restoreAction) {
		t.Errorf("レスポンスに復元フォームの送信先%qが含まれていない", restoreAction)
	}
}

func TestShow_RevisionNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	fixture := setupShowFixture(t, tx, queries, "dpr-show-notfound")
	handler := setupHandler(t, queries)

	req := newShowRequest(t, "dpr-show-notfound-space", "00000000-0000-0000-0000-000000000000", fixture.userID)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_NotSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	fixture := setupShowFixture(t, tx, queries, "dpr-show-nonmember")
	strangerID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-show-stranger@example.com").
		WithAtname("dprshowstranger").
		Build()
	handler := setupHandler(t, queries)

	req := newShowRequest(t, "dpr-show-nonmember-space", string(fixture.revision2.ID), strangerID)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_OtherMembersRevision(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	fixture := setupShowFixture(t, tx, queries, "dpr-show-othermember")

	// 同じスペースの別メンバーが、同じページに自分の下書きとリビジョンを持つ。
	otherUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-show-other@example.com").
		WithAtname("dprshowother").
		Build()
	otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(fixture.spaceID).
		WithUserID(otherUserID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(fixture.spaceID).
		WithTopicID(fixture.topicID).
		WithSpaceMemberID(otherMemberID).
		Build()
	otherDraftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(fixture.spaceID).
		WithTopicID(fixture.topicID).
		WithPageID(fixture.pageID).
		WithSpaceMemberID(otherMemberID).
		Build()
	otherRev, err := repository.NewDraftPageRevisionRepository(queries).Create(context.Background(), repository.CreateDraftPageRevisionInput{
		DraftPageID:   otherDraftPageID,
		SpaceID:       fixture.spaceID,
		SpaceMemberID: otherMemberID,
		Title:         "Other Member Title",
		Body:          "other member body\n",
		BodyHTML:      "<p>other member body</p>",
	})
	if err != nil {
		t.Fatalf("Create() (otherRev) のエラー = %v", err)
	}

	handler := setupHandler(t, queries)

	// フィクスチャのオーナーが別メンバーのリビジョンを要求した場合、隠されること (404)。
	req := newShowRequest(t, "dpr-show-othermember-space", string(otherRev.ID), fixture.userID)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}
