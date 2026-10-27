package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

	// 許可は認可ごとに作るため、失効していない許可があっても同じアプリ・メンバーに別の許可を作れる
	another, err := repo.Create(ctx, CreateOAuthGrantInput{
		OAuthApplicationID: f.oauthApplicationID,
		SpaceID:            f.spaceID,
		SpaceMemberID:      f.spaceMemberID,
		Scopes:             []model.Scope{model.ScopePageRead},
	})
	if err != nil {
		t.Fatalf("失効していない許可があるときのCreate()のエラー = %v", err)
	}
	if another.ID == grant.ID {
		t.Errorf("another.ID = %v、既存の許可とは別の許可を期待", another.ID)
	}
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
	newer := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).Build()
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

func TestOAuthGrantRepository_ListUsableBySpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()
	now := time.Now()
	past := now.Add(-time.Hour)

	f := setupOAuthFixture(t, tx, "grant-usable")
	other := setupOAuthFixture(t, tx, "grant-usable-other")
	build := func(name string, refresh func(b *testutil.OAuthRefreshTokenBuilder)) model.OAuthGrantID {
		id := f.buildGrant(t, tx)
		if refresh != nil {
			b := testutil.NewOAuthRefreshTokenBuilder(t, tx).
				WithOAuthGrantID(id).WithSpaceID(f.spaceID).WithTokenDigest("grant_usable_" + name)
			refresh(b)
			b.Build()
		}
		return id
	}

	older := build("older", func(*testutil.OAuthRefreshTokenBuilder) {})
	// 使用済みのトークン (ローテーション前のもの) があっても、有効なトークンが1つあれば出す
	newer := build("newer_used", func(b *testutil.OAuthRefreshTokenBuilder) { b.WithUsedAt(past) })
	testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(newer).WithSpaceID(f.spaceID).WithTokenDigest("grant_usable_newer").Build()
	build("unexchanged", nil)
	build("used", func(b *testutil.OAuthRefreshTokenBuilder) { b.WithUsedAt(past) })
	build("revoked", func(b *testutil.OAuthRefreshTokenBuilder) { b.WithRevokedAt(past) })
	build("expired", func(b *testutil.OAuthRefreshTokenBuilder) { b.WithExpiresAt(past) })
	revokedGrant := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).
		WithRevokedAt(now).Build()
	testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(revokedGrant).WithSpaceID(f.spaceID).WithTokenDigest("grant_usable_revoked_grant").Build()

	grants, err := repo.ListUsableBySpaceMember(ctx, f.spaceID, f.spaceMemberID, now)
	if err != nil {
		t.Fatalf("ListUsableBySpaceMember()のエラー = %v", err)
	}
	gotIDs := make([]model.OAuthGrantID, len(grants))
	for i, g := range grants {
		gotIDs[i] = g.ID
	}
	if want := []model.OAuthGrantID{newer, older}; !slices.Equal(gotIDs, want) {
		t.Errorf("許可のID = %v、期待値 = %v (有効なリフレッシュトークンを持つ失効していない許可だけ)", gotIDs, want)
	}

	otherSpace, err := repo.ListUsableBySpaceMember(ctx, other.spaceID, f.spaceMemberID, now)
	if err != nil {
		t.Fatalf("他のスペースを指定したListUsableBySpaceMember()のエラー = %v", err)
	}
	if len(otherSpace) != 0 {
		t.Errorf("他のスペースを指定したときの件数 = %d、期待値 = 0", len(otherSpace))
	}
}

func TestOAuthGrantRepository_LastUsedAt(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)
	recent := now.Add(-model.OAuthGrantLastUsedAtUpdateInterval / 2)
	stale := now.Add(-2 * model.OAuthGrantLastUsedAtUpdateInterval)

	f := setupOAuthFixture(t, tx, "grant-last-used")
	build := func(lastUsedAt *time.Time, revoked bool) model.OAuthGrantID {
		b := testutil.NewOAuthGrantBuilder(t, tx).
			WithOAuthApplicationID(f.oauthApplicationID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID)
		if lastUsedAt != nil {
			b = b.WithLastUsedAt(*lastUsedAt)
		}
		if revoked {
			b = b.WithRevokedAt(stale)
		}
		return b.Build()
	}

	tests := []struct {
		name       string
		lastUsedAt *time.Time
		revoked    bool
		// setは、間隔を問わないSetLastUsedAtで更新するか (偽ならUpdateLastUsedAt)
		set  bool
		want *time.Time
	}{
		{name: "照合: まだ使われていなければ更新する", want: &now},
		{name: "照合: 間隔が過ぎていれば更新する", lastUsedAt: &stale, want: &now},
		{name: "照合: 間隔が過ぎていなければ更新しない", lastUsedAt: &recent, want: &recent},
		{name: "発行: 間隔が過ぎていなくても更新する", lastUsedAt: &recent, set: true, want: &now},
		{name: "失効した許可は更新しない", revoked: true, set: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := build(tt.lastUsedAt, tt.revoked)
			update := repo.UpdateLastUsedAt
			if tt.set {
				update = repo.SetLastUsedAt
			}
			if err := update(ctx, id, f.spaceID, now); err != nil {
				t.Fatalf("最終利用日時の更新のエラー = %v", err)
			}
			grant, err := repo.FindByID(ctx, id, f.spaceID)
			if err != nil {
				t.Fatalf("FindByID()のエラー = %v", err)
			}
			if (grant.LastUsedAt == nil) != (tt.want == nil) || (tt.want != nil && !grant.LastUsedAt.Equal(*tt.want)) {
				t.Errorf("LastUsedAt = %v、期待値 = %v", grant.LastUsedAt, tt.want)
			}
		})
	}
}

func TestOAuthGrantRepository_Evict(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Microsecond)
	// grantStateは押し出しの前の許可の状態。lastUsedAgoが0なら未使用で、createdAgoだけ前に作る
	type grantState struct {
		name        string
		createdAgo  time.Duration
		lastUsedAgo time.Duration
	}
	recentlyUsed := func(n int) []grantState {
		states := make([]grantState, n)
		for i := range states {
			states[i] = grantState{name: fmt.Sprintf("recent%d", i), createdAgo: time.Hour, lastUsedAgo: time.Duration(i+1) * time.Second}
		}
		return states
	}

	tests := []struct {
		name   string
		grants []grantState
		// wantEvictedは押し出される許可の名前
		wantEvicted []string
	}{
		{
			name:        "上限より少なければ押し出さない",
			grants:      recentlyUsed(model.OAuthGrantMaxPerApplicationMember - 1),
			wantEvicted: nil,
		},
		{
			name: "交換されないまま猶予を過ぎた許可を、最終利用が古い許可より先に押し出す",
			grants: append([]grantState{
				{name: "stale_unused", createdAgo: 2 * model.OAuthGrantEvictionGracePeriod},
				{name: "oldest_used", createdAgo: 2 * time.Hour, lastUsedAgo: time.Hour},
				{name: "fresh_unused", createdAgo: time.Second},
			}, recentlyUsed(model.OAuthGrantMaxPerApplicationMember-3)...),
			wantEvicted: []string{"stale_unused"},
		},
		{
			name: "交換されないまま猶予を過ぎた許可が無ければ、最終利用が最も古い許可を押し出す",
			grants: append([]grantState{
				{name: "oldest_used", createdAgo: 2 * time.Hour, lastUsedAgo: time.Hour},
				{name: "fresh_unused", createdAgo: time.Second},
			}, recentlyUsed(model.OAuthGrantMaxPerApplicationMember-2)...),
			wantEvicted: []string{"oldest_used"},
		},
		{
			name: "ほかに無ければ、猶予の中の未使用の許可を古いものから押し出す",
			grants: func() []grantState {
				states := make([]grantState, model.OAuthGrantMaxPerApplicationMember)
				for i := range states {
					states[i] = grantState{name: fmt.Sprintf("fresh%d", i), createdAgo: time.Duration(i+1) * time.Second}
				}
				return states
			}(),
			wantEvicted: []string{fmt.Sprintf("fresh%d", model.OAuthGrantMaxPerApplicationMember-1)},
		},
		{
			name: "同時の作成で上限を超えていれば、上限より1件少なくなるまで押し出す",
			grants: append([]grantState{
				{name: "oldest_used", createdAgo: 2 * time.Hour, lastUsedAgo: 3 * time.Hour},
				{name: "older_used", createdAgo: 2 * time.Hour, lastUsedAgo: 2 * time.Hour},
			}, recentlyUsed(model.OAuthGrantMaxPerApplicationMember)...),
			wantEvicted: []string{"oldest_used", "older_used", fmt.Sprintf("recent%d", model.OAuthGrantMaxPerApplicationMember-1)},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
			ctx := context.Background()
			suffix := fmt.Sprintf("grant-evict-%d", i)
			f := setupOAuthFixture(t, tx, suffix)

			ids := make(map[string]model.OAuthGrantID, len(tt.grants))
			for _, g := range tt.grants {
				id := f.buildGrant(t, tx)
				var lastUsedAt *time.Time
				if g.lastUsedAgo > 0 {
					v := now.Add(-g.lastUsedAgo)
					lastUsedAt = &v
				}
				if _, err := tx.ExecContext(ctx, "UPDATE oauth_grants SET created_at = $1, last_used_at = $2 WHERE id = $3 AND space_id = $4",
					now.Add(-g.createdAgo), lastUsedAt, string(id), string(f.spaceID)); err != nil {
					t.Fatalf("許可の日時の設定に失敗: %v", err)
				}
				testutil.NewOAuthRefreshTokenBuilder(t, tx).
					WithOAuthGrantID(id).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_" + g.name + "_refresh").Build()
				testutil.NewOAuthAccessTokenBuilder(t, tx).
					WithOAuthGrantID(id).WithSpaceID(f.spaceID).WithTokenDigest(suffix + "_" + g.name + "_access").Build()
				ids[g.name] = id
			}
			// 同じメンバーのほかのアプリの許可は、数えず押し出さない
			otherAppID := testutil.NewOAuthApplicationBuilder(t, tx).
				WithSpaceID(f.spaceID).WithClientID(suffix + "-other-client").Build()
			otherGrantID := testutil.NewOAuthGrantBuilder(t, tx).
				WithOAuthApplicationID(otherAppID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).Build()

			if err := repo.Evict(ctx, EvictInput{
				OAuthApplicationID: f.oauthApplicationID, SpaceID: f.spaceID, SpaceMemberID: f.spaceMemberID, Now: now,
			}); err != nil {
				t.Fatalf("Evict()のエラー = %v", err)
			}

			var evicted []string
			for _, g := range tt.grants {
				grant, err := repo.FindByID(ctx, ids[g.name], f.spaceID)
				if err != nil {
					t.Fatalf("FindByID()のエラー = %v", err)
				}
				var tokensRevoked int
				if err := tx.QueryRowContext(ctx, `SELECT
					(SELECT count(*) FROM oauth_access_tokens WHERE oauth_grant_id = $1 AND space_id = $2 AND revoked_at IS NOT NULL) +
					(SELECT count(*) FROM oauth_refresh_tokens WHERE oauth_grant_id = $1 AND space_id = $2 AND revoked_at IS NOT NULL)`,
					string(grant.ID), string(f.spaceID)).Scan(&tokensRevoked); err != nil {
					t.Fatalf("トークンの失効の取得に失敗: %v", err)
				}
				if grant.IsRevoked() {
					evicted = append(evicted, g.name)
					if tokensRevoked != 2 {
						t.Errorf("押し出した許可%sの失効したトークン = %d件、期待値 = 2件", g.name, tokensRevoked)
					}
				} else if tokensRevoked != 0 {
					t.Errorf("押し出していない許可%sのトークンが%d件失効した", g.name, tokensRevoked)
				}
			}
			slices.Sort(evicted)
			want := slices.Clone(tt.wantEvicted)
			slices.Sort(want)
			if !slices.Equal(evicted, want) {
				t.Errorf("押し出した許可 = %v、期待値 = %v", evicted, want)
			}
			other, err := repo.FindByID(ctx, otherGrantID, f.spaceID)
			if err != nil {
				t.Fatalf("FindByID()のエラー = %v", err)
			}
			if other.IsRevoked() {
				t.Error("ほかのアプリの許可を押し出した")
			}
		})
	}
}

func TestOAuthGrantRepository_LockByID(t *testing.T) {
	t.Parallel()
	_, tx := testutil.SetupTx(t)
	repo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	f := setupOAuthFixture(t, tx, "grant-lock-id")
	id := f.buildGrant(t, tx)
	ctx := t.Context()
	locked, err := repo.LockByID(ctx, id, f.spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if locked == nil || locked.ID != id {
		t.Fatalf("ロックした許可 = %v、期待するID = %s", locked, id)
	}
	other := setupOAuthFixture(t, tx, "grant-lock-id-other")
	locked, err = repo.LockByID(ctx, id, other.spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if locked != nil {
		t.Error("別のスペースの許可をロックできました")
	}
}
