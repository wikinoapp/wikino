package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// cleanupTables lists the tables the seed rebuilds from scratch. Every run
// starts from an empty set of them, so that the data on screen always matches
// what the current code generates.
//
// [Ja] cleanupTables はシードが毎回作り直すテーブルの一覧。実行のたびにこれらを
// 空にしてから始めることで、画面に出るデータが常に現在のコードの生成結果と
// 一致するようにする。
var cleanupTables = []string{
	"attachments",
	"draft_page_revisions",
	"draft_pages",
	"email_confirmations",
	"export_statuses",
	"exports",
	"feature_flags",
	"page_attachment_references",
	"page_editors",
	"page_revisions",
	"pages",
	"password_reset_tokens",
	"rate_limits",
	"space_members",
	"spaces",
	"suggestion_comments",
	"suggestion_page_revisions",
	"suggestion_pages",
	"suggestions",
	"topic_members",
	"topics",
	"user_passwords",
	"user_sessions",
	"user_two_factor_auths",
	"users",
}

// preservedTables lists the tables the cleanup leaves alone. They hold either
// bookkeeping the database needs to stay usable (migration versions, the job
// queue's own state) or Rails-era rows the Go seed does not produce.
//
// The list is exhaustive on purpose: a test compares it together with
// cleanupTables against the tables the schema actually has, so a table added
// later has to be placed on one side or the other instead of being silently
// left behind by the cleanup.
//
// [Ja] preservedTables はクリーンアップが触らないテーブルの一覧。データベースを
// 使い続けるための管理情報 (マイグレーションのバージョン、ジョブキュー自身の状態)
// か、Go 版のシードが作らない Rails 期のデータのいずれかを持つ。
//
// この一覧を網羅的にしているのは意図的で、cleanupTables と合わせてスキーマの
// 実際のテーブルと突き合わせるテストがある。後から追加されたテーブルは、
// クリーンアップから黙って漏れるのではなく、どちらかへ必ず振り分けることになる。
var preservedTables = []string{
	"active_storage_attachments",
	"active_storage_blobs",
	"active_storage_variant_records",
	"ar_internal_metadata",
	"river_job",
	"river_leader",
	"river_migration",
	"river_notification",
	"river_queue",
	"schema_migrations",
}

// cleanup empties every table in cleanupTables.
//
// The tables are truncated by one statement with CASCADE rather than deleted
// row by row. TRUNCATE is far faster on the volumes the seed creates, and
// CASCADE settles the foreign keys without the superuser privilege that
// disabling constraints through session_replication_role would require.
// Naming all of the tables in a single statement also takes their locks at
// once, so the cleanup cannot stall halfway with the data half removed.
//
// [Ja] cleanup は cleanupTables のテーブルをすべて空にする。
//
// 1 行ずつ削除するのではなく、1 文の TRUNCATE と CASCADE で切り詰める。シードが
// 作る規模では TRUNCATE のほうが大幅に速く、CASCADE を使えば外部キーの解決に、
// session_replication_role による制約の無効化が要求するスーパーユーザー権限が
// 要らない。全テーブルを 1 文にまとめることでロックの取得も一度で済み、データが
// 半分だけ消えた状態で途中停止することがなくなる。
func cleanup(ctx context.Context, dbtx query.DBTX) error {
	if _, err := dbtx.ExecContext(ctx, cleanupSQL()); err != nil {
		return fmt.Errorf("テーブルのクリーンアップに失敗: %w", err)
	}

	return nil
}

// cleanupSQL builds the TRUNCATE statement. Table names cannot be bound as
// placeholders, so they are quoted as identifiers instead; they come from the
// package-level list above and never from input.
//
// [Ja] cleanupSQL は TRUNCATE 文を組み立てる。テーブル名はプレースホルダーで
// 渡せないため、識別子としてクォートする。名前はいずれも上のパッケージレベルの
// 一覧に由来し、入力から来ることはない。
func cleanupSQL() string {
	quoted := make([]string, 0, len(cleanupTables))
	for _, table := range cleanupTables {
		quoted = append(quoted, pq.QuoteIdentifier(table))
	}

	// #nosec G201 -- the interpolated names are quoted identifiers from
	// cleanupTables, a fixed package-level list.
	//
	// [Ja] #nosec G201 -- 埋め込む名前は固定のパッケージレベルの一覧
	// cleanupTables をクォートした識別子であり、外部入力ではない。
	return fmt.Sprintf("TRUNCATE TABLE %s CASCADE", strings.Join(quoted, ", "))
}
