package usecase

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// oauthTokenTestCodeVerifierは、oauthAuthorizationTestCodeChallengeに対応するcode_verifier (RFC 7636 Appendix B)
const oauthTokenTestCodeVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

// oauthTokenTestRedirectURIは認可要求で使ったリダイレクトURI
const oauthTokenTestRedirectURI = "http://127.0.0.1:53123/callback"

// oauthTokenTestClientSecretはconfidentialクライアントのシークレット
const oauthTokenTestClientSecret = "wks_test_client_secret"

// oauthTokenFixtureOptionsは、トークン要求のフィクスチャの既定値 (フラグ有効・`oauth_grant:write`
// を持つメンバー・スペースのpublicクライアント・失効していない許可) から変える点
type oauthTokenFixtureOptions struct {
	memberScopes   []model.Scope
	flagDisabled   bool
	confidential   bool
	officialClient bool
}

// oauthTokenFixtureは、スペースのメンバーがOAuthアプリに与えた許可
type oauthTokenFixture struct {
	patMemberFixture
	identifier model.SpaceIdentifier
	appID      model.OAuthApplicationID
	clientID   string
	grantID    model.OAuthGrantID
}

// setupOAuthTokenFixtureは、スペースkeyのメンバーと、そのメンバーがOAuthアプリに与えた許可を作る
func setupOAuthTokenFixture(t *testing.T, tx *sql.Tx, key string, opts oauthTokenFixtureOptions) oauthTokenFixture {
	t.Helper()

	memberScopes := opts.memberScopes
	if memberScopes == nil {
		memberScopes = []model.Scope{model.ScopePageWrite, model.ScopeOAuthGrantWrite}
	}
	f := setupPATMember(t, tx, key, memberScopes, !opts.flagDisabled)

	clientID := key + "-client"
	appBuilder := testutil.NewOAuthApplicationBuilder(t, tx).
		WithClientID(clientID).
		WithRedirectURIs([]string{"http://127.0.0.1/callback"})
	if opts.officialClient {
		appBuilder = appBuilder.AsOfficialClient()
	} else {
		appBuilder = appBuilder.WithSpaceID(f.spaceID)
	}
	if opts.confidential {
		appBuilder = appBuilder.WithConfidentialClientSecretDigest(auth.DigestOpaqueToken(oauthTokenTestClientSecret))
	}
	appID := appBuilder.Build()

	grantID := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(appID).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageWrite}).
		Build()

	return oauthTokenFixture{
		patMemberFixture: f,
		identifier:       model.SpaceIdentifier(key),
		appID:            appID,
		clientID:         clientID,
		grantID:          grantID,
	}
}

// buildCodeは許可に属する認可コードを作り、コードの値を返す
func (f oauthTokenFixture) buildCode(t *testing.T, tx *sql.Tx, modify func(b *testutil.OAuthAuthorizationCodeBuilder)) string {
	t.Helper()

	code := "code-" + string(f.grantID)
	b := testutil.NewOAuthAuthorizationCodeBuilder(t, tx).
		WithOAuthGrantID(f.grantID).
		WithSpaceID(f.spaceID).
		WithCodeDigest(auth.DigestOpaqueToken(code)).
		WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageWrite}).
		WithRedirectURI(oauthTokenTestRedirectURI).
		WithCodeChallenge(oauthAuthorizationTestCodeChallenge)
	if modify != nil {
		modify(b)
	}
	b.Build()
	return code
}

// buildRefreshTokenは許可に属するリフレッシュトークンを作り、トークンの値を返す
func (f oauthTokenFixture) buildRefreshToken(t *testing.T, tx *sql.Tx, value string, modify func(b *testutil.OAuthRefreshTokenBuilder)) string {
	t.Helper()

	token := string(auth.OAuthRefreshTokenPrefix) + value
	b := testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(f.grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest(auth.DigestOpaqueToken(token)).
		WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageWrite}).
		WithExpiresAt(time.Now().Add(24 * time.Hour))
	if modify != nil {
		modify(b)
	}
	b.Build()
	return token
}

// codeParamsは、fの認可コードcodeを交換する、検証を通るトークン要求のパラメーターを返す
func (f oauthTokenFixture) codeParams(code string) OAuthTokenParams {
	return OAuthTokenParams{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  oauthTokenTestRedirectURI,
		CodeVerifier: oauthTokenTestCodeVerifier,
		ClientID:     f.clientID,
	}
}

// refreshParamsは、fのリフレッシュトークンtokenで更新する、検証を通るトークン要求のパラメーターを返す
func (f oauthTokenFixture) refreshParams(token string) OAuthTokenParams {
	return OAuthTokenParams{
		GrantType:    "refresh_token",
		RefreshToken: token,
		ClientID:     f.clientID,
	}
}

func newCreateOAuthTokenUsecaseForTest(q *query.Queries) *CreateOAuthTokenUsecase {
	return NewCreateOAuthTokenUsecase(
		oauthAuthorizationTestConfig,
		repository.NewOAuthApplicationRepository(q),
		repository.NewOAuthGrantRepository(q),
		repository.NewOAuthAuthorizationCodeRepository(q),
		repository.NewOAuthAccessTokenRepository(q),
		repository.NewOAuthRefreshTokenRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewUserRepository(q),
		repository.NewFeatureFlagRepository(q),
	)
}

// assertOAuthTokenErrorは、errがcodeのOAuthTokenErrorであることを確かめる
func assertOAuthTokenError(t *testing.T, output *CreateOAuthTokenOutput, err error, code model.OAuthTokenErrorCode) {
	t.Helper()

	if output != nil {
		t.Errorf("output = %+v、期待値 = nil", output)
	}
	oe := model.AsOAuthTokenError(err)
	if oe == nil {
		t.Fatalf("エラー = %v、期待値 = %s のOAuthTokenError", err, code)
	}
	if oe.Code != code {
		t.Errorf("Code = %s、期待値 = %s", oe.Code, code)
	}
}

// assertIssuedTokensは、発行したトークンが許可に属し、期待したスコープで保存されていることを確かめる
func assertIssuedTokens(t *testing.T, q *query.Queries, f oauthTokenFixture, output *CreateOAuthTokenOutput, wantScopes []model.Scope) {
	t.Helper()

	ctx := context.Background()
	if !strings.HasPrefix(output.AccessToken, string(auth.OAuthAccessTokenPrefix)) {
		t.Errorf("AccessToken = %q、接頭辞 %q を期待", output.AccessToken, auth.OAuthAccessTokenPrefix)
	}
	if !strings.HasPrefix(output.RefreshToken, string(auth.OAuthRefreshTokenPrefix)) {
		t.Errorf("RefreshToken = %q、接頭辞 %q を期待", output.RefreshToken, auth.OAuthRefreshTokenPrefix)
	}
	if output.ExpiresIn != time.Hour {
		t.Errorf("ExpiresIn = %v、期待値 = %v", output.ExpiresIn, time.Hour)
	}
	if !slices.Equal(output.Scopes, wantScopes) {
		t.Errorf("Scopes = %v、期待値 = %v", output.Scopes, wantScopes)
	}

	accessToken, err := repository.NewOAuthAccessTokenRepository(q).FindByTokenDigest(ctx, auth.DigestOpaqueToken(output.AccessToken))
	if err != nil {
		t.Fatalf("アクセストークンの取得に失敗: %v", err)
	}
	if accessToken == nil || accessToken.OAuthGrantID != f.grantID || !accessToken.IsActive(time.Now()) {
		t.Fatalf("アクセストークン = %+v、許可 %v の有効なトークンを期待", accessToken, f.grantID)
	}
	if !slices.Equal(accessToken.Scopes, wantScopes) {
		t.Errorf("保存したアクセストークンのScopes = %v、期待値 = %v", accessToken.Scopes, wantScopes)
	}

	refreshToken, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(ctx, auth.DigestOpaqueToken(output.RefreshToken))
	if err != nil {
		t.Fatalf("リフレッシュトークンの取得に失敗: %v", err)
	}
	if refreshToken == nil || refreshToken.OAuthGrantID != f.grantID || !refreshToken.IsActive(time.Now()) {
		t.Fatalf("リフレッシュトークン = %+v、許可 %v の有効なトークンを期待", refreshToken, f.grantID)
	}
}

// assertGrantTokensRevokedは、許可に属するアクセストークンとリフレッシュトークンがすべて失効していることを確かめる
func assertGrantTokensRevoked(t *testing.T, tx *sql.Tx, grantID model.OAuthGrantID) {
	t.Helper()

	for _, table := range []string{"oauth_access_tokens", "oauth_refresh_tokens"} {
		var active int
		if err := tx.QueryRowContext(context.Background(),
			"SELECT count(*) FROM "+table+" WHERE oauth_grant_id = $1 AND revoked_at IS NULL", string(grantID),
		).Scan(&active); err != nil {
			t.Fatalf("%sの件数の取得に失敗: %v", table, err)
		}
		if active != 0 {
			t.Errorf("%sの失効していないトークン = %d件、期待値 = 0件", table, active)
		}
	}
}

func TestCreateOAuthTokenUsecase_Execute_AuthorizationCode(t *testing.T) {
	t.Parallel()

	t.Run("認可コードをアクセストークンとリフレッシュトークンに交換する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-code", oauthTokenFixtureOptions{})
		code := f.buildCode(t, tx, nil)

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), f.codeParams(code))
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})

		stored, err := repository.NewOAuthAuthorizationCodeRepository(q).FindByCodeDigest(context.Background(), auth.DigestOpaqueToken(code))
		if err != nil {
			t.Fatalf("認可コードの取得に失敗: %v", err)
		}
		if !stored.IsUsed() {
			t.Error("交換した認可コードのIsUsed() = false、期待値 = true")
		}
	})

	t.Run("confidentialクライアントはシークレットで認証する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-confidential", oauthTokenFixtureOptions{confidential: true})
		params := f.codeParams(f.buildCode(t, tx, nil))
		params.ClientSecret = oauthTokenTestClientSecret

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})
	})

	t.Run("公式クライアントは許可のスペースのコードを交換できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-official", oauthTokenFixtureOptions{officialClient: true})
		params := f.codeParams(f.buildCode(t, tx, nil))
		params.Resource = "https://example.com/api/v1/spaces/cot-official"

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})
	})

	t.Run("使用済みのコードの再提示は拒否し、許可のトークンをすべて失効させる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-replay", oauthTokenFixtureOptions{})
		code := f.buildCode(t, tx, nil)
		uc := newCreateOAuthTokenUsecaseForTest(q)

		// 1回目の交換で発行したトークンと、以前の同意で発行済みのトークンを、再提示で失効させる
		first, err := uc.Execute(context.Background(), f.codeParams(code))
		if err != nil {
			t.Fatalf("1回目のExecute()のエラー = %v", err)
		}
		f.buildRefreshToken(t, tx, "cot-replay-earlier", nil)

		output, err := uc.Execute(context.Background(), f.codeParams(code))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)
		assertGrantTokensRevoked(t, tx, f.grantID)

		principal, err := newAuthenticateAPITokenUC(tx).Execute(context.Background(), first.AccessToken)
		if err != nil {
			t.Fatalf("トークン照合のエラー = %v", err)
		}
		if principal != nil {
			t.Error("再提示の後も1回目に発行したアクセストークンでAPIを呼べる")
		}
	})

	t.Run("別のクライアントは使用済みのコードを送ってもトークンを失効させられない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-replay-other", oauthTokenFixtureOptions{})
		code := f.buildCode(t, tx, func(b *testutil.OAuthAuthorizationCodeBuilder) { b.WithUsedAt(time.Now()) })
		f.buildRefreshToken(t, tx, "cot-replay-other-active", nil)
		testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID("cot-replay-other-attacker").Build()
		params := f.codeParams(code)
		params.ClientID = "cot-replay-other-attacker"

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)

		refreshToken, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken("wkr_cot-replay-other-active"))
		if err != nil {
			t.Fatalf("リフレッシュトークンの取得に失敗: %v", err)
		}
		if refreshToken.IsRevoked() {
			t.Error("別のクライアントの要求で許可のリフレッシュトークンが失効した")
		}
	})

	// publicクライアントはclient_idだけで認証を通るため、コードだけを拾った第三者でも同じclient_idを送れる
	for _, replay := range []struct {
		name   string
		key    string
		modify func(p *OAuthTokenParams)
	}{
		{name: "code_verifierが合わない", key: "cot-replay-verifier", modify: func(p *OAuthTokenParams) { p.CodeVerifier = strings.Repeat("a", 43) }},
		{name: "redirect_uriが違う", key: "cot-replay-redirect", modify: func(p *OAuthTokenParams) { p.RedirectURI = "http://127.0.0.1:60000/callback" }},
	} {
		t.Run("publicクライアントでも"+replay.name+"使用済みのコードの再提示ではトークンを失効させない", func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			key := replay.key
			f := setupOAuthTokenFixture(t, tx, key, oauthTokenFixtureOptions{})
			code := f.buildCode(t, tx, func(b *testutil.OAuthAuthorizationCodeBuilder) { b.WithUsedAt(time.Now()) })
			token := f.buildRefreshToken(t, tx, key+"-active", nil)
			params := f.codeParams(code)
			replay.modify(&params)

			output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
			assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)

			refreshToken, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(token))
			if err != nil {
				t.Fatalf("リフレッシュトークンの取得に失敗: %v", err)
			}
			if refreshToken.IsRevoked() {
				t.Error("コードを発行した要求と合わない再提示で許可のリフレッシュトークンが失効した")
			}
		})
	}

	tests := []struct {
		name string
		opts oauthTokenFixtureOptions
		// modifyCodeは認可コードの既定値 (期限内・未使用) から変える点
		modifyCode func(b *testutil.OAuthAuthorizationCodeBuilder)
		// modifyParamsは検証を通るトークン要求から変える点
		modifyParams func(p *OAuthTokenParams)
		// setupは要求の前に済ませておく操作
		setup    func(t *testing.T, tx *sql.Tx, f oauthTokenFixture)
		wantCode model.OAuthTokenErrorCode
	}{
		{
			name:         "パラメーターの重複",
			modifyParams: func(p *OAuthTokenParams) { p.DuplicatedParams = []string{"code"} },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "grant_typeが無い",
			modifyParams: func(p *OAuthTokenParams) { p.GrantType = "" },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "対応していないgrant_type",
			modifyParams: func(p *OAuthTokenParams) { p.GrantType = "password" },
			wantCode:     model.OAuthTokenErrorUnsupportedGrantType,
		},
		{
			name:         "code_verifierが無い",
			modifyParams: func(p *OAuthTokenParams) { p.CodeVerifier = "" },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "code_verifierの形式が不正",
			modifyParams: func(p *OAuthTokenParams) { p.CodeVerifier = "short" },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "redirect_uriが無い",
			modifyParams: func(p *OAuthTokenParams) { p.RedirectURI = "" },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "client_idが無い",
			modifyParams: func(p *OAuthTokenParams) { p.ClientID = "" },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "登録されていないコード",
			modifyParams: func(p *OAuthTokenParams) { p.Code = "unknown" },
			wantCode:     model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:         "登録されていないクライアント",
			modifyParams: func(p *OAuthTokenParams) { p.ClientID = "unknown" },
			wantCode:     model.OAuthTokenErrorInvalidClient,
		},
		{
			name:         "publicクライアントがシークレットを送った",
			modifyParams: func(p *OAuthTokenParams) { p.ClientSecret = oauthTokenTestClientSecret },
			wantCode:     model.OAuthTokenErrorInvalidClient,
		},
		{
			name:     "confidentialクライアントがシークレットを送らない",
			opts:     oauthTokenFixtureOptions{confidential: true},
			wantCode: model.OAuthTokenErrorInvalidClient,
		},
		{
			name:         "confidentialクライアントのシークレットが違う",
			opts:         oauthTokenFixtureOptions{confidential: true},
			modifyParams: func(p *OAuthTokenParams) { p.ClientSecret = "wks_wrong" },
			wantCode:     model.OAuthTokenErrorInvalidClient,
		},
		{
			name: "同じスペースの別のアプリに発行されたコード",
			setup: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) {
				testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID(f.clientID + "-other").Build()
			},
			modifyParams: func(p *OAuthTokenParams) { p.ClientID += "-other" },
			wantCode:     model.OAuthTokenErrorInvalidGrant,
		},
		{
			name: "別のスペースのアプリはコードのスペースで使えない",
			setup: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) {
				other := testutil.NewSpaceBuilder(t, tx).WithIdentifier(f.clientID + "-other-space").Build()
				testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(other).WithClientID(f.clientID + "-other-space").Build()
			},
			modifyParams: func(p *OAuthTokenParams) { p.ClientID += "-other-space" },
			wantCode:     model.OAuthTokenErrorInvalidClient,
		},
		{
			name:       "期限切れのコード",
			modifyCode: func(b *testutil.OAuthAuthorizationCodeBuilder) { b.WithExpiresAt(time.Now().Add(-time.Second)) },
			wantCode:   model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:         "redirect_uriが認可要求と違う",
			modifyParams: func(p *OAuthTokenParams) { p.RedirectURI = "http://127.0.0.1:60000/callback" },
			wantCode:     model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:         "code_verifierがcode_challengeと合わない",
			modifyParams: func(p *OAuthTokenParams) { p.CodeVerifier = strings.Repeat("a", 43) },
			wantCode:     model.OAuthTokenErrorInvalidGrant,
		},
		{
			name: "許可が失効している",
			setup: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) {
				if _, err := tx.ExecContext(context.Background(), "UPDATE oauth_grants SET revoked_at = NOW() WHERE id = $1", string(f.grantID)); err != nil {
					t.Fatalf("許可の失効に失敗: %v", err)
				}
			},
			wantCode: model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:     "持ち主がoauth_grant:writeを持たない",
			opts:     oauthTokenFixtureOptions{memberScopes: []model.Scope{model.ScopePageWrite, model.ScopeOAuthGrantRead}},
			wantCode: model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:     "持ち主のフィーチャーフラグが無効",
			opts:     oauthTokenFixtureOptions{flagDisabled: true},
			wantCode: model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:         "resourceが別のスペースを指す",
			modifyParams: func(p *OAuthTokenParams) { p.Resource = "https://example.com/api/v1/spaces/other-space" },
			wantCode:     model.OAuthTokenErrorInvalidTarget,
		},
		{
			name:         "resourceの形式が不正",
			modifyParams: func(p *OAuthTokenParams) { p.Resource = "https://example.com/s/cot" },
			wantCode:     model.OAuthTokenErrorInvalidTarget,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			key := "cot-reject-" + string(rune('a'+i))
			f := setupOAuthTokenFixture(t, tx, key, tt.opts)
			code := f.buildCode(t, tx, tt.modifyCode)
			if tt.setup != nil {
				tt.setup(t, tx, f)
			}
			params := f.codeParams(code)
			if tt.modifyParams != nil {
				tt.modifyParams(&params)
			}

			output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
			assertOAuthTokenError(t, output, err, tt.wantCode)

			// 拒否した要求ではトークンを発行しない
			var issued int
			if err := tx.QueryRowContext(context.Background(),
				"SELECT count(*) FROM oauth_access_tokens WHERE oauth_grant_id = $1", string(f.grantID),
			).Scan(&issued); err != nil {
				t.Fatalf("アクセストークンの件数の取得に失敗: %v", err)
			}
			if issued != 0 {
				t.Errorf("発行したアクセストークン = %d件、期待値 = 0件", issued)
			}
		})
	}
}

func TestCreateOAuthTokenUsecase_Execute_RefreshToken(t *testing.T) {
	t.Parallel()

	t.Run("リフレッシュトークンをローテーションし、新しいアクセストークンを発行する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-refresh", oauthTokenFixtureOptions{})
		token := f.buildRefreshToken(t, tx, "cot-refresh", nil)
		uc := newCreateOAuthTokenUsecaseForTest(q)

		output, err := uc.Execute(context.Background(), f.refreshParams(token))
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})

		previous, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(token))
		if err != nil {
			t.Fatalf("リフレッシュトークンの取得に失敗: %v", err)
		}
		if !previous.IsUsed() {
			t.Error("元のリフレッシュトークンのIsUsed() = false、期待値 = true")
		}

		// ローテーションで得たトークンで続けて更新できる
		if _, err := uc.Execute(context.Background(), f.refreshParams(output.RefreshToken)); err != nil {
			t.Fatalf("ローテーションで得たトークンでのExecute()のエラー = %v", err)
		}
	})

	t.Run("スコープを狭めた要求では、アクセストークンだけを狭める", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-refresh-narrow", oauthTokenFixtureOptions{})
		params := f.refreshParams(f.buildRefreshToken(t, tx, "cot-refresh-narrow", nil))
		params.Scope = "topic:read"

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead})

		rotated, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(output.RefreshToken))
		if err != nil {
			t.Fatalf("リフレッシュトークンの取得に失敗: %v", err)
		}
		if want := []model.Scope{model.ScopeTopicRead, model.ScopePageWrite}; !slices.Equal(rotated.Scopes, want) {
			t.Errorf("ローテーションしたリフレッシュトークンのScopes = %v、期待値 = %v", rotated.Scopes, want)
		}
	})

	t.Run("使用済みのリフレッシュトークンの再提示は拒否し、許可のトークンをすべて失効させる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-refresh-replay", oauthTokenFixtureOptions{})
		token := f.buildRefreshToken(t, tx, "cot-refresh-replay", nil)
		uc := newCreateOAuthTokenUsecaseForTest(q)

		rotated, err := uc.Execute(context.Background(), f.refreshParams(token))
		if err != nil {
			t.Fatalf("1回目のExecute()のエラー = %v", err)
		}

		output, err := uc.Execute(context.Background(), f.refreshParams(token))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)
		assertGrantTokensRevoked(t, tx, f.grantID)

		// ローテーションで正規のクライアントが得たトークンも失効している
		output, err = uc.Execute(context.Background(), f.refreshParams(rotated.RefreshToken))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)
	})

	t.Run("別のクライアントは使用済みのリフレッシュトークンを送ってもトークンを失効させられない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthTokenFixture(t, tx, "cot-refresh-replay-other", oauthTokenFixtureOptions{})
		used := f.buildRefreshToken(t, tx, "cot-refresh-replay-other-used", func(b *testutil.OAuthRefreshTokenBuilder) { b.WithUsedAt(time.Now()) })
		active := f.buildRefreshToken(t, tx, "cot-refresh-replay-other-active", nil)
		testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID("cot-refresh-replay-other-attacker").Build()
		params := f.refreshParams(used)
		params.ClientID = "cot-refresh-replay-other-attacker"

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)

		refreshToken, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(active))
		if err != nil {
			t.Fatalf("リフレッシュトークンの取得に失敗: %v", err)
		}
		if refreshToken.IsRevoked() {
			t.Error("別のクライアントの要求で許可のリフレッシュトークンが失効した")
		}
	})

	tests := []struct {
		name string
		opts oauthTokenFixtureOptions
		// modifyTokenはリフレッシュトークンの既定値 (期限内・未使用・失効していない) から変える点
		modifyToken  func(b *testutil.OAuthRefreshTokenBuilder)
		modifyParams func(p *OAuthTokenParams)
		// setupは要求の前に済ませておく操作
		setup    func(t *testing.T, tx *sql.Tx, f oauthTokenFixture)
		wantCode model.OAuthTokenErrorCode
	}{
		{
			name:         "refresh_tokenが無い",
			modifyParams: func(p *OAuthTokenParams) { p.RefreshToken = "" },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "登録されていないリフレッシュトークン",
			modifyParams: func(p *OAuthTokenParams) { p.RefreshToken = "wkr_unknown" },
			wantCode:     model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:         "登録されていないクライアント",
			modifyParams: func(p *OAuthTokenParams) { p.ClientID = "unknown" },
			wantCode:     model.OAuthTokenErrorInvalidClient,
		},
		{
			name:        "期限切れのリフレッシュトークン",
			modifyToken: func(b *testutil.OAuthRefreshTokenBuilder) { b.WithExpiresAt(time.Now().Add(-time.Second)) },
			wantCode:    model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:        "失効したリフレッシュトークン",
			modifyToken: func(b *testutil.OAuthRefreshTokenBuilder) { b.WithRevokedAt(time.Now()) },
			wantCode:    model.OAuthTokenErrorInvalidGrant,
		},
		{
			name: "同じスペースの別のアプリに発行されたトークン",
			setup: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) {
				testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID(f.clientID + "-other").Build()
			},
			modifyParams: func(p *OAuthTokenParams) { p.ClientID += "-other" },
			wantCode:     model.OAuthTokenErrorInvalidGrant,
		},
		{
			name: "許可が失効している",
			setup: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) {
				if _, err := tx.ExecContext(context.Background(), "UPDATE oauth_grants SET revoked_at = NOW() WHERE id = $1", string(f.grantID)); err != nil {
					t.Fatalf("許可の失効に失敗: %v", err)
				}
			},
			wantCode: model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:         "元の範囲を超えるスコープ",
			modifyParams: func(p *OAuthTokenParams) { p.Scope = "page:read page:write topic:read" },
			wantCode:     model.OAuthTokenErrorInvalidScope,
		},
		{
			name:         "トークンに付与できないスコープ",
			modifyParams: func(p *OAuthTokenParams) { p.Scope = "space:admin" },
			wantCode:     model.OAuthTokenErrorInvalidScope,
		},
		{
			name:     "持ち主がoauth_grant:writeを持たない",
			opts:     oauthTokenFixtureOptions{memberScopes: []model.Scope{model.ScopePageWrite}},
			wantCode: model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:         "resourceが別のスペースを指す",
			modifyParams: func(p *OAuthTokenParams) { p.Resource = "https://example.com/api/v1/spaces/other-space" },
			wantCode:     model.OAuthTokenErrorInvalidTarget,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			key := "cot-refresh-reject-" + string(rune('a'+i))
			f := setupOAuthTokenFixture(t, tx, key, tt.opts)
			token := f.buildRefreshToken(t, tx, key, tt.modifyToken)
			if tt.setup != nil {
				tt.setup(t, tx, f)
			}
			params := f.refreshParams(token)
			if tt.modifyParams != nil {
				tt.modifyParams(&params)
			}

			output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
			assertOAuthTokenError(t, output, err, tt.wantCode)

			// 拒否した要求ではリフレッシュトークンをローテーションしない
			stored, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(token))
			if err != nil {
				t.Fatalf("リフレッシュトークンの取得に失敗: %v", err)
			}
			if stored.IsUsed() {
				t.Error("拒否した要求でリフレッシュトークンが使用済みになった")
			}
		})
	}
}

func TestVerifyPKCES256(t *testing.T) {
	t.Parallel()

	if !verifyPKCES256(oauthTokenTestCodeVerifier, oauthAuthorizationTestCodeChallenge) {
		t.Error("RFC 7636 Appendix Bの値で一致しない")
	}
	if verifyPKCES256(oauthTokenTestCodeVerifier+"x", oauthAuthorizationTestCodeChallenge) {
		t.Error("異なるcode_verifierで一致した")
	}
}
