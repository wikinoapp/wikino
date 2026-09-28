package usecase

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newGetSpaceSettingsUsecaseForTestはテストのトランザクションを使うユースケースを組み立てる。
func newGetSpaceSettingsUsecaseForTest(q *query.Queries) *GetSpaceSettingsUsecase {
	return NewGetSpaceSettingsUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
	)
}

// newSpaceSettingsUserは、並行するテストのトランザクションが一意制約で待ち合わないよう、
// keyから一意なメールアドレスとアットネームを持つユーザーを作る。
func newSpaceSettingsUser(t *testing.T, tx *sql.Tx, key string) model.UserID {
	t.Helper()

	return testutil.NewUserBuilder(t, tx).
		WithEmail(key + "@example.com").
		WithAtname(strings.ReplaceAll(key, "-", "_")).
		Build()
}

func TestGetSpaceSettingsUsecase_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		scopes      []model.Scope
		flagEnabled bool
		want        *SpaceSettingsItems
		wantErrCode model.AppErrorCode
	}{
		{
			name:   "space:adminは既存の項目を開け、フラグが無効ならトークン管理の項目は出ない",
			key:    "gsset-admin",
			scopes: []model.Scope{model.ScopeSpaceAdmin},
			want:   &SpaceSettingsItems{CanUpdateSpace: true},
		},
		{
			name:        "space:adminでフラグが有効ならトークン管理の項目も出る",
			key:         "gsset-admin-flag",
			scopes:      []model.Scope{model.ScopeSpaceAdmin},
			flagEnabled: true,
			want:        &SpaceSettingsItems{CanUpdateSpace: true, CanShowPersonalAccessTokens: true, CanShowOAuthGrants: true, CanShowOAuthApplications: true},
		},
		{
			name:   "space:writeだけで既存の項目を開ける",
			key:    "gsset-write",
			scopes: []model.Scope{model.ScopeSpaceWrite},
			want:   &SpaceSettingsItems{CanUpdateSpace: true},
		},
		{
			name:        "personal_access_token:readだけでフラグが有効なら個人アクセストークンの項目だけを開ける",
			key:         "gsset-pat",
			scopes:      []model.Scope{model.ScopePersonalAccessTokenRead},
			flagEnabled: true,
			want:        &SpaceSettingsItems{CanShowPersonalAccessTokens: true},
		},
		{
			name:        "oauth_grant:readだけでフラグが有効なら連携中のアプリの項目だけを開ける",
			key:         "gsset-oauth",
			scopes:      []model.Scope{model.ScopeOAuthGrantRead},
			flagEnabled: true,
			want:        &SpaceSettingsItems{CanShowOAuthGrants: true},
		},
		{
			name:        "oauth_application:readだけでフラグが有効ならOAuthアプリの項目だけを開ける",
			key:         "gsset-oauth-app",
			scopes:      []model.Scope{model.ScopeOAuthApplicationRead},
			flagEnabled: true,
			want:        &SpaceSettingsItems{CanShowOAuthApplications: true},
		},
		{
			name:        "oauth_application:readだけでフラグが無効なら開けない",
			key:         "gsset-oauth-app-noflag",
			scopes:      []model.Scope{model.ScopeOAuthApplicationRead},
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "personal_access_token:readだけでフラグが無効なら開けない",
			key:         "gsset-pat-noflag",
			scopes:      []model.Scope{model.ScopePersonalAccessTokenRead},
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "どの項目のスコープも持たないメンバーは開けない",
			key:         "gsset-reader",
			scopes:      []model.Scope{model.ScopeSpaceRead, model.ScopePageWrite},
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			uc := newGetSpaceSettingsUsecaseForTest(q)

			userID := newSpaceSettingsUser(t, tx, tt.key)
			spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(tt.key).Build()
			testutil.NewSpaceMemberBuilder(t, tx).
				WithSpaceID(spaceID).
				WithUserID(userID).
				WithScopes(tt.scopes).
				Build()
			if tt.flagEnabled {
				testutil.NewFeatureFlagBuilder(t, tx).
					WithUserID(userID).
					WithName(string(model.FeatureFlagPublicAPI)).
					Build()
			}

			output, err := uc.Execute(context.Background(), GetSpaceSettingsInput{
				SpaceIdentifier: model.SpaceIdentifier(tt.key),
				UserID:          userID,
			})

			if tt.want == nil {
				assertAppErrCode(t, err, tt.wantErrCode)
				return
			}
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.Space.ID != spaceID {
				t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, spaceID)
			}
			if output.Items != *tt.want {
				t.Errorf("Items = %+v、期待値 = %+v", output.Items, *tt.want)
			}
		})
	}
}

func TestGetSpaceSettingsUsecase_Execute_見つからない(t *testing.T) {
	t.Parallel()

	t.Run("スペースのメンバーではないユーザー", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		uc := newGetSpaceSettingsUsecaseForTest(q)

		userID := newSpaceSettingsUser(t, tx, "gsset-outsider")
		testutil.NewSpaceBuilder(t, tx).WithIdentifier("gsset-outsider").Build()

		_, err := uc.Execute(context.Background(), GetSpaceSettingsInput{
			SpaceIdentifier: "gsset-outsider",
			UserID:          userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("無効になったメンバー", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		uc := newGetSpaceSettingsUsecaseForTest(q)

		userID := newSpaceSettingsUser(t, tx, "gsset-inactive")
		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("gsset-inactive").Build()
		testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(userID).
			WithActive(false).
			Build()

		_, err := uc.Execute(context.Background(), GetSpaceSettingsInput{
			SpaceIdentifier: "gsset-inactive",
			UserID:          userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})

	t.Run("存在しないスペース", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		uc := newGetSpaceSettingsUsecaseForTest(q)

		userID := newSpaceSettingsUser(t, tx, "gsset-missing")

		_, err := uc.Execute(context.Background(), GetSpaceSettingsInput{
			SpaceIdentifier: "gsset-missing",
			UserID:          userID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}
