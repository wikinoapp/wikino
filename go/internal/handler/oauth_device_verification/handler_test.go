package oauth_device_verification_test

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	oauthdeviceverificationhandler "github.com/wikinoapp/wikino/go/internal/handler/oauth_device_verification"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// deviceFixtureは、連携先のスペースのメンバーのユーザーと、そのスペースのOAuthアプリ「連携するCLI」が
// page:readとpage:writeを要求したデバイス認可
type deviceFixture struct {
	userID           model.UserID
	spaceID          model.SpaceID
	deviceCodeDigest string

	// userCodeは区切りの無い形のユーザーコード
	userCode string
}

// setupDeviceFixtureは、identifierを識別子に持つスペースと、指定したロールを持つメンバーと、アプリと、
// 許可待ちのデバイス認可をコミットして作る。flagEnabledが真ならメンバーのユーザーに公開APIの
// フィーチャーフラグを有効にする
func setupDeviceFixture(t *testing.T, identifier string, role model.SpaceRole, flagEnabled bool) deviceFixture {
	t.Helper()

	userCode, err := auth.GenerateOAuthUserCode()
	if err != nil {
		t.Fatalf("GenerateOAuthUserCode()のエラー = %v", err)
	}

	var f deviceFixture
	testutil.CommitFixture(t, func(tx *sql.Tx) {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail(identifier + "@example.com").
			WithAtname(strings.ReplaceAll(identifier, "-", "_")).
			Build()
		spaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier(identifier).
			WithName("連携先のスペース").
			Build()
		testutil.NewSpaceMemberBuilder(t, tx).
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
		appID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(spaceID).
			WithName("連携するCLI").
			WithClientID(identifier + "-client").
			Build()
		testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
			WithOAuthApplicationID(appID).
			WithSpaceID(spaceID).
			WithScopes([]model.Scope{model.ScopePageRead, model.ScopePageWrite}).
			WithDeviceCodeDigest(identifier + "-device-code").
			WithUserCodeDigest(auth.DigestOpaqueToken(userCode)).
			Build()
		f = deviceFixture{userID: userID, spaceID: spaceID, deviceCodeDigest: identifier + "-device-code", userCode: userCode}
	})
	return f
}

// setupHandlerはテスト用のDBに直接つなぐハンドラーを組み立てる
func setupHandler(t *testing.T) *oauthdeviceverificationhandler.Handler {
	t.Helper()

	q := query.New(testutil.GetTestDB())
	cfg := &config.Config{Env: "test", Domain: "example.com"}
	featureFlagRepo := repository.NewFeatureFlagRepository(q)
	deviceRepo := repository.NewOAuthDeviceAuthorizationRepository(q)
	appRepo := repository.NewOAuthApplicationRepository(q)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)

	verificationValidator := validator.NewOAuthDeviceVerificationValidator(deviceRepo, appRepo, spaceRepo)
	return oauthdeviceverificationhandler.NewHandler(
		cfg,
		newLimiter(),
		usecase.NewGetOAuthDeviceVerificationNewUsecase(featureFlagRepo),
		usecase.NewGetOAuthDeviceConsentUsecase(featureFlagRepo, spaceMemberRepo, repository.NewTopicMemberRepository(q), verificationValidator),
		usecase.NewUpdateOAuthDeviceAuthorizationUsecase(testutil.GetTestDB(), featureFlagRepo, deviceRepo, repository.NewOAuthGrantRepository(q), spaceMemberRepo, verificationValidator, validator.NewOAuthAuthorizationCreateValidator()),
	)
}

// newLimiterはテスト用のDBに直接つなぐレート制限を返す
func newLimiter() *ratelimit.Limiter {
	return ratelimit.NewLimiter(repository.NewRateLimitRepository(query.New(testutil.GetTestDB())))
}

// newRequestはログイン中のユーザーと画面を描画するロケールを載せたリクエストを組み立てる。
// GETではparamsをクエリに、POSTではフォームの本文にする。userIDが空のときはユーザーを載せない。
// 送信元のIPアドレスはremoteIPにする。IPアドレス単位のレート制限のカウンターがテストの間で
// 共有されないよう、テストごとに異なる値を渡す
func newRequest(t *testing.T, method string, userID model.UserID, remoteIP string, params url.Values) *http.Request {
	t.Helper()

	target := "/oauth/device"
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader(params.Encode())
	} else if len(params) > 0 {
		target += "?" + params.Encode()
	}
	req := httptest.NewRequest(method, target, body)
	req.RemoteAddr = remoteIP + ":12345"
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	ctx := req.Context()
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "verifying_user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// assertResponseHeadersは、ユーザーコードの入力と同意の画面のすべての応答に付けるヘッダーを確かめる
func assertResponseHeaders(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()

	want := map[string]string{
		"Content-Security-Policy": "frame-ancestors 'none'",
		"X-Frame-Options":         "DENY",
		"X-Robots-Tag":            "noindex",
		"Cache-Control":           "no-store",
		"Referrer-Policy":         "no-referrer",
	}
	for name, value := range want {
		if got := rr.Header().Get(name); got != value {
			t.Errorf("%s = %q、期待値 = %q", name, got, value)
		}
	}
}

// assertBodyContainsは、応答の本文がwantをすべて含むことを確かめる
func assertBodyContains(t *testing.T, rr *httptest.ResponseRecorder, want ...string) {
	t.Helper()

	body := rr.Body.String()
	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("本文に%qが含まれていない", w)
		}
	}
}
