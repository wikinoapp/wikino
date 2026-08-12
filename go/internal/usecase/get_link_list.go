package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetLinkListUsecase aggregates the link list shown for a page, together with the backlinks of
// each listed page.
//
// [Ja] GetLinkListUsecase はページのリンク一覧と、一覧した各ページのバックリンクを集約する
// 読み取り UseCase。
type GetLinkListUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	draftPageRepo   *repository.DraftPageRepository
}

// NewGetLinkListUsecase は GetLinkListUsecase を生成する
func NewGetLinkListUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
) *GetLinkListUsecase {
	return &GetLinkListUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		draftPageRepo:   draftPageRepo,
	}
}

// GetLinkListInput holds the input parameters for fetching the link list.
// UserID is nil when the user is not signed in (a public topic's link list is viewable without
// signing in).
//
// [Ja] GetLinkListInput はリンク一覧取得の入力パラメータ。
// UserID は未ログイン時に nil になる (公開トピックのリンク一覧は未ログインでも閲覧できる)。
type GetLinkListInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          *model.UserID
	CurrentPage     int32
	LinkLimit       int32
	BacklinkLimit   int32
}

// GetLinkListOutput はリンク一覧取得の出力
type GetLinkListOutput struct {
	Space            *model.Space
	SpaceMember      *model.SpaceMember
	Page             *model.Page
	TopicMember      *model.TopicMember
	LinkedPages      []*model.Page
	LinkedTotalCount int64
	BacklinksPerPage map[model.PageID]*EditLinkBacklinks
	TopicMap         map[model.TopicID]*model.Topic
	CanUpdatePage    bool
}

// Execute fetches the link list. It returns a *model.AppError with AppErrCodeResourceNotFound
// whenever the page must not be shown to the current viewer, so that the handler cannot tell
// "hidden" and "missing" apart in the response.
//
// [Ja] Execute はリンク一覧を取得する。現在の閲覧者に見せてはいけない場合は
// AppErrCodeResourceNotFound の *model.AppError を返し、レスポンス上で「隠している」と
// 「存在しない」を区別しないようにする。
func (uc *GetLinkListUsecase) Execute(ctx context.Context, input GetLinkListInput) (*GetLinkListOutput, error) {
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
	// The link list follows the member's own draft while it exists, so that links added in the
	// editor show up before the draft is saved. Guests have no draft and always see the saved page.
	//
	// [Ja] リンク一覧は下書きがある間はそちらを優先し、エディタで足したリンクを保存前から
	// 反映する。ゲストは下書きを持たないため常に保存済みのページを見る。
	linkedPageIDs := data.page.LinkedPageIDs
	if data.spaceMember != nil {
		draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, data.page.ID, data.spaceMember.ID, data.space.ID)
		if err != nil {
			return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
		}
		if draftPage != nil {
			linkedPageIDs = draftPage.LinkedPageIDs
		}
	}

	canUpdatePage := access.authorizer(data.page.TopicID).CanUpdatePage()

	if len(linkedPageIDs) == 0 {
		return &GetLinkListOutput{
			Space:         data.space,
			SpaceMember:   data.spaceMember,
			Page:          data.page,
			TopicMember:   data.topicMember,
			CanUpdatePage: canUpdatePage,
		}, nil
	}

	visibility := access.visibility()

	paginatedLinks, err := uc.pageRepo.FindLinkedPagesPaginated(ctx, linkedPageIDs, data.space.ID, visibility, input.CurrentPage, input.LinkLimit)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの取得に失敗: %w", err)
	}

	excludePageIDs := buildExcludePageIDs(data.page.ID, paginatedLinks.Pages)

	backlinkPaginatedMap, err := uc.pageRepo.FindBacklinksForPages(ctx, paginatedLinks.Pages, data.space.ID, visibility, input.BacklinkLimit, excludePageIDs)
	if err != nil {
		return nil, fmt.Errorf("バックリンクの取得に失敗: %w", err)
	}

	var allPageSlices [][]*model.Page
	allPageSlices = append(allPageSlices, paginatedLinks.Pages)
	for _, paginated := range backlinkPaginatedMap {
		allPageSlices = append(allPageSlices, paginated.Pages)
	}

	topicMap := access.topicMapForPages(allPageSlices...)

	backlinksPerPage := make(map[model.PageID]*EditLinkBacklinks, len(backlinkPaginatedMap))
	for pageID, paginated := range backlinkPaginatedMap {
		backlinksPerPage[pageID] = &EditLinkBacklinks{
			Pages:      paginated.Pages,
			TotalCount: paginated.TotalCount,
		}
	}

	return &GetLinkListOutput{
		Space:            data.space,
		SpaceMember:      data.spaceMember,
		Page:             data.page,
		TopicMember:      data.topicMember,
		LinkedPages:      paginatedLinks.Pages,
		LinkedTotalCount: paginatedLinks.TotalCount,
		BacklinksPerPage: backlinksPerPage,
		TopicMap:         topicMap,
		CanUpdatePage:    canUpdatePage,
	}, nil
}

func (uc *GetLinkListUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}
