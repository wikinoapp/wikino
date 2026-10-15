package page

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// createFlagsは `page create` のフラグの値。
type createFlags struct {
	topic  string
	title  string
	body   *cmdutil.BodyFlags
	dryRun *bool
}

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "create",
		Short:   loc.T("page_create_short"),
		Long:    loc.T("page_create_long"),
		Example: loc.T("page_create_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("page_create_output"),
			cmdutil.AnnotationErrors: loc.T("page_create_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 0),
	}

	flags := &createFlags{}
	cmd.Flags().StringVar(&flags.topic, "topic", "", loc.T("page_create_topic_flag"))
	cmd.Flags().StringVar(&flags.title, "title", "", loc.T("page_create_title_flag"))
	flags.body = cmdutil.AddBodyFlags(cmd, loc)
	flags.dryRun = cmdutil.AddDryRunFlag(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runCreate(f, cmd, flags)
	}

	return cmd
}

func runCreate(f *cmdutil.Factory, cmd *cobra.Command, flags *createFlags) error {
	loc := f.Localizer

	// APIはトピック・タイトル・本文のすべてを求める。本文の指定忘れで空のページを作らないよう、
	// 本文も空文字列を含めて明示させる。
	for _, name := range []string{"topic", "title"} {
		if !cmd.Flags().Changed(name) {
			return cmdutil.NewUsageError(loc, cmd, loc.T("usage_error_flag_required", map[string]any{"Flag": "--" + name}))
		}
	}
	if err := flags.body.Validate(loc); err != nil {
		return err
	}
	if !flags.body.Changed() {
		return cmdutil.NewUsageError(loc, cmd, loc.T("page_create_error_body_required"))
	}
	topic, err := cmdutil.ParseNumber(loc, "--topic", flags.topic)
	if err != nil {
		return err
	}

	if *flags.dryRun {
		if err := cmdutil.ValidateDryRunFlag(loc, cmd); err != nil {
			return err
		}
	}

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: getFields})
	if err != nil {
		return err
	}

	body, err := flags.body.Read(f)
	if err != nil {
		return err
	}

	var dryRun *apiutil.DryRun
	if *flags.dryRun {
		dryRun = &apiutil.DryRun{}
	}
	client, space, err := f.APIClientWithDryRun(cmd, dryRun)
	if err != nil {
		return err
	}

	// 書き込みで意図しないリンク先のページが作られないよう、要求の前にリンクの解決を確かめる。
	// プレビューは読み取り専用のため、`--dry-run` でも送る。
	links, err := cmdutil.PreviewWikilinks(cmd.Context(), loc, client, space, topic, []string{*body})
	if err != nil {
		return err
	}

	resp, err := client.CreatePageWithResponse(cmd.Context(), space, apigen.NewPage{
		TopicNumber: topic,
		Title:       flags.title,
		Body:        *body,
	})
	if errors.Is(err, apiutil.ErrDryRun) {
		return cmdutil.PrintWriteDryRun(f.IOStreams, loc, printer, dryRun, links)
	}
	if err != nil {
		// 作成は冪等でないため、結果が分からないときは作成済みかを確かめてから送り直させる。
		listCommand := output.JoinCommand([]string{"wikino", "page", "list", "--topic", formatNumber(topic), "--title", flags.title})
		return cmdutil.ConvertWithHints(loc, err, map[string]string{
			output.CodeNotFound:         loc.T("page_create_error_not_found_hint", map[string]any{"Topic": formatNumber(topic)}),
			output.CodeValidationFailed: loc.T("page_create_error_validation_failed_hint", map[string]any{"ListCommand": listCommand}),
			output.CodeNetwork:          loc.T("page_create_error_unknown_result_hint", map[string]any{"ListCommand": listCommand}),
			output.CodeServerError:      loc.T("page_create_error_unknown_result_hint", map[string]any{"ListCommand": listCommand}),
		})
	}
	if resp.JSON201 == nil || resp.Headers201 == nil {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}

	item := getItem{Page: *resp.JSON201, ETag: resp.Headers201.ETag}
	return printWritten(f, printer, item, loc.T("page_create_done", map[string]any{"Number": formatNumber(item.Number)}), links)
}
