package usecase

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newCreateOAuthApplicationUsecaseForTest(q *query.Queries) *CreateOAuthApplicationUsecase {
	return NewCreateOAuthApplicationUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
		validator.NewOAuthApplicationCreateValidator(),
	)
}

func TestCreateOAuthApplicationUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("confidentialクライアントはシークレットを発行し、ダイジェストだけを保存する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "coa-confidential", []model.Scope{model.ScopeOAuthApplicationWrite}, true)

		output, err := newCreateOAuthApplicationUsecaseForTest(q).Execute(context.Background(), CreateOAuthApplicationInput{
			SpaceIdentifier: "coa-confidential",
			UserID:          f.userID,
			Name:            "Webアプリ",
			RedirectURIs:    "https://example.com/callback\nhttps://example.com/callback2",
			ClientType:      "confidential",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if !strings.HasPrefix(output.ClientSecret, string(auth.OAuthClientSecretPrefix)) {
			t.Errorf("ClientSecret = %q、期待値は接頭辞%qで始まる", output.ClientSecret, auth.OAuthClientSecretPrefix)
		}

		stored, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), output.Application.ID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
		}
		if stored == nil {
			t.Fatal("登録したアプリがデータベースに無い")
		}
		if stored.ClientSecretDigest == nil || *stored.ClientSecretDigest != auth.DigestOpaqueToken(output.ClientSecret) {
			t.Errorf("ClientSecretDigest = %v、期待値はシークレットのダイジェスト", stored.ClientSecretDigest)
		}
		if !stored.IsConfidential() {
			t.Error("IsConfidential() = false、期待値 = true")
		}
		if stored.ClientID == "" || stored.ClientID == output.ClientSecret {
			t.Errorf("ClientID = %q、期待値はシークレットと異なる値", stored.ClientID)
		}
		if stored.CreatedSpaceMemberID == nil || *stored.CreatedSpaceMemberID != f.spaceMemberID {
			t.Errorf("CreatedSpaceMemberID = %v、期待値 = %v", stored.CreatedSpaceMemberID, f.spaceMemberID)
		}
		if stored.Name != "Webアプリ" {
			t.Errorf("Name = %q、期待値 = %q", stored.Name, "Webアプリ")
		}
		if want := []string{"https://example.com/callback", "https://example.com/callback2"}; !slices.Equal(stored.RedirectURIs, want) {
			t.Errorf("RedirectURIs = %q、期待値 = %q", stored.RedirectURIs, want)
		}
	})

	t.Run("publicクライアントはシークレットを持たない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "coa-public", []model.Scope{model.ScopeOAuthApplicationWrite}, true)

		output, err := newCreateOAuthApplicationUsecaseForTest(q).Execute(context.Background(), CreateOAuthApplicationInput{
			SpaceIdentifier: "coa-public",
			UserID:          f.userID,
			Name:            "CLI",
			RedirectURIs:    "http://127.0.0.1/callback",
			ClientType:      "public",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.ClientSecret != "" {
			t.Errorf("ClientSecret = %q、期待値 = 空文字列", output.ClientSecret)
		}
		if output.Application.ClientSecretDigest != nil || output.Application.IsConfidential() {
			t.Errorf("登録したアプリ = %+v、期待値はシークレットを持たないpublicクライアント", output.Application)
		}
	})
}

func TestCreateOAuthApplicationUsecase_Execute_登録できない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         string
		scopes      []model.Scope
		flagEnabled bool
		wantErrCode model.AppErrorCode
	}{
		{
			name:        "oauth_application:readだけを持つ",
			key:         "coa-readonly",
			scopes:      []model.Scope{model.ScopeOAuthApplicationRead},
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "oauth_application:deleteだけを持つ",
			key:         "coa-deleteonly",
			scopes:      []model.Scope{model.ScopeOAuthApplicationDelete},
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "フィーチャーフラグが無効",
			key:         "coa-noflag",
			scopes:      []model.Scope{model.ScopeOAuthApplicationWrite},
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			f := setupPATMember(t, tx, tt.key, tt.scopes, tt.flagEnabled)

			_, err := newCreateOAuthApplicationUsecaseForTest(q).Execute(context.Background(), CreateOAuthApplicationInput{
				SpaceIdentifier: model.SpaceIdentifier(tt.key),
				UserID:          f.userID,
				Name:            "Webアプリ",
				RedirectURIs:    "https://example.com/callback",
				ClientType:      "confidential",
			})
			assertAppErrCode(t, err, tt.wantErrCode)
			assertNoOAuthApplication(t, q, f)
		})
	}

	t.Run("入力が不正ならValidationErrorを返し登録しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "coa-invalid", []model.Scope{model.ScopeOAuthApplicationWrite}, true)

		ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
		_, err := newCreateOAuthApplicationUsecaseForTest(q).Execute(ctx, CreateOAuthApplicationInput{
			SpaceIdentifier: "coa-invalid",
			UserID:          f.userID,
			Name:            "Webアプリ",
			RedirectURIs:    "http://example.com/callback",
			ClientType:      "confidential",
		})
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("Execute()のエラー = %v、期待値 = ValidationError", err)
		}
		if !ve.HasFieldError("redirect_uris") {
			t.Error("redirect_urisのフィールドエラーが無い")
		}
		assertNoOAuthApplication(t, q, f)
	})
}

// assertNoOAuthApplicationは、スペースにアプリが1つも作られていないことを確かめる。
func assertNoOAuthApplication(t *testing.T, q *query.Queries, f patMemberFixture) {
	t.Helper()

	apps, err := repository.NewOAuthApplicationRepository(q).ListBySpace(context.Background(), f.spaceID)
	if err != nil {
		t.Fatalf("ListBySpace()のエラー = %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("作られたアプリの件数 = %d、期待値 = 0", len(apps))
	}
}
