package usecase

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newUpdateOAuthApplicationUsecaseForTest(q *query.Queries) *UpdateOAuthApplicationUsecase {
	return NewUpdateOAuthApplicationUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
		validator.NewOAuthApplicationUpdateValidator(),
	)
}

func TestUpdateOAuthApplicationUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("他のメンバーが作成したアプリの名前とリダイレクトURIを更新する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "uoa-update", []model.Scope{model.ScopeOAuthApplicationWrite}, true)
		// 作成したメンバーがいなくなったアプリ。閲覧者が作成したものでなくても編集できる
		appID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithClientID("uoa-update-client").
			WithConfidentialClientSecretDigest("uoa_update_digest").
			Build()

		output, err := newUpdateOAuthApplicationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthApplicationInput{
			SpaceIdentifier:    "uoa-update",
			UserID:             f.userID,
			OAuthApplicationID: appID,
			ExpectedVersion:    1,
			Name:               "編集したアプリ",
			RedirectURIs:       "https://example.com/new\nhttps://example.com/new",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		stored, err := repository.NewOAuthApplicationRepository(q).FindByIDAndSpaceID(context.Background(), appID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
		}
		if stored.Name != "編集したアプリ" || output.Application.Name != "編集したアプリ" {
			t.Errorf("Name = %q、期待値 = %q", stored.Name, "編集したアプリ")
		}
		if want := []string{"https://example.com/new"}; !slices.Equal(stored.RedirectURIs, want) {
			t.Errorf("RedirectURIs = %q、期待値 = %q", stored.RedirectURIs, want)
		}
		if !stored.IsConfidential() || stored.ClientSecretDigest == nil || *stored.ClientSecretDigest != "uoa_update_digest" {
			t.Errorf("種別かシークレットが変わった: %+v", stored)
		}
	})
}

func TestUpdateOAuthApplicationUsecase_Execute_更新できない(t *testing.T) {
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
			name:        "oauth_application:readだけを持つ",
			key:         "uoa-readonly",
			scopes:      []model.Scope{model.ScopeOAuthApplicationRead},
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "oauth_application:deleteだけを持つ",
			key:         "uoa-deleteonly",
			scopes:      []model.Scope{model.ScopeOAuthApplicationDelete},
			flagEnabled: true,
			wantErrCode: model.AppErrCodeForbidden,
		},
		{
			name:        "フィーチャーフラグが無効",
			key:         "uoa-noflag",
			scopes:      []model.Scope{model.ScopeOAuthApplicationWrite},
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "別のスペースのアプリ",
			key:         "uoa-otherspace",
			scopes:      []model.Scope{model.ScopeOAuthApplicationWrite},
			flagEnabled: true,
			otherSpace:  true,
			wantErrCode: model.AppErrCodeResourceNotFound,
		},
		{
			name:        "削除されたアプリ",
			key:         "uoa-discarded",
			scopes:      []model.Scope{model.ScopeOAuthApplicationWrite},
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
				WithClientID(tt.key + "-client").
				WithName("元の名前")
			if tt.discarded {
				builder = builder.WithDiscardedAt(time.Now())
			}
			appID := builder.Build()

			_, err := newUpdateOAuthApplicationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthApplicationInput{
				SpaceIdentifier:    model.SpaceIdentifier(tt.key),
				UserID:             f.userID,
				OAuthApplicationID: appID,
				ExpectedVersion:    1,
				Name:               "編集したアプリ",
				RedirectURIs:       "https://example.com/new",
			})
			assertAppErrCode(t, err, tt.wantErrCode)
			assertOAuthApplicationName(t, tx, appID, "元の名前")
		})
	}

	t.Run("入力が不正ならValidationErrorを返し更新しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupPATMember(t, tx, "uoa-invalid", []model.Scope{model.ScopeOAuthApplicationWrite}, true)
		appID := testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithClientID("uoa-invalid-client").
			WithName("元の名前").
			Build()

		ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
		_, err := newUpdateOAuthApplicationUsecaseForTest(q).Execute(ctx, UpdateOAuthApplicationInput{
			SpaceIdentifier:    "uoa-invalid",
			UserID:             f.userID,
			OAuthApplicationID: appID,
			ExpectedVersion:    1,
			Name:               "編集したアプリ",
			RedirectURIs:       "http://example.com/callback",
		})
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatalf("Execute()のエラー = %v、期待値 = ValidationError", err)
		}
		if !ve.HasFieldError("redirect_uris") {
			t.Error("redirect_urisのフィールドエラーが無い")
		}
		assertOAuthApplicationName(t, tx, appID, "元の名前")
	})
}

func TestUpdateOAuthApplicationUsecase_Execute_版が古い(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	f := setupPATMember(t, tx, "uoa-stale", []model.Scope{model.ScopeOAuthApplicationWrite}, true)
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("uoa-stale-client").
		WithName("元の名前").
		Build()
	uc := newUpdateOAuthApplicationUsecaseForTest(q)

	// 版1のフォームから先に保存し、版を2へ進める
	if _, err := uc.Execute(context.Background(), UpdateOAuthApplicationInput{
		SpaceIdentifier:    "uoa-stale",
		UserID:             f.userID,
		OAuthApplicationID: appID,
		ExpectedVersion:    1,
		Name:               "先に保存した名前",
		RedirectURIs:       "https://example.com/first",
	}); err != nil {
		t.Fatalf("先の保存のExecute()のエラー = %v", err)
	}

	_, err := uc.Execute(context.Background(), UpdateOAuthApplicationInput{
		SpaceIdentifier:    "uoa-stale",
		UserID:             f.userID,
		OAuthApplicationID: appID,
		ExpectedVersion:    1,
		Name:               "古いフォームの名前",
		RedirectURIs:       "https://example.com/second",
	})
	assertAppErrCode(t, err, model.AppErrCodeConflict)
	assertOAuthApplicationName(t, tx, appID, "先に保存した名前")
}

// assertOAuthApplicationNameは、アプリの名前がwantのままであることを確かめる。別のスペースの
// アプリや削除されたアプリも確かめられるよう、スペースと削除日時を問わずにIDで引く。
func assertOAuthApplicationName(t *testing.T, tx *sql.Tx, appID model.OAuthApplicationID, want string) {
	t.Helper()

	var name string
	if err := tx.QueryRowContext(context.Background(), `SELECT name FROM oauth_applications WHERE id = $1`, string(appID)).Scan(&name); err != nil {
		t.Fatalf("アプリの名前の取得に失敗: %v", err)
	}
	if name != want {
		t.Errorf("アプリの名前 = %q、期待値 = %q", name, want)
	}
}
