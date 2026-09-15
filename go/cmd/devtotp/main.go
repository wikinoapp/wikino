// devtotpコマンドはdevユーザーの現在のTOTPコードを出力し、ブラウザ確認が
// サインインフローの2要素認証ステップを通過できるようにする。
//
// 対象ユーザーはWIKINO_DEVTOTP_ATNAMEで指定する。これはアプリケーション自身が
// 表示する識別子であり、シードが作るアカウントに限らず開発用データベースにある
// どのアカウントでもコードを生成できるようにするため。すでにメールアドレスを
// 持っている呼び出し側は、代わりにWIKINO_DEVTOTP_EMAILで指定する。
// scripts/browse.shがそれにあたり、`wikino devcreds` を通して名簿から
// メールアドレスを読んでいる。指定するのは2つのうちちょうど1つ。
//
// それ以外の設定は通常の環境変数 (DATABASE_URLなど) から読む。TOTPのsecretは
// 本プロセス内で読み出して使い切り、`ps` が他プロセスに見せるargvには出さない。
// 標準出力に書くのは6桁のコードだけ。
//
// 任意のユーザーのワンタイムコードを生成できる強力なコマンドのため、APP_ENVが
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

// atnameEnvKeyとemailEnvKeyは対象ユーザーの識別子を渡す環境変数名。
const (
	atnameEnvKey = "WIKINO_DEVTOTP_ATNAME"
	emailEnvKey  = "WIKINO_DEVTOTP_EMAIL"
)

// userSelectorはコードを生成する対象のアカウントを指す。値を持つのは
// フィールドのうちちょうど1つで、それを保証するのはselectorFromEnvである。
// 2つの識別子を1つの値として扱うことが、その保証を各所の検索に散らさず
// 一箇所に留める方法になる。
type userSelector struct {
	atname string
	email  string
}

// Stringはエラーメッセージ用にセレクタを文字列にする。atnameには
// アプリケーション自身が表示する "@" を付け、出力の中で2種類の識別子が
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

// runは設定されたユーザーの現在のTOTPコードをoutに書く。呼び出し側が
// 標準出力をそのままコードとして扱えるよう、診断情報はoutではなくロガーへ出す。
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

// ensureNotProductionは本番環境での実行を拒否する。
func ensureNotProduction(cfg *config.Config) error {
	if cfg.IsProduction() {
		return errors.New("devtotpは本番環境では実行できません")
	}

	return nil
}

// selectorFromEnvは対象ユーザーの識別子を環境変数から読む。2つの識別子が
// 揃っているときは片方を選ばずに拒否する。両方を設定した呼び出し側は2つの
// アカウントを念頭に置いており、黙って片方を採ると誤ったアカウントのコードを
// 返すことになるため。
func selectorFromEnv(getenv func(string) string) (userSelector, error) {
	atname := getenv(atnameEnvKey)
	email := getenv(emailEnvKey)

	switch {
	case atname != "" && email != "":
		return userSelector{}, fmt.Errorf("環境変数 %sと %sは同時に指定できません", atnameEnvKey, emailEnvKey)
	case atname != "":
		return userSelector{atname: atname}, nil
	case email != "":
		return userSelector{email: email}, nil
	default:
		return userSelector{}, fmt.Errorf("環境変数 %sまたは %sのどちらかが必要です", atnameEnvKey, emailEnvKey)
	}
}

// codeForUserは指定されたユーザーのTOTP secretを読み出し、指定時刻に
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

// findUserはセレクタが持っているほうの識別子でアカウントを引く。
func findUser(ctx context.Context, users *repository.UserRepository, target userSelector) (*model.User, error) {
	if target.atname != "" {
		return users.FindByAtname(ctx, target.atname)
	}

	return users.FindByEmail(ctx, target.email)
}
