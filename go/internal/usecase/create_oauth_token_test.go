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

// oauthTokenFixtureOptionsは、トークン要求のフィクスチャの既定値 (フラグ有効・編集者のロール
// (`oauth_grant:write` を持つ) のメンバー・スペースのpublicクライアント・失効していない許可) から変える点
type oauthTokenFixtureOptions struct {
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

	f := setupPATMember(t, tx, key, model.SpaceRoleEditor, !opts.flagDisabled)

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
		testutil.GetTestDB(),
		oauthAuthorizationTestConfig,
		repository.NewOAuthApplicationRepository(q),
		repository.NewOAuthGrantRepository(q),
		repository.NewOAuthAuthorizationCodeRepository(q),
		repository.NewOAuthRefreshTokenRepository(q),
		repository.NewOAuthDeviceAuthorizationRepository(q),
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

	// 発行と同じトランザクションで、許可の最終利用日時を発行の時刻にする
	grant, err := repository.NewOAuthGrantRepository(q).FindByID(ctx, f.grantID, f.spaceID)
	if err != nil {
		t.Fatalf("許可の取得に失敗: %v", err)
	}
	if grant.LastUsedAt == nil || !grant.LastUsedAt.Equal(accessToken.CreatedAt) {
		t.Errorf("許可のLastUsedAt = %v、期待値 = %v (発行の時刻)", grant.LastUsedAt, accessToken.CreatedAt)
	}
}

// assertGrantRevokedWithTokensは、許可と、許可に属するアクセストークン・リフレッシュトークンが
// すべて失効していることを確かめる
func assertGrantRevokedWithTokens(t *testing.T, tx *sql.Tx, grantID model.OAuthGrantID) {
	t.Helper()

	var grantRevoked bool
	if err := tx.QueryRowContext(context.Background(),
		"SELECT revoked_at IS NOT NULL FROM oauth_grants WHERE id = $1", string(grantID),
	).Scan(&grantRevoked); err != nil {
		t.Fatalf("許可の取得に失敗: %v", err)
	}
	if !grantRevoked {
		t.Error("許可が失効していない")
	}
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

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-code", oauthTokenFixtureOptions{})
		code := f.buildCodeDB(t, nil)

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

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-confidential", oauthTokenFixtureOptions{confidential: true})
		params := f.codeParams(f.buildCodeDB(t, nil))
		params.ClientSecret = oauthTokenTestClientSecret

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})
	})

	t.Run("公式クライアントは許可のスペースのコードを交換できる", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-official", oauthTokenFixtureOptions{officialClient: true})
		params := f.codeParams(f.buildCodeDB(t, nil))
		params.Resource = "https://example.com/api/v1/spaces/cot-official"

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})
	})

	// 認可要求は大文字と小文字だけが違う `resource` でも通るため、交換でも通す
	t.Run("resourceの識別子が大文字と小文字だけ違ってもコードを交換できる", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-code-case", oauthTokenFixtureOptions{})
		params := f.codeParams(f.buildCodeDB(t, nil))
		params.Resource = "https://example.com/api/v1/spaces/COT-Code-Case"

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})
	})

	t.Run("使用済みのコードの再提示は拒否し、許可と許可のトークンをすべて失効させる", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-replay", oauthTokenFixtureOptions{})
		code := f.buildCodeDB(t, nil)
		uc := newCreateOAuthTokenUsecaseForTest(q)

		// 1回目の交換で発行したトークンと、以前の同意で発行済みのトークンを、再提示で失効させる
		first, err := uc.Execute(context.Background(), f.codeParams(code))
		if err != nil {
			t.Fatalf("1回目のExecute()のエラー = %v", err)
		}
		f.buildRefreshTokenDB(t, "cot-replay-earlier", nil)

		output, err := uc.Execute(context.Background(), f.codeParams(code))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)
		testutil.CommitFixture(t, func(tx *sql.Tx) { assertGrantRevokedWithTokens(t, tx, f.grantID) })

		principal, err := newAuthenticateAPITokenUCForOAuthTest(q).Execute(context.Background(), first.AccessToken)
		if err != nil {
			t.Fatalf("トークン照合のエラー = %v", err)
		}
		if principal != nil {
			t.Error("再提示の後も1回目に発行したアクセストークンでAPIを呼べる")
		}
	})

	t.Run("別のクライアントは使用済みのコードを送ってもトークンを失効させられない", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-replay-other", oauthTokenFixtureOptions{})
		code := f.buildCodeDB(t, func(b *testutil.OAuthAuthorizationCodeBuilder) { b.WithUsedAt(time.Now()) })
		f.buildRefreshTokenDB(t, "cot-replay-other-active", nil)
		testutil.CommitFixture(t, func(tx *sql.Tx) {
			testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID("cot-replay-other-attacker").Build()
		})
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

			db := testutil.GetTestDB()
			q := query.New(db)
			key := replay.key
			f := setupOAuthTokenFixtureDB(t, key, oauthTokenFixtureOptions{})
			code := f.buildCodeDB(t, func(b *testutil.OAuthAuthorizationCodeBuilder) { b.WithUsedAt(time.Now()) })
			token := f.buildRefreshTokenDB(t, key+"-active", nil)
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
			// 許可のスコープは作成後に変わらないが、不変条件の確認として範囲を超えるコードを拒否する
			name: "コードのスコープが今の許可の範囲を超える",
			setup: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) {
				if _, err := tx.ExecContext(context.Background(), "UPDATE oauth_grants SET scopes = '{topic:read}' WHERE id = $1", string(f.grantID)); err != nil {
					t.Fatalf("許可のスコープの更新に失敗: %v", err)
				}
			},
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

			db := testutil.GetTestDB()
			q := query.New(db)
			key := "cot-reject-" + string(rune('a'+i))
			f := setupOAuthTokenFixtureDB(t, key, tt.opts)
			code := f.buildCodeDB(t, tt.modifyCode)
			if tt.setup != nil {
				testutil.CommitFixture(t, func(tx *sql.Tx) { tt.setup(t, tx, f) })
			}
			params := f.codeParams(code)
			if tt.modifyParams != nil {
				tt.modifyParams(&params)
			}

			output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
			assertOAuthTokenError(t, output, err, tt.wantCode)

			// 拒否した要求ではトークンを発行しない
			var issued int
			if err := db.QueryRowContext(context.Background(),
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

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-refresh", oauthTokenFixtureOptions{})
		token := f.buildRefreshTokenDB(t, "cot-refresh", nil)
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

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-refresh-narrow", oauthTokenFixtureOptions{})
		params := f.refreshParams(f.buildRefreshTokenDB(t, "cot-refresh-narrow", nil))
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

	t.Run("resourceの識別子が大文字と小文字だけ違っても更新できる", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-refresh-case", oauthTokenFixtureOptions{})
		params := f.refreshParams(f.buildRefreshTokenDB(t, "cot-refresh-case", nil))
		params.Resource = "https://example.com/api/v1/spaces/COT-Refresh-Case"

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})
	})

	t.Run("使用済みのリフレッシュトークンの再提示は拒否し、許可と許可のトークンをすべて失効させる", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-refresh-replay", oauthTokenFixtureOptions{})
		token := f.buildRefreshTokenDB(t, "cot-refresh-replay", nil)
		uc := newCreateOAuthTokenUsecaseForTest(q)
		// 同じアプリ・メンバーの別の許可 (別の端末のログイン) は、再提示の影響を受けない
		another := f
		testutil.CommitFixture(t, func(tx *sql.Tx) {
			another.grantID = testutil.NewOAuthGrantBuilder(t, tx).WithOAuthApplicationID(f.appID).WithSpaceID(f.spaceID).
				WithSpaceMemberID(f.spaceMemberID).WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageWrite}).Build()
		})
		anotherToken := another.buildRefreshTokenDB(t, "cot-refresh-replay-another", nil)

		rotated, err := uc.Execute(context.Background(), f.refreshParams(token))
		if err != nil {
			t.Fatalf("1回目のExecute()のエラー = %v", err)
		}

		output, err := uc.Execute(context.Background(), f.refreshParams(token))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)
		testutil.CommitFixture(t, func(tx *sql.Tx) { assertGrantRevokedWithTokens(t, tx, f.grantID) })

		// ローテーションで正規のクライアントが得たトークンも失効している
		output, err = uc.Execute(context.Background(), f.refreshParams(rotated.RefreshToken))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)

		if _, err := uc.Execute(context.Background(), another.refreshParams(anotherToken)); err != nil {
			t.Errorf("別の許可のリフレッシュトークンでのExecute()のエラー = %v", err)
		}
	})

	t.Run("別のクライアントは使用済みのリフレッシュトークンを送ってもトークンを失効させられない", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-refresh-replay-other", oauthTokenFixtureOptions{})
		used := f.buildRefreshTokenDB(t, "cot-refresh-replay-other-used", func(b *testutil.OAuthRefreshTokenBuilder) { b.WithUsedAt(time.Now()) })
		active := f.buildRefreshTokenDB(t, "cot-refresh-replay-other-active", nil)
		testutil.CommitFixture(t, func(tx *sql.Tx) {
			testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID("cot-refresh-replay-other-attacker").Build()
		})
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
			name:         "resourceが別のスペースを指す",
			modifyParams: func(p *OAuthTokenParams) { p.Resource = "https://example.com/api/v1/spaces/other-space" },
			wantCode:     model.OAuthTokenErrorInvalidTarget,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.GetTestDB()
			q := query.New(db)
			key := "cot-refresh-reject-" + string(rune('a'+i))
			f := setupOAuthTokenFixtureDB(t, key, tt.opts)
			token := f.buildRefreshTokenDB(t, key, tt.modifyToken)
			if tt.setup != nil {
				testutil.CommitFixture(t, func(tx *sql.Tx) { tt.setup(t, tx, f) })
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

func setupOAuthTokenFixtureDB(t *testing.T, key string, opts oauthTokenFixtureOptions) oauthTokenFixture {
	t.Helper()
	var f oauthTokenFixture
	testutil.CommitFixture(t, func(tx *sql.Tx) { f = setupOAuthTokenFixture(t, tx, key, opts) })
	return f
}

func (f oauthTokenFixture) buildCodeDB(t *testing.T, modify func(*testutil.OAuthAuthorizationCodeBuilder)) string {
	t.Helper()
	var code string
	testutil.CommitFixture(t, func(tx *sql.Tx) { code = f.buildCode(t, tx, modify) })
	return code
}

func (f oauthTokenFixture) buildRefreshTokenDB(t *testing.T, value string, modify func(*testutil.OAuthRefreshTokenBuilder)) string {
	t.Helper()
	var token string
	testutil.CommitFixture(t, func(tx *sql.Tx) { token = f.buildRefreshToken(t, tx, value, modify) })
	return token
}

func newAuthenticateAPITokenUCForOAuthTest(q *query.Queries) *AuthenticateAPITokenUsecase {
	return NewAuthenticateAPITokenUsecase(repository.NewPersonalAccessTokenRepository(q), repository.NewOAuthAccessTokenRepository(q), repository.NewOAuthGrantRepository(q), repository.NewSpaceRepository(q), repository.NewSpaceMemberRepository(q), repository.NewUserRepository(q), repository.NewFeatureFlagRepository(q))
}

// buildDeviceAuthorizationDBは、許可済みで許可に結び付いたデバイス認可の要求を作り、デバイスコードの値を返す
func (f oauthTokenFixture) buildDeviceAuthorizationDB(t *testing.T, modify func(b *testutil.OAuthDeviceAuthorizationBuilder)) string {
	t.Helper()

	deviceCode := "device-" + string(f.grantID)
	testutil.CommitFixture(t, func(tx *sql.Tx) {
		b := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
			WithOAuthApplicationID(f.appID).
			WithSpaceID(f.spaceID).
			WithOAuthGrantID(f.grantID).
			WithStatus(model.OAuthDeviceAuthorizationStatusApproved).
			WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageWrite}).
			WithDeviceCodeDigest(auth.DigestOpaqueToken(deviceCode)).
			WithUserCodeDigest("user-code-" + string(f.grantID))
		if modify != nil {
			modify(b)
		}
		b.Build()
	})
	return deviceCode
}

// pendingDeviceAuthorizationは、デバイス認可の要求を許可待ち (許可を持たない) にする
func pendingDeviceAuthorization(b *testutil.OAuthDeviceAuthorizationBuilder) {
	b.WithStatus(model.OAuthDeviceAuthorizationStatusPending)
	b.WithoutOAuthGrantID()
}

// deviceCodeParamsは、fのデバイスコードdeviceCodeを交換する、検証を通るトークン要求のパラメーターを返す
func (f oauthTokenFixture) deviceCodeParams(deviceCode string) OAuthTokenParams {
	return OAuthTokenParams{
		GrantType:  "urn:ietf:params:oauth:grant-type:device_code",
		DeviceCode: deviceCode,
		ClientID:   f.clientID,
	}
}

// findDeviceAuthorizationは、デバイスコードの値で要求を取得する
func findDeviceAuthorization(t *testing.T, q *query.Queries, deviceCode string) *model.OAuthDeviceAuthorization {
	t.Helper()

	d, err := repository.NewOAuthDeviceAuthorizationRepository(q).FindByDeviceCodeDigest(context.Background(), auth.DigestOpaqueToken(deviceCode))
	if err != nil {
		t.Fatalf("デバイス認可の要求の取得に失敗: %v", err)
	}
	return d
}

func TestCreateOAuthTokenUsecase_Execute_DeviceCode(t *testing.T) {
	t.Parallel()

	t.Run("許可済みのデバイスコードをアクセストークンとリフレッシュトークンに交換する", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-device", oauthTokenFixtureOptions{officialClient: true})
		deviceCode := f.buildDeviceAuthorizationDB(t, nil)
		params := f.deviceCodeParams(deviceCode)
		params.Resource = "https://example.com/api/v1/spaces/cot-device"
		// 要求の `scope` は使わず、同意で決まったスコープのトークンを発行する
		params.Scope = "page:read"

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertIssuedTokens(t, q, f, output, []model.Scope{model.ScopeTopicRead, model.ScopePageWrite})

		if d := findDeviceAuthorization(t, q, deviceCode); d.Status != model.OAuthDeviceAuthorizationStatusExchanged {
			t.Errorf("Status = %v、期待値 = 交換済み", d.Status)
		}
	})

	t.Run("許可待ちならauthorization_pendingを返し、ポーリングを記録する", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-device-pending", oauthTokenFixtureOptions{})
		deviceCode := f.buildDeviceAuthorizationDB(t, pendingDeviceAuthorization)

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), f.deviceCodeParams(deviceCode))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorAuthorizationPending)

		if d := findDeviceAuthorization(t, q, deviceCode); d.LastPolledAt == nil || d.Interval != 5*time.Second {
			t.Errorf("要求 = %+v、ポーリングを記録して間隔を変えないことを期待", d)
		}
	})

	t.Run("許可待ちのまま間隔より早いとslow_downを返し、間隔を5秒延ばす", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-device-slow", oauthTokenFixtureOptions{})
		deviceCode := f.buildDeviceAuthorizationDB(t, func(b *testutil.OAuthDeviceAuthorizationBuilder) {
			pendingDeviceAuthorization(b)
			b.WithLastPolledAt(time.Now().Add(-2 * time.Second))
		})

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), f.deviceCodeParams(deviceCode))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorSlowDown)

		if d := findDeviceAuthorization(t, q, deviceCode); d.Interval != 10*time.Second {
			t.Errorf("Interval = %v、期待値 = 10s", d.Interval)
		}
	})

	t.Run("交換済みのデバイスコードの再提示は拒否し、許可と許可のトークンをすべて失効させる", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-device-replay", oauthTokenFixtureOptions{})
		deviceCode := f.buildDeviceAuthorizationDB(t, nil)
		uc := newCreateOAuthTokenUsecaseForTest(q)

		first, err := uc.Execute(context.Background(), f.deviceCodeParams(deviceCode))
		if err != nil {
			t.Fatalf("1回目のExecute()のエラー = %v", err)
		}

		output, err := uc.Execute(context.Background(), f.deviceCodeParams(deviceCode))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)
		testutil.CommitFixture(t, func(tx *sql.Tx) { assertGrantRevokedWithTokens(t, tx, f.grantID) })

		principal, err := newAuthenticateAPITokenUCForOAuthTest(q).Execute(context.Background(), first.AccessToken)
		if err != nil {
			t.Fatalf("トークン照合のエラー = %v", err)
		}
		if principal != nil {
			t.Error("再提示の後も1回目に発行したアクセストークンでAPIを呼べる")
		}
	})

	t.Run("交換済みのデバイスコードは期限が切れた後の再提示でも拒否し、許可と許可のトークンをすべて失効させる", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthTokenFixtureDB(t, "cot-device-replay-expired", oauthTokenFixtureOptions{})
		// 交換済みの判定を期限切れより先に行い、`expired_token` で失効を逃さないことを確かめる
		deviceCode := f.buildDeviceAuthorizationDB(t, func(b *testutil.OAuthDeviceAuthorizationBuilder) {
			b.WithStatus(model.OAuthDeviceAuthorizationStatusExchanged).WithExpiresAt(time.Now().Add(-time.Second))
		})
		testutil.CommitFixture(t, func(tx *sql.Tx) {
			testutil.NewOAuthAccessTokenBuilder(t, tx).WithOAuthGrantID(f.grantID).WithSpaceID(f.spaceID).
				WithTokenDigest("replay-expired-access-" + string(f.grantID)).Build()
			testutil.NewOAuthRefreshTokenBuilder(t, tx).WithOAuthGrantID(f.grantID).WithSpaceID(f.spaceID).
				WithTokenDigest("replay-expired-refresh-" + string(f.grantID)).Build()
		})

		output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), f.deviceCodeParams(deviceCode))
		assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)
		testutil.CommitFixture(t, func(tx *sql.Tx) { assertGrantRevokedWithTokens(t, tx, f.grantID) })
	})

	tests := []struct {
		name         string
		opts         oauthTokenFixtureOptions
		modifyDevice func(b *testutil.OAuthDeviceAuthorizationBuilder)
		setup        func(t *testing.T, tx *sql.Tx, f oauthTokenFixture)
		modifyParams func(p *OAuthTokenParams)
		wantCode     model.OAuthTokenErrorCode
	}{
		{
			name:         "device_codeが無い",
			modifyParams: func(p *OAuthTokenParams) { p.DeviceCode = "" },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "client_idが無い",
			modifyParams: func(p *OAuthTokenParams) { p.ClientID = "" },
			wantCode:     model.OAuthTokenErrorInvalidRequest,
		},
		{
			name:         "登録されていないデバイスコード",
			modifyParams: func(p *OAuthTokenParams) { p.DeviceCode = "unknown" },
			wantCode:     model.OAuthTokenErrorInvalidGrant,
		},
		{
			name:         "登録されていないクライアント",
			modifyParams: func(p *OAuthTokenParams) { p.ClientID = "unknown" },
			wantCode:     model.OAuthTokenErrorInvalidClient,
		},
		{
			name: "publicクライアントがシークレットを送った",
			modifyParams: func(p *OAuthTokenParams) {
				p.ClientSecret = "wks_unexpected"
				p.ClientSecretProvided = true
			},
			wantCode: model.OAuthTokenErrorInvalidClient,
		},
		{
			name: "同じスペースの別のアプリが要求したデバイスコード",
			setup: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) {
				testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID(f.clientID + "-other").Build()
			},
			modifyParams: func(p *OAuthTokenParams) { p.ClientID += "-other" },
			wantCode:     model.OAuthTokenErrorInvalidGrant,
		},
		{
			name: "利用者が拒否した",
			modifyDevice: func(b *testutil.OAuthDeviceAuthorizationBuilder) {
				b.WithStatus(model.OAuthDeviceAuthorizationStatusDenied).WithoutOAuthGrantID()
			},
			wantCode: model.OAuthTokenErrorAccessDenied,
		},
		{
			// 拒否の判定を期限切れより先に行い、利用者の選択をそのまま伝える
			name: "拒否した後に期限が切れた",
			modifyDevice: func(b *testutil.OAuthDeviceAuthorizationBuilder) {
				b.WithStatus(model.OAuthDeviceAuthorizationStatusDenied).WithoutOAuthGrantID().
					WithExpiresAt(time.Now().Add(-time.Second))
			},
			wantCode: model.OAuthTokenErrorAccessDenied,
		},
		{
			name: "許可待ちのまま期限が切れた",
			modifyDevice: func(b *testutil.OAuthDeviceAuthorizationBuilder) {
				pendingDeviceAuthorization(b)
				b.WithExpiresAt(time.Now().Add(-time.Second))
			},
			wantCode: model.OAuthTokenErrorExpiredToken,
		},
		{
			name:         "許可済みのまま期限が切れた",
			modifyDevice: func(b *testutil.OAuthDeviceAuthorizationBuilder) { b.WithExpiresAt(time.Now().Add(-time.Second)) },
			wantCode:     model.OAuthTokenErrorExpiredToken,
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
			// 許可のスコープは作成後に変わらないが、不変条件の確認として範囲を超える要求を拒否する
			name: "要求のスコープが今の許可の範囲を超える",
			setup: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) {
				if _, err := tx.ExecContext(context.Background(), "UPDATE oauth_grants SET scopes = '{topic:read}' WHERE id = $1", string(f.grantID)); err != nil {
					t.Fatalf("許可のスコープの更新に失敗: %v", err)
				}
			},
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
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.GetTestDB()
			q := query.New(db)
			key := "cot-device-reject-" + string(rune('a'+i))
			f := setupOAuthTokenFixtureDB(t, key, tt.opts)
			deviceCode := f.buildDeviceAuthorizationDB(t, tt.modifyDevice)
			if tt.setup != nil {
				testutil.CommitFixture(t, func(tx *sql.Tx) { tt.setup(t, tx, f) })
			}
			params := f.deviceCodeParams(deviceCode)
			if tt.modifyParams != nil {
				tt.modifyParams(&params)
			}

			output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
			assertOAuthTokenError(t, output, err, tt.wantCode)

			// 拒否した要求ではトークンを発行せず、要求も交換済みにしない
			var issued int
			if err := db.QueryRowContext(context.Background(),
				"SELECT count(*) FROM oauth_access_tokens WHERE oauth_grant_id = $1", string(f.grantID),
			).Scan(&issued); err != nil {
				t.Fatalf("アクセストークンの件数の取得に失敗: %v", err)
			}
			if issued != 0 {
				t.Errorf("発行したアクセストークン = %d件、期待値 = 0件", issued)
			}
			if d := findDeviceAuthorization(t, q, deviceCode); d.Status == model.OAuthDeviceAuthorizationStatusExchanged {
				t.Error("拒否した要求でデバイス認可の要求が交換済みになった")
			}
		})
	}
}

// 要求・許可のロック待機後に期限切れを拒否し、要求と既存のトークンを変更しないことを確かめる。
func TestCreateOAuthTokenUsecase_DeviceCodeExpiryDuringLock(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct{ name, table, queryName string }{
		{name: "許可のロック待機", table: "oauth_grants", queryName: "LockOAuthGrantByID"},
		{name: "要求のロック待機", table: "oauth_device_authorizations", queryName: "LockOAuthDeviceAuthorizationByID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := testutil.GetTestDB()
			q := query.New(db)
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			f := setupOAuthTokenFixtureDB(t, "cot-expiry-lock-"+tt.table, oauthTokenFixtureOptions{})
			deviceCode := f.buildDeviceAuthorizationDB(t, nil)
			device := findDeviceAuthorization(t, q, deviceCode)
			accessDigest := "expiry-access-" + string(f.grantID)
			refreshDigest := "expiry-refresh-" + string(f.grantID)
			testutil.CommitFixture(t, func(tx *sql.Tx) {
				testutil.NewOAuthAccessTokenBuilder(t, tx).WithOAuthGrantID(f.grantID).WithSpaceID(f.spaceID).WithTokenDigest(accessDigest).Build()
				testutil.NewOAuthRefreshTokenBuilder(t, tx).WithOAuthGrantID(f.grantID).WithSpaceID(f.spaceID).WithTokenDigest(refreshDigest).Build()
			})
			expiresAt := time.Now().Add(5 * time.Second)
			if _, err := db.ExecContext(ctx, "UPDATE oauth_device_authorizations SET expires_at=$1 WHERE id=$2 AND space_id=$3", expiresAt, string(device.ID), string(f.spaceID)); err != nil {
				t.Fatal(err)
			}
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback() }()
			var blockingPID int
			if err := tx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockingPID); err != nil {
				t.Fatal(err)
			}
			lockID, lock := string(device.ID), "FOR UPDATE"
			if tt.table == "oauth_grants" {
				lockID, lock = string(f.grantID), "FOR NO KEY UPDATE"
			}
			// 表名とロック句は固定したテストケースだけから取り、ユーザー入力は使わない。
			if _, err := tx.ExecContext(ctx, "SELECT id FROM "+tt.table+" WHERE id=$1 AND space_id=$2 "+lock, lockID, string(f.spaceID)); err != nil {
				t.Fatal(err)
			}
			type result struct {
				output *CreateOAuthTokenOutput
				err    error
			}
			done := make(chan result, 1)
			go func() {
				output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(ctx, f.deviceCodeParams(deviceCode))
				done <- result{output, err}
			}()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				var blocked bool
				if err := db.QueryRowContext(ctx, `SELECT EXISTS (
     SELECT 1 FROM pg_stat_activity WHERE datname=current_database()
      AND $1=ANY(pg_blocking_pids(pid)) AND query LIKE '%' || $2 || '%'
    )`, blockingPID, tt.queryName).Scan(&blocked); err != nil {
					t.Fatal(err)
				}
				if blocked {
					break
				}
				select {
				case r := <-done:
					t.Fatalf("ロック待機前に処理が終了: %v", r.err)
				case <-ctx.Done():
					t.Fatalf("ロック待機を確認できない: %v", ctx.Err())
				case <-ticker.C:
				}
			}
			timer := time.NewTimer(time.Until(expiresAt) + 10*time.Millisecond)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-ctx.Done():
				t.Fatalf("期限切れまで待機できない: %v", ctx.Err())
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			select {
			case r := <-done:
				assertOAuthTokenError(t, r.output, r.err, model.OAuthTokenErrorExpiredToken)
			case <-ctx.Done():
				t.Fatalf("交換処理が完了しない: %v", ctx.Err())
			}
			stored := findDeviceAuthorization(t, q, deviceCode)
			if stored.Status != model.OAuthDeviceAuthorizationStatusApproved || !stored.UpdatedAt.Equal(device.UpdatedAt) {
				t.Errorf("期限切れの要求が変更された: %+v", stored)
			}
			for _, token := range []struct{ table, digest string }{
				{table: "oauth_access_tokens", digest: accessDigest},
				{table: "oauth_refresh_tokens", digest: refreshDigest},
			} {
				var count, unchanged int
				// 表名は固定値だけを使い、発行と既存トークンの失効の両方を検出する。
				if err := db.QueryRowContext(ctx, "SELECT count(*), count(*) FILTER (WHERE token_digest=$1 AND revoked_at IS NULL) FROM "+token.table+" WHERE oauth_grant_id=$2 AND space_id=$3", token.digest, string(f.grantID), string(f.spaceID)).Scan(&count, &unchanged); err != nil {
					t.Fatal(err)
				}
				if count != 1 || unchanged != 1 {
					t.Errorf("%s: 件数=%d、既存の未失効トークン=%d、期待値はともに1", token.table, count, unchanged)
				}
			}
		})
	}
}
