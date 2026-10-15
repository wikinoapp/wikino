package page

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
)

func newCmdBacklinks(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "backlinks <page_number>",
		Short:   loc.T("page_backlinks_short"),
		Long:    loc.T("page_backlinks_long"),
		Example: loc.T("page_backlinks_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("page_backlinks_output"),
			cmdutil.AnnotationErrors: loc.T("page_backlinks_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
	}
	pageFlags := cmdutil.AddListFlags(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runLinkList(f, cmd, args[0], pageFlags, loc.T("page_backlinks_empty_hint"),
			func(ctx context.Context, client *apiclient.Client, space string, number int32, cursor *string, limit *int32) (*apigen.PageList, int, error) {
				resp, err := client.ListPageBacklinksWithResponse(ctx, space, number, &apigen.ListPageBacklinksParams{Cursor: cursor, Limit: limit})
				if err != nil {
					return nil, 0, err
				}
				return resp.JSON200, resp.StatusCode(), nil
			})
	}

	return cmd
}
