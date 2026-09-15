package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/seed"
)

// runSeedは開発用データベースにシードデータを投入し、プロセスの終了コードを
// 返す。
func runSeed(ctx context.Context) int {
	if err := seedDatabase(ctx, os.Getenv("APP_ENV")); err != nil {
		slog.ErrorContext(ctx, "シードデータの投入に失敗しました", "error", err)

		return 1
	}

	return 0
}

// seedDatabaseはconfig.Loadが開発環境の既定値を補う前のAPP_ENVを検証し、
// データベース接続を開いて処理をinternal/seedに委ねる。終了せずにエラーを返す
// のは、deferしたCloseを必ず走らせるため。os.Exitはdeferした処理を飛ばして
// しまう。
func seedDatabase(ctx context.Context, appEnv string) error {
	if err := seed.EnsureDevEnv(appEnv); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗: %w", err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseDSN())
	if err != nil {
		return fmt.Errorf("データベースへの接続に失敗: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.WarnContext(ctx, "データベース接続のクローズに失敗しました", "error", err)
		}
	}()

	// 進捗はslogが書いているのと同じ標準エラー出力へ送る。make seedは
	// op run経由で実行され、opは標準出力と標準エラー出力を別々に中継するため、
	// 標準出力へ書いた進捗は前後のログ行と順序が入れ替わって届く。同じストリームに
	// 載せれば、両者は書いた順のまま並ぶ。このコマンドの標準出力を読む利用側は無い
	// (wikino seedは機械可読な出力を持たない) ため、進捗を移しても取り残される
	// 呼び出し側は無い。
	return seed.NewRunner(db, cfg, os.Stderr).Run(ctx)
}
