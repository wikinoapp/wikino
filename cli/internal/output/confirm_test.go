package output

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
)

var testConfirmation = Confirmation{
	Changes: []string{
		"下書きの内容でページ12「議事録」を公開します",
		"公開後、この下書きと編集履歴は削除されます",
	},
	Command: []string{"wikino", "draft", "publish", "12", "--if-match", `"d41d8cd9"`, "--yes"},
}

func TestConfirmNonInteractive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stdinTTY  bool
		stdoutTTY bool
		environ   []string
		want      string
	}{
		{
			name: "非TTYでは確認プロトコルのJSONを1行で出す",
			want: `{"status":"confirmation_required","changes":["下書きの内容でページ12「議事録」を公開します","公開後、この下書きと編集履歴は削除されます"],"confirm_command":"wikino draft publish 12 --if-match '\"d41d8cd9\"' --yes"}` + "\n",
		},
		{
			name:      "TTYでもエージェントの環境変数があれば対話しない",
			stdinTTY:  true,
			stdoutTTY: true,
			environ:   []string{"CLAUDECODE=1"},
			want:      "{\n  \"status\": \"confirmation_required\",\n  \"changes\": [\n    \"下書きの内容でページ12「議事録」を公開します\",\n    \"公開後、この下書きと編集履歴は削除されます\"\n  ],\n  \"confirm_command\": \"wikino draft publish 12 --if-match '\\\"d41d8cd9\\\"' --yes\"\n}\n",
		},
		{
			name:      "標準入力がTTYでなければ対話しない",
			stdinTTY:  false,
			stdoutTTY: false,
			want:      `{"status":"confirmation_required","changes":["下書きの内容でページ12「議事録」を公開します","公開後、この下書きと編集履歴は削除されます"],"confirm_command":"wikino draft publish 12 --if-match '\"d41d8cd9\"' --yes"}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ios, stdin, stdout, stderr := iostreams.Test()
			ios.SetStdinTTY(tt.stdinTTY)
			ios.SetStdoutTTY(tt.stdoutTTY)
			ios.SetEnv(tt.environ)
			stdin.WriteString("y\n")

			err := Confirm(ios, i18n.NewLocalizer(i18n.LangJa), testConfirmation)

			if got := ExitCodeOf(err); got != ExitConfirmationRequired {
				t.Errorf("終了コード = %d、期待値 = %d (エラー: %v)", got, ExitConfirmationRequired, err)
			}
			var silent *SilentError
			if !errors.As(err, &silent) {
				t.Errorf("標準エラーに何も出さないエラーを期待したが %T だった", err)
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("標準出力 =\n%s\n期待値 =\n%s", got, tt.want)
			}
			if stderr.Len() != 0 {
				t.Errorf("標準エラーにプロンプトを出さないことを期待したが %q だった", stderr.String())
			}
			if stdin.String() != "y\n" {
				t.Errorf("標準入力を読まないことを期待したが、残りが %q だった", stdin.String())
			}
		})
	}
}

func TestConfirmInteractive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		approved bool
	}{
		{name: "yで承認する", input: "y\n", approved: true},
		{name: "yesで承認する", input: " YES \n", approved: true},
		{name: "改行の無いyでも承認する", input: "y", approved: true},
		{name: "nで拒否する", input: "n\n", approved: false},
		{name: "空の入力は拒否する", input: "\n", approved: false},
		{name: "入力が閉じられたら拒否する", input: "", approved: false},
		{name: "y以外の語は拒否する", input: "yeah\n", approved: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ios, stdin, stdout, stderr := iostreams.Test()
			ios.SetStdinTTY(true)
			ios.SetStdoutTTY(true)
			stdin.WriteString(tt.input)

			err := Confirm(ios, i18n.NewLocalizer(i18n.LangJa), testConfirmation)

			wantPrompt := "- 下書きの内容でページ12「議事録」を公開します\n- 公開後、この下書きと編集履歴は削除されます\n実行しますか? [y/N]: "
			if got := stderr.String(); got != wantPrompt {
				t.Errorf("標準エラー = %q、期待値 = %q", got, wantPrompt)
			}
			if stdout.Len() != 0 {
				t.Errorf("標準出力に何も出さないことを期待したが %q だった", stdout.String())
			}

			if tt.approved {
				if err != nil {
					t.Errorf("承認を期待したが %v だった", err)
				}
				return
			}

			var cliErr *Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("拒否のエラーを期待したが %v (%T) だった", err, err)
			}
			if cliErr.Code != CodeConfirmationDeclined || cliErr.ExitCode != ExitConfirmationRequired {
				t.Errorf("コード = %q・終了コード = %d", cliErr.Code, cliErr.ExitCode)
			}
			if cliErr.Message != "操作を中止しました。" {
				t.Errorf("メッセージ = %q", cliErr.Message)
			}
		})
	}
}

func TestJoinCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "引用の要らない引数はそのまま並べる",
			args: []string{"wikino", "draft", "delete", "12", "--space=my-space", "--yes"},
			want: "wikino draft delete 12 --space=my-space --yes",
		},
		{
			name: "二重引用符を含むETagは単一引用符で囲む",
			args: []string{"wikino", "--if-match", `W/"abc"`},
			want: `wikino --if-match 'W/"abc"'`,
		},
		{
			name: "単一引用符を含む値は閉じてからエスケープする",
			args: []string{"wikino", "--title", "it's"},
			want: `wikino --title 'it'\''s'`,
		},
		{
			name: "空白・シェルの記号・日本語を含む値を囲む",
			args: []string{"wikino", "--title", "議事録 $(rm -rf ~) `x`; *"},
			want: "wikino --title '議事録 $(rm -rf ~) `x`; *'",
		},
		{
			name: "空の値は空の引用符にする",
			args: []string{"wikino", "--title", ""},
			want: "wikino --title ''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := JoinCommand(tt.args); got != tt.want {
				t.Errorf("JoinCommand() = %s、期待値 = %s", got, tt.want)
			}
		})
	}
}

func TestConfirmPromptInEnglish(t *testing.T) {
	t.Parallel()

	ios, stdin, _, stderr := iostreams.Test()
	ios.SetStdinTTY(true)
	ios.SetStdoutTTY(true)
	stdin.WriteString("n\n")

	err := Confirm(ios, i18n.NewLocalizer(i18n.LangEn), Confirmation{Changes: []string{"Delete the draft"}})
	if err == nil || err.Error() != "Operation canceled." {
		t.Errorf("エラー = %v", err)
	}
	if !strings.HasSuffix(stderr.String(), "Proceed? [y/N]: ") {
		t.Errorf("標準エラー = %q", stderr.String())
	}
}

func TestConfirmControlCharacters(t *testing.T) {
	t.Parallel()

	change := "下書き「危険\x1b[2J\x1b[H安全\r上書き\n別の行\t末尾」を公開します"

	t.Run("対話の確認では制御文字を空白に置き換える", func(t *testing.T) {
		t.Parallel()

		ios, stdin, stdout, stderr := iostreams.Test()
		ios.SetStdinTTY(true)
		ios.SetStdoutTTY(true)
		stdin.WriteString("n\n")

		err := Confirm(ios, i18n.NewLocalizer(i18n.LangJa), Confirmation{Changes: []string{change}})
		if got := ExitCodeOf(err); got != ExitConfirmationRequired {
			t.Errorf("終了コード = %d、期待値 = %d", got, ExitConfirmationRequired)
		}
		want := "- 下書き「危険 [2J [H安全 上書き 別の行 末尾」を公開します\n実行しますか? [y/N]: "
		if got := stderr.String(); got != want {
			t.Errorf("確認の出力 = %q、期待値 = %q", got, want)
		}
		if stdout.Len() != 0 {
			t.Errorf("標準出力に何も出さないことを期待したが %q だった", stdout.String())
		}
	})

	t.Run("対話の確認では双方向テキストの制御文字も空白に置き換える", func(t *testing.T) {
		t.Parallel()

		ios, stdin, _, stderr := iostreams.Test()
		ios.SetStdinTTY(true)
		ios.SetStdoutTTY(true)
		stdin.WriteString("n\n")

		bidi := "ページ12「議事録\u202eすまし除削\u202c」を公開します"
		_ = Confirm(ios, i18n.NewLocalizer(i18n.LangJa), Confirmation{Changes: []string{bidi}})
		want := "- ページ12「議事録 すまし除削 」を公開します\n実行しますか? [y/N]: "
		if got := stderr.String(); got != want {
			t.Errorf("確認の出力 = %q、期待値 = %q", got, want)
		}
	})

	t.Run("非対話のJSONは制御文字をエスケープして元の値を保つ", func(t *testing.T) {
		t.Parallel()

		ios, _, stdout, stderr := iostreams.Test()
		err := Confirm(ios, i18n.NewLocalizer(i18n.LangJa), Confirmation{Changes: []string{change}})
		if got := ExitCodeOf(err); got != ExitConfirmationRequired {
			t.Errorf("終了コード = %d、期待値 = %d", got, ExitConfirmationRequired)
		}
		var result confirmationRequired
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("確認のJSONを読めません: %v", err)
		}
		if len(result.Changes) != 1 || result.Changes[0] != change {
			t.Errorf("変更文 = %q、期待値 = %q", result.Changes, []string{change})
		}
		if stderr.Len() != 0 {
			t.Errorf("標準エラーに何も出さないことを期待したが %q だった", stderr.String())
		}
	})
}
