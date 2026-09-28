package usecase

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetOAuthGrantsUsecaseForTest(q *query.Queries) *GetOAuthGrantsUsecase {
	return NewGetOAuthGrantsUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthGrantRepository(q),
		repository.NewOAuthApplicationRepository(q),
	)
}

func TestGetOAuthGrantsUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("自分の失効していない許可を新しい順に、許可したアプリと一緒に返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "gog-list", []model.Scope{model.ScopeOAuthGrantRead}, true)
		other := setupPATMember(t, tx, "gog-list-other", []model.Scope{model.ScopeSpaceAdmin}, true)

		spaceAppID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(f.spaceID).WithName("スペースのアプリ").WithClientID("gog-list-space-client").Build()
		officialAppID := testutil.NewOAuthApplicationBuilder(t, tx).
			AsOfficialClient().WithName("Wikino CLI").WithClientID("gog-list-official-client").Build()
		older := testutil.NewOAuthGrantBuilder(t, tx).
			WithOAuthApplicationID(spaceAppID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).Build()
		newer := testutil.NewOAuthGrantBuilder(t, tx).
			WithOAuthApplicationID(officialAppID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).Build()
		// 失効した許可と、他のメンバーの許可は出ない
		testutil.NewOAuthGrantBuilder(t, tx).
			WithOAuthApplicationID(spaceAppID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).
			WithRevokedAt(time.Now()).Build()
		otherAppID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(other.spaceID).WithClientID("gog-list-other-client").Build()
		testutil.NewOAuthGrantBuilder(t, tx).
			WithOAuthApplicationID(otherAppID).WithSpaceID(other.spaceID).WithSpaceMemberID(other.spaceMemberID).Build()

		output, err := newGetOAuthGrantsUsecaseForTest(q).Execute(context.Background(), GetOAuthGrantsInput{
			SpaceIdentifier: "gog-list",
			UserID:          f.userID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		gotIDs := make([]model.OAuthGrantID, len(output.Grants))
		for i, g := range output.Grants {
			gotIDs[i] = g.ID
		}
		if want := []model.OAuthGrantID{newer, older}; !slices.Equal(gotIDs, want) {
			t.Errorf("許可のID = %v、期待値 = %v", gotIDs, want)
		}
		for appID, wantName := range map[model.OAuthApplicationID]string{spaceAppID: "スペースのアプリ", officialAppID: "Wikino CLI"} {
			app, ok := output.Applications[appID]
			if !ok {
				t.Errorf("Applicationsにアプリ%vが含まれていない", appID)
				continue
			}
			if app.Name != wantName {
				t.Errorf("アプリ%vの名前 = %q、期待値 = %q", appID, app.Name, wantName)
			}
		}
		if output.CanDelete {
			t.Error("CanDelete = true、oauth_grant:readだけのメンバーはfalseを期待")
		}
	})

	t.Run("oauth_grant:deleteだけのメンバーも一覧を開け、解除できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "gog-delete-only", []model.Scope{model.ScopeOAuthGrantDelete}, true)

		output, err := newGetOAuthGrantsUsecaseForTest(q).Execute(context.Background(), GetOAuthGrantsInput{
			SpaceIdentifier: "gog-delete-only",
			UserID:          f.userID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if !output.CanDelete {
			t.Error("CanDelete = false、期待値 = true")
		}
	})
}

func TestGetOAuthGrantsUsecase_Execute_開けない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		scopes      []model.Scope
		flagEnabled bool
		outsider    bool
		wantErrCode model.AppErrorCode
	}{
		{
			name:        "oauth_grant:readを持たない",
			key:         "gog-noscope",
			scopes:      []model.Scope{model.ScopePersonalAccessTokenRead},
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "フィーチャーフラグが無効",
			key:         "gog-noflag",
			scopes:      []model.Scope{model.ScopeSpaceAdmin},
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "スペースのメンバーではない",
			key:         "gog-outsider",
			scopes:      []model.Scope{model.ScopeSpaceAdmin},
			flagEnabled: true,
			outsider:    true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			f := setupPATMember(t, tx, tt.key, tt.scopes, tt.flagEnabled)
			userID := f.userID
			if tt.outsider {
				userID = setupPATMember(t, tx, tt.key+"-other", tt.scopes, tt.flagEnabled).userID
			}

			_, err := newGetOAuthGrantsUsecaseForTest(q).Execute(context.Background(), GetOAuthGrantsInput{
				SpaceIdentifier: model.SpaceIdentifier(tt.key),
				UserID:          userID,
			})
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != tt.wantErrCode {
				t.Errorf("Execute()のエラー = %v、期待値 = %v", err, tt.wantErrCode)
			}
		})
	}
}
