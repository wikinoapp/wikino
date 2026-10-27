package usecase

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newCreateOAuthAuthorizationUsecaseForTest(q *query.Queries) *CreateOAuthAuthorizationUsecase {
	return NewCreateOAuthAuthorizationUsecase(
		testutil.GetTestDB(), repository.NewOAuthGrantRepository(q),
		oauthAuthorizationTestConfig,
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewOAuthAuthorizationCodeRepository(q),
		validator.NewOAuthAuthorizationCreateValidator(),
	)
}

// buildUsedOAuthGrantsDBは、アプリappIDへのメンバーの失効していない許可をn件、最終利用日時の古い順に
// 作ってコミットし、IDを古い順に返す
func buildUsedOAuthGrantsDB(t *testing.T, appID model.OAuthApplicationID, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, n int) []model.OAuthGrantID {
	t.Helper()

	ids := make([]model.OAuthGrantID, n)
	testutil.CommitFixture(t, func(tx *sql.Tx) {
		for i := range ids {
			ids[i] = testutil.NewOAuthGrantBuilder(t, tx).
				WithOAuthApplicationID(appID).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
				WithLastUsedAt(time.Now().Add(-time.Duration(n-i) * time.Hour)).Build()
		}
	})
	return ids
}

// assertOAuthGrantsAfterEvictionは、上限まで許可があるところに1件作った後、既存の許可existing
// (最終利用日時の古い順) のうち先頭だけが押し出され、失効していない許可が上限の件数に収まっていることを
// 確かめる
func assertOAuthGrantsAfterEviction(t *testing.T, q *query.Queries, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, existing []model.OAuthGrantID) {
	t.Helper()

	ctx := context.Background()
	grantRepo := repository.NewOAuthGrantRepository(q)
	for i, id := range existing {
		grant, err := grantRepo.FindByID(ctx, id, spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if want := i == 0; grant.IsRevoked() != want {
			t.Errorf("%d番目に古い許可のIsRevoked() = %v、期待値 = %v", i+1, grant.IsRevoked(), want)
		}
	}
	grants, err := grantRepo.ListUnrevokedBySpaceMember(ctx, spaceID, spaceMemberID)
	if err != nil {
		t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
	}
	if len(grants) != model.OAuthGrantMaxPerApplicationMember {
		t.Errorf("失効していない許可の件数 = %d、期待値 = %d", len(grants), model.OAuthGrantMaxPerApplicationMember)
	}
}

func TestCreateOAuthAuthorizationUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("許可すると、選んだスコープで許可に属する認可コードを発行し、そのダイジェストだけを保存する", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthAuthorizationFixtureDB(t, "coauth-approve", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)
		params.RedirectURI = "http://127.0.0.1:53123/callback"
		params.DeviceName = " my-laptop "

		before := time.Now()
		output, err := newCreateOAuthAuthorizationUsecaseForTest(q).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:        f.userID,
			Params:        params,
			Approved:      true,
			GrantedScopes: []string{"topic:read"},
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
		if want := []model.Scope{model.ScopeTopicRead}; !slices.Equal(code.Scopes, want) {
			t.Errorf("code.Scopes = %v、期待値 = %v (要求のうち選んだスコープだけ)", code.Scopes, want)
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
			t.Fatalf("grant = %+v、期待値はログイン中のユーザーのメンバーの許可", grant)
		}
		if want := []model.Scope{model.ScopeTopicRead}; !slices.Equal(grant.Scopes, want) {
			t.Errorf("grant.Scopes = %v、期待値 = %v", grant.Scopes, want)
		}
		// 認可要求の端末名は、前後の空白を除いて許可に保存する
		if grant.DeviceName == nil || *grant.DeviceName != "my-laptop" {
			t.Errorf("grant.DeviceName = %v、期待値 = %q", grant.DeviceName, "my-laptop")
		}
	})

	t.Run("同じアプリ・メンバーの許可が上限に達していれば、最終利用が最も古い許可を押し出してから作る", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthAuthorizationFixtureDB(t, "coauth-evict", model.SpaceRoleEditor, true)
		app, err := repository.NewOAuthApplicationRepository(q).FindByClientID(context.Background(), f.clientID)
		if err != nil || app == nil {
			t.Fatalf("FindByClientID() = (%v, %v)、アプリを期待", app, err)
		}
		existing := buildUsedOAuthGrantsDB(t, app.ID, f.spaceID, f.spaceMemberID, model.OAuthGrantMaxPerApplicationMember)

		if _, err := newCreateOAuthAuthorizationUsecaseForTest(q).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:        f.userID,
			Params:        validOAuthAuthorizationParams(f.clientID),
			Approved:      true,
			GrantedScopes: []string{"page:write"},
		}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertOAuthGrantsAfterEviction(t, q, f.spaceID, f.spaceMemberID, existing)
	})

	t.Run("再認可すると既存の許可を変えずに、選んだスコープで別の許可を作る", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		q := query.New(db)
		f := setupOAuthAuthorizationFixtureDB(t, "coauth-another", model.SpaceRoleEditor, true)
		uc := newCreateOAuthAuthorizationUsecaseForTest(q)
		grantRepo := repository.NewOAuthGrantRepository(q)

		var grantIDs []model.OAuthGrantID
		for _, granted := range [][]string{{"topic:read", "page:write"}, {"page:write"}} {
			output, err := uc.Execute(context.Background(), CreateOAuthAuthorizationInput{
				UserID:        f.userID,
				Params:        validOAuthAuthorizationParams(f.clientID),
				Approved:      true,
				GrantedScopes: granted,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			code, err := repository.NewOAuthAuthorizationCodeRepository(q).FindByCodeDigest(context.Background(), auth.DigestOpaqueToken(output.Code))
			if err != nil {
				t.Fatalf("FindByCodeDigest()のエラー = %v", err)
			}
			grantIDs = append(grantIDs, code.OAuthGrantID)
		}
		if grantIDs[0] == grantIDs[1] {
			t.Fatalf("許可のID = %v、再認可で別の許可を作ることを期待", grantIDs)
		}

		for i, want := range [][]model.Scope{{model.ScopeTopicRead, model.ScopePageWrite}, {model.ScopePageWrite}} {
			grant, err := grantRepo.FindByID(context.Background(), grantIDs[i], f.spaceID)
			if err != nil {
				t.Fatalf("FindByID()のエラー = %v", err)
			}
			if grant == nil || grant.IsRevoked() || !model.HasAllScopes(grant.Scopes, want) || len(grant.Scopes) != len(want) {
				t.Errorf("%d件目の許可 = %+v、期待値 = 失効しておらずスコープが %v の許可", i+1, grant, want)
			}
		}
	})

	for _, tt := range []struct {
		name    string
		granted []string
	}{
		{name: "スコープを1つも選ばずに許可すると、ValidationErrorを返す", granted: nil},
		{name: "要求に無いスコープを送信すると、ValidationErrorを返す", granted: []string{"topic:read", "draft_page:read"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.GetTestDB()
			f := setupOAuthAuthorizationFixtureDB(t, "coauth-invalid-scopes-"+strconv.Itoa(len(tt.granted)), model.SpaceRoleViewer, true)

			output, err := newCreateOAuthAuthorizationUsecaseForTest(query.New(db)).Execute(context.Background(), CreateOAuthAuthorizationInput{
				UserID:        f.userID,
				Params:        validOAuthAuthorizationParams(f.clientID),
				Approved:      true,
				GrantedScopes: tt.granted,
			})
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("granted_scopes") {
				t.Errorf("エラー = %v、期待値 = granted_scopesのValidationError", err)
			}
			assertNoOAuthGrant(t, db, f)
		})
	}

	t.Run("拒否するとaccess_deniedを返し、コードを発行しない", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		f := setupOAuthAuthorizationFixtureDB(t, "coauth-deny", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)

		_, err := newCreateOAuthAuthorizationUsecaseForTest(query.New(db)).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   f.userID,
			Params:   params,
			Approved: false,
		})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorAccessDenied, params)
		assertNoOAuthGrant(t, db, f)
	})

	t.Run("メンバーでないユーザーが許可を送信するとaccess_deniedを返す", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		f := setupOAuthAuthorizationFixtureDB(t, "coauth-not-member", model.SpaceRoleViewer, true)
		var outsider patMemberFixture
		testutil.CommitFixture(t, func(tx *sql.Tx) {
			outsider = setupPATMember(t, tx, "coauth-not-member-outsider", model.SpaceRoleViewer, true)
		})
		params := validOAuthAuthorizationParams(f.clientID)

		_, err := newCreateOAuthAuthorizationUsecaseForTest(query.New(db)).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   outsider.userID,
			Params:   params,
			Approved: true,
		})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorAccessDenied, params)
		assertNoOAuthGrant(t, db, f)
	})

	t.Run("送信された要求も表示と同じく検証する", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		f := setupOAuthAuthorizationFixtureDB(t, "coauth-invalid", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)
		params.RedirectURI = "https://evil.example/callback"

		_, err := newCreateOAuthAuthorizationUsecaseForTest(query.New(db)).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   f.userID,
			Params:   params,
			Approved: true,
		})
		if oe := model.AsOAuthAuthorizationError(err); oe == nil || oe.IsRedirectable() {
			t.Errorf("エラー = %v、期待値 = クライアントへ戻さない*model.OAuthAuthorizationError", err)
		}
		assertNoOAuthGrant(t, db, f)
	})

	t.Run("フィーチャーフラグが無効なら見つからないとして答える", func(t *testing.T) {
		t.Parallel()

		db := testutil.GetTestDB()
		f := setupOAuthAuthorizationFixtureDB(t, "coauth-noflag", model.SpaceRoleViewer, false)

		_, err := newCreateOAuthAuthorizationUsecaseForTest(query.New(db)).Execute(context.Background(), CreateOAuthAuthorizationInput{
			UserID:   f.userID,
			Params:   validOAuthAuthorizationParams(f.clientID),
			Approved: true,
		})
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
		assertNoOAuthGrant(t, db, f)
	})
}

// assertNoOAuthGrantは、フィクスチャのアプリに許可が作られていないことを確かめる
func assertNoOAuthGrant(t *testing.T, db *sql.DB, f oauthAuthorizationFixture) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(),
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

func setupOAuthAuthorizationFixtureDB(t *testing.T, key string, role model.SpaceRole, flagEnabled bool) oauthAuthorizationFixture {
	t.Helper()
	var f oauthAuthorizationFixture
	testutil.CommitFixture(t, func(tx *sql.Tx) { f = setupOAuthAuthorizationFixture(t, tx, key, role, flagEnabled) })
	return f
}
