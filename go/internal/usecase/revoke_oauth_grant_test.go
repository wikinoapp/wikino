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

func newRevokeOAuthGrantUsecaseForTest(q *query.Queries) *RevokeOAuthGrantUsecase {
	return NewRevokeOAuthGrantUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthGrantRepository(q),
		repository.NewOAuthApplicationRepository(q),
	)
}

// 許可を解除した後に、照合がトークンを受け付けなくなるところまでを通す。
// 照合はoauth_grant:writeを持つメンバーのトークンしか受け付けないため、:writeと:deleteを持つ
// 編集者 (フィクスチャの既定) で確かめる
func TestRevokeOAuthGrantUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	ctx := context.Background()
	opts := defaultOAuthAccessTokenAuthFixtureOptions()
	f, token := setupOAuthAccessTokenAuthFixture(t, tx, "rog-revoke", opts)
	authenticateUC := newAuthenticateAPITokenUC(tx)

	principal, err := authenticateUC.Execute(ctx, token)
	if err != nil || principal == nil {
		t.Fatalf("解除前のExecute() = (%v, %v)、トークンを受け付けることを期待", principal, err)
	}

	grants, err := repository.NewOAuthGrantRepository(q).ListUnrevokedBySpaceMember(ctx, f.spaceID, f.spaceMemberID)
	if err != nil || len(grants) != 1 {
		t.Fatalf("解除前のListUnrevokedBySpaceMember() = (%v, %v)、1件を期待", grants, err)
	}
	grantID := grants[0].ID
	testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest("rog_revoke_refresh_digest").Build()

	output, err := newRevokeOAuthGrantUsecaseForTest(q).Execute(ctx, RevokeOAuthGrantInput{
		SpaceIdentifier: "rog-revoke",
		UserID:          f.userID,
		OAuthGrantID:    grantID,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output.Grant.ID != grantID || !output.Grant.IsRevoked() {
		t.Errorf("解除した許可 = %+v、期待値は失効日時を持つID %v の許可", output.Grant, grantID)
	}
	if output.Application == nil || output.Application.ClientID != "rog-revoke-client" {
		t.Errorf("Application = %+v、許可したアプリを期待", output.Application)
	}

	principal, err = authenticateUC.Execute(ctx, token)
	if err != nil {
		t.Fatalf("解除後のExecute()のエラー = %v", err)
	}
	if principal != nil {
		t.Errorf("principal = %+v、解除した連携のトークンは受け付けないことを期待", principal)
	}
	refreshToken, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(ctx, "rog_revoke_refresh_digest")
	if err != nil {
		t.Fatalf("FindByTokenDigest()のエラー = %v", err)
	}
	if !refreshToken.IsRevoked() {
		t.Error("リフレッシュトークンが失効していない")
	}
}

func TestRevokeOAuthGrantUsecase_Execute_閲覧者も解除できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupPATMember(t, tx, "rog-viewer", model.SpaceRoleViewer, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID("rog-viewer-client").Build()
	grantID := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(appID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.spaceMemberID).Build()

	output, err := newRevokeOAuthGrantUsecaseForTest(q).Execute(context.Background(), RevokeOAuthGrantInput{
		SpaceIdentifier: "rog-viewer",
		UserID:          f.userID,
		OAuthGrantID:    grantID,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if !output.Grant.IsRevoked() {
		t.Error("許可が失効していない")
	}
}

func TestRevokeOAuthGrantUsecase_Execute_解除できない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		role        model.SpaceRole
		flagEnabled bool
		// otherMemberが真なら、同じスペースの他のメンバーの許可を解除しようとする
		otherMember bool
		revoked     bool
		invalidID   bool
		wantErrCode model.AppErrorCode
	}{
		{
			name:        "フィーチャーフラグが無効",
			key:         "rog-noflag",
			role:        model.SpaceRoleAdmin,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "他のメンバーの許可",
			key:         "rog-other",
			role:        model.SpaceRoleAdmin,
			flagEnabled: true,
			otherMember: true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "解除済みの許可",
			key:         "rog-revoked",
			role:        model.SpaceRoleAdmin,
			flagEnabled: true,
			revoked:     true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "UUIDでないID",
			key:         "rog-invalid",
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
			ownerMemberID := f.spaceMemberID
			if tt.otherMember {
				otherUserID := testutil.NewUserBuilder(t, tx).
					WithEmail(tt.key + "-owner@example.com").
					WithAtname("rog_owner").
					Build()
				ownerMemberID = testutil.NewSpaceMemberBuilder(t, tx).
					WithSpaceID(f.spaceID).WithUserID(otherUserID).Build()
			}
			appID := testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithClientID(tt.key + "-client").Build()
			grantBuilder := testutil.NewOAuthGrantBuilder(t, tx).
				WithOAuthApplicationID(appID).WithSpaceID(f.spaceID).WithSpaceMemberID(ownerMemberID)
			if tt.revoked {
				grantBuilder = grantBuilder.WithRevokedAt(time.Now())
			}
			grantID := grantBuilder.Build()
			if tt.invalidID {
				grantID = "not-a-uuid"
			}

			_, err := newRevokeOAuthGrantUsecaseForTest(q).Execute(context.Background(), RevokeOAuthGrantInput{
				SpaceIdentifier: model.SpaceIdentifier(tt.key),
				UserID:          f.userID,
				OAuthGrantID:    grantID,
			})
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != tt.wantErrCode {
				t.Errorf("Execute()のエラー = %v、期待値 = %v", err, tt.wantErrCode)
			}
		})
	}
}
