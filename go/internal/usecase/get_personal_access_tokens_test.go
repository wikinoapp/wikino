package usecase

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// patMemberFixtureは個人アクセストークンのユースケースのテストで使う、スペースとそのメンバー
type patMemberFixture struct {
	userID        model.UserID
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
}

// setupPATMemberは、keyを識別子に持つスペースと、roleのロールを持つメンバーを作る。flagEnabledが
// 真ならメンバーのユーザーに公開APIのフィーチャーフラグを有効にする。
func setupPATMember(t *testing.T, tx *sql.Tx, key string, role model.SpaceRole, flagEnabled bool) patMemberFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(key + "@example.com").
		WithAtname(strings.ReplaceAll(key, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(key).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(role).
		Build()
	if flagEnabled {
		testutil.NewFeatureFlagBuilder(t, tx).
			WithUserID(userID).
			WithName(string(model.FeatureFlagPublicAPI)).
			Build()
	}

	return patMemberFixture{userID: userID, spaceID: spaceID, spaceMemberID: spaceMemberID}
}

func newGetPersonalAccessTokensUsecaseForTest(q *query.Queries) *GetPersonalAccessTokensUsecase {
	return NewGetPersonalAccessTokensUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewPersonalAccessTokenRepository(q),
	)
}

func TestGetPersonalAccessTokensUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("自分の失効していないトークンを新しい順に返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "gpat-list", model.SpaceRoleAdmin, true)
		other := setupPATMember(t, tx, "gpat-list-other", model.SpaceRoleAdmin, true)

		older := testutil.NewPersonalAccessTokenBuilder(t, tx).
			WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).WithTokenDigest("gpat_older").Build()
		newer := testutil.NewPersonalAccessTokenBuilder(t, tx).
			WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).WithTokenDigest("gpat_newer").
			WithExpiresAt(time.Now().Add(-time.Hour)).Build()
		testutil.NewPersonalAccessTokenBuilder(t, tx).
			WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).WithTokenDigest("gpat_revoked").
			WithRevokedAt(time.Now()).Build()
		testutil.NewPersonalAccessTokenBuilder(t, tx).
			WithSpaceID(other.spaceID).WithSpaceMemberID(other.spaceMemberID).WithTokenDigest("gpat_other").Build()

		output, err := newGetPersonalAccessTokensUsecaseForTest(q).Execute(context.Background(), GetPersonalAccessTokensInput{
			SpaceIdentifier: "gpat-list",
			UserID:          f.userID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := make([]model.PersonalAccessTokenID, len(output.Tokens))
		for i, token := range output.Tokens {
			got[i] = token.ID
		}
		if want := []model.PersonalAccessTokenID{newer, older}; !slices.Equal(got, want) {
			t.Errorf("TokensのID = %v、期待値 = %v", got, want)
		}
		if output.Space.ID != f.spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
		}
		if !output.CanCreate {
			t.Error("CanCreate = false、期待値 = true")
		}
		if !output.CanDelete {
			t.Error("CanDelete = false、期待値 = true")
		}
	})
}

func TestGetPersonalAccessTokensUsecase_Execute_開けない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		role        model.SpaceRole
		flagEnabled bool
		identifier  model.SpaceIdentifier
		wantErrCode model.AppErrorCode
	}{
		{
			name:        "フィーチャーフラグが無効",
			key:         "gpat-noflag",
			role:        model.SpaceRoleAdmin,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "存在しないスペース",
			key:         "gpat-missing",
			role:        model.SpaceRoleAdmin,
			flagEnabled: true,
			identifier:  "gpat-missing-none",
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			f := setupPATMember(t, tx, tt.key, tt.role, tt.flagEnabled)
			identifier := tt.identifier
			if identifier == "" {
				identifier = model.SpaceIdentifier(tt.key)
			}

			_, err := newGetPersonalAccessTokensUsecaseForTest(q).Execute(context.Background(), GetPersonalAccessTokensInput{
				SpaceIdentifier: identifier,
				UserID:          f.userID,
			})
			assertAppErrCode(t, err, tt.wantErrCode)
		})
	}

	t.Run("スペースのメンバーではないユーザー", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		setupPATMember(t, tx, "gpat-owner", model.SpaceRoleAdmin, true)
		outsider := setupPATMember(t, tx, "gpat-outsider", model.SpaceRoleAdmin, true)

		_, err := newGetPersonalAccessTokensUsecaseForTest(q).Execute(context.Background(), GetPersonalAccessTokensInput{
			SpaceIdentifier: "gpat-owner",
			UserID:          outsider.userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}
