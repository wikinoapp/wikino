package bulk_restored_page_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/bulk_restored_page"
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

// newCreateRequestは、選んだページのIDをフォームに載せた一括復元のリクエストを組み立てる。
// userIDが空のときはユーザーを載せず、未ログインの経路へ到達できるようにする。
func newCreateRequest(t *testing.T, identifier string, pageIDs []model.PageID, userID model.UserID) *http.Request {
	t.Helper()

	form := url.Values{}
	for _, id := range pageIDs {
		form.Add("page_ids", string(id))
	}
	req := httptest.NewRequest(http.MethodPost, "/s/"+identifier+"/bulk_restored_pages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "bulk-restored-page-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// 一括復元のUseCaseは自分でトランザクションを開くため、テストのデータはDBにコミットする。
func setupHandler(db *sql.DB) *bulk_restored_page.Handler {
	queries := query.New(db)
	pageRepo := repository.NewPageRepository(queries)

	return bulk_restored_page.NewHandler(
		session.NewFlashManager("", false, true),
		usecase.NewRestorePagesUsecase(
			db,
			repository.NewSpaceRepository(queries),
			repository.NewSpaceMemberRepository(queries),
			pageRepo,
			repository.NewTopicRepository(queries),
			repository.NewTopicMemberRepository(queries),
			validator.NewBulkRestoredPageCreateValidator(pageRepo),
		),
	)
}

// trashedPageは、指定したロールのメンバーがいるスペースに、ゴミ箱に入れたページを1つ作る。
func trashedPage(t *testing.T, db *sql.DB, identifier string, spaceRole model.SpaceRole) (model.UserID, model.PageID) {
	t.Helper()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(identifier).
		Build()
	testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(spaceRole).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("ゴミ箱のトピック").
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("ゴミ箱のページ").
		WithTrashedAt(time.Now().Add(-time.Hour)).
		Build()

	return userID, pageID
}

// flashOfはレスポンスが設定したフラッシュメッセージを返す。
func flashOf(t *testing.T, rr *httptest.ResponseRecorder) session.FlashMessage {
	t.Helper()

	for _, c := range rr.Result().Cookies() {
		if c.Name != session.FlashCookieName || c.Value == "" {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(c.Value)
		if err != nil {
			t.Fatalf("フラッシュのデコードに失敗: %v", err)
		}
		var flash session.FlashMessage
		if err := json.Unmarshal(data, &flash); err != nil {
			t.Fatalf("フラッシュの読み取りに失敗: %v", err)
		}
		return flash
	}
	t.Fatal("フラッシュが設定されていない")
	return session.FlashMessage{}
}

func isTrashed(t *testing.T, db *sql.DB, pageID model.PageID) bool {
	t.Helper()

	var trashed bool
	if err := db.QueryRowContext(context.Background(), "SELECT trashed_at IS NOT NULL FROM pages WHERE id = $1", string(pageID)).Scan(&trashed); err != nil {
		t.Fatalf("ページの取得に失敗: %v", err)
	}
	return trashed
}

func TestCreate_復元してゴミ箱の一覧へ戻す(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	identifier := "bulk-restored-page-success"
	userID, pageID := trashedPage(t, db, identifier, model.SpaceRoleAdmin)

	rr := httptest.NewRecorder()
	setupHandler(db).Create(rr, newCreateRequest(t, identifier, []model.PageID{pageID}, userID))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got, want := rr.Header().Get("Location"), "/s/"+identifier+"/trash"; got != want {
		t.Errorf("Location = %q、期待値 = %q", got, want)
	}
	if flash := flashOf(t, rr); flash.Type != session.FlashSuccess || flash.Message != "ページを復元しました" {
		t.Errorf("フラッシュ = %+v、成功の「ページを復元しました」を期待", flash)
	}
	if isTrashed(t, db, pageID) {
		t.Error("ページがゴミ箱から戻っていない")
	}
}

func TestCreate_ページを選ばなければエラーを伝えてゴミ箱の一覧へ戻す(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	identifier := "bulk-restored-page-empty"
	userID, pageID := trashedPage(t, db, identifier, model.SpaceRoleAdmin)

	rr := httptest.NewRecorder()
	setupHandler(db).Create(rr, newCreateRequest(t, identifier, nil, userID))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got, want := rr.Header().Get("Location"), "/s/"+identifier+"/trash"; got != want {
		t.Errorf("Location = %q、期待値 = %q", got, want)
	}
	if flash := flashOf(t, rr); flash.Type != session.FlashError || flash.Message != "復元するページを選んでください" {
		t.Errorf("フラッシュ = %+v、エラーの「復元するページを選んでください」を期待", flash)
	}
	if !isTrashed(t, db, pageID) {
		t.Error("選んでいないページがゴミ箱から戻った")
	}
}

func TestCreate_ページを戻せない閲覧者には404を返す(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	identifier := "bulk-restored-page-viewer"
	userID, pageID := trashedPage(t, db, identifier, model.SpaceRoleViewer)

	rr := httptest.NewRecorder()
	setupHandler(db).Create(rr, newCreateRequest(t, identifier, []model.PageID{pageID}, userID))

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
	if !isTrashed(t, db, pageID) {
		t.Error("権限が無いのにページがゴミ箱から戻った")
	}
}

func TestCreate_未ログインならログインへリダイレクトする(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	setupHandler(testutil.GetTestDB()).Create(rr, newCreateRequest(t, "bulk-restored-page-guest", nil, ""))

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
	}
}
