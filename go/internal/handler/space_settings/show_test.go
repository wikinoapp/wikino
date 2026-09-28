package space_settings_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	spacesettings "github.com/wikinoapp/wikino/go/internal/handler/space_settings"
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
func newSettingsRequest(t *testing.T, method, identifier string, userID model.UserID) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, "/s/"+identifier+"/settings", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "settings-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// setupSettingsHandlerは渡したクエリを使うハンドラーを組み立てる。
func setupSettingsHandler(t *testing.T, queries *query.Queries) *spacesettings.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost"}

	return spacesettings.NewHandler(
		cfg,
		usecase.NewGetSpaceSettingsUsecase(
			repository.NewSpaceRepository(queries),
			repository.NewSpaceMemberRepository(queries),
			repository.NewFeatureFlagRepository(queries),
		),
	)
}

// settingsSpaceはスペースと、渡したスコープを持つメンバー1人を用意し、メンバーのユーザーを返す。
// scopesがnilのときはビルダーの既定 (space:admin) になる。
func settingsSpace(t *testing.T, tx *sql.Tx, identifier string, scopes []model.Scope) model.UserID {
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
	if scopes != nil {
		memberBuilder = memberBuilder.WithScopes(scopes)
	}
	memberBuilder.Build()

	return userID
}

// existingItemPathsは既存の項目 (一般・エクスポート・添付ファイル・削除) のリンク先を返す。
func existingItemPaths(identifier string) []string {
	return []string{
		`href="/s/` + identifier + `/settings/general"`,
		`href="/s/` + identifier + `/settings/exports/new"`,
		`href="/s/` + identifier + `/settings/attachments"`,
		`href="/s/` + identifier + `/settings/deletion/new"`,
	}
}

func TestShow_space_writeを持つメンバーに既存の項目が表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-settings-show"
	userID := settingsSpace(t, tx, identifier, nil)

	req := newSettingsRequest(t, http.MethodGet, identifier, userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	wants := append(existingItemPaths(identifier),
		"<title>設定 | 設定のスペース</title>",
		"基本情報",
		"エクスポート",
		"添付ファイル",
		"スペースの削除",
	)
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
}

func TestShow_HEADでも200が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-settings-head"
	userID := settingsSpace(t, tx, identifier, nil)

	req := newSettingsRequest(t, http.MethodHead, identifier, userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
}

func TestShow_トークン管理の権限だけを持つメンバーはフラグが有効なら開けるが既存の項目は出ない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-settings-pat"
	userID := settingsSpace(t, tx, identifier, []model.Scope{model.ScopePersonalAccessTokenRead})
	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName(string(model.FeatureFlagPublicAPI)).
		Build()

	req := newSettingsRequest(t, http.MethodGet, identifier, userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, notWant := range existingItemPaths(identifier) {
		if strings.Contains(body, notWant) {
			t.Errorf("space:writeを持たないメンバーのレスポンスに%qが含まれている", notWant)
		}
	}
	if !strings.Contains(body, `href="/s/`+identifier+`/settings/personal_access_tokens"`) {
		t.Error("レスポンスに個人アクセストークンへのリンクが含まれていない")
	}
	if strings.Contains(body, `href="/s/`+identifier+`/settings/oauth_grants"`) {
		t.Error("oauth_grant:readを持たないメンバーのレスポンスに連携中のアプリへのリンクが含まれている")
	}
}

func TestShow_連携の閲覧権限だけを持つメンバーはフラグが有効なら開けるが既存の項目は出ない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-settings-oauth-grant"
	userID := settingsSpace(t, tx, identifier, []model.Scope{model.ScopeOAuthGrantRead})
	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName(string(model.FeatureFlagPublicAPI)).
		Build()

	req := newSettingsRequest(t, http.MethodGet, identifier, userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, notWant := range existingItemPaths(identifier) {
		if strings.Contains(body, notWant) {
			t.Errorf("space:writeを持たないメンバーのレスポンスに%qが含まれている", notWant)
		}
	}
	if !strings.Contains(body, `href="/s/`+identifier+`/settings/oauth_grants"`) {
		t.Error("レスポンスに連携中のアプリへのリンクが含まれていない")
	}
	if strings.Contains(body, `href="/s/`+identifier+`/settings/personal_access_tokens"`) {
		t.Error("personal_access_token:readを持たないメンバーのレスポンスに個人アクセストークンへのリンクが含まれている")
	}
}

func TestShow_OAuthアプリの閲覧権限だけを持つメンバーはフラグが有効なら開けるが既存の項目は出ない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-settings-oauth-app"
	userID := settingsSpace(t, tx, identifier, []model.Scope{model.ScopeOAuthApplicationRead})
	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName(string(model.FeatureFlagPublicAPI)).
		Build()

	req := newSettingsRequest(t, http.MethodGet, identifier, userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, notWant := range existingItemPaths(identifier) {
		if strings.Contains(body, notWant) {
			t.Errorf("space:writeを持たないメンバーのレスポンスに%qが含まれている", notWant)
		}
	}
	if !strings.Contains(body, `href="/s/`+identifier+`/settings/oauth_applications"`) {
		t.Error("レスポンスにOAuthアプリへのリンクが含まれていない")
	}
	if strings.Contains(body, `href="/s/`+identifier+`/settings/personal_access_tokens"`) {
		t.Error("personal_access_token:readを持たないメンバーのレスポンスに個人アクセストークンへのリンクが含まれている")
	}
}

// フラグが無効なら、トークン管理のスコープを持っていても個人アクセストークン・連携中のアプリ・OAuthアプリの項目を出さない。
func TestShow_フラグが無効なら公開APIの項目へのリンクを出さない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-settings-pat-link-noflag"
	userID := settingsSpace(t, tx, identifier, nil)

	req := newSettingsRequest(t, http.MethodGet, identifier, userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if strings.Contains(rr.Body.String(), "/settings/personal_access_tokens") {
		t.Error("フラグが無効なメンバーのレスポンスに個人アクセストークンへのリンクが含まれている")
	}
	if strings.Contains(rr.Body.String(), "/settings/oauth_grants") {
		t.Error("フラグが無効なメンバーのレスポンスに連携中のアプリへのリンクが含まれている")
	}
	if strings.Contains(rr.Body.String(), "/settings/oauth_applications") {
		t.Error("フラグが無効なメンバーのレスポンスにOAuthアプリへのリンクが含まれている")
	}
}

func TestShow_到達できない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		scopes     []model.Scope
		outsider   bool
		missing    bool
	}{
		{
			name:       "どの項目の権限も持たないメンバー",
			identifier: "space-settings-reader",
			scopes:     []model.Scope{model.ScopeSpaceRead, model.ScopePageWrite},
		},
		{
			name:       "トークン管理の権限だけを持つがフラグが無効なメンバー",
			identifier: "space-settings-pat-noflag",
			scopes:     []model.Scope{model.ScopePersonalAccessTokenRead, model.ScopeOAuthGrantRead},
		},
		{
			name:       "OAuthアプリの権限だけを持つがフラグが無効なメンバー",
			identifier: "space-settings-oauth-app-noflag",
			scopes:     []model.Scope{model.ScopeOAuthApplicationRead},
		},
		{
			name:       "スペースのメンバーではないユーザー",
			identifier: "space-settings-outsider",
			outsider:   true,
		},
		{
			name:       "存在しないスペース",
			identifier: "space-settings-missing",
			missing:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID := settingsSpace(t, tx, tt.identifier, tt.scopes)
			if tt.outsider {
				userID = testutil.NewUserBuilder(t, tx).
					WithEmail(tt.identifier + "-other@example.com").
					WithAtname(strings.ReplaceAll(tt.identifier, "-", "_") + "_other").
					Build()
			}
			identifier := tt.identifier
			if tt.missing {
				identifier = tt.identifier + "-none"
			}

			req := newSettingsRequest(t, http.MethodGet, identifier, userID)
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
	identifier := "space-settings-anon"
	settingsSpace(t, tx, identifier, nil)

	req := newSettingsRequest(t, http.MethodGet, identifier, "")
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}

// 経路はこの画面自身で終わるため、末尾の項目はリンクではなくaria-currentを持つラベルになる。
func TestShow_パンくずが現在地の項目で終わる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-settings-crumb"
	userID := settingsSpace(t, tx, identifier, nil)

	req := newSettingsRequest(t, http.MethodGet, identifier, userID)
	rr := httptest.NewRecorder()
	setupSettingsHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	breadcrumb := settingsBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/s/` + identifier + `"`,
		`aria-current="page"`,
		"設定",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/`+identifier+`/settings"`) {
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
