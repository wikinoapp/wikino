// Package cmdutilは、複数のコマンドで共通に使うcobraの組み立ての部品を提供する。
package cmdutil

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// newUsageErrorは、使い方の誤り (未知のサブコマンド・フラグの誤りなど) を表すエラーを返す。
// 入口はこのエラーを終了コード2に対応させる。
// ヒントでは推測で補正したコマンドを示さず、`--help` を案内するだけにする。
func newUsageError(loc *i18n.Localizer, cmd *cobra.Command, message string) *output.Error {
	return &output.Error{
		Code:     output.CodeUsage,
		Message:  message,
		Hint:     loc.T("usage_error_hint", map[string]any{"Command": cmd.CommandPath()}),
		ExitCode: output.ExitUsage,
	}
}

// NewUnknownSubcommandErrorは、cmdにnameというサブコマンドが無いことを表すエラーを返す。
// cobraの「Did you mean」の候補は出さない。エージェントが候補を見て「ほぼ成功した」と取り違えうるため。
func NewUnknownSubcommandError(loc *i18n.Localizer, cmd *cobra.Command, name string) *output.Error {
	return newUsageError(loc, cmd, loc.T("usage_error_unknown_subcommand", map[string]any{"Name": name}))
}

// NewFlagErrorは、cmdのフラグの解釈に失敗したことを表す翻訳済みのエラーを返す。
func NewFlagError(loc *i18n.Localizer, cmd *cobra.Command, detail error) *output.Error {
	err := newUsageError(loc, cmd, flagErrorMessage(loc, detail))
	err.Err = detail

	return err
}

// pflagの型付きエラーから入力値だけを取り出し、説明文は翻訳を通す。
func flagErrorMessage(loc *i18n.Localizer, err error) string {
	var unknown *pflag.NotExistError
	var required *pflag.ValueRequiredError
	var invalid *pflag.InvalidValueError
	var syntax *pflag.InvalidSyntaxError

	switch {
	case errors.As(err, &unknown):
		// pflagの短縮フラグの組は、指定された組全体ではなく未知の短縮フラグから後ろの残りである。
		// 残りが未知の短縮フラグだけなら、同じ値を括弧で繰り返さない。
		name, shorts := unknown.GetSpecifiedName(), unknown.GetSpecifiedShortnames()
		switch shorts {
		case "":
			return loc.T("usage_error_unknown_flag", map[string]any{"Flag": "--" + name})
		case name:
			return loc.T("usage_error_unknown_flag", map[string]any{"Flag": "-" + name})
		default:
			return loc.T("usage_error_unknown_shorthand", map[string]any{"Flag": "-" + name, "Group": "-" + shorts})
		}
	case errors.As(err, &required):
		name := "--" + required.GetSpecifiedName()
		if required.GetSpecifiedShortnames() != "" {
			name = "-" + required.GetSpecifiedName()
		}
		return loc.T("usage_error_flag_value_required", map[string]any{"Flag": name})
	case errors.As(err, &invalid):
		return loc.T("usage_error_flag_invalid_value", map[string]any{
			"Flag":  "--" + invalid.GetFlag().Name,
			"Value": invalid.GetValue(), "Type": invalid.GetFlag().Value.Type(),
		})
	case errors.As(err, &syntax):
		return loc.T("usage_error_flag_invalid_syntax", map[string]any{"Flag": syntax.GetSpecifiedFlag()})
	default:
		// 未分類のエラーもライブラリの英語の説明をそのまま利用者に出さない。
		return loc.T("usage_error_invalid_flag")
	}
}

func newSubcommandRequiredError(loc *i18n.Localizer, cmd *cobra.Command) *output.Error {
	return newUsageError(loc, cmd, loc.T("usage_error_subcommand_required"))
}

// NewUsageErrorは、cmdの使い方の誤り (終了コード2) をmessageで返す。ヒントは `--help` の案内にする。
func NewUsageError(loc *i18n.Localizer, cmd *cobra.Command, message string) *output.Error {
	return newUsageError(loc, cmd, message)
}

// ExactArgsは、位置引数がちょうどn個であることを求める。cobra.ExactArgsのエラーは英語のため使わない。
func ExactArgs(loc *i18n.Localizer, n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return newUsageError(loc, cmd, loc.T("usage_error_arg_count", map[string]any{"Want": n, "Got": len(args)}))
		}

		return nil
	}
}

// MaximumNArgsは、位置引数がn個以下であることを求める。cobra.MaximumNArgsのエラーは英語のため使わない。
func MaximumNArgs(loc *i18n.Localizer, n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return newUsageError(loc, cmd, loc.T("usage_error_too_many_args", map[string]any{"Max": n, "Got": len(args)}))
		}

		return nil
	}
}
