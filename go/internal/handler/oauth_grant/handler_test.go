package oauth_grant_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	oauthgranthandler "github.com/wikinoapp/wikino/go/internal/handler/oauth_grant"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// grantMemberは画面を操作するメンバーのユーザーと、そのスペース・メンバーのID
type grantMember struct {
	userID        model.UserID
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
}

// setupGrantMemberは、identifierを識別子に持つスペースと、scopesを持つメンバーを作る。
// flagEnabledが真ならメンバーのユーザーに公開APIのフィーチャーフラグを有効にする。
func setupGrantMember(t *testing.T, tx *sql.Tx, identifier string, scopes []model.Scope, flagEnabled bool) grantMember {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("連携のスペース").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes(scopes).
		Build()
	if flagEnabled {
		testutil.NewFeatureFlagBuilder(t, tx).
			WithUserID(userID).
			WithName(string(model.FeatureFlagPublicAPI)).
			Build()
	}

	return grantMember{userID: userID, spaceID: spaceID, spaceMemberID: spaceMemberID}
}

// buildGrantは、メンバーspaceMemberIDがスペースのアプリappNameに与えた許可を作る
func (m grantMember) buildGrant(t *testing.T, tx *sql.Tx, spaceMemberID model.SpaceMemberID, appName, clientID string, scopes []model.Scope) model.OAuthGrantID {
	t.Helper()

	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).WithName(appName).WithClientID(clientID).Build()
	return testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(appID).WithSpaceID(m.spaceID).WithSpaceMemberID(spaceMemberID).
		WithScopes(scopes).Build()
}

// setupHandlerは渡したクエリを使うハンドラーを組み立てる。
func setupHandler(t *testing.T, q *query.Queries) *oauthgranthandler.Handler {
	t.Helper()

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	featureFlagRepo := repository.NewFeatureFlagRepository(q)
	oauthGrantRepo := repository.NewOAuthGrantRepository(q)
	oauthApplicationRepo := repository.NewOAuthApplicationRepository(q)

	return oauthgranthandler.NewHandler(
		&config.Config{Env: "test", Domain: "localhost"},
		session.NewFlashManager("", false, true),
		usecase.NewGetOAuthGrantsUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, oauthGrantRepo, oauthApplicationRepo),
		usecase.NewRevokeOAuthGrantUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, oauthGrantRepo, oauthApplicationRepo),
	)
}

// newRequestはchiのURLパラメータ・ログイン中のユーザー・画面を描画するロケールを載せた
// リクエストを組み立てる。userIDが空のときはユーザーを載せず、未ログインの経路へ到達できるようにする。
func newRequest(t *testing.T, method, path, identifier string, userID model.UserID) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)
	if grantID, ok := strings.CutPrefix(path, "/s/"+identifier+"/settings/oauth_grants/"); ok {
		rctx.URLParams.Add("oauth_grant_id", grantID)
	}

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "grant-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}
