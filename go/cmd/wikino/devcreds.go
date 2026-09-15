package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/wikinoapp/wikino/go/internal/seed"
)

// runDevCredentialsは、シードが作成したアカウント1件のサインイン用資格情報を
// 出力し、プロセスの終了コードを返す。
//
// アカウントは名簿で持つ役割で指定する。これは生成器がそのアカウントを名指しする
// ときの名前でもある。位置ではなく役割で指定することが、名簿の手前に誰かが足された
// ときにも、同じ指定が同じアカウントを指し続ける理由になる。
//
// 2つの値はメールアドレスを先に、1行ずつ標準出力へ書く。シェルが何も解釈せずに
// 変数へ読み込めるようにするためで、scripts/browse.shはcmd/devtotpが出力する
// コードと同じようにこれを読む。この形でパスワードを返すことが、`ps` が同じマシンの
// 他のすべてのプロセスへ見せるargvに、パスワードを載せずに済ませる方法になる。
func runDevCredentials(
	ctx context.Context,
	args []string,
	appEnv string,
	findCredentials func(string) (*seed.Credentials, error),
	out io.Writer,
	stderr io.Writer,
) int {
	if len(args) != 1 {
		// 書き込みエラーを捨てる理由はrunと同じ。
		_, _ = fmt.Fprint(stderr, "使い方: wikino devcreds <role>\n")

		return exitUsage
	}

	credentials, err := devCredentials(appEnv, args[0], findCredentials)
	if err != nil {
		slog.ErrorContext(ctx, "開発用アカウントの資格情報の取得に失敗しました", "error", err)

		return 1
	}

	if _, err := fmt.Fprintf(out, "%s\n%s\n", credentials.Email, credentials.Password); err != nil {
		slog.ErrorContext(ctx, "資格情報の出力に失敗しました", "error", err)

		return 1
	}

	return 0
}

// devCredentialsは、役割を引く前に生のAPP_ENVを検証する。ガードが見るのが、
// config.Loadが未設定時に補う開発環境の既定値ではなく、環境が実際に持っている値で
// あるようにするため。引くこと自体に設定は要らない。名簿はデータベースではなく
// ファイルであるため。
func devCredentials(
	appEnv string,
	role string,
	findCredentials func(string) (*seed.Credentials, error),
) (*seed.Credentials, error) {
	if err := seed.EnsureDevEnv(appEnv); err != nil {
		return nil, err
	}

	return findCredentials(role)
}
