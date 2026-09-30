package usecase

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// authenticateAPITokenFixtureはトークン認証のテストで使うスペース・メンバー・トークン
type authenticateAPITokenFixture struct {
	userID        model.UserID
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
	tokenID       model.PersonalAccessTokenID
	token         string
}

// authenticateAPITokenFixtureOptionsはフィクスチャの既定値 (フラグ有効・有効なメンバー・
// `personal_access_token:write` を持つ・有効期限内のトークン) から変える点
type authenticateAPITokenFixtureOptions struct {
	memberScopes   []model.Scope
	memberActive   bool
	tokenScopes    []model.Scope
	flagDisabled   bool
	spaceDiscarded bool
	userDiscarded  bool
	expiresAt      time.Time
	revokedAt      *time.Time
}

func defaultAuthenticateAPITokenFixtureOptions() authenticateAPITokenFixtureOptions {
	return authenticateAPITokenFixtureOptions{
		memberScopes: []model.Scope{model.ScopePageWrite, model.ScopePersonalAccessTokenWrite},
		memberActive: true,
		tokenScopes:  []model.Scope{model.ScopePageRead},
		expiresAt:    time.Now().Add(24 * time.Hour),
	}
}

func setupAuthenticateAPITokenFixture(t *testing.T, tx *sql.Tx, opts authenticateAPITokenFixtureOptions) authenticateAPITokenFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).Build()
	if opts.userDiscarded {
		if _, err := tx.ExecContext(context.Background(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(userID)); err != nil {
			t.Fatalf("ユーザーの退会に失敗: %v", err)
		}
	}

	spaceBuilder := testutil.NewSpaceBuilder(t, tx)
	if opts.spaceDiscarded {
		spaceBuilder = spaceBuilder.WithDiscarded()
	}
	spaceID := spaceBuilder.Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes(opts.memberScopes).
		WithActive(opts.memberActive).
		Build()

	if !opts.flagDisabled {
		testutil.NewFeatureFlagBuilder(t, tx).
			WithUserID(userID).
			WithName(string(model.FeatureFlagPublicAPI)).
			Build()
	}

	token, err := auth.GenerateOpaqueToken(auth.PersonalAccessTokenPrefix)
	if err != nil {
		t.Fatalf("GenerateOpaqueToken()のエラー = %v", err)
	}
	tokenBuilder := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithTokenDigest(auth.DigestOpaqueToken(token)).
		WithScopes(opts.tokenScopes).
		WithExpiresAt(opts.expiresAt)
	if opts.revokedAt != nil {
		tokenBuilder = tokenBuilder.WithRevokedAt(*opts.revokedAt)
	}
	tokenID := tokenBuilder.Build()

	return authenticateAPITokenFixture{
		userID:        userID,
		spaceID:       spaceID,
		spaceMemberID: spaceMemberID,
		tokenID:       tokenID,
		token:         token,
	}
}

func newAuthenticateAPITokenUC(tx *sql.Tx) *AuthenticateAPITokenUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewAuthenticateAPITokenUsecase(
		repository.NewPersonalAccessTokenRepository(q),
		repository.NewOAuthAccessTokenRepository(q),
		repository.NewOAuthGrantRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewUserRepository(q),
		repository.NewFeatureFlagRepository(q),
	)
}

// findPersonalAccessTokenLastUsedAtはトークンの最終使用日時を返す
func findPersonalAccessTokenLastUsedAt(t *testing.T, tx *sql.Tx, id model.PersonalAccessTokenID) sql.NullTime {
	t.Helper()

	var lastUsedAt sql.NullTime
	if err := tx.QueryRowContext(context.Background(), "SELECT last_used_at FROM personal_access_tokens WHERE id = $1", string(id)).Scan(&lastUsedAt); err != nil {
		t.Fatalf("最終使用日時の取得に失敗: %v", err)
	}
	return lastUsedAt
}

func TestAuthenticateAPITokenUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	opts := defaultAuthenticateAPITokenFixtureOptions()
	// page:writeはpage:readを含意し、トークンに付与できないスコープは落とす
	opts.tokenScopes = []model.Scope{model.ScopePageWrite, model.ScopeSpaceAdmin}
	f := setupAuthenticateAPITokenFixture(t, tx, opts)

	principal, err := newAuthenticateAPITokenUC(tx).Execute(context.Background(), f.token)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if principal == nil {
		t.Fatal("principal = nil、主体を期待")
	}

	if principal.User.ID != f.userID {
		t.Errorf("User.ID = %s、期待値 = %s", principal.User.ID, f.userID)
	}
	if principal.Space.ID != f.spaceID {
		t.Errorf("Space.ID = %s、期待値 = %s", principal.Space.ID, f.spaceID)
	}
	if principal.SpaceMember.ID != f.spaceMemberID {
		t.Errorf("SpaceMember.ID = %s、期待値 = %s", principal.SpaceMember.ID, f.spaceMemberID)
	}
	if principal.TokenKind != model.APITokenKindPersonalAccessToken {
		t.Errorf("TokenKind = %s、期待値 = %s", principal.TokenKind, model.APITokenKindPersonalAccessToken)
	}
	wantScopes := map[model.Scope]bool{
		model.ScopePageWrite: true,
		model.ScopePageRead:  true,
	}
	if len(principal.Scopes) != len(wantScopes) {
		t.Errorf("Scopes = %v、期待値 = %v", principal.Scopes, wantScopes)
	}
	for _, s := range principal.Scopes {
		if !wantScopes[s] {
			t.Errorf("Scopes = %v に予期しないスコープ %s が含まれる", principal.Scopes, s)
		}
	}

	if lastUsedAt := findPersonalAccessTokenLastUsedAt(t, tx, f.tokenID); !lastUsedAt.Valid {
		t.Error("last_used_at = NULL、更新を期待")
	}
}

func TestAuthenticateAPITokenUsecase_Execute_Rejected(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-time.Hour)

	tests := []struct {
		name   string
		modify func(opts *authenticateAPITokenFixtureOptions)
		// tokenは照合に送るトークン。nilならフィクスチャのトークンを送る
		token func(f authenticateAPITokenFixture) string
	}{
		{
			name:  "登録されていないトークン",
			token: func(authenticateAPITokenFixture) string { return string(auth.PersonalAccessTokenPrefix) + "unknown" },
		},
		{
			name:  "接頭辞の無いトークン",
			token: func(f authenticateAPITokenFixture) string { return f.token[len(auth.PersonalAccessTokenPrefix):] },
		},
		{
			name:   "期限切れのトークン",
			modify: func(opts *authenticateAPITokenFixtureOptions) { opts.expiresAt = past },
		},
		{
			name:   "失効したトークン",
			modify: func(opts *authenticateAPITokenFixtureOptions) { opts.revokedAt = &past },
		},
		{
			name:   "持ち主のフィーチャーフラグが無効",
			modify: func(opts *authenticateAPITokenFixtureOptions) { opts.flagDisabled = true },
		},
		{
			name:   "持ち主がスペースの有効なメンバーでない",
			modify: func(opts *authenticateAPITokenFixtureOptions) { opts.memberActive = false },
		},
		{
			name: "持ち主がpersonal_access_token:writeを外された",
			modify: func(opts *authenticateAPITokenFixtureOptions) {
				opts.memberScopes = []model.Scope{model.ScopePageWrite, model.ScopePersonalAccessTokenRead, model.ScopePersonalAccessTokenDelete}
			},
		},
		{
			name:   "持ち主が退会済み",
			modify: func(opts *authenticateAPITokenFixtureOptions) { opts.userDiscarded = true },
		},
		{
			name:   "スペースが削除済み",
			modify: func(opts *authenticateAPITokenFixtureOptions) { opts.spaceDiscarded = true },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			opts := defaultAuthenticateAPITokenFixtureOptions()
			if tt.modify != nil {
				tt.modify(&opts)
			}
			f := setupAuthenticateAPITokenFixture(t, tx, opts)
			token := f.token
			if tt.token != nil {
				token = tt.token(f)
			}

			principal, err := newAuthenticateAPITokenUC(tx).Execute(context.Background(), token)
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if principal != nil {
				t.Errorf("principal = %+v、nilを期待", principal)
			}
			if lastUsedAt := findPersonalAccessTokenLastUsedAt(t, tx, f.tokenID); lastUsedAt.Valid {
				t.Errorf("last_used_at = %v、受け付けないトークンでは更新しないことを期待", lastUsedAt.Time)
			}
		})
	}
}

// oauthAccessTokenAuthFixtureOptionsは、OAuthのアクセストークンの照合のフィクスチャの既定値
// (フラグ有効・有効なメンバー・`oauth_grant:write` を持つ・失効していない許可・有効期限内のトークン) から変える点
type oauthAccessTokenAuthFixtureOptions struct {
	memberScopes   []model.Scope
	memberActive   bool
	tokenScopes    []model.Scope
	flagDisabled   bool
	grantRevokedAt *time.Time
	expiresAt      time.Time
	revokedAt      *time.Time
}

func defaultOAuthAccessTokenAuthFixtureOptions() oauthAccessTokenAuthFixtureOptions {
	return oauthAccessTokenAuthFixtureOptions{
		memberScopes: []model.Scope{model.ScopePageWrite, model.ScopeOAuthGrantWrite},
		memberActive: true,
		tokenScopes:  []model.Scope{model.ScopePageRead},
		expiresAt:    time.Now().Add(time.Hour),
	}
}

// setupOAuthAccessTokenAuthFixtureは、スペースのOAuthアプリの許可に属するアクセストークンを作り、
// トークンの値を返す
// keyは並行するテストの間で一意にし、スペースの識別子とアプリのクライアントIDに使う
func setupOAuthAccessTokenAuthFixture(t *testing.T, tx *sql.Tx, key string, opts oauthAccessTokenAuthFixtureOptions) (authenticateAPITokenFixture, string) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(key + "@example.com").
		WithAtname(strings.ReplaceAll(key, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(key).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes(opts.memberScopes).
		WithActive(opts.memberActive).
		Build()
	if !opts.flagDisabled {
		testutil.NewFeatureFlagBuilder(t, tx).
			WithUserID(userID).
			WithName(string(model.FeatureFlagPublicAPI)).
			Build()
	}

	appID := testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(spaceID).WithClientID(key + "-client").Build()
	grantBuilder := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(appID).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID)
	if opts.grantRevokedAt != nil {
		grantBuilder = grantBuilder.WithRevokedAt(*opts.grantRevokedAt)
	}
	grantID := grantBuilder.Build()

	token, err := auth.GenerateOpaqueToken(auth.OAuthAccessTokenPrefix)
	if err != nil {
		t.Fatalf("GenerateOpaqueToken()のエラー = %v", err)
	}
	tokenBuilder := testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(spaceID).
		WithTokenDigest(auth.DigestOpaqueToken(token)).
		WithScopes(opts.tokenScopes).
		WithExpiresAt(opts.expiresAt)
	if opts.revokedAt != nil {
		tokenBuilder = tokenBuilder.WithRevokedAt(*opts.revokedAt)
	}
	tokenBuilder.Build()

	return authenticateAPITokenFixture{
		userID:        userID,
		spaceID:       spaceID,
		spaceMemberID: spaceMemberID,
	}, token
}

func TestAuthenticateAPITokenUsecase_Execute_OAuthAccessToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	opts := defaultOAuthAccessTokenAuthFixtureOptions()
	// page:writeはpage:readを含意し、トークンに付与できないスコープは落とす
	opts.tokenScopes = []model.Scope{model.ScopePageWrite, model.ScopeSpaceAdmin}
	f, token := setupOAuthAccessTokenAuthFixture(t, tx, "aat-oauth", opts)

	principal, err := newAuthenticateAPITokenUC(tx).Execute(context.Background(), token)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if principal == nil {
		t.Fatal("principal = nil、主体を期待")
	}

	if principal.User.ID != f.userID {
		t.Errorf("User.ID = %s、期待値 = %s", principal.User.ID, f.userID)
	}
	if principal.Space.ID != f.spaceID {
		t.Errorf("Space.ID = %s、期待値 = %s", principal.Space.ID, f.spaceID)
	}
	if principal.SpaceMember.ID != f.spaceMemberID {
		t.Errorf("SpaceMember.ID = %s、期待値 = %s", principal.SpaceMember.ID, f.spaceMemberID)
	}
	if principal.TokenKind != model.APITokenKindOAuthAccessToken {
		t.Errorf("TokenKind = %s、期待値 = %s", principal.TokenKind, model.APITokenKindOAuthAccessToken)
	}
	if !principal.HasScope(model.ScopePageWrite) || !principal.HasScope(model.ScopePageRead) || len(principal.Scopes) != 2 {
		t.Errorf("Scopes = %v、期待値 = [page:write page:read]", principal.Scopes)
	}
}

func TestAuthenticateAPITokenUsecase_Execute_OAuthAccessTokenRejected(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-time.Hour)

	tests := []struct {
		name   string
		modify func(opts *oauthAccessTokenAuthFixtureOptions)
		// tokenは照合に送るトークン。nilならフィクスチャのトークンを送る
		token func(token string) string
	}{
		{
			name:  "登録されていないトークン",
			token: func(string) string { return string(auth.OAuthAccessTokenPrefix) + "unknown" },
		},
		{
			name:   "期限切れのトークン",
			modify: func(opts *oauthAccessTokenAuthFixtureOptions) { opts.expiresAt = past },
		},
		{
			name:   "失効したトークン",
			modify: func(opts *oauthAccessTokenAuthFixtureOptions) { opts.revokedAt = &past },
		},
		{
			name:   "許可が失効した (連携の解除・アプリの削除)",
			modify: func(opts *oauthAccessTokenAuthFixtureOptions) { opts.grantRevokedAt = &past },
		},
		{
			name: "持ち主がoauth_grant:writeを外された",
			modify: func(opts *oauthAccessTokenAuthFixtureOptions) {
				opts.memberScopes = []model.Scope{model.ScopePageWrite, model.ScopePersonalAccessTokenWrite, model.ScopeOAuthGrantRead}
			},
		},
		{
			name:   "持ち主がスペースの有効なメンバーでない",
			modify: func(opts *oauthAccessTokenAuthFixtureOptions) { opts.memberActive = false },
		},
		{
			name:   "持ち主のフィーチャーフラグが無効",
			modify: func(opts *oauthAccessTokenAuthFixtureOptions) { opts.flagDisabled = true },
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			opts := defaultOAuthAccessTokenAuthFixtureOptions()
			if tt.modify != nil {
				tt.modify(&opts)
			}
			_, token := setupOAuthAccessTokenAuthFixture(t, tx, "aat-oauth-reject-"+string(rune('a'+i)), opts)
			if tt.token != nil {
				token = tt.token(token)
			}

			principal, err := newAuthenticateAPITokenUC(tx).Execute(context.Background(), token)
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if principal != nil {
				t.Errorf("principal = %+v、nilを期待", principal)
			}
		})
	}
}
