package usecase

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newDeleteAccountUsecaseはテスト用のデータベースを使うUseCaseを組み立てる。
func newDeleteAccountUsecase(db *sql.DB) *DeleteAccountUsecase {
	queries := query.New(db)
	return NewDeleteAccountUsecase(
		db,
		repository.NewUserRepository(queries),
		repository.NewUserSessionRepository(queries),
		repository.NewUserPasswordRepository(queries),
		repository.NewUserTwoFactorAuthRepository(queries),
		repository.NewSpaceMemberRepository(queries),
	)
}

// accountRowTablesは、物理削除でユーザーとともに消えるべき行を持つ表。
var accountRowTables = []string{
	"user_sessions",
	"user_passwords",
	"user_two_factor_auths",
	"feature_flags",
	"password_reset_tokens",
	"email_confirmations",
}

// seedAccountWithRelatedRecordsは、物理削除が消す関連レコードを1つずつ (セッションは2つ) 持つ
// ユーザーを作成する。discardedがtrueなら論理削除しておく。
func seedAccountWithRelatedRecords(t *testing.T, db *sql.DB, suffix string, discarded bool) model.UserID {
	t.Helper()

	ctx := context.Background()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-account-"+suffix+"@example.com").
		WithAtname(strings.ReplaceAll("delete_account_"+suffix, "-", "_")).
		BuildWithTwoFactorAuthAndRecoveryCodes("JBSWY3DPEHPK3PXP", true, []string{"abcd1234"})
	testutil.NewUserPasswordBuilderDB(t, db).WithUserID(userID).Build()
	for _, token := range []string{"delete-account-" + suffix + "-1", "delete-account-" + suffix + "-2"} {
		if _, err := repository.NewUserSessionRepository(queries).Create(ctx, repository.CreateInput{
			UserID:     userID,
			Token:      token,
			IPAddress:  "127.0.0.1",
			UserAgent:  "test",
			SignedInAt: time.Now(),
		}); err != nil {
			t.Fatalf("セッションの作成に失敗: %v", err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO feature_flags (user_id, name) VALUES ($1, 'go_example')`, string(userID)); err != nil {
		t.Fatalf("フィーチャーフラグの作成に失敗: %v", err)
	}
	testutil.NewPasswordResetTokenBuilderDB(t, db).
		WithUserID(userID).
		WithTokenDigest("delete-account-" + suffix).
		Build()
	testutil.NewEmailConfirmationBuilderDB(t, db).
		WithUserID(userID).
		WithEmail("delete-account-" + suffix + "-new@example.com").
		WithEvent(model.EmailConfirmationEventEmailUpdate).
		WithCode("delete-account-" + suffix).
		Build()

	if discarded {
		if _, err := db.ExecContext(ctx, `UPDATE users SET discarded_at = NOW() WHERE id = $1`, string(userID)); err != nil {
			t.Fatalf("ユーザーの論理削除に失敗: %v", err)
		}
	}

	return userID
}

// countUserRowsは、ユーザーの行と、ユーザーを参照する表ごとの行の数を返す。
func countUserRows(t *testing.T, db *sql.DB, userID model.UserID) map[string]int {
	t.Helper()

	counts := map[string]int{}
	var n int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM users WHERE id = $1`, string(userID)).Scan(&n); err != nil {
		t.Fatalf("usersの行の数の取得に失敗: %v", err)
	}
	counts["users"] = n
	for _, table := range accountRowTables {
		// 表の名前は上の固定の一覧から渡すため、文字列の連結で組み立ててよい。
		if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM `+table+` WHERE user_id = $1`, string(userID)).Scan(&n); err != nil { //nolint:gosec
			t.Fatalf("%sの行の数の取得に失敗: %v", table, err)
		}
		counts[table] = n
	}
	return counts
}

// assertUserRowsは、ユーザーとそれを参照する行がすべて残っている (remaining) か、すべて消えているかを確かめる。
func assertUserRows(t *testing.T, db *sql.DB, userID model.UserID, remaining bool) {
	t.Helper()

	for table, n := range countUserRows(t, db, userID) {
		if remaining && n == 0 {
			t.Errorf("%sの行が消えた、残ることを期待", table)
		}
		if !remaining && n != 0 {
			t.Errorf("%sの行 = %d件、期待値 = 0件", table, n)
		}
	}
}

func TestDeleteAccountUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID := seedAccountWithRelatedRecords(t, db, "success", true)
	otherID := seedAccountWithRelatedRecords(t, db, "success-other", true)

	if err := newDeleteAccountUsecase(db).Execute(t.Context(), DeleteAccountInput{UserID: userID}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	assertUserRows(t, db, userID, false)
	// 論理削除していても、ジョブの対象でないユーザーのレコードは消さない
	assertUserRows(t, db, otherID, true)
}

func TestDeleteAccountUsecase_Execute_論理削除していないユーザーは消さない(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID := seedAccountWithRelatedRecords(t, db, "kept", false)

	if err := newDeleteAccountUsecase(db).Execute(t.Context(), DeleteAccountInput{UserID: userID}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	assertUserRows(t, db, userID, true)
}

func TestDeleteAccountUsecase_Execute_スペースメンバーの行が残る間は消さない(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID := seedAccountWithRelatedRecords(t, db, "member", true)
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("delete-account-member").Build()
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	uc := newDeleteAccountUsecase(db)

	err := uc.Execute(t.Context(), DeleteAccountInput{UserID: userID})
	if !errors.Is(err, ErrAccountHasSpaceMembers) {
		t.Fatalf("Execute()のエラー = %v、ErrAccountHasSpaceMembersを期待", err)
	}
	assertUserRows(t, db, userID, true)

	// スペースの物理削除がスペースメンバーの行を消したあとの再試行で、ユーザーを消す
	if _, err := db.ExecContext(t.Context(), `DELETE FROM space_members WHERE user_id = $1 AND space_id = $2`, string(userID), string(spaceID)); err != nil {
		t.Fatalf("スペースメンバーの削除に失敗: %v", err)
	}
	if err := uc.Execute(t.Context(), DeleteAccountInput{UserID: userID}); err != nil {
		t.Fatalf("再試行のExecute()のエラー = %v", err)
	}
	assertUserRows(t, db, userID, false)
}

func TestDeleteAccountUsecase_Execute_物理削除したユーザーでは何もしない(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID := seedAccountWithRelatedRecords(t, db, "twice", true)
	uc := newDeleteAccountUsecase(db)

	if err := uc.Execute(t.Context(), DeleteAccountInput{UserID: userID}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if err := uc.Execute(t.Context(), DeleteAccountInput{UserID: userID}); err != nil {
		t.Fatalf("2回目のExecute()のエラー = %v", err)
	}
}

// usersを参照する外部キーと削除時の動作が、物理削除の範囲と一致することを確かめる。
// 表が増えたら、物理削除で消すか、ON DELETE CASCADEにするかを決めてからここに足す。
func TestDeleteAccountUsecase_外部キーの一覧が物理削除の範囲と一致する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()

	// confdeltypeは削除時の動作で、"a" はNO ACTION、"c" はCASCADEを表す。
	want := map[string]string{
		// 物理削除のUseCaseが明示的に消す
		"user_sessions.user_id -> users":         "a",
		"user_passwords.user_id -> users":        "a",
		"user_two_factor_auths.user_id -> users": "a",
		// 残っている間は物理削除を先送りし、スペースの物理削除が消すのを待つ
		"space_members.user_id -> users": "a",
		// ユーザーとともに外部キーで消える
		"feature_flags.user_id -> users":         "c",
		"password_reset_tokens.user_id -> users": "c",
		"email_confirmations.user_id -> users":   "c",
	}

	rows, err := db.QueryContext(t.Context(), `
		SELECT child.relname, a.attname, parent.relname, c.confdeltype
		FROM pg_constraint c
		JOIN pg_class child ON child.oid = c.conrelid
		JOIN pg_class parent ON parent.oid = c.confrelid
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		WHERE c.contype = 'f'
		  AND parent.relname = 'users'`,
	)
	if err != nil {
		t.Fatalf("外部キーの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	got := map[string]string{}
	for rows.Next() {
		var child, column, parent, action string
		if err := rows.Scan(&child, &column, &parent, &action); err != nil {
			t.Fatalf("外部キーの読み取りに失敗: %v", err)
		}
		got[child+"."+column+" -> "+parent] = action
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("外部キーの読み取りに失敗: %v", err)
	}

	keys := make([]string, 0, len(got)+len(want))
	for k := range got {
		keys = append(keys, k)
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		if got[k] != want[k] {
			t.Errorf("外部キー %s の削除時の動作 = %q、期待値 = %q", k, got[k], want[k])
		}
	}
}
