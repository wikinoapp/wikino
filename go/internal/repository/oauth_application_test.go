package repository

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestOAuthApplicationRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupPersonalAccessTokenFixture(t, tx, "oauth-app-create")
	secretDigest := "create_client_secret_digest"
	redirectURIs := []string{"https://example.com/callback", "https://example.com/callback2"}

	t.Run("confidentialクライアントはシークレットのダイジェストを持つ", func(t *testing.T) {
		app, err := repo.Create(ctx, CreateOAuthApplicationInput{
			SpaceID:              f.spaceID,
			CreatedSpaceMemberID: f.spaceMemberID,
			Name:                 "Webアプリ",
			ClientID:             "create-confidential-client",
			ClientSecretDigest:   &secretDigest,
			ClientType:           model.OAuthClientTypeConfidential,
			RedirectURIs:         redirectURIs,
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if app.ID == "" {
			t.Error("app.IDが空")
		}
		if app.SpaceID == nil || *app.SpaceID != f.spaceID {
			t.Errorf("app.SpaceID = %v、期待値 = %v", app.SpaceID, f.spaceID)
		}
		if app.CreatedSpaceMemberID == nil || *app.CreatedSpaceMemberID != f.spaceMemberID {
			t.Errorf("app.CreatedSpaceMemberID = %v、期待値 = %v", app.CreatedSpaceMemberID, f.spaceMemberID)
		}
		if app.Name != "Webアプリ" {
			t.Errorf("app.Name = %q、期待値 = %q", app.Name, "Webアプリ")
		}
		if app.ClientID != "create-confidential-client" {
			t.Errorf("app.ClientID = %q、期待値 = %q", app.ClientID, "create-confidential-client")
		}
		if app.ClientSecretDigest == nil || *app.ClientSecretDigest != secretDigest {
			t.Errorf("app.ClientSecretDigest = %v、期待値 = %q", app.ClientSecretDigest, secretDigest)
		}
		if !app.IsConfidential() {
			t.Error("app.IsConfidential() = false、期待値 = true")
		}
		if app.IsOfficialClient() {
			t.Error("app.IsOfficialClient() = true、期待値 = false")
		}
		if !slices.Equal(app.RedirectURIs, redirectURIs) {
			t.Errorf("app.RedirectURIs = %v、期待値 = %v", app.RedirectURIs, redirectURIs)
		}
		if app.DiscardedAt != nil {
			t.Errorf("app.DiscardedAt = %v、期待値 = nil", app.DiscardedAt)
		}
	})

	t.Run("publicクライアントはシークレットを持たない", func(t *testing.T) {
		app, err := repo.Create(ctx, CreateOAuthApplicationInput{
			SpaceID:              f.spaceID,
			CreatedSpaceMemberID: f.spaceMemberID,
			Name:                 "CLI",
			ClientID:             "create-public-client",
			ClientType:           model.OAuthClientTypePublic,
			RedirectURIs:         []string{"http://127.0.0.1/callback"},
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if app.ClientSecretDigest != nil {
			t.Errorf("app.ClientSecretDigest = %v、期待値 = nil", app.ClientSecretDigest)
		}
		if app.IsConfidential() {
			t.Error("app.IsConfidential() = true、期待値 = false")
		}
	})
}

func TestOAuthApplicationRepository_Create_Constraint(t *testing.T) {
	t.Parallel()

	secretDigest := "constraint_client_secret_digest"
	tests := []struct {
		name               string
		clientType         model.OAuthClientType
		clientSecretDigest *string
		otherSpaceMember   bool
		wantCode           string
		wantConstraint     string
	}{
		{
			name:           "シークレットの無いconfidentialクライアント",
			clientType:     model.OAuthClientTypeConfidential,
			wantCode:       "check_violation",
			wantConstraint: "chk_oauth_applications_client_secret_digest",
		},
		{
			name:               "シークレットを持つpublicクライアント",
			clientType:         model.OAuthClientTypePublic,
			clientSecretDigest: &secretDigest,
			wantCode:           "check_violation",
			wantConstraint:     "chk_oauth_applications_client_secret_digest",
		},
		{
			name:             "作成者が別のスペースのメンバー",
			clientType:       model.OAuthClientTypePublic,
			otherSpaceMember: true,
			wantCode:         "foreign_key_violation",
			wantConstraint:   "oauth_applications_created_space_member_id_space_id_fkey",
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))

			suffix := "oauth-app-constraint-" + string(rune('a'+i))
			f := setupPersonalAccessTokenFixture(t, tx, suffix)
			spaceMemberID := f.spaceMemberID
			if tt.otherSpaceMember {
				spaceMemberID = setupPersonalAccessTokenFixture(t, tx, suffix+"-other").spaceMemberID
			}

			_, err := repo.Create(context.Background(), CreateOAuthApplicationInput{
				SpaceID:              f.spaceID,
				CreatedSpaceMemberID: spaceMemberID,
				Name:                 "アプリ",
				ClientID:             suffix + "-client",
				ClientSecretDigest:   tt.clientSecretDigest,
				ClientType:           tt.clientType,
				RedirectURIs:         []string{"https://example.com/callback"},
			})
			assertConstraintViolation(t, err, tt.wantCode, tt.wantConstraint)
		})
	}
}

func TestOAuthApplicationRepository_CreatedSpaceMemberDeleted(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-member-deleted")
	if _, err := tx.ExecContext(ctx, "DELETE FROM space_members WHERE id = $1", f.spaceMemberID); err != nil {
		t.Fatalf("スペースメンバーの削除に失敗: %v", err)
	}

	app, err := repo.FindByClientID(ctx, "oauth-app-member-deleted-client")
	if err != nil {
		t.Fatalf("FindByClientID()のエラー = %v", err)
	}
	if app == nil || app.ID != f.oauthApplicationID {
		t.Fatalf("FindByClientID() = %v、期待値 = ID %v のアプリ", app, f.oauthApplicationID)
	}
	if app.SpaceID == nil || *app.SpaceID != f.spaceID {
		t.Errorf("app.SpaceID = %v、期待値 = %v", app.SpaceID, f.spaceID)
	}
	if app.CreatedSpaceMemberID != nil {
		t.Errorf("app.CreatedSpaceMemberID = %v、期待値 = nil", *app.CreatedSpaceMemberID)
	}
}

func TestOAuthApplicationRepository_FindByClientID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-find")
	officialID := testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithClientID("find-official-client").
		Build()
	testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("find-discarded-client").
		WithDiscardedAt(time.Now()).
		Build()

	tests := []struct {
		name     string
		clientID string
		want     model.OAuthApplicationID
	}{
		{name: "スペースのアプリを返す", clientID: "oauth-app-find-client", want: f.oauthApplicationID},
		{name: "公式クライアントを返す", clientID: "find-official-client", want: officialID},
		{name: "削除されたアプリは返さない", clientID: "find-discarded-client"},
		{name: "一致するアプリが無ければnilを返す", clientID: "find-missing-client"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := repo.FindByClientID(ctx, tt.clientID)
			if err != nil {
				t.Fatalf("FindByClientID()のエラー = %v", err)
			}
			switch {
			case tt.want == "" && app != nil:
				t.Errorf("FindByClientID() = %v、期待値 = nil", app)
			case tt.want != "" && (app == nil || app.ID != tt.want):
				t.Errorf("FindByClientID() = %v、期待値 = ID %v のアプリ", app, tt.want)
			}
		})
	}
}

func TestOAuthApplicationRepository_FindByClientIDAndSpaceID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-find-space")
	other := setupOAuthFixture(t, tx, "app-find-space-other")
	officialID := testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithClientID("find-space-official-client").
		Build()
	testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithClientID("find-space-discarded-official-client").
		WithDiscardedAt(time.Now()).
		Build()

	tests := []struct {
		name     string
		clientID string
		spaceID  model.SpaceID
		want     model.OAuthApplicationID
	}{
		{name: "そのスペースのアプリを返す", clientID: "oauth-app-find-space-client", spaceID: f.spaceID, want: f.oauthApplicationID},
		{name: "公式クライアントはどのスペースでも返す", clientID: "find-space-official-client", spaceID: other.spaceID, want: officialID},
		{name: "別のスペースのアプリは返さない", clientID: "oauth-app-find-space-client", spaceID: other.spaceID},
		{name: "削除された公式クライアントは返さない", clientID: "find-space-discarded-official-client", spaceID: f.spaceID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := repo.FindByClientIDAndSpaceID(ctx, tt.clientID, tt.spaceID)
			if err != nil {
				t.Fatalf("FindByClientIDAndSpaceID()のエラー = %v", err)
			}
			switch {
			case tt.want == "" && app != nil:
				t.Errorf("FindByClientIDAndSpaceID() = %v、期待値 = nil", app)
			case tt.want != "" && (app == nil || app.ID != tt.want):
				t.Errorf("FindByClientIDAndSpaceID() = %v、期待値 = ID %v のアプリ", app, tt.want)
			}
		})
	}
}

func TestOAuthApplicationRepository_FindByIDAndSpaceID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-find-id")
	other := setupOAuthFixture(t, tx, "app-find-id-other")
	discardedID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("find-id-discarded-client").
		WithDiscardedAt(time.Now()).
		Build()
	officialID := testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithClientID("find-id-official-client").
		Build()

	tests := []struct {
		name    string
		id      model.OAuthApplicationID
		spaceID model.SpaceID
		want    bool
	}{
		{name: "そのスペースのアプリを返す", id: f.oauthApplicationID, spaceID: f.spaceID, want: true},
		{name: "別のスペースのアプリは返さない", id: other.oauthApplicationID, spaceID: f.spaceID},
		{name: "削除されたアプリは返さない", id: discardedID, spaceID: f.spaceID},
		{name: "公式クライアントは返さない", id: officialID, spaceID: f.spaceID},
		{name: "UUIDでないIDは返さない", id: "not-a-uuid", spaceID: f.spaceID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := repo.FindByIDAndSpaceID(ctx, tt.id, tt.spaceID)
			if err != nil {
				t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
			}
			switch {
			case !tt.want && app != nil:
				t.Errorf("FindByIDAndSpaceID() = %v、期待値 = nil", app)
			case tt.want && (app == nil || app.ID != tt.id):
				t.Errorf("FindByIDAndSpaceID() = %v、期待値 = ID %v のアプリ", app, tt.id)
			}
		})
	}
}

func TestOAuthApplicationRepository_ListBySpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-list")
	setupOAuthFixture(t, tx, "app-list-other")
	newerID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("list-newer-client").
		Build()
	testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("list-discarded-client").
		WithDiscardedAt(time.Now()).
		Build()
	testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithClientID("list-official-client").
		Build()

	apps, err := repo.ListBySpace(ctx, f.spaceID)
	if err != nil {
		t.Fatalf("ListBySpace()のエラー = %v", err)
	}

	// 別のスペースのアプリ・削除されたアプリ・公式クライアントを除き、新しい順に並ぶ
	got := make([]model.OAuthApplicationID, len(apps))
	for i, app := range apps {
		got[i] = app.ID
	}
	want := []model.OAuthApplicationID{newerID, f.oauthApplicationID}
	if !slices.Equal(got, want) {
		t.Errorf("ListBySpace()のID = %v、期待値 = %v", got, want)
	}
}

func TestOAuthApplicationRepository_ListAvailableInSpaceByIDs(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-list-ids")
	other := setupOAuthFixture(t, tx, "app-list-ids-other")
	discardedID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("list-ids-discarded-client").
		WithDiscardedAt(time.Now()).
		Build()
	officialID := testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithClientID("list-ids-official-client").
		Build()
	unrequestedID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("list-ids-unrequested-client").
		Build()

	apps, err := repo.ListAvailableInSpaceByIDs(ctx, []model.OAuthApplicationID{
		f.oauthApplicationID, officialID, discardedID, other.oauthApplicationID,
	}, f.spaceID)
	if err != nil {
		t.Fatalf("ListAvailableInSpaceByIDs()のエラー = %v", err)
	}

	// 指定したIDのうち、スペースのアプリと公式クライアントだけを返す。削除されたアプリと
	// 別のスペースのアプリは除く
	got := make([]model.OAuthApplicationID, len(apps))
	for i, app := range apps {
		got[i] = app.ID
	}
	slices.Sort(got)
	want := []model.OAuthApplicationID{f.oauthApplicationID, officialID}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("ListAvailableInSpaceByIDs()のID = %v、期待値 = %v (指定していない%vは含まない)", got, want, unrequestedID)
	}

	empty, err := repo.ListAvailableInSpaceByIDs(ctx, nil, f.spaceID)
	if err != nil {
		t.Fatalf("IDが空のListAvailableInSpaceByIDs()のエラー = %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("IDが空のときの件数 = %d、期待値 = 0", len(empty))
	}
}

func TestOAuthApplicationRepository_Update(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-update")
	other := setupOAuthFixture(t, tx, "app-update-other")
	discardedID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("update-discarded-client").
		WithDiscardedAt(time.Now()).
		Build()
	redirectURIs := []string{"https://example.com/updated", "http://127.0.0.1/updated"}

	t.Run("名前とリダイレクトURIを更新し、それ以外は変えない", func(t *testing.T) {
		before, err := repo.FindByIDAndSpaceID(ctx, f.oauthApplicationID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
		}

		app, err := repo.Update(ctx, UpdateOAuthApplicationInput{
			ID:              f.oauthApplicationID,
			SpaceID:         f.spaceID,
			ExpectedVersion: before.Version,
			Name:            "更新したアプリ",
			RedirectURIs:    redirectURIs,
		})
		if err != nil {
			t.Fatalf("Update()のエラー = %v", err)
		}
		if app == nil {
			t.Fatal("Update() = nil、期待値は更新したアプリ")
		}
		if app.Name != "更新したアプリ" {
			t.Errorf("app.Name = %q、期待値 = %q", app.Name, "更新したアプリ")
		}
		if !slices.Equal(app.RedirectURIs, redirectURIs) {
			t.Errorf("app.RedirectURIs = %q、期待値 = %q", app.RedirectURIs, redirectURIs)
		}
		if app.ClientID != before.ClientID || app.ClientType != before.ClientType {
			t.Errorf("クライアントIDか種別が変わった: 更新前 = %+v、更新後 = %+v", before, app)
		}
		if app.Version != before.Version+1 {
			t.Errorf("app.Version = %d、期待値 = %d", app.Version, before.Version+1)
		}

		// 同じ版を前提にした2回目の更新は、先の更新で版が進んでいるため受け付けない
		stale, err := repo.Update(ctx, UpdateOAuthApplicationInput{
			ID:              f.oauthApplicationID,
			SpaceID:         f.spaceID,
			ExpectedVersion: before.Version,
			Name:            "古い版からの更新",
			RedirectURIs:    []string{"https://example.com/stale"},
		})
		if err != nil {
			t.Fatalf("古い版でのUpdate()のエラー = %v", err)
		}
		if stale != nil {
			t.Errorf("古い版でのUpdate() = %v、期待値 = nil", stale)
		}
		after, err := repo.FindByIDAndSpaceID(ctx, f.oauthApplicationID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
		}
		if after.Name != "更新したアプリ" || after.Version != app.Version {
			t.Errorf("古い版での更新の後のアプリ = %+v、期待値は1回目の更新の内容", after)
		}
	})

	tests := []struct {
		name    string
		id      model.OAuthApplicationID
		spaceID model.SpaceID
	}{
		{name: "別のスペースのアプリは更新しない", id: other.oauthApplicationID, spaceID: f.spaceID},
		{name: "削除されたアプリは更新しない", id: discardedID, spaceID: f.spaceID},
		{name: "UUIDでないIDは更新しない", id: "not-a-uuid", spaceID: f.spaceID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := repo.Update(ctx, UpdateOAuthApplicationInput{
				ID:              tt.id,
				SpaceID:         tt.spaceID,
				ExpectedVersion: 1,
				Name:            "更新しないアプリ",
				RedirectURIs:    redirectURIs,
			})
			if err != nil {
				t.Fatalf("Update()のエラー = %v", err)
			}
			if app != nil {
				t.Errorf("Update() = %v、期待値 = nil", app)
			}
		})
	}

	otherApp, err := repo.FindByIDAndSpaceID(ctx, other.oauthApplicationID, other.spaceID)
	if err != nil {
		t.Fatalf("FindByIDAndSpaceID()のエラー = %v", err)
	}
	if otherApp.Name == "更新しないアプリ" {
		t.Error("別のスペースのアプリが更新された")
	}
}

func TestOAuthApplicationRepository_UpdateClientSecretDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthApplicationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-secret")
	other := setupOAuthFixture(t, tx, "app-secret-other")
	confidentialID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("secret-confidential-client").
		WithConfidentialClientSecretDigest("old_client_secret_digest").
		Build()
	otherConfidentialID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(other.spaceID).
		WithClientID("secret-other-confidential-client").
		WithConfidentialClientSecretDigest("other_client_secret_digest").
		Build()
	discardedID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithClientID("secret-discarded-client").
		WithConfidentialClientSecretDigest("discarded_client_secret_digest").
		WithDiscardedAt(time.Now()).
		Build()

	t.Run("confidentialクライアントのダイジェストを置き換える", func(t *testing.T) {
		app, err := repo.UpdateClientSecretDigest(ctx, confidentialID, f.spaceID, "new_client_secret_digest")
		if err != nil {
			t.Fatalf("UpdateClientSecretDigest()のエラー = %v", err)
		}
		if app == nil || app.ClientSecretDigest == nil || *app.ClientSecretDigest != "new_client_secret_digest" {
			t.Errorf("UpdateClientSecretDigest() = %v、期待値は新しいダイジェストを持つアプリ", app)
		}
	})

	tests := []struct {
		name string
		id   model.OAuthApplicationID
	}{
		// setupOAuthFixtureのアプリはpublicクライアント
		{name: "publicクライアントにはダイジェストを持たせない", id: f.oauthApplicationID},
		{name: "別のスペースのアプリは置き換えない", id: otherConfidentialID},
		{name: "削除されたアプリは置き換えない", id: discardedID},
		{name: "UUIDでないIDは置き換えない", id: "not-a-uuid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := repo.UpdateClientSecretDigest(ctx, tt.id, f.spaceID, "unexpected_client_secret_digest")
			if err != nil {
				t.Fatalf("UpdateClientSecretDigest()のエラー = %v", err)
			}
			if app != nil {
				t.Errorf("UpdateClientSecretDigest() = %v、期待値 = nil", app)
			}
		})
	}
}

func TestOAuthApplicationRepository_Discard(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewOAuthApplicationRepository(q)
	grantRepo := NewOAuthGrantRepository(q)
	accessTokenRepo := NewOAuthAccessTokenRepository(q)
	refreshTokenRepo := NewOAuthRefreshTokenRepository(q)
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "app-discard")
	other := setupOAuthFixture(t, tx, "app-discard-other")
	revokedAt := time.Now().Add(-time.Hour).Truncate(time.Microsecond)

	// 削除するアプリには、有効な許可と、既に解除された許可があり、どちらにもトークンが残っている
	activeGrantID := f.buildGrant(t, tx)
	revokedGrantID := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithRevokedAt(revokedAt).
		Build()
	otherGrantID := other.buildGrant(t, tx)

	newTokens := func(grantID model.OAuthGrantID, spaceID model.SpaceID, prefix string) {
		testutil.NewOAuthAccessTokenBuilder(t, tx).
			WithOAuthGrantID(grantID).
			WithSpaceID(spaceID).
			WithTokenDigest(prefix + "_access_digest").
			Build()
		testutil.NewOAuthRefreshTokenBuilder(t, tx).
			WithOAuthGrantID(grantID).
			WithSpaceID(spaceID).
			WithTokenDigest(prefix + "_refresh_digest").
			Build()
	}
	newTokens(activeGrantID, f.spaceID, "discard_active")
	newTokens(revokedGrantID, f.spaceID, "discard_revoked_grant")
	newTokens(otherGrantID, other.spaceID, "discard_other")
	testutil.NewOAuthAccessTokenBuilder(t, tx).
		WithOAuthGrantID(activeGrantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest("discard_already_revoked_access_digest").
		WithRevokedAt(revokedAt).
		Build()

	// 別のスペースを指定したら何も変えない
	app, err := repo.Discard(ctx, f.oauthApplicationID, other.spaceID)
	if err != nil {
		t.Fatalf("別のスペースを指定したDiscard()のエラー = %v", err)
	}
	if app != nil {
		t.Fatalf("別のスペースを指定したDiscard() = %v、期待値 = nil", app)
	}
	if app, err := repo.Discard(ctx, "not-a-uuid", f.spaceID); err != nil || app != nil {
		t.Fatalf("UUIDでないIDのDiscard() = (%v, %v)、期待値 = (nil, nil)", app, err)
	}
	if grant, err := grantRepo.FindByID(ctx, activeGrantID, f.spaceID); err != nil || grant.RevokedAt != nil {
		t.Fatalf("別のスペースを指定したDiscard()の後の許可 = (%v, %v)、期待値は失効していない許可", grant, err)
	}

	app, err = repo.Discard(ctx, f.oauthApplicationID, f.spaceID)
	if err != nil {
		t.Fatalf("Discard()のエラー = %v", err)
	}
	if app == nil || app.DiscardedAt == nil {
		t.Fatalf("Discard() = %v、期待値は削除日時を持つアプリ", app)
	}
	if found, err := repo.FindByIDAndSpaceID(ctx, f.oauthApplicationID, f.spaceID); err != nil || found != nil {
		t.Errorf("削除後のFindByIDAndSpaceID() = (%v, %v)、期待値 = (nil, nil)", found, err)
	}

	// 2度目の削除は何も変えない
	if again, err := repo.Discard(ctx, f.oauthApplicationID, f.spaceID); err != nil || again != nil {
		t.Errorf("2度目のDiscard() = (%v, %v)、期待値 = (nil, nil)", again, err)
	}

	grantTests := []struct {
		name          string
		id            model.OAuthGrantID
		spaceID       model.SpaceID
		wantRevoked   bool
		wantRevokedAt *time.Time
	}{
		{name: "有効だった許可を失効する", id: activeGrantID, spaceID: f.spaceID, wantRevoked: true},
		{name: "解除済みの許可の失効日時は変えない", id: revokedGrantID, spaceID: f.spaceID, wantRevoked: true, wantRevokedAt: &revokedAt},
		{name: "別のアプリの許可は失効しない", id: otherGrantID, spaceID: other.spaceID},
	}
	for _, tt := range grantTests {
		grant, err := grantRepo.FindByID(ctx, tt.id, tt.spaceID)
		if err != nil {
			t.Fatalf("%s: FindByID()のエラー = %v", tt.name, err)
		}
		switch {
		case !tt.wantRevoked && grant.RevokedAt != nil:
			t.Errorf("%s: grant.RevokedAt = %v、期待値 = nil", tt.name, grant.RevokedAt)
		case tt.wantRevoked && grant.RevokedAt == nil:
			t.Errorf("%s: grant.RevokedAt = nil、期待値は失効日時", tt.name)
		case tt.wantRevokedAt != nil && !grant.RevokedAt.Equal(*tt.wantRevokedAt):
			t.Errorf("%s: grant.RevokedAt = %v、期待値 = %v", tt.name, grant.RevokedAt, *tt.wantRevokedAt)
		}
	}

	tokenTests := []struct {
		prefix      string
		wantRevoked bool
	}{
		{prefix: "discard_active", wantRevoked: true},
		{prefix: "discard_revoked_grant", wantRevoked: true},
		{prefix: "discard_other"},
	}
	for _, tt := range tokenTests {
		accessToken, err := accessTokenRepo.FindByTokenDigest(ctx, tt.prefix+"_access_digest")
		if err != nil {
			t.Fatalf("%s: アクセストークンのFindByTokenDigest()のエラー = %v", tt.prefix, err)
		}
		if got := accessToken.RevokedAt != nil; got != tt.wantRevoked {
			t.Errorf("%s: アクセストークンが失効しているか = %v、期待値 = %v", tt.prefix, got, tt.wantRevoked)
		}
		refreshToken, err := refreshTokenRepo.FindByTokenDigest(ctx, tt.prefix+"_refresh_digest")
		if err != nil {
			t.Fatalf("%s: リフレッシュトークンのFindByTokenDigest()のエラー = %v", tt.prefix, err)
		}
		if got := refreshToken.RevokedAt != nil; got != tt.wantRevoked {
			t.Errorf("%s: リフレッシュトークンが失効しているか = %v、期待値 = %v", tt.prefix, got, tt.wantRevoked)
		}
	}

	alreadyRevoked, err := accessTokenRepo.FindByTokenDigest(ctx, "discard_already_revoked_access_digest")
	if err != nil {
		t.Fatalf("FindByTokenDigest()のエラー = %v", err)
	}
	if alreadyRevoked.RevokedAt == nil || !alreadyRevoked.RevokedAt.Equal(revokedAt) {
		t.Errorf("失効済みのアクセストークンのRevokedAt = %v、期待値 = %v", alreadyRevoked.RevokedAt, revokedAt)
	}
}
