package markup

import (
	"context"

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
	spaceIdentifier string,
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
	spaceIdentifier string,
	resolver PageLocationResolver,
	batchFinder BatchAttachmentFinder,
) ([]string, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// 1. 全テキストをMarkdown→HTMLに変換
	htmls := make([]string, len(inputs))
	for i, input := range inputs {
		htmls[i] = RenderMarkdown(input.Body)
	}

	// 2. 全テキストからWikiリンクキーを収集し一括解決
	var allKeys []WikilinkKey
	for _, input := range inputs {
		keys := ScanWikilinks(input.Body, input.CurrentTopicName)
		allKeys = append(allKeys, keys...)
	}

	if len(allKeys) > 0 {
		uniqueKeys := deduplicateWikilinkKeys(allKeys)
		pageLocations, err := resolver.ResolveByKeys(ctx, uniqueKeys, spaceID)
		if err != nil {
			return nil, err
		}
		for i, input := range inputs {
			htmls[i] = ReplaceWikilinks(htmls[i], input.CurrentTopicName, spaceIdentifier, pageLocations)
		}
	}

	// 3. 全HTMLから添付ファイルIDを収集し一括検索
	allAttachmentIDStrings := collectAllAttachmentIDs(htmls)
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

// collectAllAttachmentIDs は複数のHTML文字列から添付ファイルIDを重複なしで収集する
func collectAllAttachmentIDs(htmls []string) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, h := range htmls {
		for _, id := range ExtractAttachmentIDs(h) {
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

func (f *mapAttachmentFinder) FindByIDAndSpace(_ context.Context, id model.AttachmentID, _ model.SpaceID) (*model.Attachment, error) {
	return f.attachments[id], nil
}
