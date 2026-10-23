package cmdutil

import (
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

// FlagYesは、確認を省いて実行するフラグの名前。版の照合を省く `--force` とは混ぜない。
const FlagYes = "yes"

// AddYesFlagは、`--yes` をcmdに足す。確認が要る操作を行うコマンドにだけ足す。
func AddYesFlag(cmd *cobra.Command, loc *i18n.Localizer) *bool {
	return cmd.Flags().Bool(FlagYes, false, loc.T("help_yes_flag"))
}

// TargetFlagArgsは、cmdで指定された接続先・スペース・メンバーのフラグを、コマンドの引数の形で返す。
// 確認プロトコルの承認後のコマンド (confirm_command) を、確認したときと同じスペースの同じメンバーで
// 実行させるために使う。これらの値は秘密でなく、トークンは引数に含めない。
func TargetFlagArgs(cmd *cobra.Command) []string {
	var args []string
	for _, name := range []string{config.FlagHost, config.FlagSpace, config.FlagMember} {
		if cmd.Flags().Changed(name) {
			args = append(args, "--"+name, cmd.Flags().Lookup(name).Value.String())
		}
	}

	return args
}
