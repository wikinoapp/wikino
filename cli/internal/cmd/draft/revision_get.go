package draft

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/output"
	"github.com/wikinoapp/wikino/cli/internal/pagefile"
)

func newCmdRevisionGet(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "get <page_number> <revision_id>",
		Short:   loc.T("draft_revision_get_short"),
		Long:    loc.T("draft_revision_get_long"),
		Example: loc.T("draft_revision_get_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_revision_get_output"),
			cmdutil.AnnotationErrors: loc.T("draft_revision_get_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 2),
	}
	md := cmdutil.AddMarkdownFlag(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runRevisionGet(f, cmd, args[0], args[1], *md)
	}

	return cmd
}

func runRevisionGet(f *cmdutil.Factory, cmd *cobra.Command, numberArg, idArg string, md bool) error {
	loc := f.Localizer

	number, err := cmdutil.ParseNumber(loc, "<page_number>", numberArg)
	if err != nil {
		return err
	}
	id, err := cmdutil.ParseUUID(loc, "<revision_id>", idArg)
	if err != nil {
		return err
	}

	var printer *output.Printer
	if md {
		if err := cmdutil.ValidateMarkdownFlag(loc, cmd); err != nil {
			return err
		}
	} else if printer, err = output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: revisionFields}); err != nil {
		return err
	}

	client, space, err := f.APIClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.GetDraftPageRevisionWithResponse(cmd.Context(), space, number, id)
	if err != nil {
		listCommand := output.JoinCommand([]string{"wikino", "draft", "revision", "list", formatNumber(number)})
		return cmdutil.ConvertNotFound(loc, err, loc.T("draft_revision_get_error_not_found_hint", map[string]any{"Command": listCommand}))
	}
	if resp.JSON200 == nil {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}
	revision := resp.JSON200

	switch {
	case md:
		// リビジョンの応答にはタイトル・本文・IDと作成日時しか無く、版も返らないため、`wikino_title` だけを書く。
		// `wikino_etag` が無いため、このファイルを `--md-file` で書き戻そうとしても受け付けられない。
		// タイトルの無いリビジョンは書くキーが無く、空のfrontmatterを置くと本文の一部に見えるため、本文だけを出す。
		body := revision.Body
		if revision.Title != nil {
			body = pagefile.WithFrontmatter(body, pagefile.Metadata{Title: revision.Title})
		}
		_, err := io.WriteString(f.IOStreams.Out, body)
		return err
	case printer.IsJSON():
		return printer.PrintItem(revision)
	default:
		return printer.PrintDetail([]output.Field{
			{Label: loc.T("draft_label_revision_id"), Value: revision.Id.String()},
			{Label: loc.T("draft_label_title"), Value: titleLabel(loc, revision.Title)},
			{Label: loc.T("draft_label_created_at"), Value: formatTime(revision.CreatedAt)},
		}, revision.Body)
	}
}
