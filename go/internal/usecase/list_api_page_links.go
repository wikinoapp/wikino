package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ListAPIPageLinksUsecaseは、公開APIでページの保存済みの本文がリンクしているページの一覧を取得するユースケース
type ListAPIPageLinksUsecase struct {
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewListAPIPageLinksUsecaseはListAPIPageLinksUsecaseを生成する
func NewListAPIPageLinksUsecase(pageRepo *repository.PageRepository, topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *ListAPIPageLinksUsecase {
	return &ListAPIPageLinksUsecase{
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// ListAPIPageLinksInputはページのリンク一覧の取得に必要な入力パラメータ
type ListAPIPageLinksInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// Afterは前のページの最後のページの位置。この位置より後ろのページを返す。nilなら最初のページを返す
	After *APIPageCursor
	// Limitは1ページの件数 (1以上)
	Limit int32
}

// ListAPIPageLinksOutputはページのリンク一覧・バックリンク一覧の取得結果
type ListAPIPageLinksOutput struct {
	// Pagesは閲覧者が開けるページを更新日時の新しい順に並べたもの
	Pages []*model.Page
	// TopicsはPagesの各ページが属するトピック
	Topics map[model.TopicID]*model.Topic
	// HasNextは次のページがあるかどうか
	HasNext bool
}

// Executeはトークンが属するスペースのページについて、保存済みの本文 (linked_page_ids) がリンクしている
// ページを更新日時の新しい順に返す。下書きにだけあるリンクは含めない。
// Web画面のリンク一覧と同じく、ゴミ箱・廃棄済みのページと閲覧者が開けないトピックのページを除き、
// Wikiリンクで自動作成された未公開のページは含める。自身へのリンクもWeb画面と同じく含める。
// ページの未存在と見られないページの扱いはページの取得 (getPage) と同じ
func (uc *ListAPIPageLinksUsecase) Execute(ctx context.Context, input ListAPIPageLinksInput) (*ListAPIPageLinksOutput, error) {
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
	if len(page.LinkedPageIDs) == 0 {
		return &ListAPIPageLinksOutput{Pages: []*model.Page{}, Topics: map[model.TopicID]*model.Topic{}}, nil
	}

	listInput := repository.ListLinkedByModifiedAtInput{
		SpaceID:         space.ID,
		PageIDs:         page.LinkedPageIDs,
		VisibleTopicIDs: access.visibility().TopicIDs,
		// 次のページがあるかを知るため、1件多く取得する
		Limit: input.Limit + 1,
	}
	if input.After != nil {
		listInput.AfterModifiedAt = &input.After.ModifiedAt
		listInput.AfterID = &input.After.PageID
	}
	pages, err := uc.pageRepo.ListLinkedByModifiedAt(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("リンク先のページの一覧の取得に失敗: %w", err)
	}
	return newAPIPageLinksOutput(pages, input.Limit, access), nil
}

// findAPIShowablePageはリンク一覧・バックリンク一覧の起点のページを取得する。
// ページの取得 (getPage) と同じく、存在しないページに加え、ゴミ箱のページ・廃棄済みのトピックのページ・
// 閲覧者が開けないトピックのページも、存在を秘匿するため未存在として扱う
func findAPIShowablePage(ctx context.Context, pageRepo *repository.PageRepository, access *topicAccess, space *model.Space, pageNumber int32) (*model.Page, error) {
	page, err := pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(pageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	// canShowPageはゴミ箱を開ける権限を持つメンバーにゴミ箱のページを見せるため、isAPIVisiblePageで先に除く
	if !isAPIVisiblePage(page) || !access.canShowPage(page) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}
	return page, nil
}

// newAPIPageLinksOutputは1件多く取得したページから、limit件の一覧と次のページの有無を組み立てる
func newAPIPageLinksOutput(pages []*model.Page, limit int32, access *topicAccess) *ListAPIPageLinksOutput {
	hasNext := len(pages) > int(limit)
	if hasNext {
		pages = pages[:limit]
	}
	return &ListAPIPageLinksOutput{
		Pages:   pages,
		Topics:  access.topicMapForPages(pages),
		HasNext: hasNext,
	}
}
