package testutil

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
)

// SetupRiverClientは、ほかのテストのジョブに干渉しない専用スキーマを用意する。
// アプリケーションと同じDBを使い、行の削除とジョブの投入の原子性を検証できる。
func SetupRiverClient(t *testing.T, db *sql.DB) (*river.Client[*sql.Tx], string) {
	t.Helper()

	schema := "river_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := db.ExecContext(t.Context(), "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("River用スキーマの作成に失敗: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("River用スキーマの削除に失敗: %v", err)
		}
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	driver := riverdatabasesql.New(db)
	migrator, err := rivermigrate.New(driver, &rivermigrate.Config{Schema: schema, Logger: logger})
	if err != nil {
		t.Fatalf("Riverのマイグレーターの作成に失敗: %v", err)
	}
	if _, err := migrator.Migrate(t.Context(), rivermigrate.DirectionUp, nil); err != nil {
		t.Fatalf("Riverのマイグレーションに失敗: %v", err)
	}
	client, err := river.NewClient(driver, &river.Config{Schema: schema, Logger: logger})
	if err != nil {
		t.Fatalf("Riverクライアントの作成に失敗: %v", err)
	}
	return client, schema
}
