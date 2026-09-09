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

// runSeed populates the development database with seed data and returns the
// process exit code.
//
// [Ja] runSeed は開発用データベースにシードデータを投入し、プロセスの終了コードを
// 返す。
func runSeed(ctx context.Context) int {
	if err := seedDatabase(ctx, os.Getenv("APP_ENV")); err != nil {
		slog.ErrorContext(ctx, "シードデータの投入に失敗しました", "error", err)

		return 1
	}

	return 0
}

// seedDatabase verifies the raw APP_ENV value before config.Load can apply its
// development default, then opens the database connection and hands the work
// to internal/seed. It returns an error instead of exiting, so that the
// deferred Close always runs: os.Exit skips deferred calls.
//
// [Ja] seedDatabase は config.Load が開発環境の既定値を補う前の APP_ENV を検証し、
// データベース接続を開いて処理を internal/seed に委ねる。終了せずにエラーを返す
// のは、defer した Close を必ず走らせるため。os.Exit は defer した処理を飛ばして
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

	// Progress goes to stderr, the stream slog already writes to. make seed
	// runs through op run, which relays stdout and stderr separately, so
	// progress written to stdout arrives out of order with the log lines around
	// it. Carried on one stream, the two stay in the order they were written.
	// Nothing reads this command's stdout — wikino seed has no machine-readable
	// output — so moving the progress off it leaves no caller behind.
	//
	// [Ja] 進捗は slog が書いているのと同じ標準エラー出力へ送る。make seed は
	// op run 経由で実行され、op は標準出力と標準エラー出力を別々に中継するため、
	// 標準出力へ書いた進捗は前後のログ行と順序が入れ替わって届く。同じストリームに
	// 載せれば、両者は書いた順のまま並ぶ。このコマンドの標準出力を読む利用側は無い
	// (wikino seed は機械可読な出力を持たない) ため、進捗を移しても取り残される
	// 呼び出し側は無い。
	return seed.NewRunner(db, cfg, os.Stderr).Run(ctx)
}
