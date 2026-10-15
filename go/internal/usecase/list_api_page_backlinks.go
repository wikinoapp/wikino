package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ListAPIPageBacklinksUsecaseは、公開APIで保存済みの本文からページにリンクしているページの一覧を取得するユースケース
type ListAPIPageBacklinksUsecase struct {
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewListAPIPageBacklinksUsecaseはListAPIPageBacklinksUsecaseを生成する
func NewListAPIPageBacklinksUsecase(pageRepo *repository.PageRepository, topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *ListAPIPageBacklinksUsecase {
	return &ListAPIPageBacklinksUsecase{
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// ListAPIPageBacklinksInputはページのバックリンク一覧の取得に必要な入力パラメータ
type ListAPIPageBacklinksInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// Afterは前のページの最後のページの位置。この位置より後ろのページを返す。nilなら最初のページを返す
	After *APIPageCursor
	// Limitは1ページの件数 (1以上)
	Limit int32
}

// Executeはトークンが属するスペースのページについて、保存済みの本文 (linked_page_ids) でそのページに
// リンクしているページを更新日時の新しい順に返す。下書きにだけあるリンクは含めない。
// 除くページと未存在の扱いはリンク一覧 (ListAPIPageLinksUsecase) と同じで、自身へのリンクも含める
func (uc *ListAPIPageBacklinksUsecase) Execute(ctx context.Context, input ListAPIPageBacklinksInput) (*ListAPIPageLinksOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	access, err := fetchAPITopicAccess(ctx, uc.topicRepo, uc.topicMemberRepo, input.Principal)
	if err != nil {
		return nil, err
	}
	page, err := findAPIShowablePage(ctx, uc.pageRepo, access, space, input.PageNumber)
	if err != nil {
		return nil, err
	}

	listInput := repository.ListBacklinkedByModifiedAtInput{
		SpaceID:         space.ID,
		PageID:          page.ID,
		VisibleTopicIDs: access.visibility().TopicIDs,
		// 次のページがあるかを知るため、1件多く取得する
		Limit: input.Limit + 1,
	}
	if input.After != nil {
		listInput.AfterModifiedAt = &input.After.ModifiedAt
		listInput.AfterID = &input.After.PageID
	}
	pages, err := uc.pageRepo.ListBacklinkedByModifiedAt(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("バックリンクの一覧の取得に失敗: %w", err)
	}
	return newAPIPageLinksOutput(pages, input.Limit, access), nil
}
