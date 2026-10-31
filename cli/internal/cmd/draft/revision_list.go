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

// revisionListFieldsは、編集履歴の一覧の `--fields` で選べる項目。リビジョンの項目に、本文の冒頭の抜粋 (excerpt) を足す。
var revisionListFields = append(slices.Clone(revisionFields), "excerpt")

// revisionListDefaultFieldsは、編集履歴の一覧で既定で出す項目。本文 (body) は `--fields` で選んだときだけ出す。
var revisionListDefaultFields = slices.DeleteFunc(slices.Clone(revisionListFields), func(name string) bool { return name == "body" })

func newCmdRevisionList(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "list <page_number>",
		Short:   loc.T("draft_revision_list_short"),
		Long:    loc.T("draft_revision_list_long"),
		Example: loc.T("draft_revision_list_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_revision_list_output"),
			cmdutil.AnnotationErrors: loc.T("draft_revision_list_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
	}
	pageFlags := cmdutil.AddListFlags(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runRevisionList(f, cmd, pageFlags, args[0])
	}

	return cmd
}

func runRevisionList(f *cmdutil.Factory, cmd *cobra.Command, pageFlags *cmdutil.ListFlags, arg string) error {
	loc := f.Localizer

	number, err := cmdutil.ParseNumber(loc, "<page_number>", arg)
	if err != nil {
		return err
	}
	if err := pageFlags.Validate(loc, cmd); err != nil {
		return err
	}

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: revisionListFields, Default: revisionListDefaultFields})
	if err != nil {
		return err
	}

	client, space, err := f.APIClient(cmd)
	if err != nil {
		return err
	}

	include := cmdutil.ListInclude(printer)

	return cmdutil.RunList(cmd.Context(), f, printer, pageFlags, cmdutil.List[apigen.DraftPageRevisionListItem]{
		Fetch: func(ctx context.Context, cursor *string) ([]apigen.DraftPageRevisionListItem, *string, error) {
			resp, err := client.ListDraftPageRevisionsWithResponse(ctx, space, number, &apigen.ListDraftPageRevisionsParams{
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
		Item: func(r apigen.DraftPageRevisionListItem) any { return r },
		Table: func(revisions []apigen.DraftPageRevisionListItem) error {
			return printRevisionTable(printer, loc, revisions)
		},
		EmptyHint:    loc.T("draft_revision_list_empty_hint"),
		NotFoundHint: loc.T("draft_error_not_found_hint"),
	})
}

func printRevisionTable(printer *output.Printer, loc *i18n.Localizer, revisions []apigen.DraftPageRevisionListItem) error {
	rows := make([][]string, 0, len(revisions))
	for _, r := range revisions {
		rows = append(rows, []string{
			r.Id.String(),
			titleLabel(loc, r.Title),
			formatTime(r.CreatedAt),
		})
	}

	return printer.PrintTable([]string{
		loc.T("draft_label_revision_id"),
		loc.T("draft_label_title"),
		loc.T("draft_label_created_at"),
	}, rows)
}
