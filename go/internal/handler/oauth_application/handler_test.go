package oauth_application_test

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
	oauthapphandler "github.com/wikinoapp/wikino/go/internal/handler/oauth_application"
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

// appMemberは画面を操作するメンバーのユーザーと、そのスペース・メンバーのID
type appMember struct {
	userID        model.UserID
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
}

// setupAppMemberは、identifierを識別子に持つスペースと、roleを持つメンバーを作る。
// flagEnabledが真ならメンバーのユーザーに公開APIのフィーチャーフラグを有効にする。
// メンバーのユーザーの名前はnameにする。
func setupAppMember(t *testing.T, tx *sql.Tx, identifier string, name string, role model.SpaceRole, flagEnabled bool) appMember {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		WithName(name).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("アプリのスペース").
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

	return appMember{userID: userID, spaceID: spaceID, spaceMemberID: spaceMemberID}
}

// setupHandlerは渡したクエリを使うハンドラーを組み立てる。
func setupHandler(t *testing.T, q *query.Queries) *oauthapphandler.Handler {
	t.Helper()

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	userRepo := repository.NewUserRepository(q)
	featureFlagRepo := repository.NewFeatureFlagRepository(q)
	appRepo := repository.NewOAuthApplicationRepository(q)

	return oauthapphandler.NewHandler(
		&config.Config{Env: "test", Domain: "localhost"},
		session.NewFlashManager("", false, true),
		usecase.NewGetOAuthApplicationsUsecase(spaceRepo, spaceMemberRepo, userRepo, featureFlagRepo, appRepo),
		usecase.NewGetOAuthApplicationNewUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo),
		usecase.NewCreateOAuthApplicationUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, appRepo, validator.NewOAuthApplicationCreateValidator()),
		usecase.NewGetOAuthApplicationUsecase(spaceRepo, spaceMemberRepo, userRepo, featureFlagRepo, appRepo),
		usecase.NewGetOAuthApplicationEditUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, appRepo),
		usecase.NewUpdateOAuthApplicationUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, appRepo, validator.NewOAuthApplicationUpdateValidator()),
		usecase.NewDeleteOAuthApplicationUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo, appRepo),
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
	if rest, ok := strings.CutPrefix(path, "/s/"+identifier+"/settings/oauth_applications/"); ok && rest != "new" {
		appID, _, _ := strings.Cut(rest, "/")
		rctx.URLParams.Add("oauth_application_id", appID)
	}

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "app-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}
