package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSuggestionBodyHTMLUsecase は編集提案の本文HTMLを生成する読み取りユースケース
type GetSuggestionBodyHTMLUsecase struct {
	topicRepo *repository.TopicRepository
	pageRepo  *repository.PageRepository
}

// NewGetSuggestionBodyHTMLUsecase は GetSuggestionBodyHTMLUsecase を生成する
func NewGetSuggestionBodyHTMLUsecase(
	topicRepo *repository.TopicRepository,
	pageRepo *repository.PageRepository,
) *GetSuggestionBodyHTMLUsecase {
	return &GetSuggestionBodyHTMLUsecase{
		topicRepo: topicRepo,
		pageRepo:  pageRepo,
	}
}

// GetSuggestionBodyHTMLInput は編集提案の本文HTML生成の入力パラメータ
type GetSuggestionBodyHTMLInput struct {
	Body             string
	CurrentTopicName string
	SpaceID          model.SpaceID
	SpaceIdentifier  model.SpaceIdentifier
}

// GetSuggestionBodyHTMLOutput は編集提案の本文HTML生成の出力パラメータ
type GetSuggestionBodyHTMLOutput struct {
	BodyHTML string
}

// Execute は編集提案の本文をMarkdownからHTMLに変換し、Wikiリンクを解決する
func (uc *GetSuggestionBodyHTMLUsecase) Execute(ctx context.Context, input GetSuggestionBodyHTMLInput) (*GetSuggestionBodyHTMLOutput, error) {
	bodyHTML := markup.RenderMarkdown(input.Body)

	pageLocations, err := resolveLinkedPages(ctx, input.Body, input.CurrentTopicName, input.SpaceID, uc.topicRepo, uc.pageRepo)
	if err != nil {
		return nil, fmt.Errorf("wikiリンクの解析に失敗しました: %w", err)
	}
	if len(pageLocations) > 0 {
		bodyHTML = markup.ReplaceWikilinks(bodyHTML, input.CurrentTopicName, input.SpaceIdentifier, pageLocations)
	}

	bodyHTML = markup.WrapStandaloneImageLinks(bodyHTML)

	return &GetSuggestionBodyHTMLOutput{
		BodyHTML: bodyHTML,
	}, nil
}

// resolveLinkedPages はWikiリンクを解析し、既存ページへのリンク情報を返す。
// resolveAndCreateLinkedPagesと異なり、リンク先ページの自動作成は行わない。
func resolveLinkedPages(
	ctx context.Context,
	body string,
	currentTopicName string,
	spaceID model.SpaceID,
	topicRepo *repository.TopicRepository,
	pageRepo *repository.PageRepository,
) ([]markup.PageLocation, error) {
	keys := markup.ScanWikilinks(body, currentTopicName)
	if len(keys) == 0 {
		return nil, nil
	}

	topicNames := uniqueTopicNames(keys)
	topics, err := topicRepo.FindBySpaceAndNames(ctx, spaceID, topicNames)
	if err != nil {
		return nil, err
	}
	topicMap := make(map[string]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.Name] = t
	}

	var pageLocations []markup.PageLocation
	seen := make(map[string]bool)

	for _, key := range keys {
		lookupKey := key.TopicName + "/" + key.PageTitle
		if seen[lookupKey] {
			continue
		}
		seen[lookupKey] = true

		topic := topicMap[key.TopicName]
		if topic == nil {
			continue
		}

		page, err := pageRepo.FindByTopicAndTitle(ctx, topic.ID, key.PageTitle, spaceID)
		if err != nil {
			return nil, err
		}
		if page == nil {
			continue
		}

		pageTitle := key.PageTitle
		if page.Title != nil {
			pageTitle = *page.Title
		}
		pageLocations = append(pageLocations, markup.PageLocation{
			Key:        key,
			TopicName:  topic.Name,
			PageID:     page.ID,
			PageNumber: int(page.Number),
			PageTitle:  pageTitle,
		})
	}

	return pageLocations, nil
}
