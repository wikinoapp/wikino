package page

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// searchFlagsは `page search` の絞り込みのフラグの値。
type searchFlags struct {
	topic  string
	status string
}

func newCmdSearch(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "search <query>...",
		Short:   loc.T("page_search_short"),
		Long:    loc.T("page_search_long"),
		Example: loc.T("page_search_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("page_search_output"),
			cmdutil.AnnotationErrors: loc.T("page_search_errors"),
		},
		// キーワードは引用符で1つにまとめても、空白で区切って並べてもよい。キーワードの有無はRunEで検査する。
		Args: cobra.ArbitraryArgs,
	}

	flags := &searchFlags{}
	cmd.Flags().StringVar(&flags.topic, "topic", "", loc.T("page_search_topic_flag"))
	cmd.Flags().StringVar(&flags.status, "status", "", loc.T("page_search_status_flag"))
	pageFlags := cmdutil.AddListFlags(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runSearch(f, cmd, args, flags, pageFlags)
	}

	return cmd
}

func runSearch(f *cmdutil.Factory, cmd *cobra.Command, args []string, flags *searchFlags, pageFlags *cmdutil.ListFlags) error {
	loc := f.Localizer

	params, err := flags.params(loc, cmd, args)
	if err != nil {
		return err
	}
	if err := pageFlags.Validate(loc, cmd); err != nil {
		return err
	}
	params.Limit = pageFlags.LimitParam()

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: listFields, Default: listDefaultFields})
	if err != nil {
		return err
	}

	client, space, err := f.APIClient(cmd)
	if err != nil {
		return err
	}

	return cmdutil.RunList(cmd.Context(), f, printer, pageFlags, cmdutil.List[apigen.Page]{
		Fetch: func(ctx context.Context, cursor *string) ([]apigen.Page, *string, error) {
			p := params
			p.Cursor = cursor
			resp, err := client.ListPagesWithResponse(ctx, space, &p)
			if err != nil {
				return nil, nil, err
			}
			if resp.JSON200 == nil {
				return nil, nil, cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
			}
			return resp.JSON200.Items, resp.JSON200.NextCursor, nil
		},
		Item: func(p apigen.Page) any { return newListItem(p) },
		Table: func(pages []apigen.Page) error {
			return printTable(printer, loc, pages)
		},
		EmptyHint: loc.T("page_search_empty_hint"),
	})
}

// paramsは、キーワードと絞り込みのフラグを要求のパラメーターにする。
// キーワードはAPIと同じく空白 (全角の空白を含む) で区切り、1つの空白でつないで `q` に載せる。
// キーワードの数の上限はAPIの規則のため、CLIでは検査せずAPIの422に任せる。
func (s *searchFlags) params(loc *i18n.Localizer, cmd *cobra.Command, args []string) (apigen.ListPagesParams, error) {
	var params apigen.ListPagesParams

	keywords := strings.Fields(strings.Join(args, " "))
	if len(keywords) == 0 {
		return params, cmdutil.NewUsageError(loc, cmd, loc.T("page_search_error_query_required"))
	}
	q := strings.Join(keywords, " ")
	params.Q = &q

	if cmd.Flags().Changed("topic") {
		topic, err := cmdutil.ParseNumber(loc, "--topic", s.topic)
		if err != nil {
			return params, err
		}
		params.TopicNumber = &topic
	}

	if cmd.Flags().Changed("status") {
		status, err := parseStatus(loc, cmd, s.status)
		if err != nil {
			return params, err
		}
		params.Status = &status
	}

	return params, nil
}
