package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetBacklinkListUsecase aggregates the backlinks of one page listed in another page's link list.
//
// [Ja] GetBacklinkListUsecase はあるページのリンク一覧に並ぶ 1 ページについて、その
// バックリンク一覧を集約する読み取り UseCase。
type GetBacklinkListUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetBacklinkListUsecase は GetBacklinkListUsecase を生成する
func NewGetBacklinkListUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetBacklinkListUsecase {
	return &GetBacklinkListUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetBacklinkListInput holds the input parameters for fetching a linked page's backlinks.
// UserID is nil when the user is not signed in.
//
// [Ja] GetBacklinkListInput はバックリンク一覧取得の入力パラメータ。
// UserID は未ログイン時に nil になる。
type GetBacklinkListInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	PageNumber       int32
	LinkedPageNumber int32
	UserID           *model.UserID
	CurrentPage      int32
	Limit            int32
}

// GetBacklinkListOutput はバックリンク一覧取得の出力
type GetBacklinkListOutput struct {
	Space         *model.Space
	SpaceMember   *model.SpaceMember
	Page          *model.Page
	TopicMember   *model.TopicMember
	LinkedPage    *model.Page
	Backlinks     []*model.Page
	TotalCount    int64
	TopicMap      map[model.TopicID]*model.Topic
	CanUpdatePage bool
}

// Execute fetches the backlinks of the linked page. Both the page holding the link list and the
// linked page itself must be visible to the current viewer; otherwise a *model.AppError with
// AppErrCodeResourceNotFound is returned.
//
// [Ja] Execute はリンク先ページのバックリンク一覧を取得する。リンク一覧を持つページと
// リンク先ページの両方が現在の閲覧者に見えることを要求し、そうでなければ
// AppErrCodeResourceNotFound の *model.AppError を返す。
func (uc *GetBacklinkListUsecase) Execute(ctx context.Context, input GetBacklinkListInput) (*GetBacklinkListOutput, error) {
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

	linkedPage, err := uc.pageRepo.FindBySpaceAndNumber(ctx, data.space.ID, model.PageNumber(input.LinkedPageNumber))
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの取得に失敗: %w", err)
	}
	if linkedPage == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// The backlinks belong to the linked page, so its own topic decides whether this list may be
	// shown at all. Without this check a viewer could read the backlinks of a page in a topic they
	// cannot open, just by walking the URL.
	//
	// [Ja] 返すバックリンクはリンク先ページのものなので、この一覧を見せてよいかはリンク先ページの
	// トピックが決める。この判定が無いと、開けないトピックのページのバックリンクを URL 直打ちで
	// 読めてしまう。
	if !access.canShowPage(linkedPage) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	excludePageIDs := []model.PageID{data.page.ID, linkedPage.ID}
	paginatedBacklinks, err := uc.pageRepo.FindBacklinkedPagesPaginated(ctx, linkedPage.ID, data.space.ID, access.visibility(), input.CurrentPage, input.Limit, excludePageIDs)
	if err != nil {
		return nil, fmt.Errorf("バックリンクの取得に失敗: %w", err)
	}

	topicMap := access.topicMapForPages(paginatedBacklinks.Pages)

	return &GetBacklinkListOutput{
		Space:         data.space,
		SpaceMember:   data.spaceMember,
		Page:          data.page,
		TopicMember:   data.topicMember,
		LinkedPage:    linkedPage,
		Backlinks:     paginatedBacklinks.Pages,
		TotalCount:    paginatedBacklinks.TotalCount,
		TopicMap:      topicMap,
		CanUpdatePage: access.authorizer(data.page.TopicID).CanUpdatePage(),
	}, nil
}

func (uc *GetBacklinkListUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}
