package repository

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestOAuthAccessTokenRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthAccessTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "access-create")
	grantID := f.buildGrant(t, tx)
	expiresAt := time.Now().Add(time.Hour).Truncate(time.Microsecond)
	scopes := []model.Scope{model.ScopePageRead}

	token, err := repo.Create(ctx, CreateOAuthAccessTokenInput{
		OAuthGrantID: grantID,
		SpaceID:      f.spaceID,
		TokenDigest:  "access_create_digest",
		Scopes:       scopes,
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if token.ID == "" {
		t.Error("token.IDが空")
	}
	if token.OAuthGrantID != grantID {
		t.Errorf("token.OAuthGrantID = %v、期待値 = %v", token.OAuthGrantID, grantID)
	}
	if token.SpaceID != f.spaceID {
		t.Errorf("token.SpaceID = %v、期待値 = %v", token.SpaceID, f.spaceID)
	}
	if token.TokenDigest != "access_create_digest" {
		t.Errorf("token.TokenDigest = %q、期待値 = %q", token.TokenDigest, "access_create_digest")
	}
	if !slices.Equal(token.Scopes, scopes) {
		t.Errorf("token.Scopes = %v、期待値 = %v", token.Scopes, scopes)
	}
	if !token.ExpiresAt.Equal(expiresAt) {
		t.Errorf("token.ExpiresAt = %v、期待値 = %v", token.ExpiresAt, expiresAt)
	}
	if token.IsRevoked() {
		t.Error("token.IsRevoked() = true、期待値 = false")
	}

	// 許可と違うスペースでは作れない (トランザクションが中断するため最後に確かめる)
	other := setupOAuthFixture(t, tx, "access-create-other")
	_, err = repo.Create(ctx, CreateOAuthAccessTokenInput{
		OAuthGrantID: grantID,
		SpaceID:      other.spaceID,
		TokenDigest:  "access_create_other_space_digest",
		Scopes:       scopes,
		ExpiresAt:    expiresAt,
	})
	assertConstraintViolation(t, err, "foreign_key_violation", "oauth_access_tokens_oauth_grant_id_space_id_fkey")
}

func TestOAuthAccessTokenRepository_FindByTokenDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthAccessTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "access-find")
	grantID := f.buildGrant(t, tx)
	revokedID := testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest("access_find_revoked_digest").
		WithRevokedAt(time.Now()).
		Build()

	t.Run("失効したトークンも返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(ctx, "access_find_revoked_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token == nil || token.ID != revokedID {
			t.Fatalf("FindByTokenDigest() = %v、期待値 = ID %v のトークン", token, revokedID)
		}
		if !token.IsRevoked() {
			t.Error("token.IsRevoked() = false、期待値 = true")
		}
	})

	t.Run("一致するトークンが無ければnilを返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(ctx, "access_find_missing_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token != nil {
			t.Errorf("FindByTokenDigest() = %v、期待値 = nil", token)
		}
	})
}

func TestOAuthAccessTokenRepository_RevokeByOAuthGrant(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthAccessTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "access-revoke")
	other := setupOAuthFixture(t, tx, "access-revoke-other")
	grantID := f.buildGrant(t, tx)
	otherGrantID := other.buildGrant(t, tx)

	revokedAt := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	now := time.Now().Truncate(time.Microsecond)
	newToken := func(grantID model.OAuthGrantID, spaceID model.SpaceID, digest string) {
		testutil.NewOAuthAccessTokenBuilder(t, tx).
			WithOAuthGrantID(grantID).
			WithSpaceID(spaceID).
			WithTokenDigest(digest).
			Build()
	}
	newToken(grantID, f.spaceID, "access_revoke_a_digest")
	newToken(grantID, f.spaceID, "access_revoke_b_digest")
	newToken(otherGrantID, other.spaceID, "access_revoke_other_digest")
	testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest("access_revoke_revoked_digest").
		WithRevokedAt(revokedAt).
		Build()

	// 許可と違うスペースを指定したら失効しない。失効していれば、後の確認で失効日時がnowではなくなる
	if err := repo.RevokeByOAuthGrant(ctx, grantID, other.spaceID, now.Add(-time.Minute)); err != nil {
		t.Fatalf("他のスペースを指定したRevokeByOAuthGrant()のエラー = %v", err)
	}
	if err := repo.RevokeByOAuthGrant(ctx, grantID, f.spaceID, now); err != nil {
		t.Fatalf("RevokeByOAuthGrant()のエラー = %v", err)
	}

	tests := []struct {
		digest        string
		wantRevokedAt *time.Time
	}{
		{digest: "access_revoke_a_digest", wantRevokedAt: &now},
		{digest: "access_revoke_b_digest", wantRevokedAt: &now},
		{digest: "access_revoke_revoked_digest", wantRevokedAt: &revokedAt},
		{digest: "access_revoke_other_digest"},
	}
	for _, tt := range tests {
		token, err := repo.FindByTokenDigest(ctx, tt.digest)
		if err != nil {
			t.Fatalf("%s: FindByTokenDigest()のエラー = %v", tt.digest, err)
		}
		switch {
		case tt.wantRevokedAt == nil && token.RevokedAt != nil:
			t.Errorf("%s: token.RevokedAt = %v、期待値 = nil", tt.digest, token.RevokedAt)
		case tt.wantRevokedAt != nil && (token.RevokedAt == nil || !token.RevokedAt.Equal(*tt.wantRevokedAt)):
			t.Errorf("%s: token.RevokedAt = %v、期待値 = %v", tt.digest, token.RevokedAt, *tt.wantRevokedAt)
		}
	}
}

func TestOAuthAccessTokenRepository_Revoke(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthAccessTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "access-revoke-one")
	other := setupOAuthFixture(t, tx, "access-revoke-one-other")
	grantID := f.buildGrant(t, tx)

	revokedAt := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	now := time.Now().Truncate(time.Microsecond)
	targetID := testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest("access_revoke_one_target_digest").Build()
	testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest("access_revoke_one_sibling_digest").Build()
	revokedID := testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest("access_revoke_one_revoked_digest").
		WithRevokedAt(revokedAt).Build()

	// トークンと違うスペースを指定したら失効しない
	if err := repo.Revoke(ctx, targetID, other.spaceID, now.Add(-time.Minute)); err != nil {
		t.Fatalf("他のスペースを指定したRevoke()のエラー = %v", err)
	}
	if err := repo.Revoke(ctx, targetID, f.spaceID, now); err != nil {
		t.Fatalf("Revoke()のエラー = %v", err)
	}
	if err := repo.Revoke(ctx, revokedID, f.spaceID, now); err != nil {
		t.Fatalf("失効済みのトークンのRevoke()のエラー = %v", err)
	}

	tests := []struct {
		digest        string
		wantRevokedAt *time.Time
	}{
		{digest: "access_revoke_one_target_digest", wantRevokedAt: &now},
		// 同じ許可のほかのアクセストークンは失効しない
		{digest: "access_revoke_one_sibling_digest"},
		{digest: "access_revoke_one_revoked_digest", wantRevokedAt: &revokedAt},
	}
	for _, tt := range tests {
		token, err := repo.FindByTokenDigest(ctx, tt.digest)
		if err != nil {
			t.Fatalf("%s: FindByTokenDigest()のエラー = %v", tt.digest, err)
		}
		switch {
		case tt.wantRevokedAt == nil && token.RevokedAt != nil:
			t.Errorf("%s: token.RevokedAt = %v、期待値 = nil", tt.digest, token.RevokedAt)
		case tt.wantRevokedAt != nil && (token.RevokedAt == nil || !token.RevokedAt.Equal(*tt.wantRevokedAt)):
			t.Errorf("%s: token.RevokedAt = %v、期待値 = %v", tt.digest, token.RevokedAt, *tt.wantRevokedAt)
		}
	}
}
