package page

import (
	"io"
	"slices"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
	"github.com/wikinoapp/wikino/cli/internal/pagefile"
)

// getFieldsは、`page get` の `--fields` で選べる項目。APIのページの項目に、応答のETagヘッダーの値を足す。
var getFields = append(slices.Clone(pageFields), "etag")

// getItemは、`page get` の出力。
type getItem struct {
	apigen.Page
	// ETagは応答のETagヘッダーの値。更新のコマンドの `--if-match` に渡す。
	ETag string `json:"etag"`
}

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "get <page_number>",
		Short:   loc.T("page_get_short"),
		Long:    loc.T("page_get_long"),
		Example: loc.T("page_get_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("page_get_output"),
			cmdutil.AnnotationErrors: loc.T("page_get_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
	}
	md := cmdutil.AddMarkdownFlag(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runGet(f, cmd, args[0], *md)
	}

	return cmd
}

func runGet(f *cmdutil.Factory, cmd *cobra.Command, arg string, md bool) error {
	loc := f.Localizer

	number, err := cmdutil.ParseNumber(loc, "<page_number>", arg)
	if err != nil {
		return err
	}

	var printer *output.Printer
	if md {
		if err := cmdutil.ValidateMarkdownFlag(loc, cmd); err != nil {
			return err
		}
	} else if printer, err = output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: getFields}); err != nil {
		return err
	}

	client, space, err := f.APIClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.GetPageWithResponse(cmd.Context(), space, number, nil)
	if err != nil {
		return cmdutil.ConvertNotFound(loc, err, loc.T("page_error_not_found_hint"))
	}
	if resp.JSON200 == nil || resp.Headers200 == nil {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}
	item := getItem{Page: *resp.JSON200, ETag: resp.Headers200.ETag}

	switch {
	case md:
		_, err := io.WriteString(f.IOStreams.Out, pagefile.WithFrontmatter(item.Body, pagefile.Metadata{
			Title:       item.Title,
			Number:      &item.Number,
			ETag:        item.ETag,
			TopicNumber: &item.TopicNumber,
			Status:      string(item.Status),
			ModifiedAt:  &item.ModifiedAt,
			URL:         item.Url,
		}))
		return err
	case printer.IsJSON():
		return printer.PrintItem(item)
	default:
		return printer.PrintDetail(detailFields(loc, item), item.Body)
	}
}

// detailFieldsは、人間向けの表示で1件のページの本文の前に並べる項目を返す。
func detailFields(loc *i18n.Localizer, item getItem) []output.Field {
	return []output.Field{
		{Label: loc.T("page_label_number"), Value: formatNumber(item.Number)},
		{Label: loc.T("page_label_title"), Value: titleLabel(loc, item.Title)},
		{Label: loc.T("page_label_topic_number"), Value: formatNumber(item.TopicNumber)},
		{Label: loc.T("page_label_status"), Value: statusLabel(loc, item.Status)},
		{Label: loc.T("page_label_modified_at"), Value: formatTime(item.ModifiedAt)},
		{Label: loc.T("page_label_url"), Value: item.Url},
		{Label: loc.T("page_label_etag"), Value: item.ETag},
	}
}
