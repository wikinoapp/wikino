package cmdutil

import (
	"io"
	"os"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
)

// 本文を渡すフラグの名前。
const (
	FlagBody     = "body"
	FlagBodyFile = "body-file"
)

// FlagTitleは、タイトルを渡すフラグの名前。ページファイル・検索置換との排他の検査に使う。
const FlagTitle = "title"

// stdinPathは、`--body-file` で標準入力から読むことを表す値。
const stdinPath = "-"

// BodyFlagsは、本文を渡すフラグ (`--body` / `--body-file`) の値。
// ページと下書きの本文を書き込むコマンドで共通に使う。
type BodyFlags struct {
	cmd  *cobra.Command
	body string
	file string
}

// AddBodyFlagsは、`--body` と `--body-file` をcmdに足す。
func AddBodyFlags(cmd *cobra.Command, loc *i18n.Localizer) *BodyFlags {
	b := &BodyFlags{cmd: cmd}
	cmd.Flags().StringVar(&b.body, FlagBody, "", loc.T("help_body_flag"))
	cmd.Flags().StringVar(&b.file, FlagBodyFile, "", loc.T("help_body_file_flag"))

	return b
}

// Changedは、本文を渡すフラグのどちらかが指定されたかを返す。
func (b *BodyFlags) Changed() bool {
	return b.cmd.Flags().Changed(FlagBody) || b.cmd.Flags().Changed(FlagBodyFile)
}

// Validateは、本文を渡すフラグが同時に指定されていないかを検査する。要求を送る前に呼ぶ。
func (b *BodyFlags) Validate(loc *i18n.Localizer) error {
	if b.cmd.Flags().Changed(FlagBody) && b.cmd.Flags().Changed(FlagBodyFile) {
		return NewUsageError(loc, b.cmd, loc.T("usage_error_flags_exclusive", map[string]any{"Flags": "--" + FlagBody + ", --" + FlagBodyFile}))
	}

	return nil
}

// Readは、指定された本文を読む。どちらのフラグも無ければnilを返す。
// `--body-file -` は標準入力から読む。本文の一部として、末尾の改行は削らない。
func (b *BodyFlags) Read(f *Factory) (*string, error) {
	loc := f.Localizer

	var body string
	switch {
	case b.cmd.Flags().Changed(FlagBody):
		body = b.body
	case b.cmd.Flags().Changed(FlagBodyFile):
		var err error
		if body, err = readTextInput(f, b.cmd, b.file); err != nil {
			return nil, err
		}
	default:
		return nil, nil
	}

	if !utf8.ValidString(body) {
		return nil, NewUsageError(loc, b.cmd, loc.T("usage_error_body_invalid_utf8"))
	}

	return &body, nil
}

// readTextInputは、本文などを渡すフラグで指定されたファイル (pathが `-` なら標準入力) を読む。
// JSONに載せると不正なバイト列はU+FFFDに置き換わり、内容が黙って変わるため、UTF-8でなければ送る前に拒否する。
func readTextInput(f *Factory, cmd *cobra.Command, path string) (string, error) {
	loc := f.Localizer

	var data []byte
	var err error
	if path == stdinPath {
		data, err = io.ReadAll(f.IOStreams.In)
	} else {
		data, err = os.ReadFile(path) // #nosec G304 -- 利用者が入力として指定したファイルを読む
	}
	if err != nil {
		return "", NewUsageError(loc, cmd, loc.T("usage_error_input_read_file", map[string]any{
			"Path": path, "Detail": err.Error(),
		}))
	}
	if !utf8.Valid(data) {
		return "", NewUsageError(loc, cmd, loc.T("usage_error_input_invalid_utf8", map[string]any{"Path": path}))
	}

	return string(data), nil
}
