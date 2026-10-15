// Package draftは、ページの下書きを扱う `wikino draft` のコマンドを提供する。
package draft

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// NewCmdDraftは `wikino draft` を返す。
func NewCmdDraft(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "draft",
		Short:   loc.T("draft_short"),
		Long:    loc.T("draft_long"),
		Example: loc.T("draft_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_output"),
			cmdutil.AnnotationErrors: loc.T("draft_errors"),
		},
	}
	cmdutil.ConfigureGroup(cmd, loc)

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdGet(f))
	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdUpdate(f))
	cmd.AddCommand(newCmdDelete(f))
	cmd.AddCommand(newCmdPublish(f))
	cmd.AddCommand(newCmdRevision(f))

	return cmd
}

// draftFieldsは、APIの下書きの項目。
var draftFields = []string{"page_number", "title", "topic_number", "modified_at", "url", "body"}

// getFieldsは、1件の下書きを出すコマンドの `--fields` で選べる項目。APIの下書きの項目に、応答のETagヘッダーの値を足す。
var getFields = append(slices.Clone(draftFields), "etag")

// getItemは、1件の下書きの出力。
type getItem struct {
	apigen.DraftPage
	// ETagは応答のETagヘッダーの値。下書きを更新するコマンドの `--if-match` に渡す。
	ETag string `json:"etag"`
}

func formatNumber(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}

// titleLabelは、人間向けの表示で使うタイトルを返す。タイトルの無い下書きは、空欄と区別できる表示にする。
func titleLabel(loc *i18n.Localizer, title *string) string {
	if title == nil {
		return loc.T("draft_value_untitled")
	}

	return *title
}

// formatTimeは、人間向けの表示で使う日時を返す。端末の利用者が読みやすいよう秒までにする。
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// detailFieldsは、人間向けの表示で1件の下書きの本文の前に並べる項目を返す。
func detailFields(loc *i18n.Localizer, item getItem) []output.Field {
	return []output.Field{
		{Label: loc.T("draft_label_page_number"), Value: formatNumber(item.PageNumber)},
		{Label: loc.T("draft_label_title"), Value: titleLabel(loc, item.Title)},
		{Label: loc.T("draft_label_topic_number"), Value: formatNumber(item.TopicNumber)},
		{Label: loc.T("draft_label_modified_at"), Value: formatTime(item.ModifiedAt)},
		{Label: loc.T("draft_label_url"), Value: item.Url},
		{Label: loc.T("draft_label_etag"), Value: item.ETag},
	}
}

// printWrittenは、作成した下書きを出す。JSONでは `draft get` と同じ項目を出す。
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
