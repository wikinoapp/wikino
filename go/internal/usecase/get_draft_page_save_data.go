package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetDraftPageSaveDataUsecase は下書き保存に必要な事前計算データを取得するユースケース
type GetDraftPageSaveDataUsecase struct {
	attachmentRepo *repository.AttachmentRepository
	topicRepo      *repository.TopicRepository
}

// NewGetDraftPageSaveDataUsecase は GetDraftPageSaveDataUsecase を生成する
func NewGetDraftPageSaveDataUsecase(
	attachmentRepo *repository.AttachmentRepository,
	topicRepo *repository.TopicRepository,
) *GetDraftPageSaveDataUsecase {
	return &GetDraftPageSaveDataUsecase{
		attachmentRepo: attachmentRepo,
		topicRepo:      topicRepo,
	}
}

// GetDraftPageSaveDataInput は下書き保存事前計算の入力パラメータ
type GetDraftPageSaveDataInput struct {
	Body             string
	SpaceID          model.SpaceID
	CurrentTopicName string
}

// GetDraftPageSaveDataOutput は下書き保存事前計算の出力パラメータ
type GetDraftPageSaveDataOutput struct {
	BodyHTML                  string
	FeaturedImageAttachmentID *model.AttachmentID
	WikilinkKeys              []markup.WikilinkKey
	TopicMap                  map[string]*model.Topic
}

// Execute は下書き保存に必要な事前計算データを取得する
func (uc *GetDraftPageSaveDataUsecase) Execute(ctx context.Context, input GetDraftPageSaveDataInput) (*GetDraftPageSaveDataOutput, error) {
	// 1. Markdownレンダリング
	bodyHTML := markup.RenderMarkdown(input.Body)

	// 2. Wikiリンクのスキャン・トピック検索
	wikilinkKeys, topicMap, err := scanAndLookupWikilinks(ctx, input.Body, input.CurrentTopicName, input.SpaceID, uc.topicRepo)
	if err != nil {
		return nil, fmt.Errorf("wikiリンクのスキャンに失敗しました: %w", err)
	}

	// 3. アイキャッチ画像の抽出
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, input.Body, input.SpaceID, uc.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の抽出に失敗しました: %w", err)
	}

	// 4. 添付ファイルフィルター
	bodyHTML, err = markup.FilterAttachments(ctx, bodyHTML, input.SpaceID, uc.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのフィルター処理に失敗しました: %w", err)
	}

	// 5. スタンドアロン画像のラッピング
	bodyHTML = markup.WrapStandaloneImageLinks(bodyHTML)

	return &GetDraftPageSaveDataOutput{
		BodyHTML:                  bodyHTML,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		WikilinkKeys:              wikilinkKeys,
		TopicMap:                  topicMap,
	}, nil
}
