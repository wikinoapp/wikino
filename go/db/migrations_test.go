package db_test

import (
	"errors"
	"os"
	"testing"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.SetupTestMain(m))
}

func TestSpaceMembersRoleNotNull(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("role-not-null-space").Build()
	userID := testutil.NewUserBuilder(t, tx).WithEmail("role-not-null@example.com").WithAtname("role_not_null").Build()

	// ロールを保存しない旧版と同じ形のINSERT
	_, err := tx.Exec(
		`INSERT INTO space_members (space_id, user_id, role, scopes, joined_at, active, created_at, updated_at)
		 VALUES ($1, $2, NULL, ARRAY['space:admin'], now(), true, now(), now())`,
		string(spaceID), string(userID),
	)

	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code.Name() != "not_null_violation" || pqErr.Column != "role" {
		t.Errorf("エラー = %v、期待値 = roleのnot_null_violation", err)
	}
}
