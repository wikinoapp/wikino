package page

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// updateFlagsは `page update` のフラグの値。
type updateFlags struct {
	title   string
	body    *cmdutil.BodyFlags
	mdFile  *cmdutil.MarkdownFileFlag
	edits   *cmdutil.EditsFlag
	ifMatch *cmdutil.IfMatchFlags
	dryRun  *bool
}

func newCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "update <page_number>",
		Short:   loc.T("page_update_short"),
		Long:    loc.T("page_update_long"),
		Example: loc.T("page_update_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("page_update_output"),
			cmdutil.AnnotationErrors: loc.T("page_update_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
	}

	flags := &updateFlags{}
	cmd.Flags().StringVar(&flags.title, cmdutil.FlagTitle, "", loc.T("page_update_title_flag"))
	flags.body = cmdutil.AddBodyFlags(cmd, loc)
	flags.mdFile = cmdutil.AddMarkdownFileFlag(cmd, loc)
	flags.edits = cmdutil.AddEditsFlag(cmd, loc)
	flags.ifMatch = cmdutil.AddIfMatchFlags(cmd, loc)
	flags.dryRun = cmdutil.AddDryRunFlag(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runUpdate(f, cmd, flags, args[0])
	}

	return cmd
}

// updateRequestは、`page update` で送る書き込みの内容。フラグの組み合わせで、
// ページの更新 (patch) か本文の検索置換 (edits) のどちらかを送る。
type updateRequest struct {
	patch apigen.PagePatch
	edits []apigen.BodyEdit
	// ifMatchは `If-Match` に送る版。空なら送らない (検索置換で `--if-match` を省いたとき)。
	ifMatch string
	// fromMarkdownFileは、ページファイル (`--md-file`) から読んだ内容か。412で案内する取り直し方が変わる。
	fromMarkdownFile bool
}

// bodiesは、Wikiリンクのプレビューの対象にする本文を返す。本文を変えない要求ならnilを返す。
// 検索置換では、置換後の全文はサーバーで作られるため、各 `new_str` を対象にしてこの編集で増えるリンクを確かめる。
func (r *updateRequest) bodies() []string {
	if r.edits != nil {
		bodies := make([]string, 0, len(r.edits))
		for _, edit := range r.edits {
			bodies = append(bodies, edit.NewStr)
		}
		return bodies
	}
	if r.patch.Body != nil {
		return []string{*r.patch.Body}
	}

	return nil
}

func runUpdate(f *cmdutil.Factory, cmd *cobra.Command, flags *updateFlags, arg string) error {
	loc := f.Localizer

	number, err := cmdutil.ParseNumber(loc, "<page_number>", arg)
	if err != nil {
		return err
	}
	getCommand := output.JoinCommand([]string{"wikino", "page", "get", formatNumber(number)})
	getMarkdownCommand := output.JoinCommand([]string{"wikino", "page", "get", formatNumber(number), "--md"})

	if err := validateUpdateFlags(loc, cmd, flags, getCommand); err != nil {
		return err
	}

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: getFields})
	if err != nil {
		return err
	}

	req, err := readUpdateRequest(f, cmd, flags, number, getMarkdownCommand)
	if err != nil {
		return err
	}

	// `--dry-run` でも、ページの取得とWikiリンクのプレビュー (読み取り) は送り、書き込みの要求だけを送らない。
	var dryRun *apiutil.DryRun
	if *flags.dryRun {
		dryRun = &apiutil.DryRun{}
	}
	client, space, err := f.APIClientWithDryRun(cmd, dryRun)
	if err != nil {
		return err
	}

	// 本文を書き込むときは、プレビューに要るトピックの番号を、`--force` では最新の版を、ページの取得で得る。
	notFoundHint := loc.T("page_error_not_found_hint")
	bodies := req.bodies()
	var links []apigen.WikilinkPreviewLink
	if bodies != nil || flags.ifMatch.Force() {
		resp, err := client.GetPageWithResponse(cmd.Context(), space, number, nil)
		if err != nil {
			return cmdutil.ConvertNotFound(loc, err, notFoundHint)
		}
		if resp.JSON200 == nil || resp.Headers200 == nil {
			return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
		}
		if flags.ifMatch.Force() {
			req.ifMatch = resp.Headers200.ETag
		}

		if bodies != nil {
			if links, err = cmdutil.PreviewWikilinks(cmd.Context(), loc, client, space, resp.JSON200.TopicNumber, bodies); err != nil {
				return err
			}
		}
	}

	item, err := sendUpdate(cmd.Context(), loc, client, space, number, req, getCommand, getMarkdownCommand)
	if errors.Is(err, apiutil.ErrDryRun) {
		return cmdutil.PrintWriteDryRun(f.IOStreams, loc, printer, dryRun, links)
	}
	if err != nil {
		return err
	}

	return printWritten(f, printer, *item, loc.T("page_update_done", map[string]any{"Number": formatNumber(item.Number)}), links)
}

// validateUpdateFlagsは、`page update` のフラグの組み合わせを要求を送る前に検査する。
func validateUpdateFlags(loc *i18n.Localizer, cmd *cobra.Command, flags *updateFlags, getCommand string) error {
	if err := flags.body.Validate(loc); err != nil {
		return err
	}
	if err := flags.mdFile.Validate(loc); err != nil {
		return err
	}
	if err := flags.edits.Validate(loc); err != nil {
		return err
	}
	// 何も変えない要求を送ると、成功しても何も起きず、指定の漏れに気付けないため拒否する。
	if !cmd.Flags().Changed(cmdutil.FlagTitle) && !flags.body.Changed() && !flags.mdFile.Changed() && !flags.edits.Changed() {
		return cmdutil.NewUsageError(loc, cmd, loc.T("page_update_error_nothing_to_update"))
	}

	// ページファイルは版をファイルから読み、検索置換は `old_str` の一致が前提の確かめになるため版を任意にする。
	switch {
	case flags.mdFile.Changed():
	case flags.edits.Changed():
		if err := flags.ifMatch.ValidateOptional(loc, getCommand); err != nil {
			return err
		}
	default:
		if err := flags.ifMatch.Validate(loc, getCommand); err != nil {
			return err
		}
	}

	if *flags.dryRun {
		if err := cmdutil.ValidateDryRunFlag(loc, cmd); err != nil {
			return err
		}
	}

	return nil
}

// readUpdateRequestは、フラグとファイルから送る内容を読む。
func readUpdateRequest(f *cmdutil.Factory, cmd *cobra.Command, flags *updateFlags, number int32, getMarkdownCommand string) (*updateRequest, error) {
	req := &updateRequest{}

	switch {
	case flags.mdFile.Changed():
		file, err := flags.mdFile.Read(f, number, getMarkdownCommand)
		if err != nil {
			return nil, err
		}
		req.patch = apigen.PagePatch{Title: file.Title, Body: &file.Body}
		req.ifMatch = file.ETag
		req.fromMarkdownFile = true
	case flags.edits.Changed():
		edits, err := flags.edits.Read(f)
		if err != nil {
			return nil, err
		}
		req.edits = edits
		if cmd.Flags().Changed(cmdutil.FlagIfMatch) {
			req.ifMatch = flags.ifMatch.Value()
		}
	default:
		if cmd.Flags().Changed(cmdutil.FlagTitle) {
			req.patch.Title = &flags.title
		}
		body, err := flags.body.Read(f)
		if err != nil {
			return nil, err
		}
		req.patch.Body = body
		if cmd.Flags().Changed(cmdutil.FlagIfMatch) {
			req.ifMatch = flags.ifMatch.Value()
		}
	}

	return req, nil
}

// sendUpdateは、ページの更新か本文の検索置換の要求を送り、更新したページを返す。
func sendUpdate(ctx context.Context, loc *i18n.Localizer, client *apiclient.Client, space string, number int32, req *updateRequest, getCommand, getMarkdownCommand string) (*getItem, error) {
	notFoundHint := loc.T("page_error_not_found_hint")

	if req.edits != nil {
		var params apigen.CreatePageBodyEditParams
		if req.ifMatch != "" {
			params.IfMatch = &req.ifMatch
		}
		resp, err := client.CreatePageBodyEditWithResponse(ctx, space, number, &params, apigen.NewBodyEdit{Edits: req.edits})
		if errors.Is(err, apiutil.ErrDryRun) {
			return nil, err
		}
		if err != nil {
			return nil, cmdutil.ConvertEditsError(loc, err, map[string]string{
				output.CodeNotFound:           notFoundHint,
				output.CodePreconditionFailed: loc.T("page_update_error_edits_precondition_failed_hint", map[string]any{"Command": getMarkdownCommand}),
				output.CodeValidationFailed:   loc.T("page_update_error_validation_failed_hint"),
			}, getMarkdownCommand)
		}
		if resp.JSON200 == nil || resp.Headers200 == nil {
			return nil, cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
		}
		return &getItem{Page: *resp.JSON200, ETag: resp.Headers200.ETag}, nil
	}

	preconditionFailedHint := loc.T("page_update_error_precondition_failed_hint", map[string]any{"Command": getCommand})
	if req.fromMarkdownFile {
		preconditionFailedHint = loc.T("page_update_error_md_file_precondition_failed_hint", map[string]any{"Command": getMarkdownCommand})
	}
	resp, err := client.UpdatePageWithApplicationMergePatchPlusJSONBodyWithResponse(ctx, space, number, &apigen.UpdatePageParams{IfMatch: &req.ifMatch}, req.patch)
	if errors.Is(err, apiutil.ErrDryRun) {
		return nil, err
	}
	if err != nil {
		return nil, cmdutil.ConvertWithHints(loc, err, map[string]string{
			output.CodeNotFound:             notFoundHint,
			output.CodePreconditionFailed:   preconditionFailedHint,
			output.CodePreconditionRequired: loc.T("usage_error_if_match_required_hint", map[string]any{"Command": getCommand}),
			output.CodeValidationFailed:     loc.T("page_update_error_validation_failed_hint"),
		})
	}
	if resp.JSON200 == nil || resp.Headers200 == nil {
		return nil, cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}

	return &getItem{Page: *resp.JSON200, ETag: resp.Headers200.ETag}, nil
}
