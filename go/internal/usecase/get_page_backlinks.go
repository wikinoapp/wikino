package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageBacklinksUsecase aggregates the backlinks of a page itself (as opposed to the backlinks
// of the pages listed in its link list).
//
// [Ja] GetPageBacklinksUsecase はページ自身のバックリンク一覧を集約する読み取り UseCase
// (リンク一覧に並ぶ各ページのバックリンクとは別物)。
type GetPageBacklinksUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetPageBacklinksUsecase は GetPageBacklinksUsecase を生成する
func NewGetPageBacklinksUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetPageBacklinksUsecase {
	return &GetPageBacklinksUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetPageBacklinksInput holds the input parameters for fetching a page's backlinks.
// UserID is nil when the user is not signed in.
//
// [Ja] GetPageBacklinksInput はページレベルのバックリンク一覧取得の入力パラメータ。
// UserID は未ログイン時に nil になる。
type GetPageBacklinksInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          *model.UserID
	CurrentPage     int32
	Limit           int32
}

// GetPageBacklinksOutput はページレベルのバックリンク一覧取得の出力
type GetPageBacklinksOutput struct {
	Space         *model.Space
	SpaceMember   *model.SpaceMember
	Page          *model.Page
	TopicMember   *model.TopicMember
	Backlinks     []*model.Page
	TotalCount    int64
	TopicMap      map[model.TopicID]*model.Topic
	CanUpdatePage bool
}

// Execute fetches a page's backlinks. It returns a *model.AppError with
// AppErrCodeResourceNotFound whenever the page must not be shown to the current viewer.
//
// [Ja] Execute はページレベルのバックリンク一覧を取得する。現在の閲覧者に見せてはいけない場合は
// AppErrCodeResourceNotFound の *model.AppError を返す。
func (uc *GetPageBacklinksUsecase) Execute(ctx context.Context, input GetPageBacklinksInput) (*GetPageBacklinksOutput, error) {
	data, err := fetchPageAccessDataAllowingGuest(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	access, err := fetchTopicAccess(ctx, uc.pageAccessRepos(), data.space.ID, data.spaceMember)
	if err != nil {
		return nil, err
	}
	if !access.canShowPage(data.page) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	paginatedBacklinks, err := uc.pageRepo.FindBacklinkedPagesPaginated(ctx, data.page.ID, data.space.ID, access.visibility(), input.CurrentPage, input.Limit, nil)
	if err != nil {
		return nil, fmt.Errorf("ページレベルのバックリンクの取得に失敗: %w", err)
	}

	topicMap := access.topicMapForPages(paginatedBacklinks.Pages)

	return &GetPageBacklinksOutput{
		Space:         data.space,
		SpaceMember:   data.spaceMember,
		Page:          data.page,
		TopicMember:   data.topicMember,
		Backlinks:     paginatedBacklinks.Pages,
		TotalCount:    paginatedBacklinks.TotalCount,
		TopicMap:      topicMap,
		CanUpdatePage: access.authorizer(data.page.TopicID).CanUpdatePage(),
	}, nil
}

func (uc *GetPageBacklinksUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}
