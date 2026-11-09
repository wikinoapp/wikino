package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// feedEntryLimitはフィードに載せるページの件数
const feedEntryLimit = 15

// feedContentReposは、フィードに載せるページの本文の描画に使うRepository
type feedContentRepos struct {
	topicRepo      *repository.TopicRepository
	pageRepo       *repository.PageRepository
	attachmentRepo *repository.AttachmentRepository
}

// renderFeedContentsは、フィードに載せるページの本文を全文のHTMLにし、ページIDごとに返す。
//
// 描画はページ表示画面と同じ経路 (Wikiリンクの解決に副作用の無いリゾルバを使うRenderHTMLBatch) を通し、
// 添付ファイルとリンクをフィードリーダーで開ける絶対URLに書き換える。
// 描画は件数によらず一定のクエリ数で行う (トピック名・Wikiリンク・添付ファイルをまとめて引く)。
func renderFeedContents(
	ctx context.Context,
	repos feedContentRepos,
	appURL string,
	space *model.Space,
	pages []*model.Page,
) (map[model.PageID]string, error) {
	if len(pages) == 0 {
		return map[model.PageID]string{}, nil
	}

	topicNames, err := feedTopicNames(ctx, repos.topicRepo, space.ID, pages)
	if err != nil {
		return nil, err
	}

	inputs := make([]markup.BatchRenderInput, len(pages))
	for i, pg := range pages {
		inputs[i] = markup.BatchRenderInput{Body: pg.Body, CurrentTopicName: topicNames[pg.TopicID]}
	}

	resolver := &previewPageLocationResolver{topicRepo: repos.topicRepo, pageRepo: repos.pageRepo}
	htmls, err := markup.RenderHTMLBatch(ctx, inputs, space.ID, space.Identifier, resolver, repos.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("フィードの本文のレンダリングに失敗: %w", err)
	}

	contents := make(map[model.PageID]string, len(pages))
	for i, pg := range pages {
		content, err := markup.AbsolutizeURLs(htmls[i], appURL, model.PageWebURL(appURL, space, pg.Number))
		if err != nil {
			return nil, fmt.Errorf("フィードの本文のURLの書き換えに失敗: %w", err)
		}
		contents[pg.ID] = content
	}

	return contents, nil
}

// feedTopicNamesは、ページの属するトピックの名前をトピックIDごとに返す。
// 本文の `[[ページ名]]` を、そのページのトピックのページとして解決するために使う。
func feedTopicNames(
	ctx context.Context,
	topicRepo *repository.TopicRepository,
	spaceID model.SpaceID,
	pages []*model.Page,
) (map[model.TopicID]string, error) {
	seen := make(map[model.TopicID]bool, len(pages))
	topicIDs := make([]model.TopicID, 0, len(pages))
	for _, pg := range pages {
		if seen[pg.TopicID] {
			continue
		}
		seen[pg.TopicID] = true
		topicIDs = append(topicIDs, pg.TopicID)
	}

	topics, err := topicRepo.FindByIDsAndSpace(ctx, topicIDs, spaceID)
	if err != nil {
		return nil, fmt.Errorf("フィードのページのトピックの取得に失敗: %w", err)
	}

	names := make(map[model.TopicID]string, len(topics))
	for _, topic := range topics {
		names[topic.ID] = topic.Name
	}

	return names, nil
}
