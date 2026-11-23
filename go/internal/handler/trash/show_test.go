package trash_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/trash"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newTrashRequestはchiのURLパラメータ・ログイン中のユーザー・画面を描画するロケールを載せた
// リクエストを組み立てる。userIDが空のときはユーザーを載せず、未ログインの経路へ到達できるようにする。
func newTrashRequest(t *testing.T, identifier, rawQuery string, userID model.UserID) *http.Request {
	t.Helper()

	target := "/s/" + identifier + "/trash"
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "trash-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

func setupHandler(t *testing.T, queries *query.Queries) *trash.Handler {
	t.Helper()

	return trash.NewHandler(
		&config.Config{Env: "test", Domain: "localhost"},
		usecase.NewGetTrashShowUsecase(
			repository.NewSpaceRepository(queries),
			repository.NewSpaceMemberRepository(queries),
			repository.NewPageRepository(queries),
			repository.NewTopicRepository(queries),
			repository.NewTopicMemberRepository(queries),
		),
	)
}

// trashSpaceはスペースとトピック番号1のトピック、渡したロールを持つメンバー1人を用意し、
// メンバーのユーザーとスペース・トピックを返す。
func trashSpace(t *testing.T, tx *sql.Tx, identifier string, spaceRole model.SpaceRole) (model.UserID, model.SpaceID, model.TopicID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("ゴミ箱のスペース").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(spaceRole).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("ゴミ箱のトピック").
		Build()

	return userID, spaceID, topicID
}

func TestShow_ゴミ箱のページを表で表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "trash-show-table"
	userID, spaceID, topicID := trashSpace(t, tx, identifier, model.SpaceRoleAdmin)
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(7).
		WithTitle("ゴミ箱のページ").
		WithTrashedAt(time.Now().Add(-time.Hour)).
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newTrashRequest(t, identifier, "", userID))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>ゴミ箱 | ゴミ箱のスペース</title>",
		`<caption class="sr-only">ゴミ箱のスペースのゴミ箱に入れたページ</caption>`,
		`<th scope="col" class="w-full">ページ</th>`,
		`<th scope="row"`,
		`name="page_ids" value="` + string(pageID) + `"`,
		`aria-label="「ゴミ箱のページ」を選択"`,
		`href="/s/` + identifier + `/pages/7"`,
		"ゴミ箱のトピック",
		`action="/s/` + identifier + `/bulk_restored_pages"`,
		`name="csrf_token"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
}

func TestShow_ページが無ければ空の状態を表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "trash-show-empty"
	userID, _, _ := trashSpace(t, tx, identifier, model.SpaceRoleAdmin)

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newTrashRequest(t, identifier, "", userID))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "ゴミ箱は空です") {
		t.Error("空の状態のメッセージが表示されていない")
	}
	if strings.Contains(body, "<table") {
		t.Error("ページが無いのに表が表示された")
	}
}

func TestShow_開けない閲覧者には404を返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		rawQuery   string
		spaceRole  model.SpaceRole
	}{
		{name: "どのトピックでもpage_trash:readを持たないメンバー", identifier: "trash-show-viewer", spaceRole: model.SpaceRoleViewer},
		{name: "総ページ数を超えるページ番号", identifier: "trash-show-out-of-range", rawQuery: "page=2", spaceRole: model.SpaceRoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID, _, _ := trashSpace(t, tx, tt.identifier, tt.spaceRole)

			rr := httptest.NewRecorder()
			setupHandler(t, queries).Show(rr, newTrashRequest(t, tt.identifier, tt.rawQuery, userID))

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestShow_未ログインならログインへリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newTrashRequest(t, "trash-show-guest", "", ""))

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
	}
}
