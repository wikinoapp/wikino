package oauth_application_client_secret_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	clientsecrethandler "github.com/wikinoapp/wikino/go/internal/handler/oauth_application_client_secret"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// clientSecretPatternは再発行した直後の画面に出るクライアントシークレットを取り出す
var clientSecretPattern = regexp.MustCompile(`value="(wks_[A-Za-z0-9_-]+)"`)

// setupMemberは、identifierを識別子に持つスペースと、scopesを持ち公開APIのフィーチャーフラグが
// 有効なメンバーを作り、ユーザーとスペースのIDを返す。
func setupMember(t *testing.T, tx *sql.Tx, identifier string, scopes []model.Scope) (model.UserID, model.SpaceID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("アプリのスペース").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes(scopes).
		Build()
	testutil.NewFeatureFlagBuilder(t, tx).
		WithUserID(userID).
		WithName(string(model.FeatureFlagPublicAPI)).
		Build()

	return userID, spaceID
}

// setupHandlerは渡したクエリを使うハンドラーを組み立てる。
func setupHandler(t *testing.T, q *query.Queries) *clientsecrethandler.Handler {
	t.Helper()

	return clientsecrethandler.NewHandler(
		&config.Config{Env: "test", Domain: "localhost"},
		usecase.NewRegenerateOAuthApplicationClientSecretUsecase(
			repository.NewSpaceRepository(q),
			repository.NewSpaceMemberRepository(q),
			repository.NewFeatureFlagRepository(q),
			repository.NewOAuthApplicationRepository(q),
		),
	)
}

// newRequestはchiのURLパラメータ・ログイン中のユーザー・画面を描画するロケールを載せた
// 再発行のリクエストを組み立てる。
func newRequest(t *testing.T, identifier string, appID model.OAuthApplicationID, userID model.UserID) *http.Request {
	t.Helper()

	path := "/s/" + identifier + "/settings/oauth_applications/" + string(appID) + "/client_secret"
	req := httptest.NewRequest(http.MethodPost, path, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)
	rctx.URLParams.Add("oauth_application_id", string(appID))

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "app-user"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

func TestCreate_新しいシークレットをその応答でだけ表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-secret"
	userID, spaceID := setupMember(t, tx, identifier, []model.Scope{model.ScopeOAuthApplicationWrite})
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("連携するWebアプリ").
		WithClientID("oauth-app-secret-client").
		WithConfidentialClientSecretDigest("oauth_app_secret_old_digest").
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, newRequest(t, identifier, appID, userID))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
	}

	body := rr.Body.String()
	secret := clientSecretPattern.FindStringSubmatch(body)
	if secret == nil {
		t.Fatal("レスポンスにクライアントシークレットが含まれていない")
	}
	for _, want := range []string{
		"<title>シークレットを再発行しました | 連携するWebアプリ | アプリのスペース</title>",
		"二度と表示できない",
		`href="/s/` + identifier + `/settings/oauth_applications/` + string(appID) + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), appID, spaceID)
	if err != nil {
		t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
	}
	if app.ClientSecretDigest == nil || *app.ClientSecretDigest != auth.DigestOpaqueToken(secret[1]) {
		t.Errorf("ClientSecretDigest = %v、期待値は表示したシークレットのダイジェスト", app.ClientSecretDigest)
	}
}

func TestCreate_再発行できないメンバーとアプリには404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-secret-denied"
	writerID, spaceID := setupMember(t, tx, identifier, []model.Scope{model.ScopeOAuthApplicationWrite})
	readerID, readerSpaceID := setupMember(t, tx, identifier+"-reader", []model.Scope{model.ScopeOAuthApplicationRead})
	publicID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(spaceID).
		WithClientID("oauth-app-secret-denied-public").
		Build()
	readerAppID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(readerSpaceID).
		WithClientID("oauth-app-secret-denied-reader").
		WithConfidentialClientSecretDigest("oauth_app_secret_denied_digest").
		Build()

	// サブテストはフィクスチャのトランザクションを共有するため、並行に走らせない。
	for _, tt := range []struct {
		name       string
		identifier string
		userID     model.UserID
		appID      model.OAuthApplicationID
	}{
		{name: "oauth_application:readだけを持つ", identifier: identifier + "-reader", userID: readerID, appID: readerAppID},
		{name: "publicクライアント", identifier: identifier, userID: writerID, appID: publicID},
		{name: "別のスペースのアプリ", identifier: identifier, userID: writerID, appID: readerAppID},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			setupHandler(t, q).Create(rr, newRequest(t, tt.identifier, tt.appID, tt.userID))

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
			if clientSecretPattern.MatchString(rr.Body.String()) {
				t.Error("404のレスポンスにシークレットが含まれている")
			}
		})
	}

	app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), readerAppID, readerSpaceID)
	if err != nil {
		t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
	}
	if app.ClientSecretDigest == nil || *app.ClientSecretDigest != "oauth_app_secret_denied_digest" {
		t.Errorf("ClientSecretDigest = %v、期待値は元のダイジェスト", app.ClientSecretDigest)
	}
}
