package seed

import (
	"context"
	"fmt"

	dbmigration "github.com/wikinoapp/wikino/go/db"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// registerOfficialOAuthApplicationsは、本番ではマイグレーションが登録する公式クライアントを登録する。
// クリーンアップがoauth_applicationsを空にし、db/schema.sqlから作ったデータベースにはもともと
// 無いため、開発環境で公式CLIからログインできるよう、マイグレーションと同じSQLで毎回登録し直す。
func registerOfficialOAuthApplications(ctx context.Context, dbtx query.DBTX) error {
	if _, err := dbtx.ExecContext(ctx, dbmigration.OfficialCLIOAuthApplicationSQL()); err != nil {
		return fmt.Errorf("公式CLIのOAuthクライアントの登録に失敗: %w", err)
	}

	return nil
}
