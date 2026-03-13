package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTopicDetailUsecase はトピック詳細画面のデータ取得ユースケース
type GetTopicDetailUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	pageRepo        *repository.PageRepository
}

// NewGetTopicDetailUsecase は GetTopicDetailUsecase を生成する
func NewGetTopicDetailUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	pageRepo *repository.PageRepository,
) *GetTopicDetailUsecase {
	return &GetTopicDetailUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		pageRepo:        pageRepo,
	}
}

// GetTopicDetailInput はトピック詳細取得の入力パラメータ
type GetTopicDetailInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          *model.UserID
	Page            int32
	PageLimit       int32
}

// GetTopicDetailOutput はトピック詳細取得の出力
type GetTopicDetailOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	Topic       *model.Topic
	TopicMember *model.TopicMember
	PinnedPages []*model.Page
	Pages       []*model.Page
	TotalCount  int64
}

// Execute はトピック詳細画面に必要なデータを取得する
func (uc *GetTopicDetailUsecase) Execute(ctx context.Context, input GetTopicDetailInput) (*GetTopicDetailOutput, error) {
	// スペースを取得
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	// ログインユーザーのスペースメンバーを取得（未ログインならnil）
	var spaceMember *model.SpaceMember
	if input.UserID != nil {
		spaceMember, err = uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, *input.UserID)
		if err != nil {
			return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
		}
	}

	// トピックを取得
	topic, err := uc.topicRepo.FindBySpaceAndNumber(ctx, space.ID, input.TopicNumber)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, nil
	}

	// トピックメンバーを取得
	var topicMember *model.TopicMember
	if spaceMember != nil {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	// 権限チェック: 非公開トピックはスペースオーナーまたはトピックメンバーのみ閲覧可能
	if topic.Visibility == model.TopicVisibilityPrivate {
		if spaceMember == nil || (spaceMember.Role != model.SpaceMemberRoleOwner && topicMember == nil) {
			return nil, nil
		}
	}

	// ピン留めページを取得
	pinnedPages, err := uc.pageRepo.FindPinnedByTopic(ctx, topic.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("ピン留めページの取得に失敗: %w", err)
	}

	// 通常ページをページネーションで取得
	paginatedResult, err := uc.pageRepo.FindRegularByTopicPaginated(ctx, topic.ID, space.ID, input.Page, input.PageLimit)
	if err != nil {
		return nil, fmt.Errorf("通常ページの取得に失敗: %w", err)
	}

	return &GetTopicDetailOutput{
		Space:       space,
		SpaceMember: spaceMember,
		Topic:       topic,
		TopicMember: topicMember,
		PinnedPages: pinnedPages,
		Pages:       paginatedResult.Pages,
		TotalCount:  paginatedResult.TotalCount,
	}, nil
}
