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

// personalAccessTokenFixtureはどの個人アクセストークンのテストでも必要になるスペースと
// スペースメンバー
type personalAccessTokenFixture struct {
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
}

// setupPersonalAccessTokenFixtureはスペースとそのメンバーを作成する。テストは並行に走るため、
// 一意性のある列 (メールアドレス・アットネーム・スペース識別子) をsuffixで区別する。
func setupPersonalAccessTokenFixture(t *testing.T, tx *sql.Tx, suffix string) personalAccessTokenFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pat-" + suffix + "@example.com").
		WithAtname("pat_" + suffix).
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("pat-" + suffix + "-space").
		WithName("PAT " + suffix).
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	return personalAccessTokenFixture{spaceID: spaceID, spaceMemberID: spaceMemberID}
}

func TestPersonalAccessTokenRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewPersonalAccessTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupPersonalAccessTokenFixture(t, tx, "create")
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Microsecond)
	scopes := []model.Scope{model.ScopePageRead, model.ScopePageWrite}

	token, err := repo.Create(ctx, CreatePersonalAccessTokenInput{
		SpaceID:        f.spaceID,
		SpaceMemberID:  f.spaceMemberID,
		Name:           "CLI",
		TokenDigest:    "create_digest",
		TokenLastChars: "wxyz",
		Scopes:         scopes,
		ExpiresAt:      expiresAt,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if token == nil {
		t.Fatal("Create()がnilを返した")
	}
	if token.ID == "" {
		t.Error("token.IDが空")
	}
	if token.SpaceID != f.spaceID {
		t.Errorf("token.SpaceID = %v、期待値 = %v", token.SpaceID, f.spaceID)
	}
	if token.SpaceMemberID != f.spaceMemberID {
		t.Errorf("token.SpaceMemberID = %v、期待値 = %v", token.SpaceMemberID, f.spaceMemberID)
	}
	if token.Name != "CLI" {
		t.Errorf("token.Name = %q、期待値 = %q", token.Name, "CLI")
	}
	if token.TokenDigest != "create_digest" {
		t.Errorf("token.TokenDigest = %q、期待値 = %q", token.TokenDigest, "create_digest")
	}
	if token.TokenLastChars != "wxyz" {
		t.Errorf("token.TokenLastChars = %q、期待値 = %q", token.TokenLastChars, "wxyz")
	}
	if !slices.Equal(token.Scopes, scopes) {
		t.Errorf("token.Scopes = %v、期待値 = %v", token.Scopes, scopes)
	}
	if !token.ExpiresAt.Equal(expiresAt) {
		t.Errorf("token.ExpiresAt = %v、期待値 = %v", token.ExpiresAt, expiresAt)
	}
	if token.LastUsedAt != nil {
		t.Errorf("token.LastUsedAt = %v、期待値 = nil", token.LastUsedAt)
	}
	if token.RevokedAt != nil {
		t.Errorf("token.RevokedAt = %v、期待値 = nil", token.RevokedAt)
	}

	other := setupPersonalAccessTokenFixture(t, tx, "create-other")
	_, err = repo.Create(ctx, CreatePersonalAccessTokenInput{
		SpaceID:        f.spaceID,
		SpaceMemberID:  other.spaceMemberID,
		Name:           "別スペースのメンバー",
		TokenDigest:    "create_other_space_digest",
		TokenLastChars: "wxyz",
		Scopes:         scopes,
		ExpiresAt:      expiresAt,
	})
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code.Name() != "foreign_key_violation" ||
		pqErr.Constraint != "personal_access_tokens_space_member_id_space_id_fkey" {
		t.Errorf("別スペースのメンバーでの作成のエラー = %v、期待値 = 複合外部キー違反", err)
	}
}

func TestPersonalAccessTokenRepository_FindByTokenDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewPersonalAccessTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupPersonalAccessTokenFixture(t, tx, "find")
	activeID := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithTokenDigest("find_active_digest").
		WithScopes([]model.Scope{model.ScopePageRead, model.ScopeTopicRead}).
		Build()
	revokedID := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithTokenDigest("find_revoked_digest").
		WithRevokedAt(time.Now()).
		Build()

	t.Run("ダイジェストが一致するトークンを返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(ctx, "find_active_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token == nil {
			t.Fatal("FindByTokenDigest()がnilを返した")
		}
		if token.ID != activeID {
			t.Errorf("token.ID = %v、期待値 = %v", token.ID, activeID)
		}
		if token.SpaceID != f.spaceID {
			t.Errorf("token.SpaceID = %v、期待値 = %v", token.SpaceID, f.spaceID)
		}
		want := []model.Scope{model.ScopePageRead, model.ScopeTopicRead}
		if !slices.Equal(token.Scopes, want) {
			t.Errorf("token.Scopes = %v、期待値 = %v", token.Scopes, want)
		}
	})

	t.Run("失効したトークンも返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(ctx, "find_revoked_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token == nil {
			t.Fatal("FindByTokenDigest()がnilを返した")
		}
		if token.ID != revokedID {
			t.Errorf("token.ID = %v、期待値 = %v", token.ID, revokedID)
		}
		if !token.IsRevoked() {
			t.Error("token.IsRevoked() = false、期待値 = true")
		}
	})

	t.Run("一致するトークンが無ければnilを返す", func(t *testing.T) {
		token, err := repo.FindByTokenDigest(ctx, "find_missing_digest")
		if err != nil {
			t.Fatalf("FindByTokenDigest()のエラー = %v", err)
		}
		if token != nil {
			t.Errorf("FindByTokenDigest() = %v、期待値 = nil", token)
		}
	})
}

func TestPersonalAccessTokenRepository_UpdateLastUsedAt(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Microsecond)
	recent := now.Add(-model.PersonalAccessTokenLastUsedAtUpdateInterval / 2)
	stale := now.Add(-model.PersonalAccessTokenLastUsedAtUpdateInterval - time.Second)

	tests := []struct {
		name       string
		lastUsedAt *time.Time
		revokedAt  *time.Time
		otherSpace bool
		want       *time.Time
	}{
		{name: "未使用なら更新する", want: &now},
		{name: "最終使用日時が間隔より古ければ更新する", lastUsedAt: &stale, want: &now},
		{name: "最終使用日時が間隔より新しければ更新しない", lastUsedAt: &recent, want: &recent},
		{name: "失効済みなら更新しない", revokedAt: &recent},
		{name: "スペースが一致しなければ更新しない", otherSpace: true},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			repo := NewPersonalAccessTokenRepository(testutil.QueriesWithTx(tx))
			ctx := context.Background()

			suffix := "touch-" + string(rune('a'+i))
			f := setupPersonalAccessTokenFixture(t, tx, suffix)
			digest := suffix + "_digest"
			builder := testutil.NewPersonalAccessTokenBuilder(t, tx).
				WithSpaceID(f.spaceID).
				WithSpaceMemberID(f.spaceMemberID).
				WithTokenDigest(digest)
			if tt.lastUsedAt != nil {
				builder = builder.WithLastUsedAt(*tt.lastUsedAt)
			}
			if tt.revokedAt != nil {
				builder = builder.WithRevokedAt(*tt.revokedAt)
			}
			id := builder.Build()

			spaceID := f.spaceID
			if tt.otherSpace {
				spaceID = setupPersonalAccessTokenFixture(t, tx, suffix+"-other").spaceID
			}

			if err := repo.UpdateLastUsedAt(ctx, id, spaceID, now); err != nil {
				t.Fatalf("UpdateLastUsedAt()のエラー = %v", err)
			}

			token, err := repo.FindByTokenDigest(ctx, digest)
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			switch {
			case tt.want == nil && token.LastUsedAt != nil:
				t.Errorf("token.LastUsedAt = %v、期待値 = nil", token.LastUsedAt)
			case tt.want != nil && (token.LastUsedAt == nil || !token.LastUsedAt.Equal(*tt.want)):
				t.Errorf("token.LastUsedAt = %v、期待値 = %v", token.LastUsedAt, *tt.want)
			}
		})
	}
}

func TestPersonalAccessTokenRepository_Revoke(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewPersonalAccessTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupPersonalAccessTokenFixture(t, tx, "revoke")
	other := setupPersonalAccessTokenFixture(t, tx, "revoke-other")
	otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithUserID(testutil.NewUserBuilder(t, tx).
			WithEmail("pat-revoke-member@example.com").
			WithAtname("pat_revoke_member").
			Build()).
		Build()

	newToken := func(digest string) model.PersonalAccessTokenID {
		return testutil.NewPersonalAccessTokenBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithSpaceMemberID(f.spaceMemberID).
			WithTokenDigest(digest).
			Build()
	}

	t.Run("自分のトークンを失効する", func(t *testing.T) {
		id := newToken("revoke_own_digest")

		token, err := repo.Revoke(ctx, id, f.spaceID, f.spaceMemberID)
		if err != nil {
			t.Fatalf("Revoke()のエラー = %v", err)
		}
		if token == nil {
			t.Fatal("Revoke()がnilを返した")
		}
		if token.ID != id {
			t.Errorf("token.ID = %v、期待値 = %v", token.ID, id)
		}
		if !token.IsRevoked() {
			t.Error("token.IsRevoked() = false、期待値 = true")
		}

		again, err := repo.Revoke(ctx, id, f.spaceID, f.spaceMemberID)
		if err != nil {
			t.Fatalf("2回目のRevoke()のエラー = %v", err)
		}
		if again != nil {
			t.Errorf("失効済みのトークンに対するRevoke() = %v、期待値 = nil", again)
		}
	})

	t.Run("失効しない場合はnilを返しトークンに触れない", func(t *testing.T) {
		tests := []struct {
			name          string
			id            model.PersonalAccessTokenID
			spaceID       model.SpaceID
			spaceMemberID model.SpaceMemberID
		}{
			{name: "他のメンバーのトークン", id: newToken("revoke_member_digest"), spaceID: f.spaceID, spaceMemberID: otherMemberID},
			{name: "他のスペースを指定した", id: newToken("revoke_space_digest"), spaceID: other.spaceID, spaceMemberID: f.spaceMemberID},
			{name: "UUIDでないID", id: "not-a-uuid", spaceID: f.spaceID, spaceMemberID: f.spaceMemberID},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				token, err := repo.Revoke(ctx, tt.id, tt.spaceID, tt.spaceMemberID)
				if err != nil {
					t.Fatalf("Revoke()のエラー = %v", err)
				}
				if token != nil {
					t.Errorf("Revoke() = %v、期待値 = nil", token)
				}
			})
		}

		for _, digest := range []string{"revoke_member_digest", "revoke_space_digest"} {
			token, err := repo.FindByTokenDigest(ctx, digest)
			if err != nil {
				t.Fatalf("FindByTokenDigest()のエラー = %v", err)
			}
			if token.IsRevoked() {
				t.Errorf("%sのトークンが失効している", digest)
			}
		}
	})
}

func TestPersonalAccessTokenRepository_ListUnrevokedBySpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewPersonalAccessTokenRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupPersonalAccessTokenFixture(t, tx, "list")
	other := setupPersonalAccessTokenFixture(t, tx, "list-other")
	otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithUserID(testutil.NewUserBuilder(t, tx).
			WithEmail("pat-list-member@example.com").
			WithAtname("pat_list_member").
			Build()).
		Build()

	older := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithTokenDigest("list_older_digest").
		Build()
	expired := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithTokenDigest("list_expired_digest").
		WithExpiresAt(time.Now().Add(-time.Hour)).
		Build()
	testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithTokenDigest("list_revoked_digest").
		WithRevokedAt(time.Now()).
		Build()
	testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(otherMemberID).
		WithTokenDigest("list_member_digest").
		Build()
	testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(other.spaceID).
		WithSpaceMemberID(other.spaceMemberID).
		WithTokenDigest("list_space_digest").
		Build()

	tokens, err := repo.ListUnrevokedBySpaceMember(ctx, f.spaceID, f.spaceMemberID)
	if err != nil {
		t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
	}

	// 失効したトークンと、他のメンバー・他のスペースのトークンは含めず、期限切れは含める。
	got := make([]model.PersonalAccessTokenID, len(tokens))
	for i, token := range tokens {
		got[i] = token.ID
	}
	want := []model.PersonalAccessTokenID{expired, older}
	if !slices.Equal(got, want) {
		t.Errorf("ListUnrevokedBySpaceMember()のID = %v、期待値 = %v", got, want)
	}

	empty, err := repo.ListUnrevokedBySpaceMember(ctx, other.spaceID, otherMemberID)
	if err != nil {
		t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("トークンが無いメンバーのListUnrevokedBySpaceMember()の件数 = %d、期待値 = 0", len(empty))
	}
}
