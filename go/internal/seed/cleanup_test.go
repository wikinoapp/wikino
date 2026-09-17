package seed

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCleanupTablesCoverTheSchema(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()

	rows, err := db.QueryContext(
		context.Background(),
		`SELECT table_name FROM information_schema.tables
         WHERE table_schema = 'public' AND table_type = 'BASE TABLE'`,
	)
	if err != nil {
		t.Fatalf("テーブル一覧の取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	classified := make(map[string]string, len(cleanupTables)+len(preservedTables))
	for _, group := range []struct {
		name   string
		tables []string
	}{
		{name: "cleanupTables", tables: cleanupTables},
		{name: "preservedTables", tables: preservedTables},
	} {
		for _, table := range group.tables {
			if previousGroup, exists := classified[table]; exists {
				t.Errorf("テーブル%sが%sと%sに重複している", table, previousGroup, group.name)

				continue
			}
			classified[table] = group.name
		}
	}

	actual := make(map[string]bool)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("テーブル名の読み取りに失敗: %v", err)
		}
		actual[table] = true

		// どちらの一覧にも無いテーブルはクリーンアップを黙って生き延び、
		// シードのたびに古い行を残してしまう。
		if _, exists := classified[table]; !exists {
			t.Errorf("テーブル%sがcleanupTablesにもpreservedTablesにも含まれていない", table)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("テーブル一覧の走査に失敗: %v", err)
	}

	for table := range classified {
		if !actual[table] {
			t.Errorf("一覧のテーブル%sがスキーマに存在しない", table)
		}
	}
}

func TestCleanupSQL(t *testing.T) {
	t.Parallel()

	got := cleanupSQL()

	if !strings.HasPrefix(got, "TRUNCATE TABLE ") {
		t.Errorf("TRUNCATE文であることを期待したが次の内容だった: %q", got)
	}
	// CASCADEは、制約の無効化が要求するスーパーユーザー権限なしに、一覧の
	// テーブル間の外部キーを解決する。
	if !strings.HasSuffix(got, " CASCADE") {
		t.Errorf("CASCADEで終わることを期待したが次の内容だった: %q", got)
	}
	for _, table := range cleanupTables {
		if !strings.Contains(got, `"`+table+`"`) {
			t.Errorf("テーブル%sが引用符付きで含まれることを期待したが次の内容だった: %q", table, got)
		}
	}
}
