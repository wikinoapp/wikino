package topic

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "list",
		Short:   loc.T("topic_list_short"),
		Long:    loc.T("topic_list_long"),
		Example: loc.T("topic_list_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("topic_list_output"),
			cmdutil.AnnotationErrors: loc.T("topic_list_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 0),
	}
	listFlags := cmdutil.AddListFlags(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runList(f, cmd, listFlags)
	}

	return cmd
}

func runList(f *cmdutil.Factory, cmd *cobra.Command, listFlags *cmdutil.ListFlags) error {
	loc := f.Localizer

	if err := listFlags.Validate(loc, cmd); err != nil {
		return err
	}
	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: topicFields})
	if err != nil {
		return err
	}

	client, space, err := f.APIClient(cmd)
	if err != nil {
		return err
	}

	return cmdutil.RunList(cmd.Context(), f, printer, listFlags, cmdutil.List[apigen.Topic]{
		Fetch: func(ctx context.Context, cursor *string) ([]apigen.Topic, *string, error) {
			resp, err := client.ListTopicsWithResponse(ctx, space, &apigen.ListTopicsParams{Cursor: cursor, Limit: listFlags.LimitParam()})
			if err != nil {
				return nil, nil, err
			}
			if resp.JSON200 == nil {
				return nil, nil, cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
			}
			return resp.JSON200.Items, resp.JSON200.NextCursor, nil
		},
		Item: func(t apigen.Topic) any { return t },
		Table: func(topics []apigen.Topic) error {
			rows := make([][]string, 0, len(topics))
			for _, t := range topics {
				rows = append(rows, []string{formatNumber(t.Number), t.Name, visibilityLabel(loc, t.Visibility)})
			}
			return printer.PrintTable([]string{loc.T("topic_label_number"), loc.T("topic_label_name"), loc.T("topic_label_visibility")}, rows)
		},
	})
}
