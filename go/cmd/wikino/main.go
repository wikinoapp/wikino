// Command wikino is Wikino's command-line entry point. The serve subcommand
// starts the HTTP server, the seed subcommand fills a development database with
// seed data, and the devcreds subcommand prints the credentials one of the
// accounts that seed creates signs in with.
//
// [Ja] wikino コマンドは Wikino のコマンドラインエントリーポイント。
// serve サブコマンドが HTTP サーバーを起動し、seed サブコマンドが開発用データ
// ベースにシードデータを投入し、devcreds サブコマンドがそのシードで作成される
// アカウント 1 件のサインイン用資格情報を出力する。
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/wikinoapp/wikino/go/internal/seed"
)

// exitUsage is the exit code used when the command line names no subcommand or
// an unknown one. It follows the convention of the Go toolchain and of getopt,
// where 2 signals a usage error and 1 signals a failure of the requested work.
//
// [Ja] exitUsage はサブコマンドが指定されていない / 未知の場合に使う終了コード。
// 使用方法の誤りを 2、依頼された処理自体の失敗を 1 とする Go ツールチェインや
// getopt の慣習に従う。
const exitUsage = 2

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches args to a subcommand and returns the process exit code. It
// takes the arguments and both output writers as parameters, rather than
// reading os.Args and writing to the process streams directly, so that the
// dispatch can be tested without terminating the test process.
//
// stdout is threaded through for devcreds, whose stdout is the machine-readable
// contract scripts/browse.sh reads. Reaching for os.Stdout inside the dispatch
// would put the one stream a caller parses out of a test's reach.
//
// No subcommand defaults to serve: an invocation that names nothing prints the
// usage and fails, so every call site has to state which subcommand it wants.
//
// [Ja] run は args をサブコマンドへ振り分け、プロセスの終了コードを返す。os.Args を
// 直接読んでプロセスのストリームへ書くのではなく、引数と 2 つの Writer を受け取るのは、
// テストプロセスを終了させずに振り分けをテストできるようにするため。
//
// 標準出力を通しているのは devcreds のためで、その標準出力は scripts/browse.sh が読む
// 機械可読の契約になっている。振り分けの中で os.Stdout を直接掴むと、呼び出し側が
// 解釈する唯一のストリームがテストから触れなくなる。
//
// サブコマンド無しのときに serve へ既定することはしない。何も指定しない実行は
// usage を表示して失敗するため、各呼び出し箇所がどのサブコマンドを使うのかを
// 明示することになる。
func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)

		return exitUsage
	}

	switch args[0] {
	case "serve":
		runServe()
	case "seed":
		return runSeed(context.Background())
	case "devcreds":
		return runDevCredentials(
			context.Background(),
			args[1:],
			os.Getenv("APP_ENV"),
			seed.FindCredentials,
			stdout,
			stderr,
		)
	default:
		// The write error is discarded on purpose: this is the diagnostic
		// channel itself, so there is nowhere left to report a failure to
		// write to it. The exit code still tells the caller what happened.
		//
		// [Ja] 書き込みエラーは意図的に捨てる。ここは診断情報の出力先そのものであり、
		// その書き込みに失敗したことを報告する先が残っていないため。何が起きたかは
		// 終了コードで呼び出し側に伝わる。
		_, _ = fmt.Fprintf(stderr, "unknown subcommand: %q\n\n", args[0])
		usage(stderr)

		return exitUsage
	}

	return 0
}

// usage writes the list of available subcommands to w. The write error is
// discarded for the same reason as in run.
//
// [Ja] usage は利用可能なサブコマンドの一覧を w に書く。書き込みエラーを捨てる理由は
// run と同じ。
func usage(w io.Writer) {
	_, _ = fmt.Fprint(w, `usage: wikino <command>

commands:
  serve      start the HTTP server
  seed       populate the development database with seed data
  devcreds   print the sign-in credentials of a seeded account
`)
}
