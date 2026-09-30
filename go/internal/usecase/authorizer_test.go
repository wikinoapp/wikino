package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// TestAuthorizerDuringRoleMigrationは列追加後に旧版と新版が書くメンバーの権限を確かめる。
func TestAuthorizerDuringRoleMigration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		create   func(t *testing.T, ctx context.Context, tx *sql.Tx, repo *repository.SpaceMemberRepository, spaceID model.SpaceID, userID model.UserID)
		wantRole model.SpaceRole
	}{
		{
			name: "旧版がロール無しで保存しても権限を維持する",
			create: func(t *testing.T, ctx context.Context, tx *sql.Tx, _ *repository.SpaceMemberRepository, spaceID model.SpaceID, userID model.UserID) {
				t.Helper()
				// release処理の列追加後にも旧版のプロセスはロールを指定せず書き込む。
				if _, err := tx.ExecContext(ctx,
					`INSERT INTO space_members (space_id, user_id, scopes, joined_at, created_at, updated_at)
					 VALUES ($1, $2, '{space:admin}', NOW(), NOW(), NOW())`,
					string(spaceID), string(userID),
				); err != nil {
					t.Fatalf("旧版のINSERTに失敗: %v", err)
				}
			},
			wantRole: "",
		},
		{
			name: "新版はロールとスコープを保存する",
			create: func(t *testing.T, ctx context.Context, _ *sql.Tx, repo *repository.SpaceMemberRepository, spaceID model.SpaceID, userID model.UserID) {
				t.Helper()
				if _, err := repo.Create(ctx, repository.CreateSpaceMemberInput{
					SpaceID: spaceID,
					UserID:  userID,
					Role:    model.SpaceRoleAdmin,
				}); err != nil {
					t.Fatalf("Create()のエラー = %v", err)
				}
			},
			wantRole: model.SpaceRoleAdmin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			ctx := context.Background()
			userID := testutil.NewUserBuilder(t, tx).Build()
			spaceID := testutil.NewSpaceBuilder(t, tx).Build()
			repo := repository.NewSpaceMemberRepository(testutil.QueriesWithTx(tx))

			tt.create(t, ctx, tx, repo, spaceID, userID)

			member, err := repo.FindActiveBySpaceAndUser(ctx, spaceID, userID)
			if err != nil {
				t.Fatalf("FindActiveBySpaceAndUser()のエラー = %v", err)
			}
			if member == nil {
				t.Fatal("スペースメンバーが見つからない")
			}
			if member.Role != tt.wantRole {
				t.Fatalf("member.Role = %q、期待値 = %q", member.Role, tt.wantRole)
			}
			if !model.HasScope(member.Scopes, model.ScopeSpaceAdmin) {
				t.Fatal("保存済みのspace:adminが読み込まれていない")
			}

			privateTopic := &model.Topic{Visibility: model.TopicVisibilityPrivate}
			authorizer := newAuthorizer(member, nil)
			if !authorizer.CanCreatePage() || !authorizer.CanShowTopic(privateTopic) || !authorizer.CanUpdateSpace() {
				t.Fatal("認可の切り替え前に既存の管理者権限が失われた")
			}

			principal := &model.APIPrincipal{SpaceMember: member, Scopes: []model.Scope{model.ScopePageWrite, model.ScopeTopicRead}}
			apiAuthorizer := newAPIAuthorizer(principal, nil)
			if !apiAuthorizer.CanCreatePage() || !apiAuthorizer.CanShowTopic(privateTopic) {
				t.Fatal("APIでも既存のメンバー権限を利用できない")
			}
			principal.Scopes = []model.Scope{model.ScopePageRead}
			if newAPIAuthorizer(principal, nil).CanCreatePage() {
				t.Fatal("トークンにない書き込み権限が付与された")
			}
		})
	}
}
