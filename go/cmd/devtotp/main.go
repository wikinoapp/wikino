// Command devtotp prints the current TOTP code of a dev user so that browser
// verification can clear the two-factor step of the sign-in flow.
//
// The target user is given by WIKINO_DEVTOTP_EMAIL and the rest of the settings
// come from the usual environment (DATABASE_URL and friends). The TOTP secret
// is read and consumed inside this process and never reaches argv, which `ps`
// exposes to other processes; only the six-digit code is written to stdout.
//
// Being able to mint a one-time code for an arbitrary user is powerful, so the
// command refuses to run when APP_ENV names the production environment.
//
// [Ja] devtotp コマンドは dev ユーザーの現在の TOTP コードを出力し、ブラウザ確認が
// サインインフローの 2 要素認証ステップを通過できるようにする。
//
// 対象ユーザーは WIKINO_DEVTOTP_EMAIL で指定し、それ以外の設定は通常の環境変数
// (DATABASE_URL など) から読む。TOTP の secret は本プロセス内で読み出して使い切り、
// `ps` が他プロセスに見せる argv には出さない。標準出力に書くのは 6 桁のコードだけ。
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
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// emailEnvKey names the environment variable holding the target user's email.
//
// [Ja] emailEnvKey は対象ユーザーのメールアドレスを渡す環境変数名。
const emailEnvKey = "WIKINO_DEVTOTP_EMAIL"

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
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
func run(ctx context.Context, out io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗: %w", err)
	}
	if err := ensureNotProduction(cfg); err != nil {
		return err
	}

	email := os.Getenv(emailEnvKey)
	if email == "" {
		return fmt.Errorf("必須の環境変数 %s が設定されていません", emailEnvKey)
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

	code, err := codeForUser(ctx, query.New(db), email, time.Now())
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

// codeForUser reads the TOTP secret of the user with the given email and
// returns the code valid at the given time. It uses the same library as the
// sign-in verification so that both sides agree on the algorithm, period and
// digit count.
//
// [Ja] codeForUser は指定メールアドレスのユーザーの TOTP secret を読み出し、指定
// 時刻に有効なコードを返す。アルゴリズム・期間・桁数が検証側とずれないよう、
// サインインの検証と同じライブラリを使う。
func codeForUser(ctx context.Context, queries *query.Queries, email string, at time.Time) (string, error) {
	user, err := repository.NewUserRepository(queries).FindByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("ユーザーの取得に失敗: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("ユーザーが見つかりません: %s", email)
	}

	twoFactorAuth, err := repository.NewUserTwoFactorAuthRepository(queries).FindEnabledByUserID(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("二要素認証設定の取得に失敗: %w", err)
	}
	if twoFactorAuth == nil {
		return "", fmt.Errorf("二要素認証が有効になっていません: %s", email)
	}

	code, err := totp.GenerateCode(twoFactorAuth.Secret, at)
	if err != nil {
		return "", fmt.Errorf("TOTPコードの生成に失敗: %w", err)
	}

	return code, nil
}
