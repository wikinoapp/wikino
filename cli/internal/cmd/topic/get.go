package topic

import (
	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer

	return &cobra.Command{
		Use:     "get <topic_number>",
		Short:   loc.T("topic_get_short"),
		Long:    loc.T("topic_get_long"),
		Example: loc.T("topic_get_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("topic_get_output"),
			cmdutil.AnnotationErrors: loc.T("topic_get_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(f, cmd, args[0])
		},
	}
}

func runGet(f *cmdutil.Factory, cmd *cobra.Command, arg string) error {
	loc := f.Localizer

	number, err := cmdutil.ParseNumber(loc, "<topic_number>", arg)
	if err != nil {
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

	resp, err := client.GetTopicWithResponse(cmd.Context(), space, number)
	if err != nil {
		return cmdutil.ConvertNotFound(loc, err, loc.T("topic_get_error_not_found_hint"))
	}
	topic := resp.JSON200
	if topic == nil {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}

	// トピックの応答はETagを返さないため、ページと違って `etag` を加えない。
	if printer.IsJSON() {
		return printer.PrintItem(topic)
	}

	return printer.PrintDetail([]output.Field{
		{Label: loc.T("topic_label_number"), Value: formatNumber(topic.Number)},
		{Label: loc.T("topic_label_name"), Value: topic.Name},
		{Label: loc.T("topic_label_visibility"), Value: visibilityLabel(loc, topic.Visibility)},
		{Label: loc.T("topic_label_description"), Value: topic.Description},
	}, "")
}
