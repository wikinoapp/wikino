package root

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

type result struct {
	stdout string
	stderr string
	err    error
}

// executeWithはルートコマンドにaddで部品を足してから実行する。
// ルートの下にまだコマンドが無いため、サブコマンドを束ねるコマンドの振る舞いは
// テスト用のコマンドを足して確かめる。
func executeWith(t *testing.T, lang string, args []string, add func(root *cobra.Command, loc *i18n.Localizer)) result {
	t.Helper()

	loc := i18n.NewLocalizer(lang)

	var helpErr error
	// JSONのヘルプはFactoryの標準出力に書くため、cobraの出力先も同じバッファにする。
	ios, _, stdout, stderr := iostreams.Test()
	cmd := newCmdRoot(newTestFactory(loc, ios), func(err error) { helpErr = err })
	if add != nil {
		add(cmd, loc)
	}

	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err := cmd.Execute()
	if err == nil {
		err = helpErr
	}

	return result{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func newTestFactory(loc *i18n.Localizer, ios *iostreams.IOStreams) *cmdutil.Factory {
	return &cmdutil.Factory{
		IOStreams: ios,
		Localizer: loc,
		Version:   "1.2.3",
		Getenv:    func(string) string { return "" },
	}
}

// addSampleGroupは、サブコマンドを束ねるsampleと、その下のgetを足す。
// 実在するコマンドと名前が重ならないよう、架空の名詞にする。
func addSampleGroup(root *cobra.Command, loc *i18n.Localizer) {
	sample := &cobra.Command{Use: "sample", Short: "サンプルを扱う"}
	cmdutil.ConfigureGroup(sample, loc)

	get := &cobra.Command{
		Use:     "get <n>",
		Short:   "サンプルを取得する",
		Example: "  wikino sample get 12",
		Args:    cobra.ExactArgs(1),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: "サンプルのJSON",
			cmdutil.AnnotationErrors: "終了コード5: サンプルが無い",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), "got "+args[0])

			return nil
		},
	}

	sample.AddCommand(get)
	root.AddCommand(sample)
}

func assertUsageError(t *testing.T, err error, wantMessage string, wantHint string) {
	t.Helper()

	var usageErr *output.Error
	if !errors.As(err, &usageErr) {
		t.Fatalf("使い方の誤りを期待したが %v (%T) だった", err, err)
	}
	if usageErr.Code != output.CodeUsage || usageErr.ExitCode != output.ExitUsage {
		t.Errorf("コード = %q・終了コード = %d、期待値 = %q・%d", usageErr.Code, usageErr.ExitCode, output.CodeUsage, output.ExitUsage)
	}
	if usageErr.Message != wantMessage {
		t.Errorf("メッセージ = %q、期待値 = %q", usageErr.Message, wantMessage)
	}
	if usageErr.Hint != wantHint {
		t.Errorf("ヒント = %q、期待値 = %q", usageErr.Hint, wantHint)
	}
}

func TestExecuteVersion(t *testing.T) {
	t.Parallel()

	ios, _, stdout, _ := iostreams.Test()

	jsonRequested, err := Execute([]string{"--version"}, newTestFactory(i18n.NewLocalizer(i18n.LangJa), ios))
	if err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if jsonRequested {
		t.Error("--jsonを指定していないのに、指定されたと返した")
	}
	if got, want := stdout.String(), "wikino version 1.2.3\n"; got != want {
		t.Errorf("標準出力 = %q、期待値 = %q", got, want)
	}
}

func TestExecuteReportsJSONFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "指定なし", args: []string{"nosuchcommand"}, want: false},
		{name: "未知のサブコマンドと--json", args: []string{"nosuchcommand", "--json"}, want: true},
		{name: "未知のフラグより前の--json", args: []string{"--json", "--nosuchflag"}, want: true},
		{name: "--json=false", args: []string{"--json=false", "nosuchcommand"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ios, _, _, _ := iostreams.Test()

			jsonRequested, err := Execute(tt.args, newTestFactory(i18n.NewLocalizer(i18n.LangJa), ios))
			if err == nil {
				t.Fatal("使い方の誤りを期待したが、nilだった")
			}
			if jsonRequested != tt.want {
				t.Errorf("--jsonの指定 = %v、期待値 = %v", jsonRequested, tt.want)
			}
		})
	}
}

func TestExecuteUsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantMessage string
		wantHint    string
	}{
		{
			name:        "サブコマンドが無い",
			args:        []string{},
			wantMessage: "サブコマンドを指定してください。",
			wantHint:    "`wikino --help` で使い方を確認してください。",
		},
		{
			name:        "未知のサブコマンド",
			args:        []string{"nosuchcommand"},
			wantMessage: `不明なサブコマンドです: "nosuchcommand"`,
			wantHint:    "`wikino --help` で使い方を確認してください。",
		},
		{
			name:        "既存のコマンドに近い未知のサブコマンドにも候補を出さない",
			args:        []string{"pag"},
			wantMessage: `不明なサブコマンドです: "pag"`,
			wantHint:    "`wikino --help` で使い方を確認してください。",
		},
		{
			name:        "未知のサブコマンドに--helpを付けてもヘルプを出さない",
			args:        []string{"nosuchcommand", "--help"},
			wantMessage: `不明なサブコマンドです: "nosuchcommand"`,
			wantHint:    "`wikino --help` で使い方を確認してください。",
		},
		{
			name:        "helpサブコマンドは使えない",
			args:        []string{"help", "sample"},
			wantMessage: `不明なサブコマンドです: "help"`,
			wantHint:    "`wikino --help` で使い方を確認してください。",
		},
		{
			name:        "未知のフラグ",
			args:        []string{"--nosuchflag"},
			wantMessage: "不明なフラグです: --nosuchflag",
			wantHint:    "`wikino --help` で使い方を確認してください。",
		},
		{
			name:        "グループだけの実行",
			args:        []string{"sample"},
			wantMessage: "サブコマンドを指定してください。",
			wantHint:    "`wikino sample --help` で使い方を確認してください。",
		},
		{
			name:        "グループの未知のサブコマンド",
			args:        []string{"sample", "nosuchcommand"},
			wantMessage: `不明なサブコマンドです: "nosuchcommand"`,
			wantHint:    "`wikino sample --help` で使い方を確認してください。",
		},
		{
			name:        "グループの未知のサブコマンドに--helpを付けてもヘルプを出さない",
			args:        []string{"sample", "nosuchcommand", "--help"},
			wantMessage: `不明なサブコマンドです: "nosuchcommand"`,
			wantHint:    "`wikino sample --help` で使い方を確認してください。",
		},
		{
			name:        "サブコマンドの未知のフラグ",
			args:        []string{"sample", "get", "12", "--nosuchflag"},
			wantMessage: "不明なフラグです: --nosuchflag",
			wantHint:    "`wikino sample get --help` で使い方を確認してください。",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := executeWith(t, i18n.LangJa, tt.args, addSampleGroup)

			assertUsageError(t, got.err, tt.wantMessage, tt.wantHint)
			if got.stdout != "" || got.stderr != "" {
				t.Errorf("何も出力しないことを期待したが、標準出力 = %q、標準エラー = %q だった", got.stdout, got.stderr)
			}
		})
	}
}

func TestExecuteUsageErrorInEnglish(t *testing.T) {
	t.Parallel()

	got := executeWith(t, i18n.LangEn, []string{"nosuchcommand"}, nil)

	assertUsageError(t, got.err, `Unknown subcommand: "nosuchcommand"`, "Run `wikino --help` to see the usage.")
}

func TestExecuteRunsSubcommand(t *testing.T) {
	t.Parallel()

	got := executeWith(t, i18n.LangJa, []string{"sample", "get", "12"}, addSampleGroup)

	if got.err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", got.err)
	}
	if got.stdout != "got 12" {
		t.Errorf("標準出力 = %q、期待値 = %q", got.stdout, "got 12")
	}
}

func TestExecuteHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		lang    string
		args    []string
		want    []string
		notWant []string
	}{
		{
			name: "ルートのヘルプ",
			lang: i18n.LangJa,
			args: []string{"--help"},
			want: []string{
				"Wikinoの公開APIを呼んで",
				"使い方:\n  wikino <command>\n",
				"例:\n  # バージョンを表示する\n  wikino --version\n",
				"コマンド:\n  api         公開APIを認証付きで直接呼ぶ\n  attachment  添付ファイルを扱う\n  auth        ログインと資格情報を扱う\n  draft       ページの下書きを扱う\n  page        ページを扱う\n  sample      サンプルを扱う\n  skill       コーディングエージェント向けのスキルを扱う\n  topic       トピックを扱う\n",
				"      --json                  TTYでも結果をJSONで出す\n",
				"      --space space           スペースの識別子 (space)\n      --version               バージョンを表示する\n",
				"出力:\n  標準出力がTTYなら人間向けの表示を",
				"エラー:\n  終了コードで失敗の種類を区別します。\n  0    成功\n  1    想定外の失敗",
				"  9    確認が必要: ",
				"  130  中断: Ctrl-C\n",
				"各コマンドの使い方は `wikino <command> --help` で確認できます。",
			},
			notWant: []string{"wikino [flags]", "help for", "version for", "completion"},
		},
		{
			name: "ルートのヘルプ (英語)",
			lang: i18n.LangEn,
			args: []string{"-h"},
			want: []string{
				"A command-line tool that reads and writes pages and drafts",
				"Usage:\n  wikino <command>\n",
				"Examples:\n  # Show the version\n  wikino --version\n",
				"Commands:\n  api         Call the public API directly with authentication\n  attachment  Work with attachments\n  auth        Manage login and credentials\n  draft       Work with page drafts\n  page        Work with pages\n  sample      サンプルを扱う\n  skill       Manage the skill for coding agents\n  topic       Work with topics\n",
				"      --json                  Print results as JSON even on a TTY\n",
				"      --space space           Space identifier (space)\n      --version               Show the version\n",
				"Output:\n  Prints human-readable output when stdout is a TTY",
				"Errors:\n  The exit code tells the kind of failure.\n  0    Success\n",
				"Use `wikino <command> --help` for more information about a command.",
			},
		},
		{
			name: "グループのヘルプ",
			lang: i18n.LangJa,
			args: []string{"sample", "--help"},
			want: []string{
				"使い方:\n  wikino sample <command>\n",
				"コマンド:\n  get         サンプルを取得する\n",
				"共通のフラグ:\n      --fields number,title   出す項目をカンマ区切りで選ぶ",
				"各コマンドの使い方は `wikino sample <command> --help` で確認できます。",
			},
		},
		{
			name: "出力とエラーの節を持つコマンドのヘルプ",
			lang: i18n.LangJa,
			args: []string{"sample", "get", "--help"},
			want: []string{
				"使い方:\n  wikino sample get <n> [flags]\n",
				"例:\n  wikino sample get 12\n",
				"出力:\nサンプルのJSON\n",
				"エラー:\n終了コード5: サンプルが無い\n",
			},
			notWant: []string{"コマンド:", "各コマンドの使い方は"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := executeWith(t, tt.lang, tt.args, addSampleGroup)

			if got.err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", got.err)
			}
			for _, w := range tt.want {
				if !strings.Contains(got.stdout, w) {
					t.Errorf("ヘルプに %q を含むことを期待したが、次の内容だった:\n%s", w, got.stdout)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(got.stdout, nw) {
					t.Errorf("ヘルプに %q を含まないことを期待したが、次の内容だった:\n%s", nw, got.stdout)
				}
			}
		})
	}
}

func TestExecuteVersionWithFlagValue(t *testing.T) {
	t.Parallel()

	got := executeWith(t, i18n.LangJa, []string{"--label", "nosuchcommand", "--version"}, func(cmd *cobra.Command, _ *i18n.Localizer) {
		cmd.Flags().String("label", "", "テスト用の値")
	})
	if got.err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", got.err)
	}
	if got.stdout != "wikino version 1.2.3\n" || got.stderr != "" {
		t.Errorf("標準出力 = %q、標準エラー = %q", got.stdout, got.stderr)
	}
}
