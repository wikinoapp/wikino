// Package pageは、ページを扱う `wikino page` のコマンドを提供する。
package page

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// NewCmdPageは `wikino page` を返す。
func NewCmdPage(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "page",
		Short:   loc.T("page_short"),
		Long:    loc.T("page_long"),
		Example: loc.T("page_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("page_output"),
			cmdutil.AnnotationErrors: loc.T("page_errors"),
		},
	}
	cmdutil.ConfigureGroup(cmd, loc)

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdGet(f))
	cmd.AddCommand(newCmdSearch(f))
	cmd.AddCommand(newCmdLinks(f))
	cmd.AddCommand(newCmdBacklinks(f))
	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdUpdate(f))

	return cmd
}

// pageFieldsは、APIのページの項目。
var pageFields = []string{"id", "number", "title", "status", "topic_number", "topic_name", "modified_at", "url", "body"}

func formatNumber(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}

// titleLabelは、人間向けの表示で使うタイトルを返す。タイトルの無いページは、空欄と区別できる表示にする。
func titleLabel(loc *i18n.Localizer, title *string) string {
	if title == nil {
		return loc.T("page_value_untitled")
	}

	return *title
}

// statusLabelは、人間向けの表示で使う公開状態の名前を返す。
func statusLabel(loc *i18n.Localizer, status apigen.PageStatus) string {
	switch status {
	case apigen.PageStatusPublished:
		return loc.T("page_value_published")
	case apigen.PageStatusUnpublished:
		return loc.T("page_value_unpublished")
	default:
		return string(status)
	}
}

// formatTimeは、人間向けの表示で使う日時を返す。端末の利用者が読みやすいよう秒までにする。
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// printWrittenは、作成・更新したページを出す。JSONでは `page get` と同じ項目を出す。
// 人間向けの表示では、書き込んだ本文を繰り返さずに項目だけを出し、完了の行 (done) を標準エラーに出す。
// 書き込みで作られた見込みのリンク先 (linksの `new_page`) は、JSONでも標準エラーに出す。
func printWritten(f *cmdutil.Factory, printer *output.Printer, item getItem, done string, links []apigen.WikilinkPreviewLink) error {
	if printer.IsJSON() {
		if err := printer.PrintItem(item); err != nil {
			return err
		}
	} else {
		if err := printer.PrintDetail(detailFields(f.Localizer, item), ""); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(f.IOStreams.ErrOut, done); err != nil {
			return err
		}
	}

	return cmdutil.PrintCreatedWikilinks(f.IOStreams, f.Localizer, links)
}
