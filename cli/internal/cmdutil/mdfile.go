package cmdutil

import (
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
	"github.com/wikinoapp/wikino/cli/internal/pagefile"
)

// FlagMarkdownFileは、`--md` で書き出したページファイルを書き戻すフラグの名前。
const FlagMarkdownFile = "md-file"

// MarkdownFileFlagは、ページファイルを書き戻すフラグ (`--md-file`) の値。
// ページと下書きの更新で共通に使う。
type MarkdownFileFlag struct {
	cmd  *cobra.Command
	path string
}

// MarkdownFileは、書き戻すページファイルから読んだ内容。
type MarkdownFile struct {
	// Titleは `wikino_title` の値。nilならタイトルを変えない。
	Title *string
	// ETagは `wikino_etag` の値。`If-Match` に送る。
	ETag string
	// BodyはWikinoのキーを取り除いた本文。
	Body string
}

// AddMarkdownFileFlagは、`--md-file` をcmdに足す。
func AddMarkdownFileFlag(cmd *cobra.Command, loc *i18n.Localizer) *MarkdownFileFlag {
	m := &MarkdownFileFlag{cmd: cmd}
	cmd.Flags().StringVar(&m.path, FlagMarkdownFile, "", loc.T("help_md_file_flag"))

	return m
}

// Changedは、`--md-file` が指定されたかを返す。
func (m *MarkdownFileFlag) Changed() bool {
	return m.cmd.Flags().Changed(FlagMarkdownFile)
}

// Validateは、`--md-file` と同時に指定できないフラグが無いかを要求を送る前に検査する。
// ファイルが本文・タイトル・版のすべてを持つため、同じ値を渡す他のフラグとは併用させない
// (どちらかが黙って無視されるのを避ける)。`--force` も、ファイルの版を照合する方針と矛盾するため併用させない。
func (m *MarkdownFileFlag) Validate(loc *i18n.Localizer) error {
	if !m.Changed() {
		return nil
	}

	return validateExclusive(loc, m.cmd, FlagMarkdownFile, FlagBody, FlagBodyFile, FlagTitle, FlagIfMatch, FlagForce, FlagEdits)
}

// Readは、ページファイルを読み、Wikinoのキーの規則を確かめる。
// numberは書き戻す先のページ番号、getCommandはページファイルを書き出すコマンド (ヒントで案内する)。
//
//   - `wikino_number` がnumberと違えば、別のページのファイルとして送らない
//   - `wikino_etag` が無いファイルは、読んだ版を必須にする方針のため受け付けない
//   - `wikino_title` が無ければタイトルを変えない
func (m *MarkdownFileFlag) Read(f *Factory, number int32, getCommand string) (*MarkdownFile, error) {
	loc := f.Localizer
	hint := loc.T("usage_error_md_file_hint", map[string]any{"Command": getCommand})

	content, err := readTextInput(f, m.cmd, m.path)
	if err != nil {
		return nil, err
	}

	file, err := pagefile.Parse(content)
	if err != nil {
		var usageErr *output.Error
		var valueErr *pagefile.InvalidValueError
		switch {
		case errors.As(err, &valueErr):
			usageErr = NewUsageError(loc, m.cmd, loc.T("usage_error_md_file_invalid_value", map[string]any{"Key": valueErr.Key}))
		case errors.Is(err, pagefile.ErrUnremovableKeys):
			usageErr = NewUsageError(loc, m.cmd, loc.T("usage_error_md_file_unremovable_keys"))
		default:
			return nil, err
		}
		usageErr.Hint = hint
		return nil, usageErr
	}

	if file.Number != nil && *file.Number != number {
		usageErr := NewUsageError(loc, m.cmd, loc.T("usage_error_md_file_number_mismatch", map[string]any{
			"FileNumber": strconv.FormatInt(int64(*file.Number), 10),
			"Number":     strconv.FormatInt(int64(number), 10),
		}))
		usageErr.Hint = hint
		return nil, usageErr
	}
	if file.ETag == "" {
		usageErr := NewUsageError(loc, m.cmd, loc.T("usage_error_md_file_etag_required"))
		usageErr.Hint = hint
		return nil, usageErr
	}
	if err := ValidateETag(loc, m.cmd, "wikino_etag", file.ETag, hint); err != nil {
		return nil, err
	}

	return &MarkdownFile{Title: file.Title, ETag: file.ETag, Body: file.Body}, nil
}

// validateExclusiveは、flagとothersのうち、cmdにあって指定されたフラグを併用していないかを検査する。
func validateExclusive(loc *i18n.Localizer, cmd *cobra.Command, flag string, others ...string) error {
	for _, other := range others {
		if cmd.Flags().Lookup(other) != nil && cmd.Flags().Changed(other) {
			return NewUsageError(loc, cmd, loc.T("usage_error_flags_exclusive", map[string]any{"Flags": "--" + flag + ", --" + other}))
		}
	}

	return nil
}
