// Package testutil はテスト用のユーティリティを提供します
package testutil

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// SetupTestDB はテスト用のデータベース接続とトランザクションをセットアップします
// テスト終了時にトランザクションは自動的にロールバックされます
func SetupTestDB(t *testing.T) (*sql.DB, *sql.Tx) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@postgresql:5432/wikino_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("データベース接続に失敗: %v", err)
	}

	// 接続確認
	if err := db.Ping(); err != nil {
		t.Fatalf("データベースへのpingに失敗: %v", err)
	}

	// トランザクション開始
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("トランザクション開始に失敗: %v", err)
	}

	// テスト終了時にロールバックとDB接続クローズ
	t.Cleanup(func() {
		_ = tx.Rollback()
		_ = db.Close()
	})

	return db, tx
}
