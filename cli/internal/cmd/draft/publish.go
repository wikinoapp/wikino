package draft

import (
	"errors"
	"fmt"
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

// publishOperationIDは、下書きの公開のoperationId。確認が要るかをOpenAPI記述の分類から決めるために使う。
const publishOperationID = "createDraftPagePublication"

// publishFieldsは、`draft publish` の `--fields` で選べる項目。`page get` と同じく、APIのページの項目に応答のETagヘッダーの値を足す。
var publishFields = []string{"id", "number", "title", "status", "topic_number", "topic_name", "modified_at", "url", "body", "etag"}

// publishItemは、`draft publish` の出力。公開したページを `page get` と同じ形で出す。
type publishItem struct {
	apigen.Page
	// ETagは応答のETagヘッダーの値。続けてページを更新するときの `--if-match` に渡す。
	ETag string `json:"etag"`
}

// publishFlagsは `draft publish` のフラグの値。
type publishFlags struct {
	ifMatch *cmdutil.IfMatchFlags
	yes     *bool
	dryRun  *bool
}

func newCmdPublish(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer
	cmd := &cobra.Command{
		Use:     "publish <page_number>",
		Short:   loc.T("draft_publish_short"),
		Long:    loc.T("draft_publish_long"),
		Example: loc.T("draft_publish_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("draft_publish_output"),
			cmdutil.AnnotationErrors: loc.T("draft_publish_errors"),
		},
		Args: cmdutil.ExactArgs(loc, 1),
	}

	flags := &publishFlags{}
	flags.ifMatch = cmdutil.AddIfMatchFlags(cmd, loc)
	flags.yes = cmdutil.AddYesFlag(cmd, loc)
	flags.dryRun = cmdutil.AddDryRunFlag(cmd, loc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runPublish(f, cmd, flags, args[0])
	}

	return cmd
}

func runPublish(f *cmdutil.Factory, cmd *cobra.Command, flags *publishFlags, arg string) error {
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

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: publishFields})
	if err != nil {
		return err
	}

	// 確認の対象はコマンドに持たず、OpenAPI記述の操作の分類から決める。
	// `--dry-run` は何も公開しないため、確認を求めない。
	op, _ := apiclient.LookupOperation(publishOperationID)
	needsConfirm := op.ConfirmationRequired && !*flags.yes && !*flags.dryRun

	// `--dry-run` でも、下書きの取得とWikiリンクのプレビュー (読み取り) は送り、公開の要求だけを送らない。
	var dryRun *apiutil.DryRun
	if *flags.dryRun {
		dryRun = &apiutil.DryRun{}
	}
	client, space, err := f.APIClientWithDryRun(cmd, dryRun)
	if err != nil {
		return err
	}

	// 公開する本文のWikiリンクを確かめるため、`--yes` でも下書きを取得する。
	// 公開に要るdraft_page:writeとpage:writeは、下書きの取得とプレビューに要る読み取りのスコープを含む。
	resp, err := client.GetDraftPageWithResponse(cmd.Context(), space, number, nil)
	if err != nil {
		return cmdutil.ConvertNotFound(loc, err, loc.T("draft_error_not_found_hint"))
	}
	if resp.JSON200 == nil || resp.Headers200 == nil {
		return cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}
	current := getItem{DraftPage: *resp.JSON200, ETag: resp.Headers200.ETag}

	ifMatch := flags.ifMatch.Value()
	switch {
	case flags.ifMatch.Force():
		ifMatch = current.ETag
	case strings.TrimSpace(ifMatch) != current.ETag:
		// 取得した下書きは、利用者が読んだ版と違う。リンク先の確認や公開の確認を今の下書きで示すと、
		// 読んでいない内容の公開を承認させることになるため、公開の要求を送ったときの412と同じく失敗させる。
		return &output.Error{
			Code:     output.CodePreconditionFailed,
			Message:  loc.T("draft_publish_error_version_changed"),
			Hint:     loc.T("draft_publish_error_precondition_failed_hint", map[string]any{"Command": getCommand}),
			ExitCode: output.ExitConflict,
		}
	}

	links, err := cmdutil.PreviewWikilinks(cmd.Context(), loc, client, space, current.TopicNumber, []string{current.Body})
	if err != nil {
		return err
	}

	if needsConfirm {
		confirmation, err := publishConfirmation(cmd, loc, client, space, current, links)
		if err != nil {
			return err
		}
		if err := output.Confirm(f.IOStreams, loc, *confirmation); err != nil {
			return err
		}
	}

	pub, err := client.CreateDraftPagePublicationWithResponse(cmd.Context(), space, number, &apigen.CreateDraftPagePublicationParams{IfMatch: &ifMatch})
	if errors.Is(err, apiutil.ErrDryRun) {
		return cmdutil.PrintWriteDryRun(f.IOStreams, loc, printer, dryRun, links)
	}
	if err != nil {
		return cmdutil.ConvertWithHints(loc, err, map[string]string{
			output.CodeNotFound:             loc.T("draft_publish_error_not_found_hint", map[string]any{"Command": output.JoinCommand([]string{"wikino", "page", "get", formatNumber(number)})}),
			output.CodePreconditionFailed:   loc.T("draft_publish_error_precondition_failed_hint", map[string]any{"Command": getCommand}),
			output.CodePreconditionRequired: loc.T("usage_error_if_match_required_hint", map[string]any{"Command": getCommand}),
			output.CodeValidationFailed:     loc.T("draft_publish_error_validation_failed_hint", map[string]any{"Command": titleCommand(number)}),
		})
	}
	if pub.JSON200 == nil || pub.Headers200 == nil {
		return cmdutil.UnexpectedResponseError(loc, pub.StatusCode())
	}
	item := publishItem{Page: *pub.JSON200, ETag: pub.Headers200.ETag}

	if printer.IsJSON() {
		if err := printer.PrintItem(item); err != nil {
			return err
		}
	} else {
		if err := printer.PrintDetail(publishDetailFields(loc, item), ""); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(f.IOStreams.ErrOut, loc.T("draft_publish_done", map[string]any{"Number": formatNumber(item.Number)})); err != nil {
			return err
		}
	}

	return cmdutil.PrintCreatedWikilinks(f.IOStreams, loc, links)
}

// titleCommandは、下書きのタイトルを直すコマンドの例を返す。版は利用者が取得し直すため、値の名前で示す。
func titleCommand(number int32) string {
	return output.JoinCommand([]string{"wikino", "draft", "update", formatNumber(number), "--" + cmdutil.FlagTitle, "<title>", "--" + cmdutil.FlagIfMatch, "<etag>"})
}

// publishConfirmationは、下書きの公開の前に利用者へ示す確認の内容を返す。
// 今のページと比べたタイトルの変更と本文の増減の行数、作成されるリンク先、公開後に下書きが消えることを示す。
// 承認後のコマンドには、確認のために取得した版を `--if-match` で固定して入れ、確認した内容と公開する内容が
// ずれないようにする (`--force` は引き継がない)。
func publishConfirmation(cmd *cobra.Command, loc *i18n.Localizer, client *apiclient.Client, space string, current getItem, links []apigen.WikilinkPreviewLink) (*output.Confirmation, error) {
	number := formatNumber(current.PageNumber)

	// タイトルの無い下書きは公開できない (422)。承認を得てから失敗させると利用者の確認が無駄になるため、確認の前に失敗させる。
	if current.Title == nil {
		return nil, &output.Error{
			Code:     output.CodeValidationFailed,
			Message:  loc.T("draft_publish_error_untitled"),
			Hint:     loc.T("draft_publish_error_validation_failed_hint", map[string]any{"Command": titleCommand(current.PageNumber)}),
			ExitCode: output.ExitValidation,
		}
	}

	resp, err := client.GetPageWithResponse(cmd.Context(), space, current.PageNumber, nil)
	if err != nil {
		return nil, cmdutil.ConvertNotFound(loc, err, loc.T("draft_error_not_found_hint"))
	}
	if resp.JSON200 == nil {
		return nil, cmdutil.UnexpectedResponseError(loc, resp.StatusCode())
	}
	page := resp.JSON200

	titleChange := loc.T("draft_publish_confirm_title_unchanged")
	if page.Title == nil || *page.Title != *current.Title {
		titleChange = loc.T("draft_publish_confirm_title_changed", map[string]any{"Old": titleLabel(loc, page.Title)})
	}
	added, deleted := countLineChanges(page.Body, current.Body)

	changes := []string{
		loc.T("draft_publish_confirm_change_publish", map[string]any{
			"Number":      number,
			"Title":       *current.Title,
			"TitleChange": titleChange,
			"Added":       strconv.Itoa(added),
			"Deleted":     strconv.Itoa(deleted),
		}),
	}
	if page.Status == apigen.PageStatusUnpublished {
		changes = append(changes, loc.T("draft_publish_confirm_change_unpublished", map[string]any{"Number": number}))
	}
	if names := cmdutil.NewPageLinkNames(links); len(names) > 0 {
		changes = append(changes, loc.T("draft_publish_confirm_change_new_pages", map[string]any{
			"Count": strconv.Itoa(len(names)),
			"Pages": strings.Join(names, ", "),
		}))
	}
	changes = append(changes, loc.T("draft_publish_confirm_change_draft_deleted"))

	command := []string{"wikino", "draft", "publish", number, "--" + cmdutil.FlagIfMatch, current.ETag}
	command = append(command, cmdutil.TargetFlagArgs(cmd)...)
	command = append(command, "--"+cmdutil.FlagYes)

	return &output.Confirmation{Changes: changes, Command: command}, nil
}

// publishDetailFieldsは、人間向けの表示で公開したページの項目を返す。`page get` と同じ項目を同じラベルで並べる。
func publishDetailFields(loc *i18n.Localizer, item publishItem) []output.Field {
	status := string(item.Status)
	switch item.Status {
	case apigen.PageStatusPublished:
		status = loc.T("page_value_published")
	case apigen.PageStatusUnpublished:
		status = loc.T("page_value_unpublished")
	}
	title := loc.T("page_value_untitled")
	if item.Title != nil {
		title = *item.Title
	}

	return []output.Field{
		{Label: loc.T("page_label_number"), Value: formatNumber(item.Number)},
		{Label: loc.T("page_label_title"), Value: title},
		{Label: loc.T("page_label_topic_name"), Value: item.TopicName},
		{Label: loc.T("page_label_topic_number"), Value: formatNumber(item.TopicNumber)},
		{Label: loc.T("page_label_status"), Value: status},
		{Label: loc.T("page_label_modified_at"), Value: formatTime(item.ModifiedAt)},
		{Label: loc.T("page_label_url"), Value: item.Url},
		{Label: loc.T("page_label_etag"), Value: item.ETag},
	}
}
