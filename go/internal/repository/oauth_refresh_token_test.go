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
		Audience:     model.OAuthAudienceMCPServer,
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
	if token.Audience != model.OAuthAudienceMCPServer {
		t.Errorf("token.Audience = %v、期待値 = %v", token.Audience, model.OAuthAudienceMCPServer)
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

	t.Run("元のトークンの宛先を新しいトークンとアクセストークンに引き継ぐ", func(t *testing.T) {
		id := newToken("refresh_rotate_mcp_digest").WithAudience(model.OAuthAudienceMCPServer).Build()

		rotated, err := repo.Rotate(ctx, rotateInput(id, f.spaceID, "refresh_rotate_mcp_new_digest"))
		if err != nil {
			t.Fatalf("Rotate()のエラー = %v", err)
		}
		if rotated == nil || rotated.Audience != model.OAuthAudienceMCPServer {
			t.Fatalf("Rotate() = %+v、期待値 = MCPサーバー宛てのトークン", rotated)
		}
		accessToken, err := accessTokenRepo.FindByTokenDigest(ctx, "refresh_rotate_mcp_new_digest_access")
		if err != nil || accessToken == nil {
			t.Fatalf("アクセストークンの取得結果 = %+v、エラー = %v", accessToken, err)
		}
		if accessToken.Audience != model.OAuthAudienceMCPServer {
			t.Errorf("アクセストークンのAudience = %v、期待値 = %v", accessToken.Audience, model.OAuthAudienceMCPServer)
		}
	})

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

func TestOAuthRefreshTokenRepository_Revoke(t *testing.T) {
	t.Parallel()

	past := time.Now().Add(-time.Hour).Truncate(time.Microsecond)

	tests := []struct {
		name string
		// buildは失効を求めるリフレッシュトークンの状態を設定する
		build func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder
		// otherSpaceが真なら、トークンと違うスペースを指定する
		otherSpace   bool
		grantRevoked bool
		wantRevoke   bool
	}{
		{name: "有効なトークンを失効し、許可と許可に属するトークンもすべて失効する", build: func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder { return b }, wantRevoke: true},
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
		{name: "失効済みの許可に残る有効なトークンも失効し、許可の日時は変えない", build: func(b *testutil.OAuthRefreshTokenBuilder) *testutil.OAuthRefreshTokenBuilder { return b }, grantRevoked: true, wantRevoke: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			repo := NewOAuthRefreshTokenRepository(q)
			accessRepo := NewOAuthAccessTokenRepository(q)
			grantRepo := NewOAuthGrantRepository(q)
			ctx := context.Background()

			suffix := fmt.Sprintf("refresh-revoke-one-%d", i)
			f := setupOAuthFixture(t, tx, suffix)
			other := setupOAuthFixture(t, tx, suffix+"-other")
			grantID := f.buildGrant(t, tx)
			now := time.Now().Truncate(time.Microsecond)
			if tt.grantRevoked {
				if _, err := tx.ExecContext(ctx, "UPDATE oauth_grants SET revoked_at = $1, updated_at = $1 WHERE id = $2 AND space_id = $3", past, string(grantID), string(f.spaceID)); err != nil {
					t.Fatalf("許可の失効状態の設定に失敗: %v", err)
				}
			}

			targetID := tt.build(testutil.NewOAuthRefreshTokenBuilder(t, tx).
				WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_target")).Build()
			testutil.NewOAuthRefreshTokenBuilder(t, tx).
				WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_sibling").Build()
			testutil.NewOAuthAccessTokenBuilder(t, tx).
				WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_access").Build()
			// 同じアプリ・メンバーの別の許可 (別の端末のログイン) のトークン
			anotherGrantID := f.buildGrant(t, tx)
			testutil.NewOAuthRefreshTokenBuilder(t, tx).
				WithOAuthGrantID(anotherGrantID).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_another_refresh").Build()
			testutil.NewOAuthAccessTokenBuilder(t, tx).
				WithOAuthGrantID(anotherGrantID).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_another_access").Build()

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
			grant, err := grantRepo.FindByID(ctx, grantID, f.spaceID)
			if err != nil {
				t.Fatalf("FindByID()のエラー = %v", err)
			}
			if grant.IsRevoked() != tt.wantRevoke {
				t.Errorf("トークンの許可のIsRevoked() = %v、期待値 = %v", grant.IsRevoked(), tt.wantRevoke)
			}
			if tt.grantRevoked && (grant.RevokedAt == nil || !grant.RevokedAt.Equal(past) || !grant.UpdatedAt.Equal(past)) {
				t.Errorf("失効済みの許可の日時が変わった: revoked_at = %v、updated_at = %v、期待値 = %v", grant.RevokedAt, grant.UpdatedAt, past)
			}
			access, err := accessRepo.FindByTokenDigest(ctx, suffix+"_access")
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			if access.IsRevoked() != tt.wantRevoke {
				t.Errorf("同じ許可のアクセストークンのIsRevoked() = %v、期待値 = %v", access.IsRevoked(), tt.wantRevoke)
			}
			sibling, err := repo.FindByTokenDigest(ctx, suffix+"_sibling")
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			if sibling.IsRevoked() != tt.wantRevoke {
				t.Errorf("同じ許可のほかのリフレッシュトークンのIsRevoked() = %v、期待値 = %v", sibling.IsRevoked(), tt.wantRevoke)
			}

			// 同じアプリ・メンバーの別の許可とそのトークンは失効しない
			anotherGrant, err := grantRepo.FindByID(ctx, anotherGrantID, f.spaceID)
			if err != nil {
				t.Fatalf("FindByID()のエラー = %v", err)
			}
			if anotherGrant.IsRevoked() {
				t.Error("同じアプリ・メンバーの別の許可が失効した")
			}
			anotherRefresh, err := repo.FindByTokenDigest(ctx, suffix+"_another_refresh")
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			anotherAccess, err := accessRepo.FindByTokenDigest(ctx, suffix+"_another_access")
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			if anotherRefresh.IsRevoked() || anotherAccess.IsRevoked() {
				t.Error("同じアプリ・メンバーの別の許可のトークンが失効した")
			}
		})
	}
}
