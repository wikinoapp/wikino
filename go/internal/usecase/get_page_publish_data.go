package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPagePublishDataUsecase はページ公開に必要な事前計算データを取得するユースケース
type GetPagePublishDataUsecase struct {
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
}

// NewGetPagePublishDataUsecase は GetPagePublishDataUsecase を生成する
func NewGetPagePublishDataUsecase(
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
) *GetPagePublishDataUsecase {
	return &GetPagePublishDataUsecase{
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
	}
}

// GetPagePublishDataInput はページ公開事前計算の入力パラメータ
type GetPagePublishDataInput struct {
	Body    string
	PageID  model.PageID
	SpaceID model.SpaceID
}

// GetPagePublishDataOutput はページ公開事前計算の出力パラメータ
type GetPagePublishDataOutput struct {
	BodyHTML                  string
	FeaturedImageAttachmentID *model.AttachmentID
	AttachmentRefsToAdd       []model.AttachmentID
	AttachmentRefsToRemove    []model.AttachmentID
}

// Execute はページ公開に必要な事前計算データを取得する
func (uc *GetPagePublishDataUsecase) Execute(ctx context.Context, input GetPagePublishDataInput) (*GetPagePublishDataOutput, error) {
	// 1. Markdownレンダリング
	bodyHTML := markup.RenderMarkdown(input.Body)

	// 2. 添付ファイル参照の差分計算（FilterAttachments前に実行。FilterAttachmentsが/attachments/{id}をdata属性に変換するため）
	toAdd, toRemove, err := calculateAttachmentRefDiff(ctx, bodyHTML, input.PageID, input.SpaceID, uc.attachmentRepo, uc.pageAttachmentRefRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイル参照の差分計算に失敗しました: %w", err)
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

	return &GetPagePublishDataOutput{
		BodyHTML:                  bodyHTML,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		AttachmentRefsToAdd:       toAdd,
		AttachmentRefsToRemove:    toRemove,
	}, nil
}
