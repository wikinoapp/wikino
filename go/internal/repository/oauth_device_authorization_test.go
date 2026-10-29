package repository

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestOAuthDeviceAuthorizationRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "device-create")
	expiresAt := time.Now().Add(model.OAuthDeviceAuthorizationLifetime).Truncate(time.Microsecond)
	scopes := []model.Scope{model.ScopePageRead, model.ScopePageWrite}
	deviceName := "my-laptop"

	d, err := repo.Create(ctx, CreateOAuthDeviceAuthorizationInput{
		OAuthApplicationID: f.oauthApplicationID,
		SpaceID:            f.spaceID,
		DeviceCodeDigest:   "device_create_device_code_digest",
		UserCodeDigest:     "device_create_user_code_digest",
		Scopes:             scopes,
		Interval:           5 * time.Second,
		ExpiresAt:          expiresAt,
		DeviceName:         &deviceName,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if d.DeviceName == nil || *d.DeviceName != deviceName {
		t.Errorf("d.DeviceName = %v、期待値 = %q", d.DeviceName, deviceName)
	}
	if d.ID == "" {
		t.Error("d.IDが空")
	}
	if d.OAuthApplicationID != f.oauthApplicationID {
		t.Errorf("d.OAuthApplicationID = %v、期待値 = %v", d.OAuthApplicationID, f.oauthApplicationID)
	}
	if d.SpaceID != f.spaceID {
		t.Errorf("d.SpaceID = %v、期待値 = %v", d.SpaceID, f.spaceID)
	}
	if d.OAuthGrantID != nil {
		t.Errorf("d.OAuthGrantID = %v、期待値 = nil", *d.OAuthGrantID)
	}
	if d.DeviceCodeDigest != "device_create_device_code_digest" {
		t.Errorf("d.DeviceCodeDigest = %q、期待値 = %q", d.DeviceCodeDigest, "device_create_device_code_digest")
	}
	if d.UserCodeDigest != "device_create_user_code_digest" {
		t.Errorf("d.UserCodeDigest = %q、期待値 = %q", d.UserCodeDigest, "device_create_user_code_digest")
	}
	if !slices.Equal(d.Scopes, scopes) {
		t.Errorf("d.Scopes = %v、期待値 = %v", d.Scopes, scopes)
	}
	if d.Status != model.OAuthDeviceAuthorizationStatusPending {
		t.Errorf("d.Status = %v、期待値 = %v", d.Status, model.OAuthDeviceAuthorizationStatusPending)
	}
	if d.Interval != 5*time.Second {
		t.Errorf("d.Interval = %v、期待値 = %v", d.Interval, 5*time.Second)
	}
	if !d.ExpiresAt.Equal(expiresAt) {
		t.Errorf("d.ExpiresAt = %v、期待値 = %v", d.ExpiresAt, expiresAt)
	}
	if d.LastPolledAt != nil {
		t.Errorf("d.LastPolledAt = %v、期待値 = nil", *d.LastPolledAt)
	}

	// ユーザーコードのダイジェストが重複すると作れない (トランザクションが中断するため最後に確かめる)
	_, err = repo.Create(ctx, CreateOAuthDeviceAuthorizationInput{
		OAuthApplicationID: f.oauthApplicationID,
		SpaceID:            f.spaceID,
		DeviceCodeDigest:   "device_create_other_device_code_digest",
		UserCodeDigest:     "device_create_user_code_digest",
		Scopes:             scopes,
		Interval:           5 * time.Second,
		ExpiresAt:          expiresAt,
	})
	assertConstraintViolation(t, err, "unique_violation", "idx_oauth_device_authorizations_user_code_digest")
}

func TestOAuthDeviceAuthorizationRepository_Create_RequiresGrantOnlyWhenApproved(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupOAuthFixture(t, tx, "device-grant-check")

	// 許可済みなのに許可を持たない要求は作れない
	_, err := tx.ExecContext(context.Background(),
		`INSERT INTO oauth_device_authorizations (oauth_application_id, space_id, device_code_digest, user_code_digest, status, interval_seconds, expires_at)
		 VALUES ($1, $2, 'device_grant_check_device', 'device_grant_check_user', 1, 5, NOW())`,
		f.oauthApplicationID, f.spaceID,
	)
	assertConstraintViolation(t, err, "check_violation", "chk_oauth_device_authorizations_oauth_grant_id")
}

func TestOAuthDeviceAuthorizationRepository_FindByDeviceCodeDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "device-find-device")
	grantID := f.buildGrant(t, tx)
	lastPolledAt := time.Now().Add(-time.Minute).Truncate(time.Microsecond)
	id := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithOAuthGrantID(grantID).
		WithStatus(model.OAuthDeviceAuthorizationStatusApproved).
		WithDeviceCodeDigest("device_find_device_code_digest").
		WithUserCodeDigest("device_find_device_user_code_digest").
		WithInterval(10 * time.Second).
		WithLastPolledAt(lastPolledAt).
		Build()

	d, err := repo.FindByDeviceCodeDigest(ctx, "device_find_device_code_digest")
	if err != nil {
		t.Fatalf("FindByDeviceCodeDigest()のエラー = %v", err)
	}
	if d == nil {
		t.Fatal("要求 = nil、非nilを期待")
	}
	if d.ID != id {
		t.Errorf("d.ID = %v、期待値 = %v", d.ID, id)
	}
	if d.OAuthGrantID == nil || *d.OAuthGrantID != grantID {
		t.Errorf("d.OAuthGrantID = %v、期待値 = %v", d.OAuthGrantID, grantID)
	}
	if d.Status != model.OAuthDeviceAuthorizationStatusApproved {
		t.Errorf("d.Status = %v、期待値 = %v", d.Status, model.OAuthDeviceAuthorizationStatusApproved)
	}
	if d.Interval != 10*time.Second {
		t.Errorf("d.Interval = %v、期待値 = %v", d.Interval, 10*time.Second)
	}
	if d.LastPolledAt == nil || !d.LastPolledAt.Equal(lastPolledAt) {
		t.Errorf("d.LastPolledAt = %v、期待値 = %v", d.LastPolledAt, lastPolledAt)
	}

	missing, err := repo.FindByDeviceCodeDigest(ctx, "device_find_missing_digest")
	if err != nil {
		t.Fatalf("FindByDeviceCodeDigest()のエラー = %v", err)
	}
	if missing != nil {
		t.Errorf("存在しないダイジェストの要求 = %v、期待値 = nil", missing)
	}
}

func TestOAuthDeviceAuthorizationRepository_FindByUserCodeDigest(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "device-find-user")
	id := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithDeviceCodeDigest("device_find_user_device_code_digest").
		WithUserCodeDigest("device_find_user_code_digest").
		Build()

	d, err := repo.FindByUserCodeDigest(ctx, "device_find_user_code_digest")
	if err != nil {
		t.Fatalf("FindByUserCodeDigest()のエラー = %v", err)
	}
	if d == nil {
		t.Fatal("要求 = nil、非nilを期待")
	}
	if d.ID != id {
		t.Errorf("d.ID = %v、期待値 = %v", d.ID, id)
	}
	if d.SpaceID != f.spaceID {
		t.Errorf("d.SpaceID = %v、期待値 = %v", d.SpaceID, f.spaceID)
	}

	missing, err := repo.FindByUserCodeDigest(ctx, "device_find_user_missing_digest")
	if err != nil {
		t.Fatalf("FindByUserCodeDigest()のエラー = %v", err)
	}
	if missing != nil {
		t.Errorf("存在しないダイジェストの要求 = %v、期待値 = nil", missing)
	}
}

func TestOAuthDeviceAuthorizationRepository_LockPending(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "device-lock-pending")
	grantID := f.buildGrant(t, tx)
	build := func(name string, status model.OAuthDeviceAuthorizationStatus, expiresAt time.Time) model.OAuthDeviceAuthorizationID {
		b := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
			WithOAuthApplicationID(f.oauthApplicationID).
			WithSpaceID(f.spaceID).
			WithStatus(status).
			WithExpiresAt(expiresAt).
			WithDeviceCodeDigest("device_lock_pending_" + name + "_device").
			WithUserCodeDigest("device_lock_pending_" + name + "_user")
		if status == model.OAuthDeviceAuthorizationStatusApproved || status == model.OAuthDeviceAuthorizationStatusExchanged {
			b = b.WithOAuthGrantID(grantID)
		}
		return b.Build()
	}
	future := time.Now().Add(time.Minute)

	pendingID := build("pending", model.OAuthDeviceAuthorizationStatusPending, future)
	otherSpace := setupOAuthFixture(t, tx, "device-lock-pending-other")
	tests := []struct {
		name    string
		id      model.OAuthDeviceAuthorizationID
		spaceID model.SpaceID
		want    bool
	}{
		{name: "許可待ちで期限内なら返す", id: pendingID, spaceID: f.spaceID, want: true},
		{name: "別のスペースを指定すると返さない", id: pendingID, spaceID: otherSpace.spaceID, want: false},
		{name: "期限切れなら返さない", id: build("expired", model.OAuthDeviceAuthorizationStatusPending, time.Now().Add(-time.Second)), spaceID: f.spaceID, want: false},
		{name: "許可済みなら返さない", id: build("approved", model.OAuthDeviceAuthorizationStatusApproved, future), spaceID: f.spaceID, want: false},
		{name: "拒否済みなら返さない", id: build("denied", model.OAuthDeviceAuthorizationStatusDenied, future), spaceID: f.spaceID, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := repo.LockPending(ctx, tt.id, tt.spaceID)
			if err != nil {
				t.Fatalf("LockPending()のエラー = %v", err)
			}
			if got := d != nil; got != tt.want {
				t.Errorf("要求を返したか = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestOAuthDeviceAuthorizationRepository_Deny(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "device-deny")
	id := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithDeviceCodeDigest("device_deny_device").
		WithUserCodeDigest("device_deny_user").
		Build()

	d, err := repo.Deny(ctx, id, f.spaceID)
	if err != nil {
		t.Fatalf("Deny()のエラー = %v", err)
	}
	if d == nil || d.Status != model.OAuthDeviceAuthorizationStatusDenied {
		t.Fatalf("要求 = %+v、拒否済みを期待", d)
	}

	// 拒否済みの要求は許可待ちでないため、もう一度は拒否できない
	again, err := repo.Deny(ctx, id, f.spaceID)
	if err != nil {
		t.Fatalf("Deny()のエラー = %v", err)
	}
	if again != nil {
		t.Errorf("2回目の拒否の結果 = %+v、期待値 = nil", again)
	}

	expiredID := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithExpiresAt(time.Now().Add(-time.Second)).
		WithDeviceCodeDigest("device_deny_expired_device").
		WithUserCodeDigest("device_deny_expired_user").
		Build()
	expired, err := repo.Deny(ctx, expiredID, f.spaceID)
	if err != nil {
		t.Fatalf("Deny()のエラー = %v", err)
	}
	if expired != nil {
		t.Errorf("期限切れの要求の拒否の結果 = %+v、期待値 = nil", expired)
	}
}

func TestOAuthDeviceAuthorizationRepository_ApproveWithGrant(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	grantRepo := NewOAuthGrantRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "device-approve")
	build := func(name string) model.OAuthDeviceAuthorizationID {
		return testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
			WithOAuthApplicationID(f.oauthApplicationID).
			WithSpaceID(f.spaceID).
			WithScopes([]model.Scope{model.ScopePageRead, model.ScopePageWrite, model.ScopeTopicRead}).
			WithDeviceCodeDigest("device_approve_" + name + "_device").
			WithUserCodeDigest("device_approve_" + name + "_user").
			WithDeviceName("device-" + name).
			Build()
	}
	input := ApproveOAuthDeviceAuthorizationWithGrantInput{
		ID:                 build("first"),
		OAuthApplicationID: f.oauthApplicationID,
		SpaceID:            f.spaceID,
		SpaceMemberID:      f.spaceMemberID,
		Scopes:             []model.Scope{model.ScopeTopicRead},
	}

	// 許可を作り、要求を許可済みにして選んだスコープに置き換える
	first, err := repo.ApproveWithGrant(ctx, input)
	if err != nil {
		t.Fatalf("ApproveWithGrant()のエラー = %v", err)
	}
	if first == nil || first.Status != model.OAuthDeviceAuthorizationStatusApproved || first.OAuthGrantID == nil {
		t.Fatalf("要求 = %+v、許可を持つ許可済みの要求を期待", first)
	}
	if !slices.Equal(first.Scopes, input.Scopes) {
		t.Errorf("first.Scopes = %v、期待値 = %v", first.Scopes, input.Scopes)
	}
	grantID := *first.OAuthGrantID
	grant, err := grantRepo.FindByID(ctx, grantID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if grant == nil || grant.OAuthApplicationID != f.oauthApplicationID || grant.SpaceMemberID != f.spaceMemberID {
		t.Fatalf("grant = %+v、フィクスチャのアプリ・メンバーの許可を期待", grant)
	}
	// 要求で受け取った端末名を許可へ写す
	if grant.DeviceName == nil || *grant.DeviceName != "device-first" {
		t.Errorf("grant.DeviceName = %v、期待値 = %q", grant.DeviceName, "device-first")
	}

	// 許可済みの要求はもう一度は許可できない。許可も作らない
	var grantCount int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM oauth_grants WHERE oauth_application_id = $1 AND space_id = $2", string(f.oauthApplicationID), string(f.spaceID)).Scan(&grantCount); err != nil {
		t.Fatalf("許可の件数の取得に失敗: %v", err)
	}
	again, err := repo.ApproveWithGrant(ctx, input)
	if err != nil {
		t.Fatalf("ApproveWithGrant()のエラー = %v", err)
	}
	if again != nil {
		t.Errorf("2回目の許可の結果 = %+v、期待値 = nil", again)
	}
	var grantCountAfter int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM oauth_grants WHERE oauth_application_id = $1 AND space_id = $2", string(f.oauthApplicationID), string(f.spaceID)).Scan(&grantCountAfter); err != nil {
		t.Fatalf("許可の件数の取得に失敗: %v", err)
	}
	if grantCountAfter != grantCount {
		t.Errorf("許可済みの要求への許可で許可が作られた (件数 %d → %d)", grantCount, grantCountAfter)
	}

	// 失効していない許可があっても、要求ごとに別の許可を作る。既存の許可のスコープとトークンは、
	// 今回のスコープに収まらなくても変えない
	refreshTokenID := testutil.NewOAuthRefreshTokenBuilder(t, tx).
		WithOAuthGrantID(grantID).
		WithSpaceID(f.spaceID).
		WithTokenDigest("device_approve_refresh_digest").
		WithScopes([]model.Scope{model.ScopeTopicRead}).
		Build()

	input.ID = build("second")
	input.Scopes = []model.Scope{model.ScopePageRead, model.ScopePageWrite}
	second, err := repo.ApproveWithGrant(ctx, input)
	if err != nil {
		t.Fatalf("ApproveWithGrant()のエラー = %v", err)
	}
	if second == nil || second.OAuthGrantID == nil || *second.OAuthGrantID == grantID {
		t.Fatalf("要求 = %+v、既存の許可 %vとは別の許可を結び付けることを期待", second, grantID)
	}
	grant, err = grantRepo.FindByID(ctx, *second.OAuthGrantID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if grant == nil || !slices.Equal(grant.Scopes, input.Scopes) {
		t.Errorf("grant = %+v、期待値 = スコープが %v の許可", grant, input.Scopes)
	}
	firstGrant, err := grantRepo.FindByID(ctx, grantID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if want := []model.Scope{model.ScopeTopicRead}; firstGrant == nil || firstGrant.IsRevoked() || !slices.Equal(firstGrant.Scopes, want) {
		t.Errorf("既存の許可 = %+v、期待値 = 失効しておらずスコープが %v のまま", firstGrant, want)
	}
	var revoked bool
	if err := tx.QueryRowContext(ctx, "SELECT revoked_at IS NOT NULL FROM oauth_refresh_tokens WHERE id = $1", string(refreshTokenID)).Scan(&revoked); err != nil {
		t.Fatalf("oauth_refresh_tokensの取得に失敗: %v", err)
	}
	if revoked {
		t.Error("既存の許可のリフレッシュトークンが失効した")
	}
}

func TestOAuthDeviceAuthorizationRepository_RecordPoll(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "device-poll")
	id := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithDeviceCodeDigest("device_poll_device").
		WithUserCodeDigest("device_poll_user").
		Build()
	find := func() *model.OAuthDeviceAuthorization {
		t.Helper()
		d, err := repo.FindByDeviceCodeDigest(ctx, "device_poll_device")
		if err != nil {
			t.Fatalf("FindByDeviceCodeDigest()のエラー = %v", err)
		}
		return d
	}
	start := time.Now().Truncate(time.Microsecond)

	tests := []struct {
		name         string
		now          time.Time
		wantTooFast  bool
		wantInterval time.Duration
	}{
		// 初回のポーリングは前回が無いため、間隔を問わない
		{name: "初回", now: start, wantTooFast: false, wantInterval: 5 * time.Second},
		{name: "間隔より早い", now: start.Add(4 * time.Second), wantTooFast: true, wantInterval: 10 * time.Second},
		// 延ばした間隔は、延ばしたポーリングの日時から数える
		{name: "延ばした間隔より早い", now: start.Add(13 * time.Second), wantTooFast: true, wantInterval: 15 * time.Second},
		{name: "間隔を空けた", now: start.Add(28 * time.Second), wantTooFast: false, wantInterval: 15 * time.Second},
	}
	for _, tt := range tests {
		tooFast, err := repo.RecordPoll(ctx, id, f.spaceID, 5*time.Second, tt.now)
		if err != nil {
			t.Fatalf("%s: RecordPoll()のエラー = %v", tt.name, err)
		}
		if tooFast != tt.wantTooFast {
			t.Errorf("%s: tooFast = %v、期待値 = %v", tt.name, tooFast, tt.wantTooFast)
		}
		d := find()
		if d.Interval != tt.wantInterval {
			t.Errorf("%s: Interval = %v、期待値 = %v", tt.name, d.Interval, tt.wantInterval)
		}
		if d.LastPolledAt == nil || !d.LastPolledAt.Equal(tt.now) {
			t.Errorf("%s: LastPolledAt = %v、期待値 = %v", tt.name, d.LastPolledAt, tt.now)
		}
	}

	// 許可待ちでない要求へのポーリングは記録しない
	if _, err := repo.Deny(ctx, id, f.spaceID); err != nil {
		t.Fatalf("Deny()のエラー = %v", err)
	}
	tooFast, err := repo.RecordPoll(ctx, id, f.spaceID, 5*time.Second, start.Add(29*time.Second))
	if err != nil {
		t.Fatalf("RecordPoll()のエラー = %v", err)
	}
	if tooFast {
		t.Error("拒否済みの要求へのポーリングのtooFast = true、期待値 = false")
	}
	if d := find(); d.Interval != 15*time.Second || !d.LastPolledAt.Equal(start.Add(28*time.Second)) {
		t.Errorf("拒否済みの要求 = %+v、ポーリングを記録しないことを期待", d)
	}

	// 別のスペースを指定しても記録しない
	other := setupOAuthFixture(t, tx, "device-poll-other")
	pendingID := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithDeviceCodeDigest("device_poll_other_device").
		WithUserCodeDigest("device_poll_other_user").
		Build()
	if _, err := repo.RecordPoll(ctx, pendingID, other.spaceID, 5*time.Second, start); err != nil {
		t.Fatalf("RecordPoll()のエラー = %v", err)
	}
	d, err := repo.FindByDeviceCodeDigest(ctx, "device_poll_other_device")
	if err != nil {
		t.Fatalf("FindByDeviceCodeDigest()のエラー = %v", err)
	}
	if d.LastPolledAt != nil {
		t.Errorf("別のスペースを指定したLastPolledAt = %v、期待値 = nil", *d.LastPolledAt)
	}
}

func TestOAuthDeviceAuthorizationRepository_Exchange(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupOAuthFixture(t, tx, "device-exchange")
	grantID := testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.spaceMemberID).
		WithScopes([]model.Scope{model.ScopePageRead, model.ScopeTopicRead}).
		Build()
	scopes := []model.Scope{model.ScopePageRead}
	build := func(name string, status model.OAuthDeviceAuthorizationStatus) model.OAuthDeviceAuthorizationID {
		b := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
			WithOAuthApplicationID(f.oauthApplicationID).
			WithSpaceID(f.spaceID).
			WithScopes(scopes).
			WithStatus(status).
			WithDeviceCodeDigest("device_exchange_" + name + "_device").
			WithUserCodeDigest("device_exchange_" + name + "_user")
		if status == model.OAuthDeviceAuthorizationStatusApproved {
			b = b.WithOAuthGrantID(grantID)
		}
		return b.Build()
	}
	now := time.Now()
	input := ExchangeOAuthDeviceAuthorizationInput{
		ID:                    build("approved", model.OAuthDeviceAuthorizationStatusApproved),
		SpaceID:               f.spaceID,
		AccessTokenDigest:     "device_exchange_access_digest",
		AccessTokenExpiresAt:  now.Add(time.Hour),
		RefreshTokenDigest:    "device_exchange_refresh_digest",
		RefreshTokenExpiresAt: now.Add(24 * time.Hour),
		Audience:              model.OAuthAudienceMCPServer,
		Now:                   now,
	}

	exchanged, err := repo.Exchange(ctx, input)
	if err != nil {
		t.Fatalf("Exchange()のエラー = %v", err)
	}
	if exchanged == nil || exchanged.Status != model.OAuthDeviceAuthorizationStatusExchanged {
		t.Fatalf("要求 = %+v、交換済みを期待", exchanged)
	}

	// 要求の許可・スコープと、入力の宛先でトークンを作る
	for _, tt := range []struct {
		table  string
		digest string
	}{
		{table: "oauth_access_tokens", digest: input.AccessTokenDigest},
		{table: "oauth_refresh_tokens", digest: input.RefreshTokenDigest},
	} {
		var gotGrantID string
		var gotScopes []byte
		var gotAudience int32
		if err := tx.QueryRowContext(ctx, "SELECT oauth_grant_id, scopes, audience FROM "+tt.table+" WHERE token_digest = $1 AND space_id = $2", tt.digest, string(f.spaceID)).
			Scan(&gotGrantID, &gotScopes, &gotAudience); err != nil {
			t.Fatalf("%sの取得に失敗: %v", tt.table, err)
		}
		if gotGrantID != string(grantID) || string(gotScopes) != "{page:read}" || model.OAuthAudience(gotAudience) != input.Audience {
			t.Errorf("%s = (%s, %s, %d)、期待値 = (%s, {page:read}, %d)", tt.table, gotGrantID, gotScopes, gotAudience, grantID, input.Audience)
		}
	}

	// 交換済み・許可待ちの要求は交換できず、トークンも作らない
	for _, tt := range []struct {
		name string
		id   model.OAuthDeviceAuthorizationID
	}{
		{name: "交換済み", id: input.ID},
		{name: "許可待ち", id: build("pending", model.OAuthDeviceAuthorizationStatusPending)},
	} {
		in := input
		in.ID = tt.id
		in.AccessTokenDigest = "device_exchange_" + string(tt.id) + "_access"
		in.RefreshTokenDigest = "device_exchange_" + string(tt.id) + "_refresh"
		got, err := repo.Exchange(ctx, in)
		if err != nil {
			t.Fatalf("%s: Exchange()のエラー = %v", tt.name, err)
		}
		if got != nil {
			t.Errorf("%s: 交換の結果 = %+v、期待値 = nil", tt.name, got)
		}
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM oauth_access_tokens WHERE token_digest = $1", in.AccessTokenDigest).Scan(&count); err != nil {
			t.Fatalf("%s: アクセストークンの件数の取得に失敗: %v", tt.name, err)
		}
		if count != 0 {
			t.Errorf("%s: 作られたアクセストークン = %d件、期待値 = 0件", tt.name, count)
		}
	}
}

func TestOAuthDeviceAuthorizationRepository_LockByID(t *testing.T) {
	t.Parallel()
	_, tx := testutil.SetupTx(t)
	repo := NewOAuthDeviceAuthorizationRepository(testutil.QueriesWithTx(tx))
	f := setupOAuthFixture(t, tx, "device-lock-id")
	other := setupOAuthFixture(t, tx, "device-lock-id-other")
	grantID := f.buildGrant(t, tx)
	// 期限切れの交換済みも返し、期限切れと再提示を呼び出し側で区別できることを確かめる。
	expiresAt := time.Now().Add(-time.Second).Truncate(time.Microsecond)
	id := testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
		WithOAuthApplicationID(f.oauthApplicationID).WithSpaceID(f.spaceID).
		WithOAuthGrantID(grantID).WithStatus(model.OAuthDeviceAuthorizationStatusExchanged).
		WithDeviceCodeDigest("device_lock_id_device").WithUserCodeDigest("device_lock_id_user").
		WithExpiresAt(expiresAt).Build()
	locked, err := repo.LockByID(t.Context(), id, f.spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if locked == nil || locked.ID != id || locked.Status != model.OAuthDeviceAuthorizationStatusExchanged || !locked.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("ロックした要求 = %+v、期限切れの交換済み要求を期待", locked)
	}
	locked, err = repo.LockByID(t.Context(), id, other.spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if locked != nil {
		t.Error("別のスペースの要求をロックできました")
	}
}
