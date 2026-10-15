package page

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func newCmdLinks(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "links <page_number>",
		Short:   loc.T("page_links_short"),
		Long:    loc.T("page_links_long"),
		Example: loc.T("page_links_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("page_links_output"),
			cmdutil.AnnotationErrors: loc.T("page_links_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
	}
	pageFlags := cmdutil.AddListFlags(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runLinkList(f, cmd, args[0], pageFlags, loc.T("page_links_empty_hint"),
			func(ctx context.Context, client *apiclient.Client, space string, number int32, cursor *string, limit *int32) (*apigen.PageList, int, error) {
				resp, err := client.ListPageLinksWithResponse(ctx, space, number, &apigen.ListPageLinksParams{Cursor: cursor, Limit: limit})
				if err != nil {
					return nil, 0, err
				}
				return resp.JSON200, resp.StatusCode(), nil
			})
	}

	return cmd
}

// fetchLinkListは、ページを起点にした一覧 (リンク先・バックリンク) の1ページを取得する。
type fetchLinkList func(ctx context.Context, client *apiclient.Client, space string, number int32, cursor *string, limit *int32) (*apigen.PageList, int, error)

// runLinkListは、`page links` と `page backlinks` の共通の処理。
// 2つは呼ぶAPIだけが違い、引数・ページ送り・出力の項目を `page list` と揃える。
func runLinkList(f *cmdutil.Factory, cmd *cobra.Command, arg string, pageFlags *cmdutil.ListFlags, emptyHint string, fetch fetchLinkList) error {
	loc := f.Localizer

	number, err := cmdutil.ParseNumber(loc, "<page_number>", arg)
	if err != nil {
		return err
	}
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

	limit := pageFlags.LimitParam()

	return cmdutil.RunList(cmd.Context(), f, printer, pageFlags, cmdutil.List[apigen.Page]{
		Fetch: func(ctx context.Context, cursor *string) ([]apigen.Page, *string, error) {
			list, status, err := fetch(ctx, client, space, number, cursor, limit)
			if err != nil {
				return nil, nil, err
			}
			if list == nil {
				return nil, nil, cmdutil.UnexpectedResponseError(loc, status)
			}
			return list.Items, list.NextCursor, nil
		},
		Item: func(p apigen.Page) any { return newListItem(p) },
		Table: func(pages []apigen.Page) error {
			return printTable(printer, loc, pages)
		},
		EmptyHint:    emptyHint,
		NotFoundHint: loc.T("page_error_not_found_hint"),
	})
}
