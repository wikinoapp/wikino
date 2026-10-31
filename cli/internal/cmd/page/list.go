package page

import (
	"context"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// listFieldsは、ページの一覧の `--fields` で選べる項目。ページの項目に、本文の冒頭の抜粋 (excerpt) を足す。
var listFields = append(slices.Clone(pageFields), "excerpt")

// listDefaultFieldsは、ページの一覧で既定で出す項目。本文 (body) は `--fields` で選んだときだけ出す。
// エージェントが一覧を見るたびに、各ページの本文がコンテキストに入るのを避けるためである。
var listDefaultFields = slices.DeleteFunc(slices.Clone(listFields), func(name string) bool { return name == "body" })

// listFlagsは `page list` の絞り込みのフラグの値。
type listFlags struct {
	topic         string
	status        string
	title         string
	modifiedSince string
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "list",
		Short:   loc.T("page_list_short"),
		Long:    loc.T("page_list_long"),
		Example: loc.T("page_list_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("page_list_output"),
			cmdutil.AnnotationErrors: loc.T("page_list_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 0),
	}

	flags := &listFlags{}
	cmd.Flags().StringVar(&flags.topic, "topic", "", loc.T("page_list_topic_flag"))
	cmd.Flags().StringVar(&flags.status, "status", "", loc.T("page_list_status_flag"))
	cmd.Flags().StringVar(&flags.title, "title", "", loc.T("page_list_title_flag"))
	cmd.Flags().StringVar(&flags.modifiedSince, "modified-since", "", loc.T("page_list_modified_since_flag"))
	pageFlags := cmdutil.AddListFlags(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runList(f, cmd, flags, pageFlags)
	}

	return cmd
}

func runList(f *cmdutil.Factory, cmd *cobra.Command, flags *listFlags, pageFlags *cmdutil.ListFlags) error {
	loc := f.Localizer

	params, err := flags.params(loc, cmd)
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
	params.Include = cmdutil.ListInclude(printer)

	client, space, err := f.APIClient(cmd)
	if err != nil {
		return err
	}

	return cmdutil.RunList(cmd.Context(), f, printer, pageFlags, cmdutil.List[apigen.PageListItem]{
		Fetch: func(ctx context.Context, cursor *string) ([]apigen.PageListItem, *string, error) {
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
		Item: func(p apigen.PageListItem) any { return p },
		Table: func(pages []apigen.PageListItem) error {
			return printTable(printer, loc, pages)
		},
	})
}

// paramsは、絞り込みのフラグを要求のパラメーターにする。値の誤りは要求を送る前に使い方の誤りにする。
func (l *listFlags) params(loc *i18n.Localizer, cmd *cobra.Command) (apigen.ListPagesParams, error) {
	var params apigen.ListPagesParams

	if cmd.Flags().Changed("topic") {
		topic, err := cmdutil.ParseNumber(loc, "--topic", l.topic)
		if err != nil {
			return params, err
		}
		params.TopicNumber = &topic
	}

	if cmd.Flags().Changed("status") {
		status, err := parseStatus(loc, cmd, l.status)
		if err != nil {
			return params, err
		}
		params.Status = &status
	}

	if cmd.Flags().Changed("title") {
		params.Title = &l.title
	}

	if cmd.Flags().Changed("modified-since") {
		since, err := time.Parse(time.RFC3339Nano, l.modifiedSince)
		if err != nil {
			return params, cmdutil.NewUsageError(loc, cmd, loc.T("page_list_error_invalid_modified_since", map[string]any{"Value": l.modifiedSince}))
		}
		params.ModifiedSince = &since
	}

	return params, nil
}

// parseStatusは、`--status` の値を読む。未知の値は要求を送る前に使い方の誤りにする。
func parseStatus(loc *i18n.Localizer, cmd *cobra.Command, value string) (apigen.ListPagesParamsStatus, error) {
	status := apigen.ListPagesParamsStatus(value)
	switch status {
	case apigen.ListPagesParamsStatusPublished, apigen.ListPagesParamsStatusUnpublished, apigen.ListPagesParamsStatusAll:
		return status, nil
	default:
		return "", cmdutil.NewUsageError(loc, cmd, loc.T("page_error_invalid_status", map[string]any{"Value": value}))
	}
}

func printTable(printer *output.Printer, loc *i18n.Localizer, pages []apigen.PageListItem) error {
	rows := make([][]string, 0, len(pages))
	for _, p := range pages {
		rows = append(rows, []string{
			formatNumber(p.Number),
			titleLabel(loc, p.Title),
			p.TopicName,
			formatNumber(p.TopicNumber),
			statusLabel(loc, p.Status),
			formatTime(p.ModifiedAt),
		})
	}

	return printer.PrintTable([]string{
		loc.T("page_label_number"),
		loc.T("page_label_title"),
		loc.T("page_label_topic_name"),
		loc.T("page_label_topic_number"),
		loc.T("page_label_status"),
		loc.T("page_label_modified_at"),
	}, rows)
}
