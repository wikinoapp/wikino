package testutil_test

import (
	"os"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.SetupTestMain(m))
}

func TestGetTestDB(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	if db == nil {
		t.Fatal("GetTestDBがnilを返しました")
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("DB接続のpingに失敗: %v", err)
	}
}

func TestSetupTx(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	if db == nil {
		t.Fatal("SetupTxが返したdbがnilです")
	}
	if tx == nil {
		t.Fatal("SetupTxが返したtxがnilです")
	}

	// トランザクション内でクエリが実行できることを確認
	var result int
	err := tx.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("トランザクション内でのクエリに失敗: %v", err)
	}
	if result != 1 {
		t.Errorf("SELECT 1の結果が%d、期待値は1", result)
	}
}

func TestSetupTx_TransactionIsolation(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	// トランザクション内でデータをINSERTしても、他のトランザクションには見えない
	_, err := tx.Exec(
		`INSERT INTO users (email, atname, name, description, locale, time_zone, joined_at, created_at, updated_at)
		 VALUES ('isolation-test@example.com', 'isolation_user', 'Isolation User', '', 0, 'Asia/Tokyo', NOW(), NOW(), NOW())`,
	)
	if err != nil {
		t.Fatalf("INSERT に失敗: %v", err)
	}

	// トランザクション内ではデータが見える
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE email = 'isolation-test@example.com'").Scan(&count)
	if err != nil {
		t.Fatalf("SELECT COUNT に失敗: %v", err)
	}
	if count != 1 {
		t.Errorf("トランザクション内でINSERTしたデータが見えない: count=%d", count)
	}

	// テスト終了時にCleanupでロールバックされるため、
	// データはDBに残らない（他のテストに影響しない）
}

func TestSetupTx_MultipleTransactions(t *testing.T) {
	t.Parallel()

	// 同じテスト内で複数のトランザクションを作成できることを確認
	db1, tx1 := testutil.SetupTx(t)
	db2, tx2 := testutil.SetupTx(t)

	// 同じDB接続プールを共有していることを確認
	if db1 != db2 {
		t.Error("SetupTxが返すDBは同じ接続プールであるべき")
	}

	// それぞれのトランザクションは独立して動作する
	var r1, r2 int
	if err := tx1.QueryRow("SELECT 1").Scan(&r1); err != nil {
		t.Fatalf("tx1のクエリに失敗: %v", err)
	}
	if err := tx2.QueryRow("SELECT 2").Scan(&r2); err != nil {
		t.Fatalf("tx2のクエリに失敗: %v", err)
	}
	if r1 != 1 || r2 != 2 {
		t.Errorf("結果が不正: r1=%d, r2=%d", r1, r2)
	}
}
