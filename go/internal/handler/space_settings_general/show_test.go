package space_settings_general_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	settingsgeneral "github.com/wikinoapp/wikino/go/internal/handler/space_settings_general"
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

// settingsGeneralPathはこれらのテストが対象にする画面のパス。
func settingsGeneralPath(identifier string) string {
	return "/s/" + identifier + "/settings/general"
}

// newSettingsGeneralRequestは一般設定の画面向けに、chiのURLパラメータ・CSRFトークン・
// ログイン中のユーザー・画面を描画するロケールを載せたリクエストを組み立てる。userIDが空のときは
// ユーザーを載せず、テストが未ログインの経路へ到達できるようにする。
func newSettingsGeneralRequest(t *testing.T, method, identifier string, userID model.UserID, form map[string]string) *http.Request {
	t.Helper()

	path := settingsGeneralPath(identifier)

	var req *http.Request
	if form == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		values := url.Values{}
		for key, value := range form {
			values.Set(key, value)
		}
		req = httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "space-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// setupSettingsGeneralHandlerは渡したプールを使うハンドラーを組み立てる。画面も保存処理も
// 自身でトランザクションを開かないため、テストはフィクスチャをトランザクションに閉じ込めてよい。
func setupSettingsGeneralHandler(t *testing.T, queries *query.Queries) *settingsgeneral.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost"}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)

	return settingsgeneral.NewHandler(
		cfg,
		session.NewFlashManager("", false, true),
		usecase.NewGetSpaceSettingsGeneralUsecase(spaceRepo, spaceMemberRepo),
		usecase.NewUpdateSpaceUsecase(spaceRepo, spaceMemberRepo, validator.NewSpaceUpdateValidator(spaceRepo)),
	)
}

// settingsGeneralSpaceはスペースと、渡したロールを持つメンバー1人を用意し、テストがそれらを指すための
// 値を返す。roleが空のときはビルダーの既定 (管理者) になる。
func settingsGeneralSpace(t *testing.T, tx *sql.Tx, identifier string, role model.SpaceRole) (model.UserID, model.SpaceID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("開発チーム").
		Build()
	memberBuilder := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if role != "" {
		memberBuilder = memberBuilder.WithRole(role)
	}
	memberBuilder.Build()

	return userID, spaceID
}

func TestShow_保存済みの値が入ったフォームが表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-general-show"
	userID, _ := settingsGeneralSpace(t, tx, identifier, "")

	req := newSettingsGeneralRequest(t, http.MethodGet, identifier, userID, nil)
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		`<title>基本情報 | 開発チーム</title>`,
		`action="/s/` + identifier + `/settings/general"`,
		`name="_method" value="PATCH"`,
		`name="csrf_token" value="test-csrf-token"`,
		`value="` + identifier + `"`,
		`value="開発チーム"`,
		`href="/s/` + identifier + `/settings"`,
		"保存する",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
}

func TestShow_HEADでも200が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-general-head"
	userID, _ := settingsGeneralSpace(t, tx, identifier, "")

	req := newSettingsGeneralRequest(t, http.MethodHead, identifier, userID, nil)
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
}

func TestShow_到達できない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		role       model.SpaceRole
		outsider   bool
	}{
		{name: "スペースを変更する権限が無いメンバー (編集者)", identifier: "space-general-editor", role: model.SpaceRoleEditor},
		{name: "スペースを変更する権限が無いメンバー (閲覧者)", identifier: "space-general-viewer", role: model.SpaceRoleViewer},
		{name: "スペースのメンバーではないユーザー", identifier: "space-general-outsider", outsider: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID, _ := settingsGeneralSpace(t, tx, tt.identifier, tt.role)
			if tt.outsider {
				userID = testutil.NewUserBuilder(t, tx).
					WithEmail(tt.identifier + "-other@example.com").
					WithAtname(strings.ReplaceAll(tt.identifier, "-", "_") + "_other").
					Build()
			}

			req := newSettingsGeneralRequest(t, http.MethodGet, tt.identifier, userID, nil)
			rr := httptest.NewRecorder()
			setupSettingsGeneralHandler(t, queries).Show(rr, req)

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
	identifier := "space-general-anonymous"
	settingsGeneralSpace(t, tx, identifier, "")

	req := newSettingsGeneralRequest(t, http.MethodGet, identifier, "", nil)
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}

func TestShow_パンくずが設定のトップを通って現在地の項目で終わる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-general-breadcrumb"
	userID, _ := settingsGeneralSpace(t, tx, identifier, "")

	req := newSettingsGeneralRequest(t, http.MethodGet, identifier, userID, nil)
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Show(rr, req)

	body := rr.Body.String()
	settingsLink := `href="/s/` + identifier + `/settings"`
	current := `aria-current="page"`
	settingsIndex := strings.Index(body, settingsLink)
	currentIndex := strings.Index(body, current)
	if settingsIndex < 0 || currentIndex < 0 {
		t.Fatalf("パンくずに設定のトップへのリンク (%v) と現在地の項目 (%v) の両方が必要", settingsIndex >= 0, currentIndex >= 0)
	}
	if settingsIndex > currentIndex {
		t.Error("パンくずの現在地の項目が設定のトップへのリンクより前にある")
	}
}
