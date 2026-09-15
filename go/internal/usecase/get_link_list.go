package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetLinkListUsecaseはページのリンク一覧と、一覧した各ページのバックリンクを集約する
// 読み取りUseCase。
type GetLinkListUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	draftPageRepo   *repository.DraftPageRepository
}

// NewGetLinkListUsecaseはGetLinkListUsecaseを生成する
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

// LinkListSourceは一覧が編集中の下書きと保存済みページのどちらを使うかを選ぶ。ゼロ値は
// LinkListSourceDraftのため、本フィールドを省いた呼び出し元は下書き優先の一覧になる。
type LinkListSource uint8

const (
	// LinkListSourceDraftは現在のメンバーの下書きがあれば優先する。
	LinkListSourceDraft LinkListSource = iota

	// LinkListSourceSavedは常に公開済みページの保存済みリンクを使う。
	LinkListSourceSaved
)

// GetLinkListInputはリンク一覧取得の入力パラメータ。
// UserIDは未ログイン時にnilになる (公開トピックのリンク一覧は未ログインでも閲覧できる)。
type GetLinkListInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          *model.UserID
	CurrentPage     int32
	LinkLimit       int32
	BacklinkLimit   int32
	Source          LinkListSource
}

// GetLinkListOutputはリンク一覧取得の出力
type GetLinkListOutput struct {
	Space            *model.Space
	SpaceMember      *model.SpaceMember
	Page             *model.Page
	TopicMember      *model.TopicMember
	LinkedPages      []*model.Page
	LinkedTotalCount int64
	BacklinksPerPage map[model.PageID]*LinkedPageBacklinks
	TopicMap         map[model.TopicID]*model.Topic
	CanUpdatePage    bool
}

// Executeはリンク一覧を取得する。現在の閲覧者に見せてはいけない場合は
// AppErrCodeResourceNotFoundの *model.AppErrorを返し、レスポンス上で「隠している」と
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
	// 編集画面は下書きがある間はそちらを優先する。一方、公開ページ表示画面は
	// LinkListSourceSavedを渡し、一覧の全ページを同じ公開済みデータ集合に保つ。
	linkedPageIDs := data.page.LinkedPageIDs
	if input.Source != LinkListSourceSaved && data.spaceMember != nil {
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

	// 本エンドポイントは2画面が初回に描画するのと同じ一覧の続きを返すため、クエリをここで繰り返す
	// のではなく同じ手順で解決する。
	listing, err := fetchLinkedPageListing(ctx, uc.pageRepo, linkedPageListingInput{
		PageID:        data.page.ID,
		LinkedPageIDs: linkedPageIDs,
		SpaceID:       data.space.ID,
		Visibility:    access.visibility(),
		LinkPage:      input.CurrentPage,
		LinkLimit:     input.LinkLimit,
		BacklinkLimit: input.BacklinkLimit,
	})
	if err != nil {
		return nil, err
	}

	backlinksPerPage, backlinkGroups := newLinkedPageBacklinksMap(listing.backlinks)

	allPageSlices := append([][]*model.Page{listing.paginatedLinks.Pages}, backlinkGroups...)
	topicMap := access.topicMapForPages(allPageSlices...)

	return &GetLinkListOutput{
		Space:            data.space,
		SpaceMember:      data.spaceMember,
		Page:             data.page,
		TopicMember:      data.topicMember,
		LinkedPages:      listing.paginatedLinks.Pages,
		LinkedTotalCount: listing.paginatedLinks.TotalCount,
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
