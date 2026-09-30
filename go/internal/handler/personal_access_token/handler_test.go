package personal_access_token_test

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	pathandler "github.com/wikinoapp/wikino/go/internal/handler/personal_access_token"
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

// patMemberは画面を操作するメンバーのユーザーと、そのスペース・メンバーのID
type patMember struct {
	userID        model.UserID
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
}

// setupPATMemberは、identifierを識別子に持つスペースと、roleを持つメンバーを作る。
// flagEnabledが真ならメンバーのユーザーに公開APIのフィーチャーフラグを有効にする。
func setupPATMember(t *testing.T, tx *sql.Tx, identifier string, role model.SpaceRole, flagEnabled bool) patMember {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("トークンのスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(role).
		Build()
	if flagEnabled {
		testutil.NewFeatureFlagBuilder(t, tx).
			WithUserID(userID).
			WithName(string(model.FeatureFlagPublicAPI)).
			Build()
	}

	return patMember{userID: userID, spaceID: spaceID, spaceMemberID: spaceMemberID}
}

// setupHandlerは渡したクエリを使うハンドラーを組み立てる。
func setupHandler(t *testing.T, q *query.Queries) *pathandler.Handler {
	t.Helper()

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	featureFlagRepo := repository.NewFeatureFlagRepository(q)
	patRepo := repository.NewPersonalAccessTokenRepository(q)

	return pathandler.NewHandler(
		&config.Config{Env: "test", Domain: "localhost"},
		session.NewFlashManager("", false, true),
		usecase.NewGetPersonalAccessTokensUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, patRepo),
		usecase.NewGetPersonalAccessTokenNewUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo),
		usecase.NewCreatePersonalAccessTokenUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, patRepo, validator.NewPersonalAccessTokenCreateValidator()),
		usecase.NewRevokePersonalAccessTokenUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, patRepo),
	)
}

// newRequestはchiのURLパラメータ・ログイン中のユーザー・画面を描画するロケールを載せた
// リクエストを組み立てる。formがnilでなければフォームの本文として送る。userIDが空のときは
// ユーザーを載せず、未ログインの経路へ到達できるようにする。
func newRequest(t *testing.T, method, path, identifier string, userID model.UserID, form url.Values) *http.Request {
	t.Helper()

	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)
	if tokenID, ok := strings.CutPrefix(path, "/s/"+identifier+"/settings/personal_access_tokens/"); ok && tokenID != "new" {
		rctx.URLParams.Add("personal_access_token_id", tokenID)
	}

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "pat-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// breadcrumbはパンくずのナビゲーション部分だけのマークアップを返す。
func breadcrumb(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if start == -1 {
		t.Fatal("レスポンスにパンくずのナビゲーションが含まれていない")
	}
	end := strings.Index(body[start:], "</nav>")
	if end == -1 {
		t.Fatal("パンくずのナビゲーションに閉じタグが無い")
	}

	return body[start : start+end]
}
