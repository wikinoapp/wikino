package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSuggestionNewUsecase は編集提案作成画面のデータ取得ユースケース
type GetSuggestionNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	draftPageRepo   *repository.DraftPageRepository
}

// NewGetSuggestionNewUsecase は GetSuggestionNewUsecase を生成する
func NewGetSuggestionNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
) *GetSuggestionNewUsecase {
	return &GetSuggestionNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		draftPageRepo:   draftPageRepo,
	}
}

// GetSuggestionNewInput は編集提案作成画面のデータ取得の入力パラメータ
type GetSuggestionNewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          model.UserID
}

// GetSuggestionNewOutput は編集提案作成画面のデータ取得の出力パラメータ
type GetSuggestionNewOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	Topic       *model.Topic
	TopicMember *model.TopicMember
	DraftPages  []*model.DraftPage
}

// Execute は編集提案作成画面用のデータを取得する
func (uc *GetSuggestionNewUsecase) Execute(ctx context.Context, input GetSuggestionNewInput) (*GetSuggestionNewOutput, error) {
	// スペースを取得
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	// スペースメンバーを取得（編集提案作成にはスペースメンバーであることが必須）
	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, nil
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
	topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	// 非公開トピックの場合、スペースオーナーまたはトピックメンバーのみアクセス可能
	if topic.Visibility == model.TopicVisibilityPrivate {
		if spaceMember.Role != model.SpaceMemberRoleOwner && topicMember == nil {
			return nil, nil
		}
	}

	// トピック内の自分の下書きページ一覧を取得（編集提案にリンクされていないもの）
	draftPages, err := uc.draftPageRepo.ListByMemberAndTopic(ctx, spaceMember.ID, topic.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得に失敗: %w", err)
	}

	return &GetSuggestionNewOutput{
		Space:       space,
		SpaceMember: spaceMember,
		Topic:       topic,
		TopicMember: topicMember,
		DraftPages:  draftPages,
	}, nil
}
