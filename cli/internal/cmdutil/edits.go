package cmdutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// FlagEditsは、本文の検索置換の編集を渡すフラグの名前。
const FlagEdits = "edits"

// EditsFlagは、本文の検索置換の編集を渡すフラグ (`--edits`) の値。
// ページと下書きの本文の検索置換で共通に使う。
type EditsFlag struct {
	cmd  *cobra.Command
	path string
}

// AddEditsFlagは、`--edits` をcmdに足す。
func AddEditsFlag(cmd *cobra.Command, loc *i18n.Localizer) *EditsFlag {
	e := &EditsFlag{cmd: cmd}
	cmd.Flags().StringVar(&e.path, FlagEdits, "", loc.T("help_edits_flag"))

	return e
}

// Changedは、`--edits` が指定されたかを返す。
func (e *EditsFlag) Changed() bool {
	return e.cmd.Flags().Changed(FlagEdits)
}

// Validateは、`--edits` と同時に指定できないフラグが無いかを要求を送る前に検査する。
// 検索置換は本文の一部だけを変え、タイトルを変えられないため、本文の全体やタイトルを渡すフラグとは併用させない。
// `--force` は、`--if-match` を省けば版を照合しないため意味を持たず、併用させない。
func (e *EditsFlag) Validate(loc *i18n.Localizer) error {
	if !e.Changed() {
		return nil
	}

	return validateExclusive(loc, e.cmd, FlagEdits, FlagBody, FlagBodyFile, FlagTitle, FlagMarkdownFile, FlagForce)
}

// editsInputは、`--edits` で読むJSON。公開APIの要求の本文と同じ形にする。
// 項目の書き間違い (`old_string` など) や欠落で、空の `new_str` として一致した箇所を消す事故を防ぐため、
// 必須の項目はポインターで受けて有無を確かめる。
type editsInput struct {
	Edits *[]editInput `json:"edits"`
}

type editInput struct {
	OldStr     *string `json:"old_str"`
	NewStr     *string `json:"new_str"`
	ReplaceAll *bool   `json:"replace_all,omitempty"`
}

// Readは、`--edits` のファイル (`-` なら標準入力) から編集の列を読む。
// 未知の項目・必須の項目の欠落・空の `old_str`・空の編集の列は、要求を送る前に使い方の誤りにする。
func (e *EditsFlag) Read(f *Factory) ([]apigen.BodyEdit, error) {
	loc := f.Localizer

	content, err := readTextInput(f, e.cmd, e.path)
	if err != nil {
		return nil, err
	}

	invalid := func(detail string) error {
		usageErr := NewUsageError(loc, e.cmd, loc.T("usage_error_edits_invalid", map[string]any{"Detail": detail}))
		usageErr.Hint = loc.T("usage_error_edits_invalid_hint")
		return usageErr
	}

	decoder := json.NewDecoder(bytes.NewReader([]byte(content)))
	decoder.DisallowUnknownFields()
	var input editsInput
	if err := decoder.Decode(&input); err != nil {
		return nil, invalid(err.Error())
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, invalid(loc.T("usage_error_edits_trailing_data"))
	}
	if input.Edits == nil || len(*input.Edits) == 0 {
		return nil, invalid(loc.T("usage_error_edits_empty"))
	}

	edits := make([]apigen.BodyEdit, 0, len(*input.Edits))
	for i, edit := range *input.Edits {
		position := map[string]any{"Index": strconv.Itoa(i)}
		switch {
		case edit.OldStr == nil:
			return nil, invalid(loc.T("usage_error_edits_missing", map[string]any{"Index": strconv.Itoa(i), "Key": "old_str"}))
		case edit.NewStr == nil:
			return nil, invalid(loc.T("usage_error_edits_missing", map[string]any{"Index": strconv.Itoa(i), "Key": "new_str"}))
		case *edit.OldStr == "":
			return nil, invalid(loc.T("usage_error_edits_empty_old_str", position))
		}
		edits = append(edits, apigen.BodyEdit{OldStr: *edit.OldStr, NewStr: *edit.NewStr, ReplaceAll: edit.ReplaceAll})
	}

	return edits, nil
}

// editPointerは、検索置換の422の `errors` の `pointer` (例: `#/edits/2/old_str`) から、編集の位置と項目を読む。
var editPointer = regexp.MustCompile(`^#?/edits/(\d+)/(old_str|new_str)$`)

// ConvertEditsErrorは、検索置換のAPIのエラーをCLIのエラーにする。hintsはConvertWithHintsと同じく
// エラーのコードごとのヒント。422で適用できなかった編集が分かれば、何番目の編集かを示すヒントにする
// (一致の件数はAPIのメッセージに含まれる)。getCommandは今の本文を確かめるコマンド。
func ConvertEditsError(loc *i18n.Localizer, err error, hints map[string]string, getCommand string) error {
	var apiErr *apiclient.Error
	isAPIError := errors.As(err, &apiErr)

	err = ConvertWithHints(loc, err, hints)
	var cliErr *output.Error
	if !isAPIError || !errors.As(err, &cliErr) || cliErr.Code != output.CodeValidationFailed {
		return err
	}

	for _, fe := range apiErr.FieldErrors {
		match := editPointer.FindStringSubmatch(fe.Pointer)
		if match == nil {
			continue
		}
		index, convErr := strconv.Atoi(match[1])
		if convErr != nil {
			continue
		}
		params := map[string]any{
			"Ordinal": strconv.Itoa(index + 1),
			"Index":   match[1],
			"Command": getCommand,
		}
		if match[2] == "old_str" {
			cliErr.Hint = loc.T("error_edits_old_str_hint", params)
		} else {
			cliErr.Hint = loc.T("error_edits_new_str_hint", params)
		}
		break
	}

	return err
}
