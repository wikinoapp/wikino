package draft

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// deleteOperationIDは、下書きの削除のoperationId。確認が要るかをOpenAPI記述の分類から決めるために使う。
const deleteOperationID = "deleteDraftPage"

// deleteFieldsは、`draft delete` の `--fields` で選べる項目。
var deleteFields = []string{"page_number", "deleted"}

// deleteItemは、`draft delete` の出力。APIの削除の応答は本文を持たないため、削除した下書きのページ番号を示す。
type deleteItem struct {
	PageNumber int32 `json:"page_number"`
	Deleted    bool  `json:"deleted"`
}

// deleteFlagsは `draft delete` のフラグの値。
type deleteFlags struct {
	ifMatch *cmdutil.IfMatchFlags
	yes     *bool
	dryRun  *bool
}

func newCmdDelete(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "delete <page_number>",
		Short:   loc.T("draft_delete_short"),
		Long:    loc.T("draft_delete_long"),
		Example: loc.T("draft_delete_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_delete_output"),
			cmdutil.AnnotationErrors: loc.T("draft_delete_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
	}

	flags := &deleteFlags{}
	flags.ifMatch = cmdutil.AddIfMatchFlags(cmd, loc)
	flags.yes = cmdutil.AddYesFlag(cmd, loc)
	flags.dryRun = cmdutil.AddDryRunFlag(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runDelete(f, cmd, flags, args[0])
	}

	return cmd
}

func runDelete(f *cmdutil.Factory, cmd *cobra.Command, flags *deleteFlags, arg string) error {
	loc := f.Localizer

	number, err := cmdutil.ParseNumber(loc, "<page_number>", arg)
	if err != nil {
		return err
	}
	getCommand := output.JoinCommand([]string{"wikino", "draft", "get", formatNumber(number)})

	if err := flags.ifMatch.Validate(loc, getCommand); err != nil {
		return err
	}
	if *flags.dryRun {
		if err := cmdutil.ValidateDryRunFlag(loc, cmd); err != nil {
			return err
		}
	}

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: deleteFields})
	if err != nil {
		return err
	}

	// 確認の対象はコマンドに持たず、OpenAPI記述の操作の分類から決める。
	// `--dry-run` は何も削除しないため、確認を求めない。
	op, _ := apiclient.LookupOperation(deleteOperationID)
	needsConfirm := op.ConfirmationRequired && !*flags.yes && !*flags.dryRun

	// `--dry-run` でも、確認と `--force` のための下書きの取得 (読み取り) は送り、削除の要求だけを送らない。
	var dryRun *apiutil.DryRun
	if *flags.dryRun {
		dryRun = &apiutil.DryRun{}
	}
	client, space, err := f.APIClientWithDryRun(cmd, dryRun)
	if err != nil {
		return err
	}

	ifMatch := flags.ifMatch.Value()
	if needsConfirm || flags.ifMatch.Force() {
		resp, err := client.GetDraftPageWithResponse(cmd.Context(), space, number, nil)
		if err != nil {
			return cmdutil.ConvertNotFound(loc, err, loc.T("draft_error_not_found_hint"))
		}
		if resp.JSON200 == nil || resp.Headers200 == nil {
			return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
		}
		current := getItem{DraftPage: *resp.JSON200, ETag: resp.Headers200.ETag}

		switch {
		case flags.ifMatch.Force():
			ifMatch = current.ETag
		case strings.TrimSpace(ifMatch) != current.ETag:
			// 確認に示す今の下書きは、利用者が読んだ版と違う。そのまま確認すると、読んでいない内容の削除を
			// 承認させることになるため、削除の要求を送ったときの412と同じく失敗させる。
			return &output.Error{
				Code:     output.CodePreconditionFailed,
				Message:  loc.T("draft_delete_error_version_changed"),
				Hint:     loc.T("draft_delete_error_precondition_failed_hint", map[string]any{"Command": getCommand}),
				ExitCode: output.ExitConflict,
			}
		}

		if needsConfirm {
			confirmation, err := deleteConfirmation(cmd, loc, client, space, current)
			if err != nil {
				return err
			}
			if err := output.Confirm(f.IOStreams, loc, *confirmation); err != nil {
				return err
			}
		}
	}

	resp, err := client.DeleteDraftPageWithResponse(cmd.Context(), space, number, &apigen.DeleteDraftPageParams{IfMatch: &ifMatch})
	if errors.Is(err, apiutil.ErrDryRun) {
		return cmdutil.PrintDryRun(f.IOStreams, loc, printer, dryRun)
	}
	if err != nil {
		return cmdutil.ConvertWithHints(loc, err, map[string]string{
			output.CodeNotFound:             loc.T("draft_delete_error_not_found_hint"),
			output.CodePreconditionFailed:   loc.T("draft_delete_error_precondition_failed_hint", map[string]any{"Command": getCommand}),
			output.CodePreconditionRequired: loc.T("usage_error_if_match_required_hint", map[string]any{"Command": getCommand}),
		})
	}
	if resp.StatusCode() != http.StatusNoContent {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}

	item := deleteItem{PageNumber: number, Deleted: true}
	if printer.IsJSON() {
		return printer.PrintItem(item)
	}
	_, err = fmt.Fprintln(f.IOStreams.ErrOut, loc.T("draft_delete_done", map[string]any{"Number": formatNumber(number)}))
	return err
}

// deleteConfirmationは、下書きの削除の前に利用者へ示す確認の内容を返す。
// 削除する下書きと、一緒に消える編集履歴の件数を示す。承認後のコマンドには、確認のために取得した版を
// `--if-match` で固定して入れ、確認した内容と削除する内容がずれないようにする (`--force` は引き継がない)。
func deleteConfirmation(cmd *cobra.Command, loc *i18n.Localizer, client *apiclient.Client, space string, current getItem) (*output.Confirmation, error) {
	revisions, err := countRevisions(cmd.Context(), loc, client, space, current.PageNumber)
	if err != nil {
		return nil, err
	}

	number := formatNumber(current.PageNumber)
	changes := []string{
		loc.T("draft_delete_confirm_change_draft", map[string]any{
			"Number":     number,
			"Title":      titleLabel(loc, current.Title),
			"ModifiedAt": formatTime(current.ModifiedAt),
		}),
		loc.T("draft_delete_confirm_change_revisions", map[string]any{"Count": strconv.Itoa(revisions)}),
		loc.T("draft_delete_confirm_change_page_unchanged"),
	}

	command := []string{"wikino", "draft", "delete", number, "--" + cmdutil.FlagIfMatch, current.ETag}
	command = append(command, cmdutil.TargetFlagArgs(cmd)...)
	command = append(command, "--"+cmdutil.FlagYes)

	return &output.Confirmation{Changes: changes, Command: command}, nil
}

// revisionPageLimitは、編集履歴の件数を数えるときに1回の要求で取得する件数。APIの上限に合わせ、要求の回数を抑える。
const revisionPageLimit int32 = 100

// countRevisionsは、ページnumberの下書きの編集履歴の件数を、`next_cursor` を辿って数える。
func countRevisions(ctx context.Context, loc *i18n.Localizer, client *apiclient.Client, space string, number int32) (int, error) {
	limit := revisionPageLimit
	fetch := func(ctx context.Context, cursor *string) ([]apigen.DraftPageRevision, *string, error) {
		resp, err := client.ListDraftPageRevisionsWithResponse(ctx, space, number, &apigen.ListDraftPageRevisionsParams{
			Cursor: cursor,
			Limit:  &limit,
		})
		if err != nil {
			return nil, nil, err
		}
		if resp.JSON200 == nil {
			return nil, nil, cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
		}
		return resp.JSON200.Items, resp.JSON200.NextCursor, nil
	}

	count := 0
	for _, err := range apiclient.All(ctx, fetch) {
		if err != nil {
			return 0, cmdutil.ConvertNotFound(loc, err, loc.T("draft_error_not_found_hint"))
		}
		count++
	}

	return count, nil
}
