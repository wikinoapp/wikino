// Package rootは `wikino` のルートコマンドを組み立てて実行する。
package root

import (
	"fmt"

	"github.com/spf13/cobra"

	apicmd "github.com/wikinoapp/wikino/cli/internal/cmd/api"
	attachmentcmd "github.com/wikinoapp/wikino/cli/internal/cmd/attachment"
	authcmd "github.com/wikinoapp/wikino/cli/internal/cmd/auth"
	draftcmd "github.com/wikinoapp/wikino/cli/internal/cmd/draft"
	pagecmd "github.com/wikinoapp/wikino/cli/internal/cmd/page"
	skillcmd "github.com/wikinoapp/wikino/cli/internal/cmd/skill"
	topiccmd "github.com/wikinoapp/wikino/cli/internal/cmd/topic"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// Executeは、argsでルートコマンドを実行し、失敗したときはその理由を返す。
// あわせて、実行したコマンドで `--json` が指定されたかを返す。
// エラーの表示と終了コードの決定は呼び出し側 (cmd/wikino) が行い、`--json` ならエラーもJSONで出す。
func Execute(args []string, f *cmdutil.Factory) (jsonRequested bool, err error) {
	var helpErr error

	ios := f.IOStreams
	cmd := newCmdRoot(f, func(err error) { helpErr = err })
	cmd.SetArgs(args)
	cmd.SetIn(ios.In)
	cmd.SetOut(ios.Out)
	cmd.SetErr(ios.ErrOut)

	executed, err := cmd.ExecuteC()
	if executed == nil {
		executed = cmd
	}
	// フラグの解析に失敗したときも、失敗した箇所より前の `--json` は解析済みのため反映される。
	jsonRequested = output.IsJSONRequested(executed.Flags())

	if err != nil {
		return jsonRequested, err
	}

	return jsonRequested, helpErr
}

func newCmdRoot(f *cmdutil.Factory, setHelpErr func(error)) *cobra.Command {
	loc, version := f.Localizer, f.Version
	cmd := &cobra.Command{
		Use:     "wikino",
		Short:   loc.T("root_short"),
		Long:    loc.T("root_long"),
		Example: loc.T("root_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("root_output"),
			cmdutil.AnnotationErrors: loc.T("root_errors", map[string]any{"ExitCodes": output.ExitCodeTable(loc)}),
		},

		// エラーと使い方はcobraに出させず、入口でまとめて出す。
		// cobraはエラーのたびに英語の使い方を標準エラーに出し、エージェントの出力を埋めるため。
		SilenceErrors: true,
		SilenceUsage:  true,

		// 未知のサブコマンドを推測で補正しない。
		DisableSuggestions: true,
		// 補完のスクリプトを出すcompletionコマンドは、コマンドとフラグを公開の契約として扱うため、
		// 必要になってから翻訳とあわせて足す。
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	cmdutil.ConfigureGroup(cmd, loc)

	// フラグの説明も表示言語に揃えるため、翻訳した文言で定義する。
	cmd.PersistentFlags().BoolP("help", "h", false, loc.T("help_flag"))
	output.AddFlags(cmd.PersistentFlags(), loc)
	config.AddFlags(cmd.PersistentFlags(), loc)
	var showVersion bool
	cmd.Flags().BoolVar(&showVersion, "version", false, loc.T("root_version_flag"))

	// バージョン表示でも未知のサブコマンドを拒否するため、解析済みの引数を先に検査する。
	groupRun := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 || !showVersion {
			return groupRun(cmd, args)
		}

		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s version %s\n", cmd.Name(), version)
		return err
	}

	cmd.SetUsageTemplate(newUsageTemplate(loc))
	cmd.SetHelpFunc(newHelpFunc(f, cmd.HelpFunc(), setHelpErr))
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return cmdutil.NewFlagError(loc, cmd, err)
	})

	// cobraのhelpサブコマンド (`wikino help <command>`) は、未知のコマンドを渡してもヘルプを出して
	// 成功するため使わない。同じ名前の隠しコマンドで、未知のサブコマンドと同じ誤りにする。
	cmd.SetHelpCommand(&cobra.Command{
		Use:    "help",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(helpCmd *cobra.Command, _ []string) error {
			return cmdutil.NewUnknownSubcommandError(loc, cmd, helpCmd.Name())
		},
	})

	cmd.AddCommand(authcmd.NewCmdAuth(f))
	cmd.AddCommand(topiccmd.NewCmdTopic(f))
	cmd.AddCommand(pagecmd.NewCmdPage(f))
	cmd.AddCommand(draftcmd.NewCmdDraft(f))
	cmd.AddCommand(attachmentcmd.NewCmdAttachment(f))
	cmd.AddCommand(apicmd.NewCmdAPI(f))
	cmd.AddCommand(skillcmd.NewCmdSkill(f))

	return cmd
}
