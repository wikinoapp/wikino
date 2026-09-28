package repository

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestOAuthRefreshTokenRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthRefreshTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "refresh-create")
	grantID := f.buildGrant(t, tx)
	expiresAt := time.Now().Add(90 * 24 * time.Hour).Truncate(time.Microsecond)
	scopes := []model.Scope{model.ScopePageRead, model.ScopePageWrite}

	token, err := repo.Create(ctx, CreateOAuthRefreshTokenInput{
		OAuthGrantID: grantID,
		SpaceID:      f.spaceID,
		TokenDigest:  "refresh_create_digest",
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
	if token.TokenDigest != "refresh_create_digest" {
		t.Errorf("token.TokenDigest = %q、期待値 = %q", token.TokenDigest, "refresh_create_digest")
	}
	if !slices.Equal(token.Scopes, scopes) {
		t.Errorf("token.Scopes = %v、期待値 = %v", token.Scopes, scopes)
	}
	if token.PreviousRefreshTokenID != nil {
		t.Errorf("token.PreviousRefreshTokenID = %v、期待値 = nil", *token.PreviousRefreshTokenID)
	}
	if !token.ExpiresAt.Equal(expiresAt) {
		t.Errorf("token.ExpiresAt = %v、期待値 = %v", token.ExpiresAt, expiresAt)
	}
	if !token.IsActive(time.Now()) {
		t.Error("token.IsActive() = false、期待値 = true")
	}

	// 許可と違うスペースでは作れない (トランザクションが中断するため最後に確かめる)
	other := setupOAuthFixture(t, tx, "refresh-create-other")
	_, err = repo.Create(ctx, CreateOAuthRefreshTokenInput{
		OAuthGrantID: grantID,
		SpaceID:      other.spaceID,
		TokenDigest:  "refresh_create_other_space_digest",
		Scopes:       scopes,
		ExpiresAt:    expiresAt,
	})
	assertConstraintViolation(t, err, "foreign_key_violation", "oauth_refresh_tokens_oauth_grant_id_space_id_fkey")
}

func TestOAuthRefreshTokenRepository_FindByTokenDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthRefreshTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "refresh-find")
	grantID := f.buildGrant(t, tx)
	usedID := testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest("refresh_find_used_digest").
		WithUsedAt(time.Now()).
		Build()

	t.Run("使用済みのトークンも返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(ctx, "refresh_find_used_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token == nil || token.ID != usedID {
			t.Fatalf("FindByTokenDigest() = %v、期待値 = ID %v のトークン", token, usedID)
		}
		if !token.IsUsed() {
			t.Error("token.IsUsed() = false、期待値 = true")
		}
	})

	t.Run("一致するトークンが無ければnilを返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(ctx, "refresh_find_missing_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token != nil {
			t.Errorf("FindByTokenDigest() = %v、期待値 = nil", token)
		}
	})
}

func TestOAuthRefreshTokenRepository_Rotate(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewOAuthRefreshTokenRepository(q)
	accessTokenRepo := NewOAuthAccessTokenRepository(q)
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "refresh-rotate")
	other := setupOAuthFixture(t, tx, "refresh-rotate-other")
	grantID := f.buildGrant(t, tx)
	scopes := []model.Scope{model.ScopePageRead, model.ScopeTopicRead}
	now := time.Now().Truncate(time.Microsecond)
	expiresAt := now.Add(90 * 24 * time.Hour)
	accessExpiresAt := now.Add(time.Hour)
	// アクセストークンのスコープは、元のスコープを狭めた要求に応じて呼び出し側が決める
	accessScopes := []model.Scope{model.ScopePageRead}

	rotateInput := func(id model.OAuthRefreshTokenID, spaceID model.SpaceID, digest string) RotateOAuthRefreshTokenInput {
		return RotateOAuthRefreshTokenInput{
			ID:                   id,
			SpaceID:              spaceID,
			TokenDigest:          digest,
			ExpiresAt:            expiresAt,
			AccessTokenDigest:    digest + "_access",
			AccessTokenScopes:    accessScopes,
			AccessTokenExpiresAt: accessExpiresAt,
			Now:                  now,
		}
	}

	newToken := func(digest string) *testutil.OAuthRefreshTokenBuilder {
		return testutil.NewOAuthRefreshTokenBuilder(t, tx).
			WithOAuthGrantID(grantID).
			WithSpaceID(f.spaceID).
			WithTokenDigest(digest).
			WithScopes(scopes)
	}

	t.Run("使用済みにして同じ許可・スコープの新しいトークンを返す", func(t *testing.T) {
		id := newToken("refresh_rotate_digest").Build()

		rotated, err := repo.Rotate(ctx, rotateInput(id, f.spaceID, "refresh_rotate_new_digest"))
		if err != nil {
			t.Fatalf("Rotate()のエラー = %v", err)
		}
		if rotated == nil {
			t.Fatal("Rotate()がnilを返した")
		}
		if rotated.ID == id {
			t.Error("rotated.IDが元のトークンと同じ")
		}
		if rotated.OAuthGrantID != grantID {
			t.Errorf("rotated.OAuthGrantID = %v、期待値 = %v", rotated.OAuthGrantID, grantID)
		}
		if rotated.SpaceID != f.spaceID {
			t.Errorf("rotated.SpaceID = %v、期待値 = %v", rotated.SpaceID, f.spaceID)
		}
		if rotated.TokenDigest != "refresh_rotate_new_digest" {
			t.Errorf("rotated.TokenDigest = %q、期待値 = %q", rotated.TokenDigest, "refresh_rotate_new_digest")
		}
		if !slices.Equal(rotated.Scopes, scopes) {
			t.Errorf("rotated.Scopes = %v、期待値 = %v", rotated.Scopes, scopes)
		}
		if rotated.PreviousRefreshTokenID == nil || *rotated.PreviousRefreshTokenID != id {
			t.Errorf("rotated.PreviousRefreshTokenID = %v、期待値 = %v", rotated.PreviousRefreshTokenID, id)
		}
		if !rotated.ExpiresAt.Equal(expiresAt) {
			t.Errorf("rotated.ExpiresAt = %v、期待値 = %v", rotated.ExpiresAt, expiresAt)
		}
		if !rotated.IsActive(now) {
			t.Error("rotated.IsActive() = false、期待値 = true")
		}

		previous, err := repo.FindByTokenDigest(ctx, "refresh_rotate_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if previous.UsedAt == nil || !previous.UsedAt.Equal(now) {
			t.Errorf("previous.UsedAt = %v、期待値 = %v", previous.UsedAt, now)
		}

		accessToken, err := accessTokenRepo.FindByTokenDigest(ctx, "refresh_rotate_new_digest_access")
		if err != nil {
			t.Fatalf("アクセストークンのFindByTokenDigest()のエラー = %v", err)
		}
		if accessToken == nil {
			t.Fatal("アクセストークンが作られていない")
		}
		if accessToken.OAuthGrantID != grantID || accessToken.SpaceID != f.spaceID {
			t.Errorf("アクセストークンの許可・スペース = (%v, %v)、期待値 = (%v, %v)", accessToken.OAuthGrantID, accessToken.SpaceID, grantID, f.spaceID)
		}
		if !slices.Equal(accessToken.Scopes, accessScopes) {
			t.Errorf("アクセストークンのScopes = %v、期待値 = %v", accessToken.Scopes, accessScopes)
		}
		if !accessToken.ExpiresAt.Equal(accessExpiresAt) {
			t.Errorf("アクセストークンのExpiresAt = %v、期待値 = %v", accessToken.ExpiresAt, accessExpiresAt)
		}

		again, err := repo.Rotate(ctx, rotateInput(id, f.spaceID, "refresh_rotate_again_digest"))
		if err != nil {
			t.Fatalf("2回目のRotate()のエラー = %v", err)
		}
		if again != nil {
			t.Errorf("使用済みのトークンに対するRotate() = %v、期待値 = nil", again)
		}
		againAccessToken, err := accessTokenRepo.FindByTokenDigest(ctx, "refresh_rotate_again_digest_access")
		if err != nil {
			t.Fatalf("アクセストークンのFindByTokenDigest()のエラー = %v", err)
		}
		if againAccessToken != nil {
			t.Errorf("使用済みのトークンでアクセストークンが作られた: %v", againAccessToken)
		}
	})

	t.Run("ローテーションしない場合はnilを返しトークンに触れない", func(t *testing.T) {
		tests := []struct {
			name    string
			id      model.OAuthRefreshTokenID
			spaceID model.SpaceID
		}{
			{name: "失効したトークン", id: newToken("refresh_rotate_revoked_digest").WithRevokedAt(now).Build(), spaceID: f.spaceID},
			{name: "期限切れのトークン", id: newToken("refresh_rotate_expired_digest").WithExpiresAt(now).Build(), spaceID: f.spaceID},
			{name: "他のスペースを指定した", id: newToken("refresh_rotate_space_digest").Build(), spaceID: other.spaceID},
		}
		for i, tt := range tests {
			digest := "refresh_rotate_rejected_" + string(rune('a'+i)) + "_digest"
			rotated, err := repo.Rotate(ctx, rotateInput(tt.id, tt.spaceID, digest))
			if err != nil {
				t.Fatalf("%s: Rotate()のエラー = %v", tt.name, err)
			}
			if rotated != nil {
				t.Errorf("%s: Rotate() = %v、期待値 = nil", tt.name, rotated)
			}

			created, err := repo.FindByTokenDigest(ctx, digest)
			if err != nil {
				t.Fatalf("%s: FindByTokenDigest()のエラー = %v", tt.name, err)
			}
			if created != nil {
				t.Errorf("%s: 新しいトークン = %v、期待値 = nil", tt.name, created)
			}
			accessToken, err := accessTokenRepo.FindByTokenDigest(ctx, digest+"_access")
			if err != nil {
				t.Fatalf("%s: アクセストークンのFindByTokenDigest()のエラー = %v", tt.name, err)
			}
			if accessToken != nil {
				t.Errorf("%s: アクセストークン = %v、期待値 = nil", tt.name, accessToken)
			}
		}

		token, err := repo.FindByTokenDigest(ctx, "refresh_rotate_space_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token.IsUsed() {
			t.Error("他のスペースを指定したトークンのIsUsed() = true、期待値 = false")
		}
	})
}

func TestOAuthRefreshTokenRepository_RevokeByOAuthGrant(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthRefreshTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "refresh-revoke")
	other := setupOAuthFixture(t, tx, "refresh-revoke-other")
	grantID := f.buildGrant(t, tx)
	otherGrantID := other.buildGrant(t, tx)

	revokedAt := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	now := time.Now().Truncate(time.Microsecond)
	testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest("refresh_revoke_active_digest").
		Build()
	testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest("refresh_revoke_used_digest").
		WithUsedAt(revokedAt).
		Build()
	testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest("refresh_revoke_revoked_digest").
		WithRevokedAt(revokedAt).
		Build()
	testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(otherGrantID).
		WithSpaceID(other.spaceID).
		WithTokenDigest("refresh_revoke_other_digest").
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
		{digest: "refresh_revoke_active_digest", wantRevokedAt: &now},
		{digest: "refresh_revoke_used_digest", wantRevokedAt: &now},
		{digest: "refresh_revoke_revoked_digest", wantRevokedAt: &revokedAt},
		{digest: "refresh_revoke_other_digest"},
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

func TestOAuthRefreshTokenRepository_Revoke(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-time.Hour).Truncate(time.Microsecond)

	tests := []struct {
		name string
		// buildは失効を求めるリフレッシュトークンの状態を設定する
		build func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder
		// otherSpaceが真なら、トークンと違うスペースを指定する
		otherSpace bool
		wantRevoke bool
	}{
		{name: "有効なトークンを失効し同じ許可のアクセストークンも失効する", build: func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder { return b }, wantRevoke: true},
		{name: "使用済みのトークンは変えない", build: func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder {
			return b.WithUsedAt(past)
		}},
		{name: "期限切れのトークンは変えない", build: func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder {
			return b.WithExpiresAt(past)
		}},
		{name: "失効済みのトークンは変えない", build: func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder {
			return b.WithRevokedAt(past)
		}},
		{name: "違うスペースを指定したら変えない", build: func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder { return b }, otherSpace: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			repo := NewOAuthRefreshTokenRepository(q)
			accessRepo := NewOAuthAccessTokenRepository(q)
			ctx := context.Background()

			suffix := fmt.Sprintf("refresh-revoke-one-%d", i)
			f := setupOAuthFixture(t, tx, suffix)
			other := setupOAuthFixture(t, tx, suffix+"-other")
			grantID := f.buildGrant(t, tx)
			now := time.Now().Truncate(time.Microsecond)

			targetID := tt.build(testutil.NewOAuthRefreshTokenBuilder(t, tx).
				WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_target")).Build()
			testutil.NewOAuthRefreshTokenBuilder(t, tx).
				WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_sibling").Build()
			testutil.NewOAuthAccessTokenBuilder(t, tx).
				WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_access").Build()
			otherGrantID := other.buildGrant(t, tx)
			testutil.NewOAuthAccessTokenBuilder(t, tx).
				WithOAuthGrantID(otherGrantID).WithSpaceID(other.spaceID).WithTokenDigest(suffix + "_other_access").Build()

			spaceID := f.spaceID
			if tt.otherSpace {
				spaceID = other.spaceID
			}
			if err := repo.Revoke(ctx, targetID, spaceID, now); err != nil {
				t.Fatalf("Revoke()のエラー = %v", err)
			}

			target, err := repo.FindByTokenDigest(ctx, suffix+"_target")
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			if revoked := target.RevokedAt != nil && target.RevokedAt.Equal(now); revoked != tt.wantRevoke {
				t.Errorf("失効を求めたトークンのRevokedAt = %v、nowで失効することの期待値 = %v", target.RevokedAt, tt.wantRevoke)
			}
			access, err := accessRepo.FindByTokenDigest(ctx, suffix+"_access")
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			if access.IsRevoked() != tt.wantRevoke {
				t.Errorf("同じ許可のアクセストークンのIsRevoked() = %v、期待値 = %v", access.IsRevoked(), tt.wantRevoke)
			}

			// 同じ許可のほかのリフレッシュトークン (別の端末のもの) と、ほかの許可のアクセストークンは失効しない
			sibling, err := repo.FindByTokenDigest(ctx, suffix+"_sibling")
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			if sibling.IsRevoked() {
				t.Error("同じ許可のほかのリフレッシュトークンが失効した")
			}
			otherAccess, err := accessRepo.FindByTokenDigest(ctx, suffix+"_other_access")
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			if otherAccess.IsRevoked() {
				t.Error("ほかの許可のアクセストークンが失効した")
			}
		})
	}
}
