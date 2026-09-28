package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetOAuthApplicationsUsecaseForTest(q *query.Queries) *GetOAuthApplicationsUsecase {
	return NewGetOAuthApplicationsUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewUserRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
	)
}

func newGetOAuthApplicationUsecaseForTest(q *query.Queries) *GetOAuthApplicationUsecase {
	return NewGetOAuthApplicationUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewUserRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
	)
}

func TestGetOAuthApplicationsUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("他のメンバーが作成したものも含めてスペースのアプリを返し、作成者を引く", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "goa-list", []model.Scope{model.ScopeOAuthApplicationRead}, true)

		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("goa-list-other@example.com").
			WithAtname("goa_list_other").
			Build()
		otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithUserID(otherUserID).
			Build()
		otherAppID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithCreatedSpaceMemberID(otherMemberID).
			WithClientID("goa-list-other-client").
			Build()
		orphanAppID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithClientID("goa-list-orphan-client").
			Build()

		output, err := newGetOAuthApplicationsUsecaseForTest(q).Execute(context.Background(), GetOAuthApplicationsInput{
			SpaceIdentifier: "goa-list",
			UserID:          f.userID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if len(output.Applications) != 2 || output.Applications[0].ID != orphanAppID || output.Applications[1].ID != otherAppID {
			t.Fatalf("Applications = %v、期待値のID = [%v %v]", output.Applications, orphanAppID, otherAppID)
		}
		if creator := output.Creators[otherMemberID]; creator == nil || creator.ID != otherUserID {
			t.Errorf("Creators[%v] = %v、期待値のユーザー = %v", otherMemberID, creator, otherUserID)
		}
		if len(output.Creators) != 1 {
			t.Errorf("Creatorsの件数 = %d、期待値 = 1 (作成者のいないアプリは引かない)", len(output.Creators))
		}
		if output.CanCreate {
			t.Error("CanCreate = true、期待値 = false (oauth_application:readだけを持つ)")
		}
	})

	t.Run("閲覧できない", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			key         string
			scopes      []model.Scope
			flagEnabled bool
			wantErrCode model.AppErrorCode
		}{
			{name: "oauth_application:readを持たない", key: "goa-noscope", scopes: []model.Scope{model.ScopePersonalAccessTokenRead}, flagEnabled: true, wantErrCode: model.AppErrCodeForbidden},
			{name: "フィーチャーフラグが無効", key: "goa-noflag", scopes: []model.Scope{model.ScopeOAuthApplicationRead}, wantErrCode: model.AppErrCodeResourceNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, tx := testutil.SetupTx(t)
				q := testutil.QueriesWithTx(tx)
				f := setupPATMember(t, tx, tt.key, tt.scopes, tt.flagEnabled)

				_, err := newGetOAuthApplicationsUsecaseForTest(q).Execute(context.Background(), GetOAuthApplicationsInput{
					SpaceIdentifier: model.SpaceIdentifier(tt.key),
					UserID:          f.userID,
				})
				assertAppErrCode(t, err, tt.wantErrCode)
			})
		}
	})
}

func TestGetOAuthApplicationUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupPATMember(t, tx, "goa-show", []model.Scope{model.ScopeOAuthApplicationRead}, true)
	other := setupPATMember(t, tx, "goa-show-other", []model.Scope{model.ScopeOAuthApplicationRead}, true)

	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithCreatedSpaceMemberID(f.spaceMemberID).
		WithClientID("goa-show-client").
		Build()
	otherAppID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(other.spaceID).
		WithClientID("goa-show-other-client").
		Build()

	// サブテストはフィクスチャのトランザクションを共有するため、並行に走らせない。
	t.Run("スペースのアプリと作成者を返す", func(t *testing.T) {
		output, err := newGetOAuthApplicationUsecaseForTest(q).Execute(context.Background(), GetOAuthApplicationInput{
			SpaceIdentifier:    "goa-show",
			UserID:             f.userID,
			OAuthApplicationID: appID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Application.ID != appID {
			t.Errorf("Application.ID = %v、期待値 = %v", output.Application.ID, appID)
		}
		if creator := output.Creators[f.spaceMemberID]; creator == nil || creator.ID != f.userID {
			t.Errorf("Creators[%v] = %v、期待値のユーザー = %v", f.spaceMemberID, creator, f.userID)
		}
	})

	t.Run("別のスペースのアプリは見つからない", func(t *testing.T) {
		_, err := newGetOAuthApplicationUsecaseForTest(q).Execute(context.Background(), GetOAuthApplicationInput{
			SpaceIdentifier:    "goa-show",
			UserID:             f.userID,
			OAuthApplicationID: otherAppID,
		})
		assertAppErrCode(t, err, model.AppErrCodeResourceNotFound)
	})
}
