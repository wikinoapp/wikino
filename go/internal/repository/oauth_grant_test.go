package repository

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// oauthFixtureはOAuthのテストで必要になるスペース・スペースメンバー・OAuthアプリ
type oauthFixture struct {
	spaceID            model.SpaceID
	spaceMemberID      model.SpaceMemberID
	oauthApplicationID model.OAuthApplicationID
}

// setupOAuthFixtureはユーザー・スペース・そのメンバー・メンバーがスペースに登録した
// OAuthアプリを作成する。テストは並行に走るため、一意性のある列 (メールアドレス・アットネーム・
// スペース識別子・クライアントID) をsuffixで区別する。
func setupOAuthFixture(t *testing.T, tx *sql.Tx, suffix string) oauthFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("oauth-" + suffix + "@example.com").
		WithAtname("oauth_" + suffix).
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("oauth-" + suffix + "-space").
		WithName("OAuth " + suffix).
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	oauthApplicationID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(spaceID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithClientID("oauth-" + suffix + "-client").
		Build()

	return oauthFixture{
		spaceID:            spaceID,
		spaceMemberID:      spaceMemberID,
		oauthApplicationID: oauthApplicationID,
	}
}

// buildGrantはフィクスチャのアプリ・メンバーの許可を作成する
func (f oauthFixture) buildGrant(t *testing.T, tx *sql.Tx) model.OAuthGrantID {
	t.Helper()

	return testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		Build()
}

// assertConstraintViolationは、errが指定した制約へのPostgreSQLの違反であることを確かめる
func assertConstraintViolation(t *testing.T, err error, codeName, constraint string) {
	t.Helper()

	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code.Name() != codeName || pqErr.Constraint != constraint {
		t.Errorf("エラー = %v、期待値 = %sの%s", err, constraint, codeName)
	}
}

func TestOAuthGrantRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "grant-create")
	scopes := []model.Scope{model.ScopePageRead, model.ScopeTopicRead}

	grant, err := repo.Create(ctx, CreateOAuthGrantInput{
		OAuthApplicationID: f.oauthApplicationID,
		SpaceID:            f.spaceID,
		SpaceMemberID:      f.spaceMemberID,
		Scopes:             scopes,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if grant.ID == "" {
		t.Error("grant.IDが空")
	}
	if grant.OAuthApplicationID != f.oauthApplicationID {
		t.Errorf("grant.OAuthApplicationID = %v、期待値 = %v", grant.OAuthApplicationID, f.oauthApplicationID)
	}
	if grant.SpaceID != f.spaceID {
		t.Errorf("grant.SpaceID = %v、期待値 = %v", grant.SpaceID, f.spaceID)
	}
	if grant.SpaceMemberID != f.spaceMemberID {
		t.Errorf("grant.SpaceMemberID = %v、期待値 = %v", grant.SpaceMemberID, f.spaceMemberID)
	}
	if !slices.Equal(grant.Scopes, scopes) {
		t.Errorf("grant.Scopes = %v、期待値 = %v", grant.Scopes, scopes)
	}
	if grant.IsRevoked() {
		t.Error("grant.IsRevoked() = true、期待値 = false")
	}

	// 失効した許可があっても、同じアプリ・メンバーに新しい許可を作れる
	revoked := setupOAuthFixture(t, tx, "grant-create-revoked")
	testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(revoked.oauthApplicationID).
		WithSpaceID(revoked.spaceID).
		WithSpaceMemberID(revoked.spaceMemberID).
		WithRevokedAt(time.Now()).
		Build()
	if _, err := repo.Create(ctx, CreateOAuthGrantInput{
		OAuthApplicationID: revoked.oauthApplicationID,
		SpaceID:            revoked.spaceID,
		SpaceMemberID:      revoked.spaceMemberID,
		Scopes:             scopes,
	}); err != nil {
		t.Errorf("失効した許可があるときのCreate()のエラー = %v", err)
	}

	// 失効していない許可が既にあれば一意制約に違反する (トランザクションが中断するため最後に確かめる)
	_, err = repo.Create(ctx, CreateOAuthGrantInput{
		OAuthApplicationID: f.oauthApplicationID,
		SpaceID:            f.spaceID,
		SpaceMemberID:      f.spaceMemberID,
		Scopes:             scopes,
	})
	assertConstraintViolation(t, err, "unique_violation", "idx_oauth_grants_unrevoked_application_member")
}

func TestOAuthGrantRepository_Create_OtherSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))

	f := setupOAuthFixture(t, tx, "grant-create-fk")
	other := setupOAuthFixture(t, tx, "grant-create-fk-other")

	_, err := repo.Create(context.Background(), CreateOAuthGrantInput{
		OAuthApplicationID: f.oauthApplicationID,
		SpaceID:            f.spaceID,
		SpaceMemberID:      other.spaceMemberID,
		Scopes:             []model.Scope{model.ScopePageRead},
	})
	assertConstraintViolation(t, err, "foreign_key_violation", "oauth_grants_space_member_id_space_id_fkey")
}

func TestOAuthGrantRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "grant-find")
	other := setupOAuthFixture(t, tx, "grant-find-other")
	id := f.buildGrant(t, tx)

	grant, err := repo.FindByID(ctx, id, f.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if grant == nil || grant.ID != id {
		t.Errorf("FindByID() = %v、期待値 = ID %v の許可", grant, id)
	}

	grant, err = repo.FindByID(ctx, id, other.spaceID)
	if err != nil {
		t.Fatalf("他のスペースを指定したFindByID()のエラー = %v", err)
	}
	if grant != nil {
		t.Errorf("他のスペースを指定したFindByID() = %v、期待値 = nil", grant)
	}
}

func TestOAuthGrantRepository_FindUnrevokedByApplicationAndSpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "grant-unrevoked")
	testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithRevokedAt(time.Now()).
		Build()

	t.Run("失効した許可しか無ければnilを返す", func(t *testing.T) {
		grant, err := repo.FindUnrevokedByApplicationAndSpaceMember(ctx, f.oauthApplicationID, f.spaceID, f.spaceMemberID)
		if err != nil {
			t.Fatalf("FindUnrevokedByApplicationAndSpaceMember()のエラー = %v", err)
		}
		if grant != nil {
			t.Errorf("FindUnrevokedByApplicationAndSpaceMember() = %v、期待値 = nil", grant)
		}
	})

	t.Run("失効していない許可を返す", func(t *testing.T) {
		id := f.buildGrant(t, tx)

		grant, err := repo.FindUnrevokedByApplicationAndSpaceMember(ctx, f.oauthApplicationID, f.spaceID, f.spaceMemberID)
		if err != nil {
			t.Fatalf("FindUnrevokedByApplicationAndSpaceMember()のエラー = %v", err)
		}
		if grant == nil || grant.ID != id {
			t.Errorf("FindUnrevokedByApplicationAndSpaceMember() = %v、期待値 = ID %v の許可", grant, id)
		}
	})
}

func TestOAuthGrantRepository_Revoke(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "grant-revoke")
	other := setupOAuthFixture(t, tx, "grant-revoke-other")

	t.Run("自分の許可を失効する", func(t *testing.T) {
		id := f.buildGrant(t, tx)

		grant, err := repo.Revoke(ctx, id, f.spaceID, f.spaceMemberID)
		if err != nil {
			t.Fatalf("Revoke()のエラー = %v", err)
		}
		if grant == nil {
			t.Fatal("Revoke()がnilを返した")
		}
		if !grant.IsRevoked() {
			t.Error("grant.IsRevoked() = false、期待値 = true")
		}

		again, err := repo.Revoke(ctx, id, f.spaceID, f.spaceMemberID)
		if err != nil {
			t.Fatalf("2回目のRevoke()のエラー = %v", err)
		}
		if again != nil {
			t.Errorf("失効済みの許可に対するRevoke() = %v、期待値 = nil", again)
		}
	})

	t.Run("許可に属するトークンも失効する", func(t *testing.T) {
		id := f.buildGrant(t, tx)
		revokedAt := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
		testutil.NewOAuthAccessTokenBuilder(t, tx).
			WithOAuthGrantID(id).WithSpaceID(f.spaceID).WithTokenDigest("grant_revoke_access_digest").Build()
		testutil.NewOAuthAccessTokenBuilder(t, tx).
			WithOAuthGrantID(id).WithSpaceID(f.spaceID).WithTokenDigest("grant_revoke_revoked_access_digest").
			WithRevokedAt(revokedAt).Build()
		testutil.NewOAuthRefreshTokenBuilder(t, tx).
			WithOAuthGrantID(id).WithSpaceID(f.spaceID).WithTokenDigest("grant_revoke_refresh_digest").Build()

		grant, err := repo.Revoke(ctx, id, f.spaceID, f.spaceMemberID)
		if err != nil {
			t.Fatalf("Revoke()のエラー = %v", err)
		}

		q := testutil.QueriesWithTx(tx)
		accessRepo := NewOAuthAccessTokenRepository(q)
		accessToken, err := accessRepo.FindByTokenDigest(ctx, "grant_revoke_access_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if accessToken.RevokedAt == nil || !accessToken.RevokedAt.Equal(*grant.RevokedAt) {
			t.Errorf("アクセストークンのRevokedAt = %v、期待値 = %v", accessToken.RevokedAt, *grant.RevokedAt)
		}
		revokedAccessToken, err := accessRepo.FindByTokenDigest(ctx, "grant_revoke_revoked_access_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if revokedAccessToken.RevokedAt == nil || !revokedAccessToken.RevokedAt.Equal(revokedAt) {
			t.Errorf("失効済みのアクセストークンのRevokedAt = %v、期待値 = %v (変えない)", revokedAccessToken.RevokedAt, revokedAt)
		}
		refreshToken, err := NewOAuthRefreshTokenRepository(q).FindByTokenDigest(ctx, "grant_revoke_refresh_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if refreshToken.RevokedAt == nil || !refreshToken.RevokedAt.Equal(*grant.RevokedAt) {
			t.Errorf("リフレッシュトークンのRevokedAt = %v、期待値 = %v", refreshToken.RevokedAt, *grant.RevokedAt)
		}
	})

	t.Run("失効しない場合はnilを返し許可とトークンに触れない", func(t *testing.T) {
		id := other.buildGrant(t, tx)
		testutil.NewOAuthAccessTokenBuilder(t, tx).
			WithOAuthGrantID(id).WithSpaceID(other.spaceID).WithTokenDigest("grant_revoke_other_access_digest").Build()

		tests := []struct {
			name          string
			id            model.OAuthGrantID
			spaceID       model.SpaceID
			spaceMemberID model.SpaceMemberID
		}{
			{name: "他のメンバーの許可", id: id, spaceID: other.spaceID, spaceMemberID: f.spaceMemberID},
			{name: "他のスペースを指定した", id: id, spaceID: f.spaceID, spaceMemberID: other.spaceMemberID},
			{name: "UUIDでないID", id: "not-a-uuid", spaceID: other.spaceID, spaceMemberID: other.spaceMemberID},
		}
		for _, tt := range tests {
			grant, err := repo.Revoke(ctx, tt.id, tt.spaceID, tt.spaceMemberID)
			if err != nil {
				t.Fatalf("%s: Revoke()のエラー = %v", tt.name, err)
			}
			if grant != nil {
				t.Errorf("%s: Revoke() = %v、期待値 = nil", tt.name, grant)
			}
		}

		grant, err := repo.FindByID(ctx, id, other.spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if grant.IsRevoked() {
			t.Error("grant.IsRevoked() = true、期待値 = false")
		}
		accessToken, err := NewOAuthAccessTokenRepository(testutil.QueriesWithTx(tx)).FindByTokenDigest(ctx, "grant_revoke_other_access_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if accessToken.IsRevoked() {
			t.Error("accessToken.IsRevoked() = true、期待値 = false")
		}
	})
}

func TestOAuthGrantRepository_ListUnrevokedBySpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "grant-list")
	other := setupOAuthFixture(t, tx, "grant-list-other")

	older := f.buildGrant(t, tx)
	// 同じアプリ・メンバーに有効な許可は1つに限るため、2件目は別のアプリの許可にする
	secondAppID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).WithClientID("oauth-grant-list-second-client").Build()
	newer := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(secondAppID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).Build()
	testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).
		WithRevokedAt(time.Now()).Build()
	other.buildGrant(t, tx)

	grants, err := repo.ListUnrevokedBySpaceMember(ctx, f.spaceID, f.spaceMemberID)
	if err != nil {
		t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
	}
	gotIDs := make([]model.OAuthGrantID, len(grants))
	for i, g := range grants {
		gotIDs[i] = g.ID
	}
	// 新しい順のため、後に作った許可が先に並ぶ
	if want := []model.OAuthGrantID{newer, older}; !slices.Equal(gotIDs, want) {
		t.Errorf("許可のID = %v、期待値 = %v (失効した許可と他のメンバーの許可は含まない)", gotIDs, want)
	}

	otherSpace, err := repo.ListUnrevokedBySpaceMember(ctx, other.spaceID, f.spaceMemberID)
	if err != nil {
		t.Fatalf("他のスペースを指定したListUnrevokedBySpaceMember()のエラー = %v", err)
	}
	if len(otherSpace) != 0 {
		t.Errorf("他のスペースを指定したときの件数 = %d、期待値 = 0", len(otherSpace))
	}
}
