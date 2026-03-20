package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetLatestPageRevisionsUsecase は各ページの最新リビジョンを取得する読み取りユースケース
type GetLatestPageRevisionsUsecase struct {
	pageRevisionRepo *repository.PageRevisionRepository
}

// NewGetLatestPageRevisionsUsecase は GetLatestPageRevisionsUsecase を生成する
func NewGetLatestPageRevisionsUsecase(
	pageRevisionRepo *repository.PageRevisionRepository,
) *GetLatestPageRevisionsUsecase {
	return &GetLatestPageRevisionsUsecase{
		pageRevisionRepo: pageRevisionRepo,
	}
}

// GetLatestPageRevisionsInput は最新ページリビジョン取得の入力パラメータ
type GetLatestPageRevisionsInput struct {
	DraftPages []*model.DraftPage
	SpaceID    model.SpaceID
}

// GetLatestPageRevisionsOutput は最新ページリビジョン取得の出力パラメータ
type GetLatestPageRevisionsOutput struct {
	PageRevisions map[model.PageID]*model.PageRevision
}

// Execute は各下書きページに対応するページの最新リビジョンを取得する
func (uc *GetLatestPageRevisionsUsecase) Execute(ctx context.Context, input GetLatestPageRevisionsInput) (*GetLatestPageRevisionsOutput, error) {
	pageRevisions := make(map[model.PageID]*model.PageRevision, len(input.DraftPages))

	for _, draftPage := range input.DraftPages {
		latestRevision, err := uc.pageRevisionRepo.FindLatestByPageID(ctx, draftPage.PageID, input.SpaceID)
		if err != nil {
			return nil, fmt.Errorf("ページリビジョンの取得に失敗しました: %w", err)
		}
		if latestRevision == nil {
			return nil, fmt.Errorf("ページ %s のリビジョンが見つかりません", draftPage.PageID)
		}

		pageRevisions[draftPage.PageID] = latestRevision
	}

	return &GetLatestPageRevisionsOutput{
		PageRevisions: pageRevisions,
	}, nil
}
