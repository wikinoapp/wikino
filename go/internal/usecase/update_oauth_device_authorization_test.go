package usecase

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newUpdateOAuthDeviceAuthorizationUsecaseForTest(q *query.Queries) *UpdateOAuthDeviceAuthorizationUsecase {
	return NewUpdateOAuthDeviceAuthorizationUsecase(
		testutil.GetTestDB(),
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthDeviceAuthorizationRepository(q),
		repository.NewOAuthGrantRepository(q),
		repository.NewSpaceMemberRepository(q),
		validator.NewOAuthDeviceVerificationValidator(repository.NewOAuthDeviceAuthorizationRepository(q), repository.NewOAuthApplicationRepository(q), repository.NewSpaceRepository(q)),
		validator.NewOAuthAuthorizationCreateValidator(),
	)
}

// findOAuthDeviceAuthorizationForTestは、フィクスチャのデバイス認可の要求を今の状態で取得する
func findOAuthDeviceAuthorizationForTest(t *testing.T, q *query.Queries, f oauthDeviceVerificationFixture) *model.OAuthDeviceAuthorization {
	t.Helper()

	d, err := repository.NewOAuthDeviceAuthorizationRepository(q).FindByDeviceCodeDigest(context.Background(), f.deviceCodeDigest)
	if err != nil {
		t.Fatalf("FindByDeviceCodeDigest()のエラー = %v", err)
	}
	if d == nil {
		t.Fatal("デバイス認可の要求が見つからない")
	}
	return d
}

func TestUpdateOAuthDeviceAuthorizationUsecase_Execute(t *testing.T) {
	t.Parallel()

	q := query.New(testutil.GetTestDB())

	t.Run("許可すると、選んだスコープで許可を作り、要求を許可済みにして許可を結び付ける", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devupdate-approve", model.SpaceRoleEditor, true, func(b *testutil.OAuthDeviceAuthorizationBuilder) {
			b.WithDeviceName("my-laptop")
		})
		output, err := newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthDeviceAuthorizationInput{
			UserID:        f.userID,
			UserCode:      f.userCode,
			Approved:      true,
			GrantedScopes: []string{"page:read"},
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if !output.Approved || output.DeniedReason != "" || output.Application.ID != f.applicationID {
			t.Fatalf("output = %+v、アプリの許可を期待", output)
		}

		d := findOAuthDeviceAuthorizationForTest(t, q, f)
		if d.Status != model.OAuthDeviceAuthorizationStatusApproved || d.OAuthGrantID == nil {
			t.Fatalf("要求 = %+v、許可を持つ許可済みの要求を期待", d)
		}
		if want := []model.Scope{model.ScopePageRead}; !slices.Equal(d.Scopes, want) {
			t.Errorf("要求のスコープ = %v、期待値 = %v (選んだスコープだけ)", d.Scopes, want)
		}
		grant, err := repository.NewOAuthGrantRepository(q).FindByID(context.Background(), *d.OAuthGrantID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if grant == nil || grant.SpaceMemberID != f.spaceMemberID || grant.OAuthApplicationID != f.applicationID {
			t.Fatalf("grant = %+v、ログイン中のユーザーのメンバーのアプリへの許可を期待", grant)
		}
		if want := []model.Scope{model.ScopePageRead}; !slices.Equal(grant.Scopes, want) {
			t.Errorf("許可のスコープ = %v、期待値 = %v", grant.Scopes, want)
		}
		// 要求で受け取った端末名を許可へ写す
		if grant.DeviceName == nil || *grant.DeviceName != "my-laptop" {
			t.Errorf("許可の端末名 = %v、期待値 = %q", grant.DeviceName, "my-laptop")
		}

		// 許可済みのコードは使用済みとして、もう受け付けない
		_, err = newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthDeviceAuthorizationInput{
			UserID:        f.userID,
			UserCode:      f.userCode,
			Approved:      true,
			GrantedScopes: []string{"page:read"},
		})
		assertInvalidOAuthDeviceUserCode(t, err)
	})

	t.Run("同じアプリ・メンバーの許可が上限に達していれば、最終利用が最も古い許可を押し出してから作る", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devupdate-evict", model.SpaceRoleEditor, true, nil)
		existing := buildUsedOAuthGrantsDB(t, f.applicationID, f.spaceID, f.spaceMemberID, model.OAuthGrantMaxPerApplicationMember)

		if _, err := newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthDeviceAuthorizationInput{
			UserID:        f.userID,
			UserCode:      f.userCode,
			Approved:      true,
			GrantedScopes: []string{"page:read"},
		}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		assertOAuthGrantsAfterEviction(t, q, f.spaceID, f.spaceMemberID, existing)
	})

	t.Run("拒否すると、許可を作らず要求を拒否済みにする", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devupdate-deny", model.SpaceRoleEditor, true, nil)
		output, err := newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthDeviceAuthorizationInput{
			UserID:   f.userID,
			UserCode: f.userCode,
			Approved: false,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Approved {
			t.Error("Approved = true、期待値 = false")
		}
		d := findOAuthDeviceAuthorizationForTest(t, q, f)
		if d.Status != model.OAuthDeviceAuthorizationStatusDenied || d.OAuthGrantID != nil {
			t.Errorf("要求 = %+v、許可を持たない拒否済みの要求を期待", d)
		}
	})

	t.Run("選んだスコープが不正なら、要求を変えずにスコープのエラーを返す", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			key     string
			granted []string
		}{
			{name: "1つも選んでいない", key: "devupdate-scope-empty", granted: nil},
			{name: "要求に無いスコープを含む", key: "devupdate-scope-extra", granted: []string{"page:read", "topic:write"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				f := setupOAuthDeviceVerificationFixture(t, tt.key, model.SpaceRoleEditor, true, nil)
				_, err := newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthDeviceAuthorizationInput{
					UserID:        f.userID,
					UserCode:      f.userCode,
					Approved:      true,
					GrantedScopes: tt.granted,
				})
				ve := model.AsValidationError(err)
				if ve == nil || !ve.HasFieldError("granted_scopes") {
					t.Fatalf("エラー = %v、granted_scopesのValidationErrorを期待", err)
				}
				if d := findOAuthDeviceAuthorizationForTest(t, q, f); d.Status != model.OAuthDeviceAuthorizationStatusPending {
					t.Errorf("要求の状態 = %v、期待値は許可待ちのまま", d.Status)
				}
			})
		}
	})

	t.Run("連携先のスペースのメンバーでなければ、許可せず理由を返す", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devupdate-outsider", model.SpaceRoleEditor, true, nil)
		outsider := setupOAuthDeviceVerificationFixture(t, "devupdate-outsider-user", model.SpaceRoleEditor, true, nil)
		output, err := newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthDeviceAuthorizationInput{
			UserID:        outsider.userID,
			UserCode:      f.userCode,
			Approved:      true,
			GrantedScopes: []string{"page:read"},
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Approved || output.DeniedReason != OAuthAuthorizationDeniedNotMember {
			t.Errorf("output = %+v、メンバーでないことを理由に許可しないことを期待", output)
		}
		if d := findOAuthDeviceAuthorizationForTest(t, q, f); d.Status != model.OAuthDeviceAuthorizationStatusPending {
			t.Errorf("要求の状態 = %v、期待値は許可待ちのまま", d.Status)
		}
	})

	t.Run("拒否済みのコードは、許可も拒否も受け付けない", func(t *testing.T) {
		t.Parallel()

		f := setupOAuthDeviceVerificationFixture(t, "devupdate-used", model.SpaceRoleEditor, true, func(b *testutil.OAuthDeviceAuthorizationBuilder) {
			b.WithStatus(model.OAuthDeviceAuthorizationStatusDenied)
		})
		for _, approved := range []bool{true, false} {
			_, err := newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(context.Background(), UpdateOAuthDeviceAuthorizationInput{
				UserID:        f.userID,
				UserCode:      f.userCode,
				Approved:      approved,
				GrantedScopes: []string{"page:read"},
			})
			assertInvalidOAuthDeviceUserCode(t, err)
		}
	})
}

// 要求のロックと許可の作成の待機を観測してから期限を過ぎさせ、変更が残らないことを確かめる。
// 同じアプリ・メンバーの既存の許可とトークンも変えない。
func TestUpdateOAuthDeviceAuthorizationUsecase_ExpiryDuringLock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		key       string
		table     string
		queryName string
	}{
		{name: "要求のロック待機", key: "devupdate-expiry-request", table: "oauth_device_authorizations", queryName: "LockPendingOAuthDeviceAuthorization"},
		{name: "許可作成の外部キー検査待機", key: "devupdate-expiry-grant-fk", table: "space_members", queryName: "ApproveOAuthDeviceAuthorizationWithGrant"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := testutil.GetTestDB()
			q := query.New(db)
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			key := tt.key
			f := setupOAuthDeviceVerificationFixture(t, key, model.SpaceRoleEditor, true, nil)
			grantRepo := repository.NewOAuthGrantRepository(q)
			originalScopes := []model.Scope{model.ScopePageRead, model.ScopePageWrite}
			existing, err := grantRepo.Create(ctx, repository.CreateOAuthGrantInput{OAuthApplicationID: f.applicationID, SpaceID: f.spaceID, SpaceMemberID: f.spaceMemberID, Scopes: originalScopes})
			if err != nil {
				t.Fatal(err)
			}
			testutil.CommitFixture(t, func(tx *sql.Tx) {
				testutil.NewOAuthAccessTokenBuilder(t, tx).WithOAuthGrantID(existing.ID).WithSpaceID(f.spaceID).WithScopes(originalScopes).WithTokenDigest(key + "-access").Build()
				testutil.NewOAuthRefreshTokenBuilder(t, tx).WithOAuthGrantID(existing.ID).WithSpaceID(f.spaceID).WithScopes(originalScopes).WithTokenDigest(key + "-refresh").Build()
			})
			expiresAt := time.Now().Add(5 * time.Second)
			if _, err := db.ExecContext(ctx, "UPDATE oauth_device_authorizations SET expires_at = $1 WHERE id = $2 AND space_id = $3", expiresAt, string(f.deviceAuthorizationID), string(f.spaceID)); err != nil {
				t.Fatal(err)
			}
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback() }()
			var blockingPID int
			if err := tx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockingPID); err != nil {
				t.Fatal(err)
			}
			lockID := string(f.deviceAuthorizationID)
			if tt.table == "space_members" {
				lockID = string(f.spaceMemberID)
			}
			// 表名は固定したテストケースだけから取り、ユーザー入力は使わない。
			if _, err := tx.ExecContext(ctx, "SELECT id FROM "+tt.table+" WHERE id = $1 AND space_id = $2 FOR UPDATE", lockID, string(f.spaceID)); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				_, err := newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(ctx, UpdateOAuthDeviceAuthorizationInput{UserID: f.userID, UserCode: f.userCode, Approved: true, GrantedScopes: []string{"page:read"}})
				done <- err
			}()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				var blocked bool
				if err := db.QueryRowContext(ctx, `SELECT EXISTS (
		     SELECT 1 FROM pg_stat_activity WHERE datname = current_database()
		      AND $1 = ANY(pg_blocking_pids(pid)) AND query LIKE '%' || $2 || '%'
		    )`, blockingPID, tt.queryName).Scan(&blocked); err != nil {
					t.Fatal(err)
				}
				if blocked {
					break
				}
				select {
				case err := <-done:
					t.Fatalf("ロック待機前に処理が終了: %v", err)
				case <-ctx.Done():
					t.Fatalf("ロック待機を確認できない: %v", ctx.Err())
				case <-ticker.C:
				}
			}
			timer := time.NewTimer(time.Until(expiresAt) + 10*time.Millisecond)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-ctx.Done():
				t.Fatalf("期限切れまで待機できない: %v", ctx.Err())
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				assertInvalidOAuthDeviceUserCode(t, err)
			case <-ctx.Done():
				t.Fatalf("承認が完了しない: %v", ctx.Err())
			}
			device := findOAuthDeviceAuthorizationForTest(t, q, f)
			if device.Status != model.OAuthDeviceAuthorizationStatusPending || device.OAuthGrantID != nil || !slices.Equal(device.Scopes, originalScopes) {
				t.Errorf("期限切れの要求が変更された: %+v", device)
			}
			grants, err := grantRepo.ListUnrevokedBySpaceMember(ctx, f.spaceID, f.spaceMemberID)
			if err != nil {
				t.Fatal(err)
			}
			if len(grants) != 1 || grants[0].ID != existing.ID || !slices.Equal(grants[0].Scopes, originalScopes) {
				t.Fatalf("失効していない許可 = %+v、期待値 = 変更されていない既存の許可 %vだけ", grants, existing.ID)
			}
			access, err := repository.NewOAuthAccessTokenRepository(q).FindByTokenDigest(ctx, key+"-access")
			if err != nil {
				t.Fatal(err)
			}
			refresh, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(ctx, key+"-refresh")
			if err != nil {
				t.Fatal(err)
			}
			if access == nil || refresh == nil || access.IsRevoked() || refresh.IsRevoked() {
				t.Error("期限切れの承認で既存のトークンが失効した")
			}
		})
	}
}
