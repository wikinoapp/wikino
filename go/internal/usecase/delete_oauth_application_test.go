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

func newDeleteOAuthApplicationUsecaseForTest(q *query.Queries) *DeleteOAuthApplicationUsecase {
	return NewDeleteOAuthApplicationUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
	)
}

func TestDeleteOAuthApplicationUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("他のメンバーが作成したアプリを削除し、その許可とトークンを失効する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		ctx := context.Background()
		f := setupPATMember(t, tx, "doa-delete", []model.Scope{model.ScopeOAuthApplicationDelete}, true)
		// 作成したメンバーがいなくなったアプリ。閲覧者が作成したものでなくても削除できる
		appID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithClientID("doa-delete-client").
			Build()
		grantID := testutil.NewOAuthGrantBuilder(t, tx).
			WithOAuthApplicationID(appID).
			WithSpaceID(f.spaceID).
			WithSpaceMemberID(f.spaceMemberID).
			Build()
		testutil.NewOAuthAccessTokenBuilder(t, tx).
			WithOAuthGrantID(grantID).
			WithSpaceID(f.spaceID).
			WithTokenDigest("doa_delete_access_digest").
			Build()
		testutil.NewOAuthRefreshTokenBuilder(t, tx).
			WithOAuthGrantID(grantID).
			WithSpaceID(f.spaceID).
			WithTokenDigest("doa_delete_refresh_digest").
			Build()

		output, err := newDeleteOAuthApplicationUsecaseForTest(q).Execute(ctx, DeleteOAuthApplicationInput{
			SpaceIdentifier:    "doa-delete",
			UserID:             f.userID,
			OAuthApplicationID: appID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Application.ID != appID || output.Application.DiscardedAt == nil {
			t.Errorf("削除したアプリ = %+v、期待値は削除日時を持つID %v のアプリ", output.Application, appID)
		}

		if app, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(ctx, appID, f.spaceID); err != nil || app != nil {
			t.Errorf("削除後のFindByIDAndSpaceID() = (%v, %v)、期待値 = (nil, nil)", app, err)
		}
		grant, err := repository.NewOAuthGrantRepository(q).FindByID(ctx, grantID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if grant.RevokedAt == nil {
			t.Error("許可が失効していない")
		}
		accessToken, err := repository.NewOAuthAccessTokenRepository(q).FindByTokenDigest(ctx, "doa_delete_access_digest")
		if err != nil {
			t.Fatalf("アクセストークンのFindByTokenDigest()のエラー = %v", err)
		}
		if accessToken.RevokedAt == nil {
			t.Error("アクセストークンが失効していない")
		}
		refreshToken, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(ctx, "doa_delete_refresh_digest")
		if err != nil {
			t.Fatalf("リフレッシュトークンのFindByTokenDigest()のエラー = %v", err)
		}
		if refreshToken.RevokedAt == nil {
			t.Error("リフレッシュトークンが失効していない")
		}
	})
}

func TestDeleteOAuthApplicationUsecase_Execute_削除できない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		scopes      []model.Scope
		flagEnabled bool
		otherSpace  bool
		discarded   bool
		wantErrCode model.AppErrorCode
	}{
		{
			name:        "oauth_application:writeだけを持つ",
			key:         "doa-writeonly",
			scopes:      []model.Scope{model.ScopeOAuthApplicationWrite},
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "フィーチャーフラグが無効",
			key:         "doa-noflag",
			scopes:      []model.Scope{model.ScopeOAuthApplicationDelete},
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "別のスペースのアプリ",
			key:         "doa-otherspace",
			scopes:      []model.Scope{model.ScopeOAuthApplicationDelete},
			flagEnabled: true,
			otherSpace:  true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "削除済みのアプリ",
			key:         "doa-discarded",
			scopes:      []model.Scope{model.ScopeOAuthApplicationDelete},
			flagEnabled: true,
			discarded:   true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			f := setupPATMember(t, tx, tt.key, tt.scopes, tt.flagEnabled)
			appSpaceID := f.spaceID
			if tt.otherSpace {
				appSpaceID = testutil.NewSpaceBuilder(t, tx).WithIdentifier(tt.key + "-other").Build()
			}
			builder := testutil.NewOAuthApplicationBuilder(t, tx).
				WithSpaceID(appSpaceID).
				WithClientID(tt.key + "-client")
			var discardedAt time.Time
			if tt.discarded {
				discardedAt = time.Now().Add(-time.Hour).Truncate(time.Microsecond)
				builder = builder.WithDiscardedAt(discardedAt)
			}
			appID := builder.Build()

			_, err := newDeleteOAuthApplicationUsecaseForTest(q).Execute(context.Background(), DeleteOAuthApplicationInput{
				SpaceIdentifier:    model.SpaceIdentifier(tt.key),
				UserID:             f.userID,
				OAuthApplicationID: appID,
			})
			assertAppErrCode(t, err, tt.wantErrCode)

			var got *time.Time
			if err := tx.QueryRowContext(context.Background(), `SELECT discarded_at FROM oauth_applications WHERE id = $1`, string(appID)).Scan(&got); err != nil {
				t.Fatalf("削除日時の取得に失敗: %v", err)
			}
			switch {
			case !tt.discarded && got != nil:
				t.Errorf("discarded_at = %v、期待値 = nil", got)
			case tt.discarded && (got == nil || !got.Equal(discardedAt)):
				t.Errorf("discarded_at = %v、期待値 = %v", got, discardedAt)
			}
		})
	}
}
