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

// waitOAuthGrantBlockedは、別接続が許可行のロック待ちに入ったことをDB側で確かめる。
// goroutineの開始だけではSQLのスナップショットが取得されたことを保証できないため、待機を観測する
func waitOAuthGrantBlocked(t *testing.T, ctx context.Context, db *sql.DB, blockingPID int) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
   SELECT 1 FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid))
  )`, blockingPID).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("許可行のロック待ちを確認できませんでした")
		case <-ticker.C:
		}
	}
}

func TestOAuthScopeConcurrency(t *testing.T) {
	t.Parallel()
	for _, refresh := range []bool{false, true} {
		for _, shrinkFirst := range []bool{false, true} {
			operation, first := "認可コード交換", "発行が先"
			if refresh {
				operation = "リフレッシュ"
			}
			if shrinkFirst {
				first = "縮小が先"
			}
			name := operation + "/" + first
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				db := testutil.GetTestDB()
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				key := fmt.Sprintf("oauth-race-%t-%t", refresh, shrinkFirst)
				f := setupOAuthTokenFixtureDB(t, key, oauthTokenFixtureOptions{})
				q := query.New(db)
				codeValue := f.buildCodeDB(t, nil)
				refreshValue := f.buildRefreshTokenDB(t, key, nil)
				// 範囲内の別端末のトークンまで、縮小や交換の拒否で失効させない。
				inRangeValue := string(auth.OAuthAccessTokenPrefix) + key + "-in-range"
				testutil.CommitFixture(t, func(tx *sql.Tx) {
					testutil.NewOAuthAccessTokenBuilder(t, tx).WithOAuthGrantID(f.grantID).WithSpaceID(f.spaceID).
						WithTokenDigest(auth.DigestOpaqueToken(inRangeValue)).WithScopes([]model.Scope{model.ScopeTopicRead}).Build()
				})
				code, err := repository.NewOAuthAuthorizationCodeRepository(q).FindByCodeDigest(ctx, auth.DigestOpaqueToken(codeValue))
				if err != nil {
					t.Fatal(err)
				}
				token, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(ctx, auth.DigestOpaqueToken(refreshValue))
				if err != nil {
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
				if _, err := repository.NewOAuthGrantRepository(q).WithTx(tx).LockByID(ctx, f.grantID, f.spaceID); err != nil {
					t.Fatal(err)
				}
				narrow := repository.CreateOAuthAuthorizationCodeWithGrantInput{
					OAuthApplicationID: f.appID, SpaceID: f.spaceID, SpaceMemberID: f.spaceMemberID,
					CodeDigest: key + "-narrow", Scopes: []model.Scope{model.ScopeTopicRead},
					RedirectURI: oauthTokenTestRedirectURI, CodeChallenge: oauthAuthorizationTestCodeChallenge,
					ExpiresAt: time.Now().Add(time.Minute),
				}
				accessValue := string(auth.OAuthAccessTokenPrefix) + key + "-issued"
				newRefreshValue := string(auth.OAuthRefreshTokenPrefix) + key + "-issued"
				now := time.Now()
				if shrinkFirst {
					// 縮小の未コミット状態では別接続の事前検証は旧スコープを読む。
					// 発行時のロック待機後に最新の許可を使って拒否できることを確かめる。
					if _, err := repository.NewOAuthAuthorizationCodeRepository(q).WithTx(tx).CreateWithGrant(ctx, narrow); err != nil {
						t.Fatal(err)
					}
				} else if refresh {
					if _, err := repository.NewOAuthRefreshTokenRepository(q).WithTx(tx).Rotate(ctx, repository.RotateOAuthRefreshTokenInput{
						ID: token.ID, SpaceID: f.spaceID, TokenDigest: auth.DigestOpaqueToken(newRefreshValue), ExpiresAt: now.Add(time.Hour),
						AccessTokenDigest: auth.DigestOpaqueToken(accessValue), AccessTokenScopes: token.Scopes, AccessTokenExpiresAt: now.Add(time.Hour), Now: now,
					}); err != nil {
						t.Fatal(err)
					}
				} else {
					if _, err := repository.NewOAuthAuthorizationCodeRepository(q).WithTx(tx).Exchange(ctx, repository.ExchangeOAuthAuthorizationCodeInput{
						ID: code.ID, SpaceID: f.spaceID, AccessTokenDigest: auth.DigestOpaqueToken(accessValue), AccessTokenExpiresAt: now.Add(time.Hour),
						RefreshTokenDigest: auth.DigestOpaqueToken(newRefreshValue), RefreshTokenExpiresAt: now.Add(time.Hour), Now: now,
					}); err != nil {
						t.Fatal(err)
					}
				}
				type result struct {
					output *CreateOAuthTokenOutput
					err    error
				}
				done := make(chan result, 1)
				go func() {
					if shrinkFirst {
						params := f.codeParams(codeValue)
						if refresh {
							params = f.refreshParams(refreshValue)
						}
						output, err := newCreateOAuthTokenUsecaseForTest(q).Execute(ctx, params)
						done <- result{output: output, err: err}
					} else {
						_, err := newCreateOAuthAuthorizationUsecaseForTest(q).Execute(ctx, CreateOAuthAuthorizationInput{
							UserID: f.userID, Params: OAuthAuthorizationParams{ClientID: f.clientID, RedirectURI: oauthTokenTestRedirectURI,
								ResponseType: "code", Scope: "topic:read page:write", CodeChallenge: oauthAuthorizationTestCodeChallenge, CodeChallengeMethod: "S256"},
							Approved: true, GrantedScopes: []string{"topic:read"},
						})
						done <- result{err: err}
					}
				}()
				waitOAuthGrantBlocked(t, ctx, db, blockingPID)
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				var got result
				select {
				case got = <-done:
				case <-ctx.Done():
					t.Fatal("競合する処理が完了しませんでした")
				}
				if shrinkFirst {
					assertOAuthTokenError(t, got.output, got.err, model.OAuthTokenErrorInvalidGrant)
					if !refresh {
						stored, err := repository.NewOAuthAuthorizationCodeRepository(q).FindByCodeDigest(ctx, auth.DigestOpaqueToken(codeValue))
						if err != nil {
							t.Fatal(err)
						}
						if stored.IsUsed() {
							t.Error("縮小に負けたコードが使用済みになりました")
						}
					}
				} else {
					if got.err != nil {
						t.Fatal(got.err)
					}
					access, err := repository.NewOAuthAccessTokenRepository(q).FindByTokenDigest(ctx, auth.DigestOpaqueToken(accessValue))
					if err != nil {
						t.Fatal(err)
					}
					renewed, err := repository.NewOAuthRefreshTokenRepository(q).FindByTokenDigest(ctx, auth.DigestOpaqueToken(newRefreshValue))
					if err != nil {
						t.Fatal(err)
					}
					if access == nil || !access.IsRevoked() || renewed == nil || !renewed.IsRevoked() {
						t.Error("ロック待機中に発行された範囲外のトークンが失効していません")
					}
					principal, err := newAuthenticateAPITokenUCForOAuthTest(q).Execute(ctx, accessValue)
					if err != nil {
						t.Fatal(err)
					}
					if principal != nil {
						t.Error("縮小後も範囲外のトークンがAPI認証を通りました")
					}
				}
				principal, err := newAuthenticateAPITokenUCForOAuthTest(q).Execute(ctx, inRangeValue)
				if err != nil {
					t.Fatal(err)
				}
				if principal == nil {
					t.Error("範囲内の別端末のアクセストークンまで失効しました")
				}
			})
		}
	}
}
