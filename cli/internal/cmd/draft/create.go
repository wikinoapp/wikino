package draft

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// flagTopicは、新しいページの下書きを作るトピックを指すフラグの名前。
const flagTopic = "topic"

// createFlagsは `draft create` のフラグの値。
type createFlags struct {
	topic  string
	title  string
	body   *cmdutil.BodyFlags
	dryRun *bool
}

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "create [<page_number>]",
		Short:   loc.T("draft_create_short"),
		Long:    loc.T("draft_create_long"),
		Example: loc.T("draft_create_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_create_output"),
			cmdutil.AnnotationErrors: loc.T("draft_create_errors"),
		},
		Args: cmdutil.MaximumNArgs(loc, 1),
	}

	flags := &createFlags{}
	cmd.Flags().StringVar(&flags.topic, flagTopic, "", loc.T("draft_create_topic_flag"))
	cmd.Flags().StringVar(&flags.title, cmdutil.FlagTitle, "", loc.T("draft_create_title_flag"))
	flags.body = cmdutil.AddBodyFlags(cmd, loc)
	flags.dryRun = cmdutil.AddDryRunFlag(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCreate(f, cmd, flags, args)
	}

	return cmd
}

// createTargetは、下書きを作成する先。既存のページ (pageNumber) か、新しいページを作るトピック (topic) のどちらか一方を持つ。
type createTarget struct {
	pageNumber int32
	topic      int32
}

// parseCreateTargetは、位置引数のページ番号と `--topic` から、下書きを作成する先を読む。
// どちらか一方だけを求め、両方や無指定は要求を送る前に使い方の誤りにする。
func parseCreateTarget(loc *i18n.Localizer, cmd *cobra.Command, flags *createFlags, args []string) (createTarget, error) {
	hasTopic := cmd.Flags().Changed(flagTopic)

	switch {
	case len(args) == 1 && hasTopic:
		return createTarget{}, cmdutil.NewUsageError(loc, cmd, loc.T("draft_create_error_page_and_topic"))
	case len(args) == 1:
		number, err := cmdutil.ParseNumber(loc, "<page_number>", args[0])
		return createTarget{pageNumber: number}, err
	case hasTopic:
		topic, err := cmdutil.ParseNumber(loc, "--"+flagTopic, flags.topic)
		return createTarget{topic: topic}, err
	default:
		return createTarget{}, cmdutil.NewUsageError(loc, cmd, loc.T("draft_create_error_target_required"))
	}
}

func runCreate(f *cmdutil.Factory, cmd *cobra.Command, flags *createFlags, args []string) error {
	loc := f.Localizer

	target, err := parseCreateTarget(loc, cmd, flags, args)
	if err != nil {
		return err
	}
	if err := flags.body.Validate(loc); err != nil {
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

	// `--dry-run` でも、ページの取得とWikiリンクのプレビュー (読み取り) は送り、作成の要求だけを送らない。
	var dryRun *apiutil.DryRun
	if *flags.dryRun {
		dryRun = &apiutil.DryRun{}
	}
	client, space, err := f.APIClientWithDryRun(cmd, dryRun)
	if err != nil {
		return err
	}

	// 書き込みで意図しないリンク先のページが作られないよう、本文を渡したときは要求の前にリンクの解決を確かめる。
	// 既存のページでは、プレビューに要るトピックの番号をページの取得で得る。
	// 本文を省くと既存のページの今の本文か空の本文になり、新しいリンク先は作られないため確かめない。
	var links []apigen.WikilinkPreviewLink
	if body != nil {
		topic := target.topic
		if target.pageNumber != 0 {
			resp, err := client.GetPageWithResponse(cmd.Context(), space, target.pageNumber, nil)
			if err != nil {
				return cmdutil.ConvertNotFound(loc, err, loc.T("draft_create_error_page_not_found_hint"))
			}
			if resp.JSON200 == nil {
				return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
			}
			topic = resp.JSON200.TopicNumber
		}

		if links, err = cmdutil.PreviewWikilinks(cmd.Context(), loc, client, space, topic, []string{*body}); err != nil {
			return err
		}
	}

	newDraft := apigen.NewDraftPage{Body: body}
	if cmd.Flags().Changed(cmdutil.FlagTitle) {
		newDraft.Title = &flags.title
	}
	if target.pageNumber != 0 {
		newDraft.PageNumber = &target.pageNumber
	} else {
		newDraft.TopicNumber = &target.topic
	}

	resp, err := client.CreateDraftPageWithResponse(cmd.Context(), space, newDraft)
	if errors.Is(err, apiutil.ErrDryRun) {
		return cmdutil.PrintWriteDryRun(f.IOStreams, loc, printer, dryRun, links)
	}
	if err != nil {
		return convertCreateError(loc, err, target)
	}
	if resp.JSON201 == nil || resp.Headers201 == nil {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}

	item := getItem{DraftPage: *resp.JSON201, ETag: resp.Headers201.ETag}
	return printWritten(f, printer, item, loc.T("draft_create_done", map[string]any{"Number": formatNumber(item.PageNumber)}), links)
}

// convertCreateErrorは、作成の要求のエラーを、作成する先に合わせたヒントを持つCLIのエラーにする。
func convertCreateError(loc *i18n.Localizer, err error, target createTarget) error {
	var hints map[string]string
	var targetField, validationHint string
	if target.pageNumber == 0 {
		// 新しいページの下書きの作成は冪等でないため、結果が分からないときは作成済みかを確かめてから送り直させる。
		topic := formatNumber(target.topic)
		hints = map[string]string{
			output.CodeNotFound:    loc.T("draft_create_error_topic_not_found_hint", map[string]any{"Topic": topic}),
			output.CodeNetwork:     loc.T("draft_create_error_topic_unknown_result_hint"),
			output.CodeServerError: loc.T("draft_create_error_topic_unknown_result_hint"),
		}
		targetField = "topic_number"
		validationHint = loc.T("draft_create_error_topic_validation_failed_hint")
	} else {
		// 既存のページの下書きは1つだけのため、送り直して409になれば作成済みと分かる。
		number := formatNumber(target.pageNumber)
		getCommand := output.JoinCommand([]string{"wikino", "draft", "get", number})
		updateCommand := output.JoinCommand([]string{"wikino", "draft", "update", number})
		hints = map[string]string{
			output.CodeConflict:    loc.T("draft_create_error_conflict_hint", map[string]any{"GetCommand": getCommand, "UpdateCommand": updateCommand}),
			output.CodeNotFound:    loc.T("draft_create_error_page_forbidden_hint", map[string]any{"Number": number}),
			output.CodeNetwork:     loc.T("draft_create_error_page_unknown_result_hint", map[string]any{"GetCommand": getCommand}),
			output.CodeServerError: loc.T("draft_create_error_page_unknown_result_hint", map[string]any{"GetCommand": getCommand}),
		}
		targetField = "page_number"
		validationHint = loc.T("draft_create_error_page_not_found_hint")
	}

	// タイトルなどの入力エラーに番号の確認を案内しないよう、作成先の項目に問題があるときだけ差し替える。
	var apiErr *apiclient.Error
	if errors.As(err, &apiErr) {
		for _, field := range apiErr.FieldErrors {
			if field.Pointer == "#/"+targetField || field.Pointer == "/"+targetField {
				hints[output.CodeValidationFailed] = validationHint
				break
			}
		}
	}

	return cmdutil.ConvertWithHints(loc, err, hints)
}
