// Command devtotp prints the current TOTP code of a dev user so that browser
// verification can clear the two-factor step of the sign-in flow.
//
// The target user is named by WIKINO_DEVTOTP_ATNAME, the identifier the
// application itself shows, so that a code can be minted for any account in the
// development database rather than only for the ones the seed creates. A caller
// that already holds an email address instead names the user by
// WIKINO_DEVTOTP_EMAIL; scripts/browse.sh is such a caller, as it reads the
// address out of the account roster through `wikino devcreds`. Exactly one of
// the two is given.
//
// The rest of the settings come from the usual environment (DATABASE_URL and
// friends). The TOTP secret is read and consumed inside this process and never
// reaches argv, which `ps` exposes to other processes; only the six-digit code
// is written to stdout.
//
// Being able to mint a one-time code for an arbitrary user is powerful, so the
// command refuses to run when APP_ENV names the production environment.
//
// [Ja] devtotp コマンドは dev ユーザーの現在の TOTP コードを出力し、ブラウザ確認が
// サインインフローの 2 要素認証ステップを通過できるようにする。
//
// 対象ユーザーは WIKINO_DEVTOTP_ATNAME で指定する。これはアプリケーション自身が
// 表示する識別子であり、シードが作るアカウントに限らず開発用データベースにある
// どのアカウントでもコードを生成できるようにするため。すでにメールアドレスを
// 持っている呼び出し側は、代わりに WIKINO_DEVTOTP_EMAIL で指定する。
// scripts/browse.sh がそれにあたり、`wikino devcreds` を通して名簿から
// メールアドレスを読んでいる。指定するのは 2 つのうちちょうど 1 つ。
//
// それ以外の設定は通常の環境変数 (DATABASE_URL など) から読む。TOTP の secret は
// 本プロセス内で読み出して使い切り、`ps` が他プロセスに見せる argv には出さない。
// 標準出力に書くのは 6 桁のコードだけ。
//
// 任意のユーザーのワンタイムコードを生成できる強力なコマンドのため、APP_ENV が
// 本番環境を指しているときは実行を拒否する。
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// atnameEnvKey and emailEnvKey name the environment variables that carry the
// target user's identifier.
//
// [Ja] atnameEnvKey と emailEnvKey は対象ユーザーの識別子を渡す環境変数名。
const (
	atnameEnvKey = "WIKINO_DEVTOTP_ATNAME"
	emailEnvKey  = "WIKINO_DEVTOTP_EMAIL"
)

// userSelector names the account whose code is minted. Exactly one of its
// fields holds a value, which is what selectorFromEnv guarantees; taking both
// identifiers as one value is what keeps that guarantee in one place instead of
// spreading it over every lookup.
//
// [Ja] userSelector はコードを生成する対象のアカウントを指す。値を持つのは
// フィールドのうちちょうど 1 つで、それを保証するのは selectorFromEnv である。
// 2 つの識別子を 1 つの値として扱うことが、その保証を各所の検索に散らさず
// 一箇所に留める方法になる。
type userSelector struct {
	atname string
	email  string
}

// String renders the selector for an error message. An atname carries the "@"
// prefix the application itself shows, so that the two kinds of identifier stay
// distinguishable in the output.
//
// [Ja] String はエラーメッセージ用にセレクタを文字列にする。atname には
// アプリケーション自身が表示する "@" を付け、出力の中で 2 種類の識別子が
// 区別できるようにする。
func (s userSelector) String() string {
	if s.atname != "" {
		return "@" + s.atname
	}

	return s.email
}

func main() {
	if err := run(context.Background(), os.Getenv, os.Stdout); err != nil {
		slog.Error("TOTPコードの生成に失敗しました", "error", err)
		os.Exit(1)
	}
}

// run writes the current TOTP code of the configured user to out. Diagnostics
// go to the logger instead of out so that callers can consume stdout as the
// code itself.
//
// [Ja] run は設定されたユーザーの現在の TOTP コードを out に書く。呼び出し側が
// 標準出力をそのままコードとして扱えるよう、診断情報は out ではなくロガーへ出す。
func run(ctx context.Context, getenv func(string) string, out io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗: %w", err)
	}
	if err := ensureNotProduction(cfg); err != nil {
		return err
	}

	target, err := selectorFromEnv(getenv)
	if err != nil {
		return err
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

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("データベースへのpingに失敗: %w", err)
	}

	code, err := codeForUser(ctx, query.New(db), target, time.Now())
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, code); err != nil {
		return fmt.Errorf("TOTPコードの出力に失敗: %w", err)
	}

	return nil
}

// ensureNotProduction rejects a production environment.
//
// [Ja] ensureNotProduction は本番環境での実行を拒否する。
func ensureNotProduction(cfg *config.Config) error {
	if cfg.IsProduction() {
		return errors.New("devtotpは本番環境では実行できません")
	}

	return nil
}

// selectorFromEnv reads the target user's identifier from the environment. It
// rejects both identifiers together rather than picking one, because a caller
// that set both has two accounts in mind and silently serving one of them would
// hand back a code for the wrong account.
//
// [Ja] selectorFromEnv は対象ユーザーの識別子を環境変数から読む。2 つの識別子が
// 揃っているときは片方を選ばずに拒否する。両方を設定した呼び出し側は 2 つの
// アカウントを念頭に置いており、黙って片方を採ると誤ったアカウントのコードを
// 返すことになるため。
func selectorFromEnv(getenv func(string) string) (userSelector, error) {
	atname := getenv(atnameEnvKey)
	email := getenv(emailEnvKey)

	switch {
	case atname != "" && email != "":
		return userSelector{}, fmt.Errorf("環境変数 %s と %s は同時に指定できません", atnameEnvKey, emailEnvKey)
	case atname != "":
		return userSelector{atname: atname}, nil
	case email != "":
		return userSelector{email: email}, nil
	default:
		return userSelector{}, fmt.Errorf("環境変数 %s または %s のどちらかが必要です", atnameEnvKey, emailEnvKey)
	}
}

// codeForUser reads the TOTP secret of the selected user and returns the code
// valid at the given time. It uses the same library as the sign-in verification
// so that both sides agree on the algorithm, period and digit count.
//
// [Ja] codeForUser は指定されたユーザーの TOTP secret を読み出し、指定時刻に
// 有効なコードを返す。アルゴリズム・期間・桁数が検証側とずれないよう、
// サインインの検証と同じライブラリを使う。
func codeForUser(ctx context.Context, queries *query.Queries, target userSelector, at time.Time) (string, error) {
	user, err := findUser(ctx, repository.NewUserRepository(queries), target)
	if err != nil {
		return "", fmt.Errorf("ユーザーの取得に失敗: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("ユーザーが見つかりません: %s", target)
	}

	twoFactorAuth, err := repository.NewUserTwoFactorAuthRepository(queries).FindEnabledByUserID(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("二要素認証設定の取得に失敗: %w", err)
	}
	if twoFactorAuth == nil {
		return "", fmt.Errorf("二要素認証が有効になっていません: %s", target)
	}

	code, err := totp.GenerateCode(twoFactorAuth.Secret, at)
	if err != nil {
		return "", fmt.Errorf("TOTPコードの生成に失敗: %w", err)
	}

	return code, nil
}

// findUser looks the account up by whichever identifier the selector holds.
//
// [Ja] findUser はセレクタが持っているほうの識別子でアカウントを引く。
func findUser(ctx context.Context, users *repository.UserRepository, target userSelector) (*model.User, error) {
	if target.atname != "" {
		return users.FindByAtname(ctx, target.atname)
	}

	return users.FindByEmail(ctx, target.email)
}
