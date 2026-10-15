package root

import (
	"bytes"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// usageTemplateは、cobraの既定の使い方のテンプレートを翻訳し、「出力」「エラー」の節を足したもの。
// 見出しは実行時の言語で埋め込んでから設定する。cobra.AddTemplateFuncで翻訳の関数を足すと
// プロセス全体で共有され、言語の違うコマンドを並列に組み立てるテストで混ざるため使わない。
//
// サブコマンドを束ねるコマンドは、cobraからは実行できるように見えるが自身では何もしないため、
// 使い方の行を `<command>` の形だけにする。
const usageTemplate = `%[1]s:{{if index .Annotations "%[12]s"}}
  {{.CommandPath}} <command>{{else if .Runnable}}
  {{.UseLine}}{{end}}{{if gt (len .Aliases) 0}}

%[2]s:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%[3]s:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%[4]s:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%[5]s:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%[6]s:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{with index .Annotations "%[7]s"}}

%[8]s:
{{.}}{{end}}{{with index .Annotations "%[9]s"}}

%[10]s:
{{.}}{{end}}{{if .HasAvailableSubCommands}}

%[11]s{{end}}
`

func newUsageTemplate(loc *i18n.Localizer) string {
	return fmt.Sprintf(
		usageTemplate,
		loc.T("help_heading_usage"),
		loc.T("help_heading_aliases"),
		loc.T("help_heading_examples"),
		loc.T("help_heading_commands"),
		loc.T("help_heading_flags"),
		loc.T("help_heading_global_flags"),
		cmdutil.AnnotationOutput,
		loc.T("help_heading_output"),
		cmdutil.AnnotationErrors,
		loc.T("help_heading_errors"),
		loc.T("help_more_information", map[string]any{"Command": "{{.CommandPath}} <command>"}),
		cmdutil.AnnotationGroup,
	)
}

// newHelpFuncは、サブコマンドを束ねるコマンドに未知のサブコマンドと `--help` が渡されたときに、
// ヘルプを出さず使い方の誤りとして失敗させるヘルプの関数を返す。
// `--json`・`--fields`・`--jq` が付いていれば、ヘルプをJSONで出す。
//
// cobraは `--help` を引数の検証より先に処理するため、`wikino unknown --help` はRunEに届かず、
// 親のコマンドのヘルプを出して成功してしまう。ヘルプの関数はエラーを返せないため、
// 誤りをsetErrで呼び出し側に渡す (gh CLIのルートのヘルプの関数と同じ方式)。
func newHelpFunc(f *cmdutil.Factory, defaultHelp func(*cobra.Command, []string), setErr func(error)) func(*cobra.Command, []string) {
	loc := f.Localizer

	return func(cmd *cobra.Command, args []string) {
		if cmdutil.IsGroup(cmd) {
			if rest := cmd.Flags().Args(); len(rest) > 0 {
				setErr(cmdutil.NewUnknownSubcommandError(loc, cmd, rest[0]))

				return
			}
		}

		if isHelpJSONRequested(cmd.Flags()) {
			if err := printHelpJSON(f, cmd); err != nil {
				setErr(err)
			}

			return
		}

		// 描画先をバッファにし、実際の出力先への書き込みの成否を呼び出し側へ返す。
		stdout := cmd.OutOrStdout()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		defaultHelp(cmd, args)
		cmd.SetOut(stdout)

		if _, err := buf.WriteTo(stdout); err != nil {
			setErr(err)
		}
	}
}

// isHelpJSONRequestedは、ヘルプをJSONで出すかを返す。
// 結果の出力と違い、標準出力がTTYでなくても人間向けのヘルプを既定にする。
// `--help` の出力は読む人のためのもので、エージェントも同じ文面を読めるようにするため。
// `--fields` と `--jq` は、結果の出力と同じくJSONを求めたものとして扱う。
func isHelpJSONRequested(fs *pflag.FlagSet) bool {
	return output.IsJSONRequested(fs) || fs.Changed(output.FlagFields) || fs.Changed(output.FlagJQ)
}

// printHelpJSONは、cmdのヘルプをJSONで出す。`--fields` と `--jq` は結果の出力と同じく適用する。
func printHelpJSON(f *cmdutil.Factory, cmd *cobra.Command) error {
	printer, err := output.NewPrinter(f.IOStreams, f.Localizer, cmd.Flags(), output.Spec{Fields: helpJSONFields})
	if err != nil {
		return err
	}

	return printer.PrintItem(newHelpJSON(cmd))
}
