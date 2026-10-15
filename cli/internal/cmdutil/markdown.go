package cmdutil

import (
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// FlagMarkdownは、1件をWikinoファイル形式のページファイルとして出すフラグの名前。
const FlagMarkdown = "md"

// AddMarkdownFlagは、`--md` をcmdに足す。
func AddMarkdownFlag(cmd *cobra.Command, loc *i18n.Localizer) *bool {
	var md bool
	cmd.Flags().BoolVar(&md, FlagMarkdown, false, loc.T("help_md_flag"))

	return &md
}

// ValidateMarkdownFlagは、`--md` とJSONの出力を選ぶフラグ (`--json`・`--fields`・`--jq`) が
// 同時に指定されていないかを検査する。`--md` の出力はJSONでないため、どちらかが黙って無視されるのを避ける。
func ValidateMarkdownFlag(loc *i18n.Localizer, cmd *cobra.Command) error {
	for _, name := range []string{output.FlagJSON, output.FlagFields, output.FlagJQ} {
		if cmd.Flags().Changed(name) {
			return NewUsageError(loc, cmd, loc.T("usage_error_flags_exclusive", map[string]any{"Flags": "--" + FlagMarkdown + ", --" + name}))
		}
	}

	return nil
}
