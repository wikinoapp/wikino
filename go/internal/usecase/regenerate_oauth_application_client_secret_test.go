package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newRegenerateOAuthApplicationClientSecretUsecaseForTest(q *query.Queries) *RegenerateOAuthApplicationClientSecretUsecase {
	return NewRegenerateOAuthApplicationClientSecretUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
	)
}

func TestRegenerateOAuthApplicationClientSecretUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("新しいシークレットを発行し、そのダイジェストだけを保存する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "roa-regenerate", model.SpaceRoleAdmin, true)
		appID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithClientID("roa-regenerate-client").
			WithConfidentialClientSecretDigest("roa_old_digest").
			Build()

		output, err := newRegenerateOAuthApplicationClientSecretUsecaseForTest(q).Execute(context.Background(), RegenerateOAuthApplicationClientSecretInput{
			SpaceIdentifier:    "roa-regenerate",
			UserID:             f.userID,
			OAuthApplicationID: appID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if !strings.HasPrefix(output.ClientSecret, string(auth.OAuthClientSecretPrefix)) {
			t.Errorf("ClientSecret = %q、期待値は接頭辞%qで始まる", output.ClientSecret, auth.OAuthClientSecretPrefix)
		}

		stored, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), appID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
		}
		if stored.ClientSecretDigest == nil || *stored.ClientSecretDigest != auth.DigestOpaqueToken(output.ClientSecret) {
			t.Errorf("ClientSecretDigest = %v、期待値は新しいシークレットのダイジェスト", stored.ClientSecretDigest)
		}
	})
}

func TestRegenerateOAuthApplicationClientSecretUsecase_Execute_再発行できない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		role        model.SpaceRole
		flagEnabled bool
		public      bool
		wantErrCode model.AppErrorCode
	}{
		{
			name:        "編集者 (OAuthアプリの権限を持たない)",
			key:         "roa-readonly",
			role:        model.SpaceRoleEditor,
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "閲覧者 (OAuthアプリの権限を持たない)",
			key:         "roa-deleteonly",
			role:        model.SpaceRoleViewer,
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "フィーチャーフラグが無効",
			key:         "roa-noflag",
			role:        model.SpaceRoleAdmin,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "publicクライアント",
			key:         "roa-public",
			role:        model.SpaceRoleAdmin,
			flagEnabled: true,
			public:      true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			f := setupPATMember(t, tx, tt.key, tt.role, tt.flagEnabled)
			builder := testutil.NewOAuthApplicationBuilder(t, tx).
				WithSpaceID(f.spaceID).
				WithClientID(tt.key + "-client")
			if !tt.public {
				builder = builder.WithConfidentialClientSecretDigest("roa_old_digest")
			}
			appID := builder.Build()

			_, err := newRegenerateOAuthApplicationClientSecretUsecaseForTest(q).Execute(context.Background(), RegenerateOAuthApplicationClientSecretInput{
				SpaceIdentifier:    model.SpaceIdentifier(tt.key),
				UserID:             f.userID,
				OAuthApplicationID: appID,
			})
			assertAppErrCode(t, err, tt.wantErrCode)

			stored, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), appID, f.spaceID)
			if err != nil {
				t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
			}
			switch {
			case tt.public && stored.ClientSecretDigest != nil:
				t.Errorf("publicクライアントのClientSecretDigest = %v、期待値 = nil", *stored.ClientSecretDigest)
			case !tt.public && (stored.ClientSecretDigest == nil || *stored.ClientSecretDigest != "roa_old_digest"):
				t.Errorf("ClientSecretDigest = %v、期待値は元のダイジェスト", stored.ClientSecretDigest)
			}
		})
	}
}
