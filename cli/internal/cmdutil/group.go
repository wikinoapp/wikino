package cmdutil

import (
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

// ヘルプの節に使うcobraのAnnotationsのキー。
// 値は翻訳済みの文言で、ヘルプのテンプレートが「出力」「エラー」の節として表示する。
const (
	AnnotationOutput = "wikino_help_output"
	AnnotationErrors = "wikino_help_errors"
)

// AnnotationGroupは、サブコマンドを束ねるだけのコマンドであることを表すAnnotationsのキー。
const AnnotationGroup = "wikino_group"

// ConfigureGroupは、cmdをサブコマンドを束ねるだけのコマンドとして設定する。
//
// cobraは、実行する処理を持たないコマンドが呼ばれるとヘルプを出して成功 (終了コード0) にし、
// ルート以外のコマンドでは未知のサブコマンドも引数として黙って受け取る。
// エージェントがこれを成功と取り違えないよう、サブコマンドが無いときと未知のときを使い方の誤りにする。
func ConfigureGroup(cmd *cobra.Command, loc *i18n.Localizer) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[AnnotationGroup] = "true"

	// 引数の検証をcobraに任せず、RunEで未知のサブコマンドとして扱う。
	// cobraの検証はルートのコマンドでだけ未知のサブコマンドを英語のエラーにするため。
	cmd.Args = cobra.ArbitraryArgs
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return NewUnknownSubcommandError(loc, cmd, args[0])
		}

		return newSubcommandRequiredError(loc, cmd)
	}
}

// IsGroupは、cmdがConfigureGroupで設定したコマンドかを返す。
func IsGroup(cmd *cobra.Command) bool {
	return cmd.Annotations[AnnotationGroup] == "true"
}
