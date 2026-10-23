package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// PreviewWikilinksは、bodiesの各本文をトピックtopicのページや下書きに書き込んだときに、
// 本文のWikiリンクがどのページを指すかを公開APIの `createWikilinkPreview` で確かめる。
// 書き込みで意図しないリンク先のページが作られる事故を防ぐため、本文を書き込む要求の前に呼ぶ。
// Wikiリンクの解釈はサーバーにだけ持たせ、CLIは解釈しない。
//
// 検索置換では各 `new_str` を1つの本文として渡し、この編集で増えるリンクを確かめる。
// 複数の本文で同じリンク先を指すリンクは、最初の1つだけを返す。空の本文はリンクを持たないため送らない。
// 失敗したときは書き込みの要求を送らずに済むよう、エラーを返す。
func PreviewWikilinks(ctx context.Context, loc *i18n.Localizer, client *apiclient.Client, space string, topic int32, bodies []string) ([]apigen.WikilinkPreviewLink, error) {
	links := []apigen.WikilinkPreviewLink{}
	seen := map[string]bool{}
	for _, body := range bodies {
		if body == "" {
			continue
		}

		resp, err := client.CreateWikilinkPreviewWithResponse(ctx, space, apigen.NewWikilinkPreview{TopicNumber: topic, Body: body})
		if err != nil {
			return nil, convertPreviewError(loc, err, topic)
		}
		if resp.JSON200 == nil {
			return nil, UnexpectedResponseError(loc, resp.StatusCode())
		}

		for _, link := range resp.JSON200.Links {
			key := string(link.Resolution) + "\x00" + link.TopicName + "\x00" + link.PageTitle
			if seen[key] {
				continue
			}
			seen[key] = true
			links = append(links, link)
		}
	}

	return links, nil
}

// convertPreviewErrorは、プレビューのエラーを、書き込みの要求を送っていないことが分かるCLIのエラーにする。
func convertPreviewError(loc *i18n.Localizer, err error, topic int32) error {
	err = ConvertWithHints(loc, err, map[string]string{
		output.CodeValidationFailed: loc.T("wikilink_preview_error_validation_failed_hint", map[string]any{"Topic": strconv.FormatInt(int64(topic), 10)}),
	})

	var cliErr *output.Error
	if errors.As(err, &cliErr) {
		cliErr.Message = loc.T("wikilink_preview_error", map[string]any{"Message": cliErr.Message})
	}

	return err
}

// writeDryRunOutputは、本文を書き込むコマンドの `--dry-run` の結果。
// 送る要求に加えて、本文のWikiリンクの解決の結果を `wikilinks` に出す。
type writeDryRunOutput struct {
	apiutil.DryRunOutput
	Wikilinks []apigen.WikilinkPreviewLink `json:"wikilinks"`
}

// PrintWriteDryRunは、本文を書き込むコマンドの `--dry-run` の結果を標準出力にJSONで出す。
// 書き込むと作られるリンク先と、要求を送っていないことは標準エラーに出す。
func PrintWriteDryRun(ios *iostreams.IOStreams, loc *i18n.Localizer, printer *output.Printer, dryRun *apiutil.DryRun, links []apigen.WikilinkPreviewLink) error {
	if links == nil {
		links = []apigen.WikilinkPreviewLink{}
	}
	if err := printer.PrintItem(writeDryRunOutput{DryRunOutput: dryRun.Output(), Wikilinks: links}); err != nil {
		return err
	}

	if err := printNewPageLinks(ios, loc, links, "wikilink_will_create"); err != nil {
		return err
	}

	_, err := fmt.Fprintln(ios.ErrOut, loc.T("dry_run_not_sent"))
	return err
}

// PrintCreatedWikilinksは、書き込みで作られた見込みのリンク先を標準エラーに出す。
// 標準出力はAPIの応答のまま保ち、JSONを読む利用者の出力を汚さない。
// プレビューと書き込みの間に他の利用者がページを作成・削除すると実際の結果とずれうるため、見込みとして出す。
func PrintCreatedWikilinks(ios *iostreams.IOStreams, loc *i18n.Localizer, links []apigen.WikilinkPreviewLink) error {
	return printNewPageLinks(ios, loc, links, "wikilink_created")
}

// NewPageLinkNamesは、linksのうち書き込むと作られるリンク先 (`new_page`) を「トピック名/ページのタイトル」の形で返す。
func NewPageLinkNames(links []apigen.WikilinkPreviewLink) []string {
	var names []string
	for _, link := range links {
		if link.Resolution == apigen.WikilinkPreviewLinkResolutionNewPage {
			names = append(names, link.TopicName+"/"+link.PageTitle)
		}
	}

	return names
}

// printNewPageLinksは、作られるリンク先があれば、messageIDの文言で1行にまとめて標準エラーに出す。
// トピック名とページのタイトルは他の利用者が書いた値のため、確認の文言と同じく1行の値として無害化する。
func printNewPageLinks(ios *iostreams.IOStreams, loc *i18n.Localizer, links []apigen.WikilinkPreviewLink, messageID string) error {
	names := NewPageLinkNames(links)
	if len(names) == 0 {
		return nil
	}

	_, err := fmt.Fprintln(ios.ErrOut, loc.T(messageID, map[string]any{
		"Count": strconv.Itoa(len(names)),
		"Pages": output.SanitizeLine(strings.Join(names, ", ")),
	}))
	return err
}
