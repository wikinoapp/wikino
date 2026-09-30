package usecase

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newCreateOAuthAuthorizationUsecaseForTest(q *query.Queries) *CreateOAuthAuthorizationUsecase {
	return NewCreateOAuthAuthorizationUsecase(
		oauthAuthorizationTestConfig,
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewOAuthAuthorizationCodeRepository(q),
	)
}

func TestCreateOAuthAuthorizationUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("許可すると、許可に属する認可コードを発行し、そのダイジェストだけを保存する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupOAuthAuthorizationFixture(t, tx, "coa-approve", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)
		params.RedirectURI = "http://127.0.0.1:53123/callback"

		before := time.Now()
		output, err := newCreateOAuthAuthorizationUsecaseForTest(q).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   f.userID,
			Params:   params,
			Approved: true,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.RedirectURI != params.RedirectURI || output.State != params.State {
			t.Errorf("RedirectURI = %q・State = %q、期待値 = %q・%q", output.RedirectURI, output.State, params.RedirectURI, params.State)
		}
		if output.Code == "" {
			t.Fatal("Codeが空")
		}

		code, err := repository.NewOAuthAuthorizationCodeRepository(q).FindByCodeDigest(context.Background(), auth.DigestOpaqueToken(output.Code))
		if err != nil {
			t.Fatalf("FindByCodeDigest()のエラー = %v", err)
		}
		if code == nil {
			t.Fatal("発行したコードのダイジェストが保存されていない")
		}
		if code.SpaceID != f.spaceID {
			t.Errorf("code.SpaceID = %v、期待値 = %v", code.SpaceID, f.spaceID)
		}
		if want := []model.Scope{model.ScopeTopicRead, model.ScopePageWrite}; !slices.Equal(code.Scopes, want) {
			t.Errorf("code.Scopes = %v、期待値 = %v", code.Scopes, want)
		}
		if code.RedirectURI != params.RedirectURI {
			t.Errorf("code.RedirectURI = %q、期待値 = %q", code.RedirectURI, params.RedirectURI)
		}
		if code.CodeChallenge != params.CodeChallenge {
			t.Errorf("code.CodeChallenge = %q、期待値 = %q", code.CodeChallenge, params.CodeChallenge)
		}
		if lifetime := code.ExpiresAt.Sub(before); lifetime < model.OAuthAuthorizationCodeLifetime || lifetime > model.OAuthAuthorizationCodeLifetime+time.Minute {
			t.Errorf("code.ExpiresAt = %v、期待値は発行の%v後", code.ExpiresAt, model.OAuthAuthorizationCodeLifetime)
		}

		grant, err := repository.NewOAuthGrantRepository(q).FindByID(context.Background(), code.OAuthGrantID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if grant == nil || grant.SpaceMemberID != f.spaceMemberID {
			t.Errorf("grant = %+v、期待値はログイン中のユーザーのメンバーの許可", grant)
		}
	})

	t.Run("拒否するとaccess_deniedを返し、コードを発行しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "coa-deny", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)

		_, err := newCreateOAuthAuthorizationUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   f.userID,
			Params:   params,
			Approved: false,
		})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorAccessDenied, params)
		assertNoOAuthGrant(t, tx, f)
	})

	t.Run("メンバーでないユーザーが許可を送信するとaccess_deniedを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "coa-not-member", model.SpaceRoleViewer, true)
		outsider := setupPATMember(t, tx, "coa-not-member-outsider", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)

		_, err := newCreateOAuthAuthorizationUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   outsider.userID,
			Params:   params,
			Approved: true,
		})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorAccessDenied, params)
		assertNoOAuthGrant(t, tx, f)
	})

	t.Run("送信された要求も表示と同じく検証する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "coa-invalid", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)
		params.RedirectURI = "https://evil.example/callback"

		_, err := newCreateOAuthAuthorizationUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   f.userID,
			Params:   params,
			Approved: true,
		})
		if oe := model.AsOAuthAuthorizationError(err); oe == nil || oe.IsRedirectable() {
			t.Errorf("エラー = %v、期待値 = クライアントへ戻さない*model.OAuthAuthorizationError", err)
		}
		assertNoOAuthGrant(t, tx, f)
	})

	t.Run("フィーチャーフラグが無効なら見つからないとして答える", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "coa-noflag", model.SpaceRoleViewer, false)

		_, err := newCreateOAuthAuthorizationUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   f.userID,
			Params:   validOAuthAuthorizationParams(f.clientID),
			Approved: true,
		})
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
		assertNoOAuthGrant(t, tx, f)
	})
}

// assertNoOAuthGrantは、フィクスチャのアプリに許可が作られていないことを確かめる
func assertNoOAuthGrant(t *testing.T, tx *sql.Tx, f oauthAuthorizationFixture) {
	t.Helper()

	var count int
	err := tx.QueryRowContext(context.Background(),
		`SELECT count(*) FROM oauth_grants g
		 INNER JOIN oauth_applications a ON a.id = g.oauth_application_id
		 WHERE a.client_id = $1 AND g.space_id = $2`,
		f.clientID, f.spaceID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("許可の数え上げに失敗: %v", err)
	}
	if count != 0 {
		t.Errorf("許可の数 = %d、期待値 = 0", count)
	}
}
