// Package testutil はテスト用のユーティリティを提供します
package testutil

import (
	"database/sql"
	"log/slog"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/auth"
)

// testDB はパッケージレベルで共有するDB接続プール
// SetupTestMainで初期化される
var testDB *sql.DB

// getTestDBURL はテスト用DBのURLを取得します
func getTestDBURL() string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@postgresql:5432/wikino_test?sslmode=disable"
	}
	return dbURL
}

// SetupTestMain はTestMain内で呼び出し、パッケージ共有のDB接続を初期化します。
// 戻り値をos.Exitに渡してください。
//
// 使用例:
//
//	func TestMain(m *testing.M) {
//	    os.Exit(testutil.SetupTestMain(m))
//	}
func SetupTestMain(m *testing.M) int {
	// テスト用にbcryptコストを下げる（DefaultCost 10 → MinCost 4 で約64倍高速化）
	auth.BcryptCost = auth.TestBcryptCost

	var err error
	testDB, err = sql.Open("postgres", getTestDBURL())
	if err != nil {
		slog.Error("テスト用DB接続に失敗", "error", err)
		return 1
	}
	defer func() { _ = testDB.Close() }()

	if err := testDB.Ping(); err != nil {
		slog.Error("テスト用DBへのpingに失敗", "error", err)
		return 1
	}

	return m.Run()
}

// GetTestDB はSetupTestMainで初期化されたDB接続プールへの参照を返します。
// SetupTestMainが呼ばれていない場合はpanicします。
func GetTestDB() *sql.DB {
	if testDB == nil {
		panic("SetupTestMainが呼ばれていません。TestMain内でtestutil.SetupTestMain(m)を呼んでください")
	}
	return testDB
}

// SetupTx はテスト用のトランザクションをセットアップします。
// SetupTestMainで初期化されたDB接続プールを使用します。
// テスト終了時にトランザクションは自動的にロールバックされます。
func SetupTx(t *testing.T) (*sql.DB, *sql.Tx) {
	t.Helper()

	if testDB == nil {
		t.Fatal("SetupTestMainが呼ばれていません。TestMain内でtestutil.SetupTestMain(m)を呼んでください")
	}

	tx, err := testDB.Begin()
	if err != nil {
		t.Fatalf("トランザクション開始に失敗: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return testDB, tx
}
