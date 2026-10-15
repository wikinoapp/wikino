// Package dbは、dbmateのマイグレーションのうちGoのコードからも実行するものを埋め込む。
//
// db/schema.sqlはスキーマだけを持ち、マイグレーションが入れたデータを含まない。開発と
// テストのデータベースはdb/schema.sqlから作るため、データを入れるマイグレーションの結果は
// そのままでは本番にしか現れない。埋め込んだup節をシードとテストから実行し、本番と同じ行を
// 登録する。
package db

import (
	_ "embed"
	"strings"
)

//go:embed migrations/20261012120000_register_official_cli_oauth_application.sql
var officialCLIOAuthApplicationMigration string

// OfficialCLIOAuthApplicationSQLは、公式CLIのOAuthクライアントを登録するマイグレーションのup節を返す
func OfficialCLIOAuthApplicationSQL() string {
	return migrationUpSQL(officialCLIOAuthApplicationMigration)
}

// migrationUpSQLは、dbmateのマイグレーションから `-- migrate:up` と `-- migrate:down` の間を取り出す
func migrationUpSQL(migration string) string {
	up, _, _ := strings.Cut(migration, "-- migrate:down")
	return strings.TrimPrefix(strings.TrimSpace(up), "-- migrate:up")
}
