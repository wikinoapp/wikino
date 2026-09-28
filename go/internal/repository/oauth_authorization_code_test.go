package repository

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestOAuthAuthorizationCodeRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthAuthorizationCodeRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "code-create")
	grantID := f.buildGrant(t, tx)
	expiresAt := time.Now().Add(5 * time.Minute).Truncate(time.Microsecond)
	scopes := []model.Scope{model.ScopePageRead, model.ScopePageWrite}

	code, err := repo.Create(ctx, CreateOAuthAuthorizationCodeInput{
		OAuthGrantID:  grantID,
		SpaceID:       f.spaceID,
		CodeDigest:    "code_create_digest",
		Scopes:        scopes,
		RedirectURI:   "http://127.0.0.1:8080/callback",
		CodeChallenge: "code_create_challenge",
		ExpiresAt:     expiresAt,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if code.ID == "" {
		t.Error("code.IDが空")
	}
	if code.OAuthGrantID != grantID {
		t.Errorf("code.OAuthGrantID = %v、期待値 = %v", code.OAuthGrantID, grantID)
	}
	if code.SpaceID != f.spaceID {
		t.Errorf("code.SpaceID = %v、期待値 = %v", code.SpaceID, f.spaceID)
	}
	if code.CodeDigest != "code_create_digest" {
		t.Errorf("code.CodeDigest = %q、期待値 = %q", code.CodeDigest, "code_create_digest")
	}
	if !slices.Equal(code.Scopes, scopes) {
		t.Errorf("code.Scopes = %v、期待値 = %v", code.Scopes, scopes)
	}
	if code.RedirectURI != "http://127.0.0.1:8080/callback" {
		t.Errorf("code.RedirectURI = %q、期待値 = %q", code.RedirectURI, "http://127.0.0.1:8080/callback")
	}
	if code.CodeChallenge != "code_create_challenge" {
		t.Errorf("code.CodeChallenge = %q、期待値 = %q", code.CodeChallenge, "code_create_challenge")
	}
	if !code.ExpiresAt.Equal(expiresAt) {
		t.Errorf("code.ExpiresAt = %v、期待値 = %v", code.ExpiresAt, expiresAt)
	}
	if code.IsUsed() {
		t.Error("code.IsUsed() = true、期待値 = false")
	}

	// 許可と違うスペースでは作れない (トランザクションが中断するため最後に確かめる)
	other := setupOAuthFixture(t, tx, "code-create-other")
	_, err = repo.Create(ctx, CreateOAuthAuthorizationCodeInput{
		OAuthGrantID:  grantID,
		SpaceID:       other.spaceID,
		CodeDigest:    "code_create_other_space_digest",
		Scopes:        scopes,
		RedirectURI:   "http://127.0.0.1:8080/callback",
		CodeChallenge: "code_create_challenge",
		ExpiresAt:     expiresAt,
	})
	assertConstraintViolation(t, err, "foreign_key_violation", "oauth_authorization_codes_oauth_grant_id_space_id_fkey")
}

func TestOAuthAuthorizationCodeRepository_CreateWithGrant(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthAuthorizationCodeRepository(testutil.QueriesWithTx(tx))
	grantRepo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "code-with-grant")
	expiresAt := time.Now().Add(5 * time.Minute).Truncate(time.Microsecond)
	input := CreateOAuthAuthorizationCodeWithGrantInput{
		OAuthApplicationID: f.oauthApplicationID,
		SpaceID:            f.spaceID,
		SpaceMemberID:      f.spaceMemberID,
		CodeDigest:         "code_with_grant_first_digest",
		Scopes:             []model.Scope{model.ScopeTopicRead},
		RedirectURI:        "http://127.0.0.1:8080/callback",
		CodeChallenge:      "code_with_grant_challenge",
		ExpiresAt:          expiresAt,
	}

	// 許可が無ければ作り、コードはその許可に属する
	first, err := repo.CreateWithGrant(ctx, input)
	if err != nil {
		t.Fatalf("CreateWithGrant()のエラー = %v", err)
	}
	if first.SpaceID != f.spaceID {
		t.Errorf("first.SpaceID = %v、期待値 = %v", first.SpaceID, f.spaceID)
	}
	if first.RedirectURI != input.RedirectURI || first.CodeChallenge != input.CodeChallenge || !first.ExpiresAt.Equal(expiresAt) {
		t.Errorf("first = %+v、入力と一致しない", first)
	}
	grant, err := grantRepo.FindByID(ctx, first.OAuthGrantID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if grant == nil || grant.OAuthApplicationID != f.oauthApplicationID || grant.SpaceMemberID != f.spaceMemberID {
		t.Fatalf("grant = %+v、フィクスチャのアプリ・メンバーの許可を期待", grant)
	}
	if want := []model.Scope{model.ScopeTopicRead}; !slices.Equal(grant.Scopes, want) {
		t.Errorf("grant.Scopes = %v、期待値 = %v", grant.Scopes, want)
	}

	// 失効していない許可があれば、同じ許可のスコープを和に広げ、コードは今回のスコープだけを持つ
	input.CodeDigest = "code_with_grant_second_digest"
	input.Scopes = []model.Scope{model.ScopePageRead, model.ScopePageWrite}
	second, err := repo.CreateWithGrant(ctx, input)
	if err != nil {
		t.Fatalf("CreateWithGrant()のエラー = %v", err)
	}
	if second.OAuthGrantID != first.OAuthGrantID {
		t.Errorf("second.OAuthGrantID = %v、期待値 = %v (既存の許可)", second.OAuthGrantID, first.OAuthGrantID)
	}
	if !slices.Equal(second.Scopes, input.Scopes) {
		t.Errorf("second.Scopes = %v、期待値 = %v", second.Scopes, input.Scopes)
	}
	grant, err = grantRepo.FindByID(ctx, first.OAuthGrantID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if want := []model.Scope{model.ScopePageRead, model.ScopePageWrite, model.ScopeTopicRead}; !slices.Equal(grant.Scopes, want) {
		t.Errorf("grant.Scopes = %v、期待値 = %v", grant.Scopes, want)
	}

	// 許可が失効していれば、新しい許可を作る
	if _, err := grantRepo.Revoke(ctx, first.OAuthGrantID, f.spaceID, f.spaceMemberID); err != nil {
		t.Fatalf("Revoke()のエラー = %v", err)
	}
	input.CodeDigest = "code_with_grant_third_digest"
	input.Scopes = []model.Scope{model.ScopePageRead}
	third, err := repo.CreateWithGrant(ctx, input)
	if err != nil {
		t.Fatalf("CreateWithGrant()のエラー = %v", err)
	}
	if third.OAuthGrantID == first.OAuthGrantID {
		t.Error("失効した許可にコードが作られた")
	}
	grant, err = grantRepo.FindByID(ctx, third.OAuthGrantID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if want := []model.Scope{model.ScopePageRead}; !slices.Equal(grant.Scopes, want) {
		t.Errorf("grant.Scopes = %v、期待値 = %v (以前の許可のスコープを引き継がない)", grant.Scopes, want)
	}
}

func TestOAuthAuthorizationCodeRepository_FindByCodeDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthAuthorizationCodeRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "code-find")
	grantID := f.buildGrant(t, tx)
	usedID := testutil.NewOAuthAuthorizationCodeBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(f.spaceID).
		WithCodeDigest("code_find_used_digest").
		WithUsedAt(time.Now()).
		Build()

	t.Run("使用済みのコードも返す", func(t *testing.T) {
		code, err := repo.FindByCodeDigest(ctx, "code_find_used_digest")
		if err != nil {
			t.Fatalf("FindByCodeDigest()のエラー = %v", err)
		}
		if code == nil || code.ID != usedID {
			t.Fatalf("FindByCodeDigest() = %v、期待値 = ID %v のコード", code, usedID)
		}
		if !code.IsUsed() {
			t.Error("code.IsUsed() = false、期待値 = true")
		}
	})

	t.Run("一致するコードが無ければnilを返す", func(t *testing.T) {
		code, err := repo.FindByCodeDigest(ctx, "code_find_missing_digest")
		if err != nil {
			t.Fatalf("FindByCodeDigest()のエラー = %v", err)
		}
		if code != nil {
			t.Errorf("FindByCodeDigest() = %v、期待値 = nil", code)
		}
	})
}

func TestOAuthAuthorizationCodeRepository_Exchange(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewOAuthAuthorizationCodeRepository(q)
	accessTokenRepo := NewOAuthAccessTokenRepository(q)
	refreshTokenRepo := NewOAuthRefreshTokenRepository(q)
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "code-exchange")
	other := setupOAuthFixture(t, tx, "code-exchange-other")
	grantID := f.buildGrant(t, tx)
	scopes := []model.Scope{model.ScopeTopicRead, model.ScopePageRead}
	now := time.Now().Truncate(time.Microsecond)
	accessExpiresAt := now.Add(time.Hour)
	refreshExpiresAt := now.Add(90 * 24 * time.Hour)

	newCode := func(digest string) *testutil.OAuthAuthorizationCodeBuilder {
		return testutil.NewOAuthAuthorizationCodeBuilder(t, tx).
			WithOAuthGrantID(grantID).
			WithSpaceID(f.spaceID).
			WithCodeDigest(digest).
			WithScopes(scopes)
	}
	exchange := func(id model.OAuthAuthorizationCodeID, spaceID model.SpaceID, suffix string) (*model.OAuthAuthorizationCode, error) {
		return repo.Exchange(ctx, ExchangeOAuthAuthorizationCodeInput{
			ID:                    id,
			SpaceID:               spaceID,
			AccessTokenDigest:     "code_exchange_access_" + suffix,
			AccessTokenExpiresAt:  accessExpiresAt,
			RefreshTokenDigest:    "code_exchange_refresh_" + suffix,
			RefreshTokenExpiresAt: refreshExpiresAt,
			Now:                   now,
		})
	}

	t.Run("未使用のコードを使用済みにし、コードの許可・スコープでトークンを作る", func(t *testing.T) {
		id := newCode("code_exchange_digest").Build()

		code, err := exchange(id, f.spaceID, "ok")
		if err != nil {
			t.Fatalf("Exchange()のエラー = %v", err)
		}
		if code == nil || code.ID != id {
			t.Fatalf("Exchange() = %v、期待値 = ID %v のコード", code, id)
		}
		if code.UsedAt == nil || !code.UsedAt.Equal(now) {
			t.Errorf("code.UsedAt = %v、期待値 = %v", code.UsedAt, now)
		}

		accessToken, err := accessTokenRepo.FindByTokenDigest(ctx, "code_exchange_access_ok")
		if err != nil {
			t.Fatalf("アクセストークンのFindByTokenDigest()のエラー = %v", err)
		}
		if accessToken == nil {
			t.Fatal("アクセストークンが作られていない")
		}
		if accessToken.OAuthGrantID != grantID || accessToken.SpaceID != f.spaceID {
			t.Errorf("アクセストークンの許可・スペース = (%v, %v)、期待値 = (%v, %v)", accessToken.OAuthGrantID, accessToken.SpaceID, grantID, f.spaceID)
		}
		if !slices.Equal(accessToken.Scopes, scopes) {
			t.Errorf("アクセストークンのScopes = %v、期待値 = %v", accessToken.Scopes, scopes)
		}
		if !accessToken.ExpiresAt.Equal(accessExpiresAt) {
			t.Errorf("アクセストークンのExpiresAt = %v、期待値 = %v", accessToken.ExpiresAt, accessExpiresAt)
		}

		refreshToken, err := refreshTokenRepo.FindByTokenDigest(ctx, "code_exchange_refresh_ok")
		if err != nil {
			t.Fatalf("リフレッシュトークンのFindByTokenDigest()のエラー = %v", err)
		}
		if refreshToken == nil {
			t.Fatal("リフレッシュトークンが作られていない")
		}
		if refreshToken.OAuthGrantID != grantID || refreshToken.SpaceID != f.spaceID {
			t.Errorf("リフレッシュトークンの許可・スペース = (%v, %v)、期待値 = (%v, %v)", refreshToken.OAuthGrantID, refreshToken.SpaceID, grantID, f.spaceID)
		}
		if !slices.Equal(refreshToken.Scopes, scopes) {
			t.Errorf("リフレッシュトークンのScopes = %v、期待値 = %v", refreshToken.Scopes, scopes)
		}
		if !refreshToken.ExpiresAt.Equal(refreshExpiresAt) {
			t.Errorf("リフレッシュトークンのExpiresAt = %v、期待値 = %v", refreshToken.ExpiresAt, refreshExpiresAt)
		}
		if refreshToken.PreviousRefreshTokenID != nil {
			t.Errorf("リフレッシュトークンのPreviousRefreshTokenID = %v、期待値 = nil", refreshToken.PreviousRefreshTokenID)
		}

		again, err := exchange(id, f.spaceID, "again")
		if err != nil {
			t.Fatalf("2回目のExchange()のエラー = %v", err)
		}
		if again != nil {
			t.Errorf("使用済みのコードに対するExchange() = %v、期待値 = nil", again)
		}
		assertNoOAuthTokens(t, accessTokenRepo, refreshTokenRepo, "again")
	})

	t.Run("交換しない場合はnilを返しトークンを作らない", func(t *testing.T) {
		tests := []struct {
			name    string
			id      model.OAuthAuthorizationCodeID
			spaceID model.SpaceID
			suffix  string
		}{
			{name: "使用済みのコード", id: newCode("code_exchange_used_digest").WithUsedAt(now).Build(), spaceID: f.spaceID, suffix: "used"},
			{name: "他のスペースを指定した", id: newCode("code_exchange_space_digest").Build(), spaceID: other.spaceID, suffix: "space"},
		}
		for _, tt := range tests {
			code, err := exchange(tt.id, tt.spaceID, tt.suffix)
			if err != nil {
				t.Fatalf("%s: Exchange()のエラー = %v", tt.name, err)
			}
			if code != nil {
				t.Errorf("%s: Exchange() = %v、期待値 = nil", tt.name, code)
			}
			assertNoOAuthTokens(t, accessTokenRepo, refreshTokenRepo, tt.suffix)
		}

		code, err := repo.FindByCodeDigest(ctx, "code_exchange_space_digest")
		if err != nil {
			t.Fatalf("FindByCodeDigest()のエラー = %v", err)
		}
		if code.IsUsed() {
			t.Error("他のスペースを指定したコードのIsUsed() = true、期待値 = false")
		}
	})
}

// assertNoOAuthTokensは、交換で作るはずだったトークンが作られていないことを確かめる
func assertNoOAuthTokens(t *testing.T, accessTokenRepo *OAuthAccessTokenRepository, refreshTokenRepo *OAuthRefreshTokenRepository, suffix string) {
	t.Helper()

	ctx := context.Background()
	accessToken, err := accessTokenRepo.FindByTokenDigest(ctx, "code_exchange_access_"+suffix)
	if err != nil {
		t.Fatalf("アクセストークンのFindByTokenDigest()のエラー = %v", err)
	}
	if accessToken != nil {
		t.Errorf("アクセストークン = %v、期待値 = nil", accessToken)
	}
	refreshToken, err := refreshTokenRepo.FindByTokenDigest(ctx, "code_exchange_refresh_"+suffix)
	if err != nil {
		t.Fatalf("リフレッシュトークンのFindByTokenDigest()のエラー = %v", err)
	}
	if refreshToken != nil {
		t.Errorf("リフレッシュトークン = %v、期待値 = nil", refreshToken)
	}
}
