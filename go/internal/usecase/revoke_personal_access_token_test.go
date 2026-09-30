package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newRevokePersonalAccessTokenUsecaseForTest(q *query.Queries) *RevokePersonalAccessTokenUsecase {
	return NewRevokePersonalAccessTokenUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewPersonalAccessTokenRepository(q),
	)
}

func TestRevokePersonalAccessTokenUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupPATMember(t, tx, "rpat-revoke", model.SpaceRoleViewer, true)
	tokenID := testutil.NewPersonalAccessTokenBuilder(t, tx).
		WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).WithTokenDigest("rpat_revoke").Build()

	output, err := newRevokePersonalAccessTokenUsecaseForTest(q).Execute(context.Background(), RevokePersonalAccessTokenInput{
		SpaceIdentifier:       "rpat-revoke",
		UserID:                f.userID,
		PersonalAccessTokenID: tokenID,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output.Token.ID != tokenID {
		t.Errorf("Token.ID = %v、期待値 = %v", output.Token.ID, tokenID)
	}
	if output.Token.RevokedAt == nil {
		t.Error("Token.RevokedAt = nil、失効日時を期待")
	}
	if output.Space.ID != f.spaceID {
		t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
	}
	assertNoPersonalAccessToken(t, q, f)
}

func TestRevokePersonalAccessTokenUsecase_Execute_失効したトークンは次の照合で受け付けない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	opts := defaultAuthenticateAPITokenFixtureOptions()
	f := setupAuthenticateAPITokenFixture(t, tx, opts)
	authenticateUC := newAuthenticateAPITokenUC(tx)

	principal, err := authenticateUC.Execute(context.Background(), f.token)
	if err != nil {
		t.Fatalf("失効前のExecute()のエラー = %v", err)
	}
	if principal == nil {
		t.Fatal("失効前のトークンを受け付けなかった")
	}

	space, err := repository.NewSpaceRepository(testutil.QueriesWithTx(tx)).FindByID(context.Background(), f.spaceID)
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if _, err := newRevokePersonalAccessTokenUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), RevokePersonalAccessTokenInput{
		SpaceIdentifier:       space.Identifier,
		UserID:                f.userID,
		PersonalAccessTokenID: f.tokenID,
	}); err != nil {
		t.Fatalf("失効のExecute()のエラー = %v", err)
	}

	principal, err = authenticateUC.Execute(context.Background(), f.token)
	if err != nil {
		t.Fatalf("失効後のExecute()のエラー = %v", err)
	}
	if principal != nil {
		t.Errorf("principal = %+v、失効したトークンは受け付けないことを期待", principal)
	}
}

func TestRevokePersonalAccessTokenUsecase_Execute_失効できない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		role        model.SpaceRole
		flagEnabled bool
		// tokenOfは失効を求めるトークンの持ち主。"other"なら同じスペースの他のメンバー
		tokenOf     string
		revoked     bool
		invalidID   bool
		wantErrCode model.AppErrorCode
	}{
		{
			name:        "フィーチャーフラグが無効",
			key:         "rpat-noflag",
			role:        model.SpaceRoleAdmin,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "他のメンバーのトークン",
			key:         "rpat-other",
			role:        model.SpaceRoleAdmin,
			flagEnabled: true,
			tokenOf:     "other",
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "既に失効したトークン",
			key:         "rpat-revoked",
			role:        model.SpaceRoleAdmin,
			flagEnabled: true,
			revoked:     true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "UUIDでないID",
			key:         "rpat-invalid-id",
			role:        model.SpaceRoleAdmin,
			flagEnabled: true,
			invalidID:   true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			f := setupPATMember(t, tx, tt.key, tt.role, tt.flagEnabled)

			spaceMemberID := f.spaceMemberID
			if tt.tokenOf == "other" {
				otherUserID := testutil.NewUserBuilder(t, tx).WithEmail(tt.key + "-other@example.com").Build()
				spaceMemberID = testutil.NewSpaceMemberBuilder(t, tx).
					WithSpaceID(f.spaceID).WithUserID(otherUserID).Build()
			}
			builder := testutil.NewPersonalAccessTokenBuilder(t, tx).
				WithSpaceID(f.spaceID).WithSpaceMemberID(spaceMemberID).WithTokenDigest(tt.key)
			revokedAt := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
			if tt.revoked {
				builder = builder.WithRevokedAt(revokedAt)
			}
			tokenID := builder.Build()
			if tt.invalidID {
				tokenID = "not-a-uuid"
			}

			_, err := newRevokePersonalAccessTokenUsecaseForTest(q).Execute(context.Background(), RevokePersonalAccessTokenInput{
				SpaceIdentifier:       model.SpaceIdentifier(tt.key),
				UserID:                f.userID,
				PersonalAccessTokenID: tokenID,
			})
			assertAppErrCode(t, err, tt.wantErrCode)

			if tt.invalidID {
				return
			}
			var gotRevokedAt *time.Time
			if err := tx.QueryRowContext(context.Background(), "SELECT revoked_at FROM personal_access_tokens WHERE id = $1 AND space_id = $2", string(tokenID), string(f.spaceID)).Scan(&gotRevokedAt); err != nil {
				t.Fatalf("失効日時の取得に失敗: %v", err)
			}
			switch {
			case tt.revoked && (gotRevokedAt == nil || !gotRevokedAt.Equal(revokedAt)):
				t.Errorf("revoked_at = %v、元の失効日時 %v のままを期待", gotRevokedAt, revokedAt)
			case !tt.revoked && gotRevokedAt != nil:
				t.Errorf("revoked_at = %v、失効しないことを期待", gotRevokedAt)
			}
		})
	}
}
