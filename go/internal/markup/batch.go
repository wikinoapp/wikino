package markup

import (
	"context"

	"github.com/yuin/goldmark/ast"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PageLocationResolver はWikiリンクキーからページ位置情報を一括解決するインターフェース。
// バッチレンダリング時にN+1クエリを防止するため、複数のキーを一括で解決する。
type PageLocationResolver interface {
	ResolveByKeys(ctx context.Context, keys []WikilinkKey, spaceID model.SpaceID) ([]PageLocation, error)
}

// BatchAttachmentFinder は添付ファイルの一括検索インターフェース。
// バッチレンダリング時にN+1クエリを防止するため、複数のIDを一括で検索する。
type BatchAttachmentFinder interface {
	FindByIDsAndSpace(ctx context.Context, ids []model.AttachmentID, spaceID model.SpaceID) ([]*model.Attachment, error)
}

// BatchRenderInput はバッチレンダリングの入力
type BatchRenderInput struct {
	Body             string
	CurrentTopicName string
}

// RenderHTML は単一テキストのHTMLをレンダリングする。
// Markdownレンダリング → HTMLサニタイズ → Wikiリンク変換 → 添付ファイルフィルター →
// スタンドアロン画像ラッピングの一連の処理を統合して実行する。
func RenderHTML(
	ctx context.Context,
	body string,
	currentTopicName string,
	spaceID model.SpaceID,
	spaceIdentifier model.SpaceIdentifier,
	resolver PageLocationResolver,
	batchFinder BatchAttachmentFinder,
) (string, error) {
	if body == "" {
		return "", nil
	}

	inputs := []BatchRenderInput{
		{Body: body, CurrentTopicName: currentTopicName},
	}

	results, err := RenderHTMLBatch(ctx, inputs, spaceID, spaceIdentifier, resolver, batchFinder)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", nil
	}

	return results[0], nil
}

// RenderHTMLBatch は複数テキストのHTMLを一括レンダリングする。
// Wikiリンクの解決と添付ファイルの検索をバッチ化してN+1クエリを防止する。
func RenderHTMLBatch(
	ctx context.Context,
	inputs []BatchRenderInput,
	spaceID model.SpaceID,
	spaceIdentifier model.SpaceIdentifier,
	resolver PageLocationResolver,
	batchFinder BatchAttachmentFinder,
) ([]string, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// 1. 全テキストをMarkdown→HTMLに変換し、あわせて添付ファイルIDを収集する。
	// The parse and the rendered HTML are shared with the attachment scan, so each body is read
	// once. The IDs are collected here rather than after the wiki-link conversion, because the
	// conversion rewrites the HTML that the scan reads.
	//
	// [Ja] 解析結果とレンダリング結果を添付ファイルの走査と共有するため、本文の読み取りは1回で済む。
	// IDをWikiリンク変換の後ではなくここで集めるのは、変換が走査の読むHTMLを書き換えるためである。
	htmls := make([]string, len(inputs))
	sources := make([][]byte, len(inputs))
	documents := make([]ast.Node, len(inputs))
	matches := make([][]WikilinkMatch, len(inputs))
	attachmentIDs := make([][]string, len(inputs))
	for i, input := range inputs {
		source, document, bodyHTML := renderBody(input.Body)
		htmls[i] = bodyHTML
		sources[i] = source
		documents[i] = document
		if document == nil {
			continue
		}
		if holdsAttachmentPath(input.Body) {
			attachmentIDs[i] = attachmentIDsOf(scanAttachmentRefMatches(source, document, bodyHTML, true))
		}

		// The matches are read from the normalized source, whose offsets are what the parse and
		// the replacement below refer to.
		//
		// [Ja] 一致は正規化後のソースから読む。解析結果と下の置換が指す位置はそちらのものである。
		matches[i] = ScanWikilinkMatches(string(source), input.CurrentTopicName)
	}

	// 2. 全テキストからWikiリンクキーを収集し一括解決
	var allKeys []WikilinkKey
	for _, bodyMatches := range matches {
		for _, match := range bodyMatches {
			allKeys = append(allKeys, match.Key)
		}
	}

	if len(allKeys) > 0 {
		uniqueKeys := deduplicateWikilinkKeys(allKeys)
		pageLocations, err := resolver.ResolveByKeys(ctx, uniqueKeys, spaceID)
		if err != nil {
			return nil, err
		}
		for i := range inputs {
			htmls[i] = replaceWikilinkMatches(sources[i], documents[i], htmls[i], matches[i], spaceIdentifier, pageLocations)
		}
	}

	// 3. 収集済みの添付ファイルIDを一括検索
	allAttachmentIDStrings := collectAllAttachmentIDs(attachmentIDs)
	if len(allAttachmentIDStrings) > 0 {
		ids := make([]model.AttachmentID, len(allAttachmentIDStrings))
		for i, id := range allAttachmentIDStrings {
			ids[i] = model.AttachmentID(id)
		}
		attachments, err := batchFinder.FindByIDsAndSpace(ctx, ids, spaceID)
		if err != nil {
			return nil, err
		}
		finder := newMapAttachmentFinder(attachments)
		for i := range htmls {
			processed, err := FilterAttachments(ctx, htmls[i], spaceID, finder)
			if err != nil {
				return nil, err
			}
			htmls[i] = processed
		}
	}

	// 4. 画像リンクのラッピング
	for i := range htmls {
		htmls[i] = WrapStandaloneImageLinks(htmls[i])
	}

	return htmls, nil
}

// deduplicateWikilinkKeys はWikiリンクキーの重複を除去する
func deduplicateWikilinkKeys(keys []WikilinkKey) []WikilinkKey {
	seen := make(map[string]bool, len(keys))
	unique := make([]WikilinkKey, 0, len(keys))
	for _, key := range keys {
		k := key.TopicName + "/" + key.PageTitle
		if !seen[k] {
			seen[k] = true
			unique = append(unique, key)
		}
	}
	return unique
}

// collectAllAttachmentIDs は本文ごとの添付ファイルIDを重複なしで1つにまとめる
func collectAllAttachmentIDs(perBody [][]string) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, bodyIDs := range perBody {
		for _, id := range bodyIDs {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// mapAttachmentFinder はマップベースのAttachmentFinder実装。
// バッチ検索結果をマップに保持し、個別の検索をO(1)で処理する。
type mapAttachmentFinder struct {
	attachments map[model.AttachmentID]*model.Attachment
}

func newMapAttachmentFinder(attachments []*model.Attachment) *mapAttachmentFinder {
	m := make(map[model.AttachmentID]*model.Attachment, len(attachments))
	for _, a := range attachments {
		m[a.ID] = a
	}
	return &mapAttachmentFinder{attachments: m}
}

func (f *mapAttachmentFinder) FindByIDAndSpace(_ context.Context, id model.AttachmentID, spaceID model.SpaceID) (*model.Attachment, error) {
	att := f.attachments[id]
	if att != nil && att.SpaceID != spaceID {
		return nil, nil
	}
	return att, nil
}
