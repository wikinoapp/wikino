package db_test

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.SetupTestMain(m))
}

// execMigrationUpは、マイグレーションファイルのmigrate:upの部分をトランザクションの中で実行する。
// テスト用DBは適用済みのスキーマから作るため、データを書き換えるマイグレーションは
// テストで用意した行に対してもう一度実行して確かめる
func execMigrationUp(t *testing.T, tx *sql.Tx, name string) {
	t.Helper()

	content, err := os.ReadFile("migrations/" + name)
	if err != nil {
		t.Fatalf("マイグレーションファイルの読み込みに失敗: %v", err)
	}
	up, _, found := strings.Cut(string(content), "-- migrate:down")
	if !found {
		t.Fatalf("マイグレーションファイルにmigrate:downがありません: %s", name)
	}
	if _, err := tx.Exec(up); err != nil {
		t.Fatalf("マイグレーションの実行に失敗: %v", err)
	}
}

func TestBackfillRoleOfSpaceMembers(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("backfill-role-space").Build()

	// ロールを保存しない旧版が作ったメンバー
	var nullRoleMemberID string
	err := tx.QueryRow(
		`INSERT INTO space_members (space_id, user_id, role, scopes, joined_at, active, created_at, updated_at)
		 VALUES ($1, $2, NULL, ARRAY['space:admin'], now(), true, now(), now())
		 RETURNING id`,
		string(spaceID), string(testutil.NewUserBuilder(t, tx).WithEmail("backfill-null@example.com").WithAtname("backfill_null").Build()),
	).Scan(&nullRoleMemberID)
	if err != nil {
		t.Fatalf("ロールの無いスペースメンバーの作成に失敗: %v", err)
	}

	editorMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(testutil.NewUserBuilder(t, tx).WithEmail("backfill-editor@example.com").WithAtname("backfill_editor").Build()).
		WithRole(model.SpaceRoleEditor).
		Build()

	execMigrationUp(t, tx, "20260930104436_backfill_role_of_space_members.sql")

	tests := []struct {
		name     string
		memberID string
		want     model.SpaceRole
	}{
		{name: "ロールの無いメンバーは管理者になる", memberID: nullRoleMemberID, want: model.SpaceRoleAdmin},
		{name: "保存済みのロールは上書きしない", memberID: string(editorMemberID), want: model.SpaceRoleEditor},
	}
	for _, tt := range tests {
		var got string
		if err := tx.QueryRow(`SELECT role FROM space_members WHERE id = $1 AND space_id = $2`, tt.memberID, string(spaceID)).Scan(&got); err != nil {
			t.Fatalf("%s: ロールの取得に失敗: %v", tt.name, err)
		}
		if model.SpaceRole(got) != tt.want {
			t.Errorf("%s: ロール = %q、期待値 = %q", tt.name, got, tt.want)
		}
	}

	var nullCount int
	if err := tx.QueryRow(`SELECT count(*) FROM space_members WHERE space_id = $1 AND role IS NULL`, string(spaceID)).Scan(&nullCount); err != nil {
		t.Fatalf("ロールの無い行の数の取得に失敗: %v", err)
	}
	if nullCount != 0 {
		t.Errorf("ロールの無い行の数 = %d、期待値 = 0", nullCount)
	}
}
