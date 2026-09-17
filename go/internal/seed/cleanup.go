package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/query"
)

// cleanupTablesはシードが毎回作り直すテーブルの一覧。実行のたびにこれらを
// 空にしてから始めることで、画面に出るデータが常に現在のコードの生成結果と
// 一致するようにする。
var cleanupTables = []string{
	"attachments",
	"draft_page_revisions",
	"draft_pages",
	"email_confirmations",
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

// preservedTablesはクリーンアップが触らないテーブルの一覧。データベースを
// 使い続けるための管理情報 (マイグレーションのバージョン、ジョブキュー自身の状態)
// か、Go版のシードが作らないRails期のデータのいずれかを持つ。
//
// この一覧を網羅的にしているのは意図的で、cleanupTablesと合わせてスキーマの
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

// cleanupはcleanupTablesのテーブルをすべて空にする。
//
// 1行ずつ削除するのではなく、1文のTRUNCATEとCASCADEで切り詰める。シードが
// 作る規模ではTRUNCATEのほうが大幅に速く、CASCADEを使えば外部キーの解決に、
// session_replication_roleによる制約の無効化が要求するスーパーユーザー権限が
// 要らない。全テーブルを1文にまとめることでロックの取得も一度で済み、データが
// 半分だけ消えた状態で途中停止することがなくなる。
func cleanup(ctx context.Context, dbtx query.DBTX) error {
	if _, err := dbtx.ExecContext(ctx, cleanupSQL()); err != nil {
		return fmt.Errorf("テーブルのクリーンアップに失敗: %w", err)
	}

	return nil
}

// cleanupSQLはTRUNCATE文を組み立てる。テーブル名はプレースホルダーで
// 渡せないため、識別子としてクォートする。名前はいずれも上のパッケージレベルの
// 一覧に由来し、入力から来ることはない。
func cleanupSQL() string {
	quoted := make([]string, 0, len(cleanupTables))
	for _, table := range cleanupTables {
		quoted = append(quoted, pq.QuoteIdentifier(table))
	}

	// #nosec G201 -- 埋め込む名前は固定のパッケージレベルの一覧
	// cleanupTablesをクォートした識別子であり、外部入力ではない。
	return fmt.Sprintf("TRUNCATE TABLE %s CASCADE", strings.Join(quoted, ", "))
}
