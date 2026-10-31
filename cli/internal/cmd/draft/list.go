package draft

import (
	"context"
	"slices"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// listFieldsは、下書きの一覧の `--fields` で選べる項目。下書きの項目に、本文の冒頭の抜粋 (excerpt) を足す。
var listFields = append(slices.Clone(draftFields), "excerpt")

// listDefaultFieldsは、下書きの一覧で既定で出す項目。本文 (body) は `--fields` で選んだときだけ出す。
// エージェントが一覧を見るたびに、各下書きの本文がコンテキストに入るのを避けるためである。
var listDefaultFields = slices.DeleteFunc(slices.Clone(listFields), func(name string) bool { return name == "body" })

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "list",
		Short:   loc.T("draft_list_short"),
		Long:    loc.T("draft_list_long"),
		Example: loc.T("draft_list_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_list_output"),
			cmdutil.AnnotationErrors: loc.T("draft_list_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 0),
	}
	pageFlags := cmdutil.AddListFlags(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runList(f, cmd, pageFlags)
	}

	return cmd
}

func runList(f *cmdutil.Factory, cmd *cobra.Command, pageFlags *cmdutil.ListFlags) error {
	loc := f.Localizer

	if err := pageFlags.Validate(loc, cmd); err != nil {
		return err
	}

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: listFields, Default: listDefaultFields})
	if err != nil {
		return err
	}

	client, space, err := f.APIClient(cmd)
	if err != nil {
		return err
	}

	include := cmdutil.ListInclude(printer)

	return cmdutil.RunList(cmd.Context(), f, printer, pageFlags, cmdutil.List[apigen.DraftPageListItem]{
		Fetch: func(ctx context.Context, cursor *string) ([]apigen.DraftPageListItem, *string, error) {
			resp, err := client.ListDraftPagesWithResponse(ctx, space, &apigen.ListDraftPagesParams{
				Include: include,
				Cursor:  cursor,
				Limit:   pageFlags.LimitParam(),
			})
			if err != nil {
				return nil, nil, err
			}
			if resp.JSON200 == nil {
				return nil, nil, cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
			}
			return resp.JSON200.Items, resp.JSON200.NextCursor, nil
		},
		Item: func(d apigen.DraftPageListItem) any { return d },
		Table: func(drafts []apigen.DraftPageListItem) error {
			return printTable(printer, loc, drafts)
		},
		EmptyHint: loc.T("draft_list_empty_hint"),
	})
}

func printTable(printer *output.Printer, loc *i18n.Localizer, drafts []apigen.DraftPageListItem) error {
	rows := make([][]string, 0, len(drafts))
	for _, d := range drafts {
		rows = append(rows, []string{
			formatNumber(d.PageNumber),
			titleLabel(loc, d.Title),
			d.TopicName,
			formatNumber(d.TopicNumber),
			formatTime(d.ModifiedAt),
		})
	}

	return printer.PrintTable([]string{
		loc.T("draft_label_page_number"),
		loc.T("draft_label_title"),
		loc.T("draft_label_topic_name"),
		loc.T("draft_label_topic_number"),
		loc.T("draft_label_modified_at"),
	}, rows)
}
