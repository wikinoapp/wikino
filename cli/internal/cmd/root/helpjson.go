package root

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
)

// helpJSONは `--help --json` で出すコマンドのヘルプ。
// 人間向けのヘルプと同じくcobraのコマンド定義だけを情報源にし、節ごとに項目を分ける。
// 項目の名前は公開の契約として扱い、1.0以降は追加だけを行う。
type helpJSON struct {
	// Commandはコマンドの完全な名前 (例: wikino page get)。
	Command string `json:"command"`
	// Usageは使い方の行。サブコマンドを束ねるコマンドは `<command>` の形にする。
	Usage          string           `json:"usage"`
	Short          string           `json:"short"`
	Long           string           `json:"long"`
	Args           []helpArg        `json:"args"`
	Flags          []helpFlag       `json:"flags"`
	InheritedFlags []helpFlag       `json:"inherited_flags"`
	Examples       string           `json:"examples"`
	Output         string           `json:"output"`
	Errors         string           `json:"errors"`
	Subcommands    []helpSubcommand `json:"subcommands"`
}

// helpArgは位置引数。
type helpArg struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	// Variadicは、同じ引数を複数並べられるか (例: `<query>...`)。
	Variadic bool `json:"variadic"`
}

// helpFlagはフラグ。
type helpFlag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand"`
	// Typeは値の型 (bool・string・int・stringSlice・stringArrayなど)。
	Type        string `json:"type"`
	Default     string `json:"default"`
	Description string `json:"description"`
}

// helpSubcommandはサブコマンドの名前と短い説明。
type helpSubcommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Short   string `json:"short"`
}

// helpJSONFieldsは、`--help --json` の `--fields` で選べる項目。
var helpJSONFields = []string{
	"command", "usage", "short", "long", "args", "flags", "inherited_flags",
	"examples", "output", "errors", "subcommands",
}

// newHelpJSONは、cmdのヘルプをJSONで出す形にする。
func newHelpJSON(cmd *cobra.Command) helpJSON {
	help := helpJSON{
		Command:        cmd.CommandPath(),
		Usage:          usageLine(cmd),
		Short:          cmd.Short,
		Long:           cmd.Long,
		Args:           parseArgs(cmd.Use),
		Flags:          helpFlags(cmd.LocalFlags()),
		InheritedFlags: helpFlags(cmd.InheritedFlags()),
		Examples:       cmd.Example,
		Output:         cmd.Annotations[cmdutil.AnnotationOutput],
		Errors:         cmd.Annotations[cmdutil.AnnotationErrors],
		Subcommands:    []helpSubcommand{},
	}

	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() {
			continue
		}
		help.Subcommands = append(help.Subcommands, helpSubcommand{
			Name:    sub.Name(),
			Command: sub.CommandPath(),
			Short:   sub.Short,
		})
	}

	return help
}

// usageLineは、人間向けのヘルプの「使い方」の行と同じ形を返す。
func usageLine(cmd *cobra.Command) string {
	if cmdutil.IsGroup(cmd) {
		return cmd.CommandPath() + " <command>"
	}

	return cmd.UseLine()
}

// parseArgsは、コマンドのUse (例: `get <page_number>`・`login [<space>]`・`search <query>...`) から位置引数を取り出す。
// cobraは位置引数の定義を持たないため、ヘルプに書いた使い方の行を情報源にする。
func parseArgs(use string) []helpArg {
	args := []helpArg{}

	fields := strings.Fields(use)
	if len(fields) == 0 {
		return args
	}

	for _, field := range fields[1:] {
		arg := helpArg{Required: true}

		if strings.HasSuffix(field, "...") {
			arg.Variadic = true
			field = strings.TrimSuffix(field, "...")
		}
		if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
			arg.Required = false
			field = strings.TrimSuffix(strings.TrimPrefix(field, "["), "]")
		}
		arg.Name = strings.TrimSuffix(strings.TrimPrefix(field, "<"), ">")

		args = append(args, arg)
	}

	return args
}

// helpFlagsは、fsのうちヘルプに出すフラグを名前の順に返す。
func helpFlags(fs *pflag.FlagSet) []helpFlag {
	flags := []helpFlag{}

	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}

		// 説明のバッククォートは人間向けのヘルプで値の名前に使う印のため、取り除いた説明を出す。
		_, usage := pflag.UnquoteUsage(f)
		flags = append(flags, helpFlag{
			Name:        f.Name,
			Shorthand:   f.Shorthand,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: usage,
		})
	})

	return flags
}
