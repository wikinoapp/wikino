package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// 同じアプリ・メンバーへの再認可 (同意の送信・デバイス認可の許可) は別の許可を作り、既存の許可には触れない。
// 既存の許可の行をトークンの発行がロックしている間でも、再認可は待たずに終わり、
// 既存の許可のトークンは新しい許可の範囲に収まらなくても使い続けられることを確かめる。
func TestOAuthGrantIndependence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
		// prepareは、既存の許可と同じアプリ・メンバーでtopic:readだけを認める再認可を用意し、それを実行する関数を返す
		prepare func(t *testing.T, q *query.Queries, f oauthTokenFixture) func(ctx context.Context) error
	}{
		{
			name: "同意の送信",
			key:  "oauth-grant-independence",
			prepare: func(t *testing.T, q *query.Queries, f oauthTokenFixture) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					_, err := newCreateOAuthAuthorizationUsecaseForTest(q).Execute(ctx, CreateOAuthAuthorizationInput{
						UserID: f.userID, Params: OAuthAuthorizationParams{ClientID: f.clientID, RedirectURI: oauthTokenTestRedirectURI,
							ResponseType: "code", Scope: "topic:read page:write", CodeChallenge: oauthAuthorizationTestCodeChallenge, CodeChallengeMethod: "S256"},
						Approved: true, GrantedScopes: []string{"topic:read"},
					})
					return err
				}
			},
		},
		{
			name: "デバイス認可の許可",
			key:  "oauth-grant-independence-device",
			prepare: func(t *testing.T, q *query.Queries, f oauthTokenFixture) func(ctx context.Context) error {
				userCode, err := auth.GenerateOAuthUserCode()
				if err != nil {
					t.Fatal(err)
				}
				testutil.CommitFixture(t, func(tx *sql.Tx) {
					testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).WithOAuthApplicationID(f.appID).WithSpaceID(f.spaceID).
						WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageWrite}).
						WithDeviceCodeDigest(string(f.identifier) + "-device-code").WithUserCodeDigest(auth.DigestOpaqueToken(userCode)).Build()
				})
				return func(ctx context.Context) error {
					output, err := newUpdateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(ctx, UpdateOAuthDeviceAuthorizationInput{
						UserID: f.userID, UserCode: userCode, Approved: true, GrantedScopes: []string{"topic:read"},
					})
					if err != nil {
						return err
					}
					if !output.Approved {
						return fmt.Errorf("デバイス認可の許可の結果 = %+v、期待値 = 許可済み", output)
					}
					return nil
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := testutil.GetTestDB()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			f := setupOAuthTokenFixtureDB(t, tt.key, oauthTokenFixtureOptions{})
			q := query.New(db)
			refreshValue := f.buildRefreshTokenDB(t, tt.key, nil)
			accessValue := string(auth.OAuthAccessTokenPrefix) + tt.key + "-access"
			testutil.CommitFixture(t, func(tx *sql.Tx) {
				testutil.NewOAuthAccessTokenBuilder(t, tx).WithOAuthGrantID(f.grantID).WithSpaceID(f.spaceID).
					WithTokenDigest(auth.DigestOpaqueToken(accessValue)).WithScopes([]model.Scope{model.ScopePageWrite}).Build()
			})
			authorize := tt.prepare(t, q, f)

			// トークンの発行 (交換・リフレッシュ) と同じく、既存の許可の行をロックしておく
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback() }()
			if _, err := repository.NewOAuthGrantRepository(q).WithTx(tx).LockByID(ctx, f.grantID, f.spaceID); err != nil {
				t.Fatal(err)
			}

			done := make(chan error, 1)
			go func() { done <- authorize(ctx) }()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal("既存の許可のロック中に再認可が完了しませんでした")
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}

			grants, err := repository.NewOAuthGrantRepository(q).ListUnrevokedBySpaceMember(ctx, f.spaceID, f.spaceMemberID)
			if err != nil {
				t.Fatal(err)
			}
			if len(grants) != 2 {
				t.Fatalf("失効していない許可の件数 = %d、期待値 = 2 (既存の許可と再認可の許可)", len(grants))
			}

			principal, err := newAuthenticateAPITokenUCForOAuthTest(q).Execute(ctx, accessValue)
			if err != nil {
				t.Fatal(err)
			}
			if principal == nil {
				t.Error("再認可で範囲を狭めた後、既存の許可のアクセストークンがAPI認証を通りませんでした")
			}
			output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(ctx, f.refreshParams(refreshValue))
			if err != nil {
				t.Fatal(err)
			}
			if output == nil || !model.HasAllScopes(output.Scopes, []model.Scope{model.ScopePageWrite}) {
				t.Errorf("既存の許可のリフレッシュの結果 = %+v、期待値 = page:writeを含むトークン", output)
			}
		})
	}
}
