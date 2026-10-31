package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newRevokeOAuthTokenUsecaseForTest(q *query.Queries) *RevokeOAuthTokenUsecase {
	return NewRevokeOAuthTokenUsecase(
		repository.NewOAuthApplicationRepository(q),
		repository.NewOAuthGrantRepository(q),
		repository.NewOAuthAccessTokenRepository(q),
		repository.NewOAuthRefreshTokenRepository(q),
	)
}

// buildAccessTokenは許可に属するアクセストークンを作り、トークンの値を返す
func (f oauthTokenFixture) buildAccessToken(t *testing.T, tx *sql.Tx, value string) string {
	t.Helper()

	token := string(auth.OAuthAccessTokenPrefix) + value
	testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(f.grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest(auth.DigestOpaqueToken(token)).
		Build()
	return token
}

// assertAccessTokenRevokedは、アクセストークンtokenの失効の有無がwantであることを確かめる
func assertAccessTokenRevoked(t *testing.T, q *query.Queries, token string, want bool) {
	t.Helper()

	got, err := repository.NewOAuthAccessTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(token))
	if err != nil {
		t.Fatalf("アクセストークンのFindByTokenDigest()のエラー = %v", err)
	}
	if got.IsRevoked() != want {
		t.Errorf("アクセストークン%sのIsRevoked() = %v、期待値 = %v", token, got.IsRevoked(), want)
	}
}

// assertRefreshTokenRevokedは、リフレッシュトークンtokenの失効の有無がwantであることを確かめる
func assertRefreshTokenRevoked(t *testing.T, q *query.Queries, token string, want bool) {
	t.Helper()

	got, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(token))
	if err != nil {
		t.Fatalf("リフレッシュトークンのFindByTokenDigest()のエラー = %v", err)
	}
	if got.IsRevoked() != want {
		t.Errorf("リフレッシュトークン%sのIsRevoked() = %v、期待値 = %v", token, got.IsRevoked(), want)
	}
}

func TestRevokeOAuthTokenUsecase_Execute_アクセストークン(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupOAuthTokenFixture(t, tx, "rot-access", oauthTokenFixtureOptions{})
	target := f.buildAccessToken(t, tx, "rot_access_target")
	sibling := f.buildAccessToken(t, tx, "rot_access_sibling")
	refresh := f.buildRefreshToken(t, tx, "rot_access_refresh", nil)

	err := newRevokeOAuthTokenUsecaseForTest(q).Execute(context.Background(), OAuthTokenRevocationParams{
		Token:    target,
		ClientID: f.clientID,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	// 失効させるのは送られたアクセストークンだけで、同じ許可のほかのトークンと許可は残す
	assertAccessTokenRevoked(t, q, target, true)
	assertAccessTokenRevoked(t, q, sibling, false)
	assertRefreshTokenRevoked(t, q, refresh, false)
	assertGrantRevoked(t, q, f, false)
}

func TestRevokeOAuthTokenUsecase_Execute_リフレッシュトークン(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupOAuthTokenFixture(t, tx, "rot-refresh", oauthTokenFixtureOptions{confidential: true})
	target := f.buildRefreshToken(t, tx, "rot_refresh_target", nil)
	sibling := f.buildRefreshToken(t, tx, "rot_refresh_sibling", nil)
	access := f.buildAccessToken(t, tx, "rot_refresh_access")

	err := newRevokeOAuthTokenUsecaseForTest(q).Execute(context.Background(), OAuthTokenRevocationParams{
		Token:                target,
		ClientID:             f.clientID,
		ClientSecret:         oauthTokenTestClientSecret,
		ClientSecretProvided: true,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	// 送られたリフレッシュトークンの許可と、許可に属するトークンをすべて失効させる
	assertRefreshTokenRevoked(t, q, target, true)
	assertAccessTokenRevoked(t, q, access, true)
	assertRefreshTokenRevoked(t, q, sibling, true)
	assertGrantRevoked(t, q, f, true)
}

// 発行が許可行をロックした後にログアウトが到着しても、互いに同じ順序でロックする。
// 待機中に発行されたトークンも、ログアウト後には許可の失効により使えないことを確かめる。
func TestRevokeOAuthTokenUsecase_Execute_リフレッシュとの同時実行(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	key := "rot-refresh-concurrent"
	f := setupOAuthTokenFixtureDB(t, key, oauthTokenFixtureOptions{})
	value := f.buildRefreshTokenDB(t, key, nil)
	refreshRepo := repository.NewOAuthRefreshTokenRepository(q)
	token, err := refreshRepo.FindByTokenDigest(ctx, auth.DigestOpaqueToken(value))
	if err != nil {
		t.Fatal(err)
	}

	// リフレッシュのrotateTokenと同じく、許可を先にロックする。
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	var issuingPID int
	if err := tx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&issuingPID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.NewOAuthGrantRepository(q).WithTx(tx).LockByID(ctx, f.grantID, f.spaceID); err != nil {
		t.Fatal(err)
	}
	revoked := make(chan error, 1)
	go func() {
		revoked <- newRevokeOAuthTokenUsecaseForTest(q).Execute(ctx, OAuthTokenRevocationParams{Token: value, ClientID: f.clientID})
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
    SELECT 1 FROM pg_stat_activity WHERE datname = current_database()
      AND $1 = ANY(pg_blocking_pids(pid)) AND query LIKE '%RevokeOAuthRefreshToken%'
  )`, issuingPID).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		select {
		case err := <-revoked:
			t.Fatalf("許可のロックを待つ前に失効が終了: %v", err)
		case <-ctx.Done():
			t.Fatalf("失効のロック待機を確認できない: %v", ctx.Err())
		case <-ticker.C:
		}
	}

	now := time.Now()
	accessValue := string(auth.OAuthAccessTokenPrefix) + key + "-issued"
	refreshValue := string(auth.OAuthRefreshTokenPrefix) + key + "-issued"
	rotated, err := refreshRepo.WithTx(tx).Rotate(ctx, repository.RotateOAuthRefreshTokenInput{
		ID: token.ID, SpaceID: f.spaceID, TokenDigest: auth.DigestOpaqueToken(refreshValue), ExpiresAt: now.Add(time.Hour),
		AccessTokenDigest: auth.DigestOpaqueToken(accessValue), AccessTokenScopes: token.Scopes, AccessTokenExpiresAt: now.Add(time.Hour), Now: now,
	})
	if err != nil || rotated == nil {
		t.Fatalf("ログアウト待機中のRotate() = %+v、エラー = %v、トークンの発行成功を期待", rotated, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-revoked:
		if err != nil {
			t.Fatalf("発行後のログアウトに失敗: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("ログアウトが完了しない: %v", ctx.Err())
	}

	assertGrantRevoked(t, q, f, true)
	principal, err := newAuthenticateAPITokenUCForOAuthTest(q).Execute(ctx, accessValue, model.OAuthAudiencePublicAPI)
	if err != nil || principal != nil {
		t.Errorf("ログアウト後のアクセストークンの照合 = %+v、エラー = %v、拒否を期待", principal, err)
	}
	output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(ctx, f.refreshParams(refreshValue))
	assertOAuthTokenError(t, output, err, model.OAuthTokenErrorInvalidGrant)
}

func TestRevokeOAuthTokenUsecase_Execute_無効なトークンは何もせず成功する(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-time.Hour)

	tests := []struct {
		name string
		// tokenOfはフィクスチャから失効を求めるトークンを作る
		tokenOf func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) string
	}{
		{
			name: "存在しないアクセストークン",
			tokenOf: func(_ *testing.T, _ *sql.Tx, _ oauthTokenFixture) string {
				return string(auth.OAuthAccessTokenPrefix) + "missing"
			},
		},
		{
			name: "存在しないリフレッシュトークン",
			tokenOf: func(_ *testing.T, _ *sql.Tx, _ oauthTokenFixture) string {
				return string(auth.OAuthRefreshTokenPrefix) + "missing"
			},
		},
		{
			name: "個人用アクセストークンの形式",
			tokenOf: func(_ *testing.T, _ *sql.Tx, _ oauthTokenFixture) string {
				return string(auth.PersonalAccessTokenPrefix) + "token"
			},
		},
		{
			name: "使用済みのリフレッシュトークン",
			tokenOf: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) string {
				return f.buildRefreshToken(t, tx, string(f.identifier)+"-used", func(b *testutil.OAuthRefreshTokenBuilder) { b.WithUsedAt(past) })
			},
		},
		{
			name: "期限切れのリフレッシュトークン",
			tokenOf: func(t *testing.T, tx *sql.Tx, f oauthTokenFixture) string {
				return f.buildRefreshToken(t, tx, string(f.identifier)+"-expired", func(b *testutil.OAuthRefreshTokenBuilder) { b.WithExpiresAt(past) })
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			key := "rot-invalid-" + string(rune('a'+i))
			f := setupOAuthTokenFixture(t, tx, key, oauthTokenFixtureOptions{})
			access := f.buildAccessToken(t, tx, key+"-access")

			err := newRevokeOAuthTokenUsecaseForTest(q).Execute(context.Background(), OAuthTokenRevocationParams{
				Token:    tt.tokenOf(t, tx, f),
				ClientID: f.clientID,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v、成功を期待", err)
			}
			// 使えないリフレッシュトークンの失効では、同じ許可のアクセストークンも失効させない
			assertAccessTokenRevoked(t, q, access, false)
		})
	}
}

func TestRevokeOAuthTokenUsecase_Execute_拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// paramsOfは、フィクスチャfと、fのアクセストークンtokenから要求のパラメーターを作る
		paramsOf func(f oauthTokenFixture, other oauthTokenFixture, token string) OAuthTokenRevocationParams
		wantCode model.OAuthTokenErrorCode
	}{
		{
			name: "トークンが無い",
			paramsOf: func(f, _ oauthTokenFixture, _ string) OAuthTokenRevocationParams {
				return OAuthTokenRevocationParams{ClientID: f.clientID}
			},
			wantCode: model.OAuthTokenErrorInvalidRequest,
		},
		{
			name: "クライアントIDが無い",
			paramsOf: func(_, _ oauthTokenFixture, token string) OAuthTokenRevocationParams {
				return OAuthTokenRevocationParams{Token: token}
			},
			wantCode: model.OAuthTokenErrorInvalidRequest,
		},
		{
			name: "パラメーターの重複",
			paramsOf: func(f, _ oauthTokenFixture, token string) OAuthTokenRevocationParams {
				return OAuthTokenRevocationParams{Token: token, ClientID: f.clientID, DuplicatedParams: []string{"token"}}
			},
			wantCode: model.OAuthTokenErrorInvalidRequest,
		},
		{
			name: "存在しないクライアント",
			paramsOf: func(_, _ oauthTokenFixture, token string) OAuthTokenRevocationParams {
				return OAuthTokenRevocationParams{Token: token, ClientID: "missing-client"}
			},
			wantCode: model.OAuthTokenErrorInvalidClient,
		},
		{
			name: "publicクライアントがシークレットを送った",
			paramsOf: func(f, _ oauthTokenFixture, token string) OAuthTokenRevocationParams {
				return OAuthTokenRevocationParams{Token: token, ClientID: f.clientID, ClientSecret: "secret", ClientSecretProvided: true}
			},
			wantCode: model.OAuthTokenErrorInvalidClient,
		},
		{
			name: "別のクライアントに発行されたトークン",
			paramsOf: func(_, other oauthTokenFixture, token string) OAuthTokenRevocationParams {
				return OAuthTokenRevocationParams{Token: token, ClientID: other.clientID}
			},
			wantCode: model.OAuthTokenErrorUnauthorizedClient,
		},
		{
			name: "存在しないトークンでもクライアントの認証に失敗すれば拒否する",
			paramsOf: func(_, _ oauthTokenFixture, _ string) OAuthTokenRevocationParams {
				return OAuthTokenRevocationParams{Token: string(auth.OAuthAccessTokenPrefix) + "missing", ClientID: "missing-client"}
			},
			wantCode: model.OAuthTokenErrorInvalidClient,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			key := "rot-reject-" + string(rune('a'+i))
			f := setupOAuthTokenFixture(t, tx, key, oauthTokenFixtureOptions{})
			// 同じスペースに登録された別のクライアント
			otherClientID := key + "-other-client"
			testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID(otherClientID).Build()
			other := f
			other.clientID = otherClientID
			token := f.buildAccessToken(t, tx, key+"-access")

			err := newRevokeOAuthTokenUsecaseForTest(q).Execute(context.Background(), tt.paramsOf(f, other, token))
			oe := model.AsOAuthTokenError(err)
			if oe == nil || oe.Code != tt.wantCode {
				t.Fatalf("Execute()のエラー = %v、期待値 = %v", err, tt.wantCode)
			}
			assertAccessTokenRevoked(t, q, token, false)
		})
	}
}

func TestRevokeOAuthTokenUsecase_Execute_別スペースのクライアント(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupOAuthTokenFixture(t, tx, "rot-cross-space-token", oauthTokenFixtureOptions{})
	other := setupOAuthTokenFixture(t, tx, "rot-cross-space-client", oauthTokenFixtureOptions{})
	access := f.buildAccessToken(t, tx, "rot_cross_space_access")
	refresh := f.buildRefreshToken(t, tx, "rot_cross_space_refresh", nil)
	uc := newRevokeOAuthTokenUsecaseForTest(q)
	ctx := context.Background()

	// 別スペースの有効なクライアントでも、発行先が異なるトークンは失効できない。
	for _, token := range []string{access, refresh} {
		err := uc.Execute(ctx, OAuthTokenRevocationParams{Token: token, ClientID: other.clientID})
		if oe := model.AsOAuthTokenError(err); oe == nil || oe.Code != model.OAuthTokenErrorUnauthorizedClient {
			t.Errorf("別スペースのトークン%sのエラー = %v、期待値 = unauthorized_client", token, err)
		}
	}
	assertAccessTokenRevoked(t, q, access, false)
	assertRefreshTokenRevoked(t, q, refresh, false)

	// トークンが見つからなくても、同じクライアントの認証は成功して200に相当するnilを返す。
	if err := uc.Execute(ctx, OAuthTokenRevocationParams{
		Token:    string(auth.OAuthAccessTokenPrefix) + "rot_cross_space_missing",
		ClientID: other.clientID,
	}); err != nil {
		t.Errorf("存在しないトークンの失効のエラー = %v、成功を期待", err)
	}
}

func TestRevokeOAuthTokenUsecase_Execute_公式クライアント(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	official := setupOAuthTokenFixture(t, tx, "rot-official", oauthTokenFixtureOptions{officialClient: true})
	spaceApp := setupOAuthTokenFixture(t, tx, "rot-official-space-app", oauthTokenFixtureOptions{})
	refresh := official.buildRefreshToken(t, tx, "rot_official_refresh", nil)
	access := official.buildAccessToken(t, tx, "rot_official_access")
	spaceAppAccess := spaceApp.buildAccessToken(t, tx, "rot_official_space_app_access")
	uc := newRevokeOAuthTokenUsecaseForTest(q)
	ctx := context.Background()

	// 公式クライアントはどのスペースでも使えるため、発行先の判定は許可のアプリとの一致だけで決まる。
	// スペースのアプリに発行されたトークンは失効できない
	err := uc.Execute(ctx, OAuthTokenRevocationParams{Token: spaceAppAccess, ClientID: official.clientID})
	if oe := model.AsOAuthTokenError(err); oe == nil || oe.Code != model.OAuthTokenErrorUnauthorizedClient {
		t.Errorf("スペースのアプリのトークンの失効のエラー = %v、期待値 = unauthorized_client", err)
	}
	assertAccessTokenRevoked(t, q, spaceAppAccess, false)

	// 自分の許可のリフレッシュトークンは失効でき、許可と同じ許可のアクセストークンも失効する
	if err := uc.Execute(ctx, OAuthTokenRevocationParams{Token: refresh, ClientID: official.clientID}); err != nil {
		t.Fatalf("公式クライアントのリフレッシュトークンの失効のエラー = %v、成功を期待", err)
	}
	assertRefreshTokenRevoked(t, q, refresh, true)
	assertAccessTokenRevoked(t, q, access, true)
	assertGrantRevoked(t, q, official, true)
}

func TestRevokeOAuthTokenUsecase_Execute_confidentialクライアントのシークレットの誤り(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupOAuthTokenFixture(t, tx, "rot-wrong-secret", oauthTokenFixtureOptions{confidential: true})
	token := f.buildRefreshToken(t, tx, "rot_wrong_secret_refresh", nil)

	for _, params := range []OAuthTokenRevocationParams{
		{Token: token, ClientID: f.clientID, ClientSecret: "wks_wrong", ClientSecretProvided: true},
		{Token: token, ClientID: f.clientID},
	} {
		err := newRevokeOAuthTokenUsecaseForTest(q).Execute(context.Background(), params)
		if oe := model.AsOAuthTokenError(err); oe == nil || oe.Code != model.OAuthTokenErrorInvalidClient {
			t.Errorf("Execute(%+v)のエラー = %v、期待値 = invalid_client", params, err)
		}
	}
	assertRefreshTokenRevoked(t, q, token, false)
}

// assertGrantRevokedは、フィクスチャの許可の失効の有無がwantであることを確かめる
func assertGrantRevoked(t *testing.T, q *query.Queries, f oauthTokenFixture, want bool) {
	t.Helper()

	grant, err := repository.NewOAuthGrantRepository(q).FindByID(context.Background(), f.grantID, f.spaceID)
	if err != nil {
		t.Fatalf("許可のFindByID()のエラー = %v", err)
	}
	if grant.IsRevoked() != want {
		t.Errorf("許可のIsRevoked() = %v、期待値 = %v", grant.IsRevoked(), want)
	}
}
