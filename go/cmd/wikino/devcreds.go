package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/wikinoapp/wikino/go/internal/seed"
)

// runDevCredentials writes the sign-in credentials of one seeded account and
// returns the process exit code.
//
// The account is named by the role it holds in the roster, which is the name
// the generators reach for it by as well. Naming it by role rather than by
// position is what keeps an invocation pointed at the same account when someone
// is added to the roster ahead of it.
//
// The two values are written to stdout one per line, the email first, so that a
// shell can read them into variables without parsing anything;
// scripts/browse.sh does exactly that, as it does with the code cmd/devtotp
// prints. Handing the password back this way is what keeps it out of argv,
// which `ps` shows to every other process on the machine.
//
// [Ja] runDevCredentials は、シードが作成したアカウント 1 件のサインイン用資格情報を
// 出力し、プロセスの終了コードを返す。
//
// アカウントは名簿で持つ役割で指定する。これは生成器がそのアカウントを名指しする
// ときの名前でもある。位置ではなく役割で指定することが、名簿の手前に誰かが足された
// ときにも、同じ指定が同じアカウントを指し続ける理由になる。
//
// 2 つの値はメールアドレスを先に、1 行ずつ標準出力へ書く。シェルが何も解釈せずに
// 変数へ読み込めるようにするためで、scripts/browse.sh は cmd/devtotp が出力する
// コードと同じようにこれを読む。この形でパスワードを返すことが、`ps` が同じマシンの
// 他のすべてのプロセスへ見せる argv に、パスワードを載せずに済ませる方法になる。
func runDevCredentials(
	ctx context.Context,
	args []string,
	appEnv string,
	findCredentials func(string) (*seed.Credentials, error),
	out io.Writer,
	stderr io.Writer,
) int {
	if len(args) != 1 {
		// The write error is discarded for the same reason as in run.
		//
		// [Ja] 書き込みエラーを捨てる理由は run と同じ。
		_, _ = fmt.Fprint(stderr, "usage: wikino devcreds <role>\n")

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

// devCredentials verifies the raw APP_ENV value before looking the role up, so
// that the guard sees what the environment actually holds rather than the
// development default config.Load would substitute for an unset value. The
// lookup itself needs no configuration: the roster is a file, not a database.
//
// [Ja] devCredentials は、役割を引く前に生の APP_ENV を検証する。ガードが見るのが、
// config.Load が未設定時に補う開発環境の既定値ではなく、環境が実際に持っている値で
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
