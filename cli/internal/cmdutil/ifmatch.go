package cmdutil

import (
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// 版の照合のフラグの名前。`--force` は版の照合を省く意味だけに使い、確認の省略 (`--yes`) と混ぜない。
const (
	FlagIfMatch = "if-match"
	FlagForce   = "force"
)

// IfMatchFlagsは、版の照合のフラグ (`--if-match` / `--force`) の値。
// ページ・下書きの更新や下書きの削除・公開で、利用者が読んだ版を前提に書き込むために共通に使う。
type IfMatchFlags struct {
	cmd   *cobra.Command
	value string
	force bool
}

// AddIfMatchFlagsは、`--if-match` と `--force` をcmdに足す。
func AddIfMatchFlags(cmd *cobra.Command, loc *i18n.Localizer) *IfMatchFlags {
	m := &IfMatchFlags{cmd: cmd}
	cmd.Flags().StringVar(&m.value, FlagIfMatch, "", loc.T("help_if_match_flag"))
	cmd.Flags().BoolVar(&m.force, FlagForce, false, loc.T("help_force_flag"))

	return m
}

// Validateは、版の照合のフラグを要求を送る前に検査する。
// どちらも無ければ、読まずに上書きする事故を防ぐため、getCommand (版を取得するコマンド) を案内して失敗させる。
func (m *IfMatchFlags) Validate(loc *i18n.Localizer, getCommand string) error {
	ifMatch := m.cmd.Flags().Changed(FlagIfMatch)

	switch {
	case ifMatch && m.force:
		return NewUsageError(loc, m.cmd, loc.T("usage_error_flags_exclusive", map[string]any{"Flags": "--" + FlagIfMatch + ", --" + FlagForce}))
	case m.force:
		return nil
	case !ifMatch:
		err := NewUsageError(loc, m.cmd, loc.T("usage_error_if_match_required"))
		err.Hint = ifMatchHint(loc, getCommand)
		return err
	default:
		return ValidateETag(loc, m.cmd, "--"+FlagIfMatch, m.value, ifMatchHint(loc, getCommand))
	}
}

// ValidateOptionalは、版の照合を任意にするコマンド (検索置換の `--edits`) で、`--if-match` を
// 指定したときだけ値を検査する。`--force` との排他はコマンドの側で検査する。
func (m *IfMatchFlags) ValidateOptional(loc *i18n.Localizer, getCommand string) error {
	if !m.cmd.Flags().Changed(FlagIfMatch) {
		return nil
	}

	return ValidateETag(loc, m.cmd, "--"+FlagIfMatch, m.value, ifMatchHint(loc, getCommand))
}

func ifMatchHint(loc *i18n.Localizer, getCommand string) string {
	return loc.T("usage_error_if_match_required_hint", map[string]any{"Command": getCommand})
}

// ValidateETagは、`If-Match` に送る版の値を検査する。nameは値を渡したフラグやキーの名前、hintは誤りのときのヒント。
// 空の値はAPIで428になり、制御文字はヘッダーの差し込みの経路になりうる。`*` は版を問わず一致し、
// 読んだ版の照合にならないため、最新の版で上書きするなら `--force` を使わせる。
func ValidateETag(loc *i18n.Localizer, cmd *cobra.Command, name, value, hint string) error {
	var err *output.Error
	switch {
	case strings.TrimSpace(value) == "" || strings.ContainsFunc(value, unicode.IsControl):
		err = NewUsageError(loc, cmd, loc.T("usage_error_if_match_invalid", map[string]any{"Name": name}))
	case strings.TrimSpace(value) == "*":
		err = NewUsageError(loc, cmd, loc.T("usage_error_if_match_wildcard", map[string]any{"Name": name}))
	default:
		return nil
	}
	err.Hint = hint

	return err
}

// Valueは、`--if-match` の値を返す。Validateを通した後に呼ぶ。
func (m *IfMatchFlags) Value() string {
	return m.value
}

// Forceは、`--force` が指定されたかを返す。trueなら、コマンドは最新の版を取得して `If-Match` に送る。
func (m *IfMatchFlags) Force() bool {
	return m.force
}
