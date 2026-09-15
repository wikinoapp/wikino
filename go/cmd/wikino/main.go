// wikinoコマンドはWikinoのコマンドラインエントリーポイント。
// serveサブコマンドがHTTPサーバーを起動し、seedサブコマンドが開発用データ
// ベースにシードデータを投入し、devcredsサブコマンドがそのシードで作成される
// アカウント1件のサインイン用資格情報を出力する。
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/wikinoapp/wikino/go/internal/seed"
)

// exitUsageはサブコマンドが指定されていない / 未知の場合に使う終了コード。
// 使用方法の誤りを2、依頼された処理自体の失敗を1とするGoツールチェインや
// getoptの慣習に従う。
const exitUsage = 2

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// runはargsをサブコマンドへ振り分け、プロセスの終了コードを返す。os.Argsを
// 直接読んでプロセスのストリームへ書くのではなく、引数と2つのWriterを受け取るのは、
// テストプロセスを終了させずに振り分けをテストできるようにするため。
//
// 標準出力を通しているのはdevcredsのためで、その標準出力はscripts/browse.shが読む
// 機械可読の契約になっている。振り分けの中でos.Stdoutを直接掴むと、呼び出し側が
// 解釈する唯一のストリームがテストから触れなくなる。
//
// サブコマンド無しのときにserveへ既定することはしない。何も指定しない実行は
// usageを表示して失敗するため、各呼び出し箇所がどのサブコマンドを使うのかを
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
		// 書き込みエラーは意図的に捨てる。ここは診断情報の出力先そのものであり、
		// その書き込みに失敗したことを報告する先が残っていないため。何が起きたかは
		// 終了コードで呼び出し側に伝わる。
		_, _ = fmt.Fprintf(stderr, "不明なサブコマンドです: %q\n\n", args[0])
		usage(stderr)

		return exitUsage
	}

	return 0
}

// usageは利用可能なサブコマンドの一覧をwに書く。書き込みエラーを捨てる理由は
// runと同じ。
func usage(w io.Writer) {
	_, _ = fmt.Fprint(w, `使い方: wikino <command>

コマンド:
  serve      HTTPサーバーを起動する
  seed       開発用データベースにシードデータを投入する
  devcreds   シードで作成したアカウントのサインイン用資格情報を出力する
`)
}
