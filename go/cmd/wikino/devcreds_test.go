package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/seed"
)

// TestRunDevCredentialsRejectsWrongArgumentCount covers the invocations that
// name no role or more than one. A shell reads this command's stdout as the
// credentials themselves, so an invocation it cannot answer has to leave that
// stream empty and say what happened on stderr.
//
// [Ja] TestRunDevCredentialsRejectsWrongArgumentCount は、役割を指定していない
// 実行と、2 つ以上指定した実行を扱う。シェルは本コマンドの標準出力を資格情報その
// ものとして読むため、答えられない実行はその出力を空のままにし、何が起きたのかは
// 標準エラー出力で告げる必要がある。
func TestRunDevCredentialsRejectsWrongArgumentCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "役割を指定していないとき", args: []string{}},
		{name: "役割を2つ以上指定したとき", args: []string{"owner", "collaborator"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out, stderr bytes.Buffer

			lookupCalled := false
			code := runDevCredentials(
				t.Context(),
				tt.args,
				"dev",
				func(string) (*seed.Credentials, error) {
					lookupCalled = true

					return &seed.Credentials{Email: "owner@example.com", Password: "seed-password"}, nil
				},
				&out,
				&stderr,
			)

			if code != exitUsage {
				t.Errorf("終了コードが %d であることを期待したが %d だった", exitUsage, code)
			}
			if out.Len() != 0 {
				t.Errorf("標準出力が空であることを期待したが %q だった", out.String())
			}
			if !strings.Contains(stderr.String(), "usage: wikino devcreds <role>") {
				t.Errorf("usageの出力を期待したが次の内容だった: %q", stderr.String())
			}
			// A role is what the lookup is asked for, so an invocation that
			// names none or names two has nothing to look up. Checking that it
			// does not is what says the argument check stands in front of the
			// roster, rather than leaving the answer to whether that file
			// happens to be readable from wherever the command was started.
			//
			// [Ja] 引きが尋ねられるのは役割であるため、役割を指定していない実行と
			// 2 つ指定した実行には、引くものが無い。引かないことを確認することが、
			// 引数の検査が名簿の手前に立っていることを言う方法になる。そうしないと、
			// その答えは、コマンドを開始した場所からそのファイルが読めるかどうかに
			// 委ねられる。
			if lookupCalled {
				t.Error("引数の個数が誤っているときは資格情報を検索しないことを期待した")
			}
		})
	}
}

// TestRunDevCredentialsWritesCredentials fixes the stdout protocol consumed by
// browse.sh: exactly two lines, with the email before the password. Diagnostics
// stay off both output streams on success.
//
// [Ja] TestRunDevCredentialsWritesCredentials は browse.sh が読む標準出力の契約を
// 固定する。メールアドレス、パスワードの順で正確に 2 行とし、成功時はどちらの
// 出力にも診断情報を混ぜない。
func TestRunDevCredentialsWritesCredentials(t *testing.T) {
	t.Parallel()

	var out, stderr bytes.Buffer
	var lookedUpRole string
	findCredentials := func(role string) (*seed.Credentials, error) {
		lookedUpRole = role

		return &seed.Credentials{
			Email:    "owner@example.com",
			Password: "seed-password",
		}, nil
	}

	code := runDevCredentials(
		t.Context(),
		[]string{"owner"},
		"dev",
		findCredentials,
		&out,
		&stderr,
	)

	if code != 0 {
		t.Errorf("終了コードが 0 であることを期待したが %d だった", code)
	}
	if lookedUpRole != "owner" {
		t.Errorf("ownerを検索することを期待したが %q だった", lookedUpRole)
	}
	if got, want := out.String(), "owner@example.com\nseed-password\n"; got != want {
		t.Errorf("標準出力が %q であることを期待したが %q だった", want, got)
	}
	if stderr.Len() != 0 {
		t.Errorf("標準エラー出力が空であることを期待したが %q だった", stderr.String())
	}
}

// TestRunDevCredentialsReportsLookupFailure covers the failure the command
// meets most often: the roster is missing, the role is misspelled, or the file
// does not pass its checks.
//
// The stdout contract holds here too. A shell reads that stream as the
// credentials themselves, so a run that has none to give has to leave it empty
// and report on stderr, which is where the logger writes.
//
// [Ja] TestRunDevCredentialsReportsLookupFailure は、本コマンドがいちばん多く出会う
// 失敗を扱う。名簿が無い、役割の綴りを間違えた、ファイルが検査を通らない、といった
// 場合である。
//
// 標準出力の契約はここでも変わらない。シェルはそのストリームを資格情報そのものとして
// 読むため、渡せるものが無い実行はそれを空のままにし、ロガーが書く先である標準エラー
// 出力で報告する必要がある。
func TestRunDevCredentialsReportsLookupFailure(t *testing.T) {
	t.Parallel()

	var out, stderr bytes.Buffer

	code := runDevCredentials(
		t.Context(),
		[]string{"onwer"},
		"dev",
		func(string) (*seed.Credentials, error) {
			return nil, errors.New(`役割 "onwer" のアカウントは名簿にありません`)
		},
		&out,
		&stderr,
	)

	if code != 1 {
		t.Errorf("終了コードが 1 であることを期待したが %d だった", code)
	}
	if out.Len() != 0 {
		t.Errorf("標準出力が空であることを期待したが %q だった", out.String())
	}
}

// TestDevCredentialsRejectsNonDevEnv checks the guard that stands in front of
// the lookup. The command prints a password, so it has to refuse anywhere the
// roster is not a development file, and it has to refuse before reading
// anything.
//
// [Ja] TestDevCredentialsRejectsNonDevEnv は、引きの手前に立つガードを確認する。
// 本コマンドはパスワードを出力するため、名簿が開発用のファイルでない場所では実行を
// 拒否する必要があり、しかも何かを読む前に拒否する必要がある。
func TestDevCredentialsRejectsNonDevEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appEnv  string
		wantErr bool
	}{
		{name: "開発環境では実行できる", appEnv: "dev", wantErr: false},
		{name: "テスト環境では実行できない", appEnv: "test", wantErr: true},
		{name: "本番環境では実行できない", appEnv: "prod", wantErr: true},
		{name: "環境が未設定のときは実行できない", appEnv: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookupCalled := false
			credentials, err := devCredentials(
				tt.appEnv,
				"owner",
				func(string) (*seed.Credentials, error) {
					lookupCalled = true

					return &seed.Credentials{Email: "owner@example.com", Password: "seed-password"}, nil
				},
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("エラーを期待したがnilだった")
				}
				if !strings.Contains(err.Error(), "開発環境でのみ実行できます") {
					t.Errorf("環境を理由とするエラーを期待したが %q だった", err)
				}
				if lookupCalled {
					t.Error("環境ガードに失敗したときは資格情報を検索しないことを期待した")
				}

				return
			}
			if err != nil {
				t.Fatalf("開発環境で資格情報を取得できることを期待したがエラーになった: %v", err)
			}
			if !lookupCalled || credentials.Email != "owner@example.com" {
				t.Errorf("開発環境で検索結果を返すことを期待したが %+v だった", credentials)
			}
		})
	}
}
