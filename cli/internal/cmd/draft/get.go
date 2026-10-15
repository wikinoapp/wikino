package draft

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/output"
	"github.com/wikinoapp/wikino/cli/internal/pagefile"
)

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "get <page_number>",
		Short:   loc.T("draft_get_short"),
		Long:    loc.T("draft_get_long"),
		Example: loc.T("draft_get_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_get_output"),
			cmdutil.AnnotationErrors: loc.T("draft_get_errors"),
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

	resp, err := client.GetDraftPageWithResponse(cmd.Context(), space, number, nil)
	if err != nil {
		return cmdutil.ConvertNotFound(loc, err, loc.T("draft_error_not_found_hint"))
	}
	if resp.JSON200 == nil || resp.Headers200 == nil {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}
	item := getItem{DraftPage: *resp.JSON200, ETag: resp.Headers200.ETag}

	switch {
	case md:
		// 下書きには公開状態が無いため、`wikino_status` は書かない。
		_, err := io.WriteString(f.IOStreams.Out, pagefile.WithFrontmatter(item.Body, pagefile.Metadata{
			Title:       item.Title,
			Number:      &item.PageNumber,
			ETag:        item.ETag,
			TopicNumber: &item.TopicNumber,
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
