package topic_settings_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	topicsettings "github.com/wikinoapp/wikino/go/internal/handler/topic_settings"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newSettingsRequestはchiのURLパラメータ・ログイン中のユーザー・画面を描画するロケールを
// 載せたリクエストを組み立てる。userIDが空のときはユーザーを載せず、テストが未ログインの経路へ
// 到達できるようにする。
func newSettingsRequest(t *testing.T, method, identifier, topicNumber string, userID model.UserID) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, "/s/"+identifier+"/topics/"+topicNumber+"/settings", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)
	rctx.URLParams.Add("topic_number", topicNumber)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "topic-settings-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// setupSettingsHandlerは渡したクエリを使うハンドラーを組み立てる。
func setupSettingsHandler(t *testing.T, queries *query.Queries) *topicsettings.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost"}

	return topicsettings.NewHandler(
		cfg,
		usecase.NewGetTopicSettingsUsecase(
			repository.NewSpaceRepository(queries),
			repository.NewSpaceMemberRepository(queries),
			repository.NewTopicRepository(queries),
			repository.NewTopicMemberRepository(queries),
		),
	)
}

// settingsTopicはスペースとトピック番号1のトピック、渡したロールを持つメンバー1人を用意し、
// メンバーのユーザーを返す。spaceRoleが空のときはビルダーの既定 (管理者) になる。
// topicRoleが空でなければ、そのロールのトピックメンバーにする。
func settingsTopic(t *testing.T, tx *sql.Tx, identifier string, spaceRole model.SpaceRole, topicRole model.TopicRole) model.UserID {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("設定のスペース").
		Build()
	memberBuilder := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if spaceRole != "" {
		memberBuilder = memberBuilder.WithRole(spaceRole)
	}
	spaceMemberID := memberBuilder.Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("設定のトピック").
		Build()
	if topicRole != "" {
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(spaceMemberID).
			WithRole(topicRole).
			Build()
	}

	return userID
}

func generalPath(identifier string) string {
	return `href="/s/` + identifier + `/topics/1/settings/general"`
}

func deletionPath(identifier string) string {
	return `href="/s/` + identifier + `/topics/1/settings/deletion/new"`
}

func TestShow_開ける項目へのリンクだけを表示する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		spaceRole   model.SpaceRole
		topicRole   model.TopicRole
		wantGeneral bool
		wantDelete  bool
	}{
		{
			name:        "スペースの管理者には一般設定と削除を表示する",
			identifier:  "topic-settings-admin",
			wantGeneral: true,
			wantDelete:  true,
		},
		{
			name:        "トピックの管理者のロールを持つ編集者には一般設定と削除を表示する",
			identifier:  "topic-settings-topic-admin",
			spaceRole:   model.SpaceRoleEditor,
			topicRole:   model.TopicRoleAdmin,
			wantGeneral: true,
			wantDelete:  true,
		},
		{
			name:        "topic:deleteを持たない編集者には一般設定だけを表示する",
			identifier:  "topic-settings-editor",
			spaceRole:   model.SpaceRoleEditor,
			wantGeneral: true,
			wantDelete:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID := settingsTopic(t, tx, tt.identifier, tt.spaceRole, tt.topicRole)

			req := newSettingsRequest(t, http.MethodGet, tt.identifier, "1", userID)
			rr := httptest.NewRecorder()
			setupSettingsHandler(t, queries).Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			if !strings.Contains(body, "<title>設定 | 設定のトピック | 設定のスペース</title>") {
				t.Error("レスポンスにトピックの設定のタイトルが含まれていない")
			}
			if got := strings.Contains(body, generalPath(tt.identifier)); got != tt.wantGeneral {
				t.Errorf("一般設定へのリンクの有無 = %v、期待値 = %v", got, tt.wantGeneral)
			}
			if got := strings.Contains(body, "基本情報"); got != tt.wantGeneral {
				t.Errorf("一般設定のラベルの有無 = %v、期待値 = %v", got, tt.wantGeneral)
			}
			if got := strings.Contains(body, deletionPath(tt.identifier)); got != tt.wantDelete {
				t.Errorf("トピックの削除へのリンクの有無 = %v、期待値 = %v", got, tt.wantDelete)
			}
			if got := strings.Contains(body, "トピックの削除"); got != tt.wantDelete {
				t.Errorf("トピックの削除のラベルの有無 = %v、期待値 = %v", got, tt.wantDelete)
			}
		})
	}
}

func TestShow_検索エンジンに載せない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-settings-noindex"
	userID := settingsTopic(t, tx, identifier, "", "")

	req := newSettingsRequest(t, http.MethodGet, identifier, "1", userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}
}

func TestShow_HEADでも200が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-settings-head"
	userID := settingsTopic(t, tx, identifier, "", "")

	req := newSettingsRequest(t, http.MethodHead, identifier, "1", userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
}

func TestShow_到達できない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		spaceRole   model.SpaceRole
		topicNumber string
		outsider    bool
	}{
		{
			name:        "開ける項目を持たない閲覧者",
			identifier:  "topic-settings-viewer",
			spaceRole:   model.SpaceRoleViewer,
			topicNumber: "1",
		},
		{
			name:        "スペースのメンバーではないユーザー",
			identifier:  "topic-settings-outsider",
			topicNumber: "1",
			outsider:    true,
		},
		{
			name:        "存在しないトピック",
			identifier:  "topic-settings-missing",
			topicNumber: "999",
		},
		{
			name:        "数値でないトピック番号",
			identifier:  "topic-settings-nan",
			topicNumber: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID := settingsTopic(t, tx, tt.identifier, tt.spaceRole, "")
			if tt.outsider {
				userID = testutil.NewUserBuilder(t, tx).
					WithEmail(tt.identifier + "-other@example.com").
					WithAtname(strings.ReplaceAll(tt.identifier, "-", "_") + "_other").
					Build()
			}

			req := newSettingsRequest(t, http.MethodGet, tt.identifier, tt.topicNumber, userID)
			rr := httptest.NewRecorder()
			setupSettingsHandler(t, queries).Show(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestShow_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-settings-anon"
	settingsTopic(t, tx, identifier, "", "")

	req := newSettingsRequest(t, http.MethodGet, identifier, "1", "")
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}

// 経路はトピックを通ってこの画面自身で終わるため、末尾の項目はリンクではなくaria-currentを持つラベルになる。
func TestShow_パンくずがトピックを通って現在地の項目で終わる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-settings-crumb"
	userID := settingsTopic(t, tx, identifier, "", "")

	req := newSettingsRequest(t, http.MethodGet, identifier, "1", userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	breadcrumb := settingsBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/s/` + identifier + `"`,
		`href="/s/` + identifier + `/topics/1"`,
		"設定のトピック",
		`aria-current="page"`,
		"設定",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/`+identifier+`/topics/1/settings"`) {
		t.Error("現在の設定のパンくずの項目がリンクになっている")
	}
}

// settingsBreadcrumbはパンくずのナビゲーション部分だけのマークアップを返す。経路に
// ついての検証が、画面の他の場所に出た同じ文字列で満たされてしまうのを防ぐ。
func settingsBreadcrumb(t *testing.T, body string) string {
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
