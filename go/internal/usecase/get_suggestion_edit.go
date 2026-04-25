package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSuggestionEditUsecase は編集提案編集フォーム用のデータ取得ユースケース
type GetSuggestionEditUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	suggestionRepo  *repository.SuggestionRepository
	userRepo        *repository.UserRepository
}

// NewGetSuggestionEditUsecase は GetSuggestionEditUsecase を生成する
func NewGetSuggestionEditUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	userRepo *repository.UserRepository,
) *GetSuggestionEditUsecase {
	return &GetSuggestionEditUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		suggestionRepo:  suggestionRepo,
		userRepo:        userRepo,
	}
}

// GetSuggestionEditInput は編集提案編集フォーム用データ取得の入力パラメータ
type GetSuggestionEditInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	UserID           model.UserID
}

// GetSuggestionEditOutput は編集提案編集フォーム用データ取得の出力
type GetSuggestionEditOutput struct {
	Space                      *model.Space
	Topic                      *model.Topic
	Suggestion                 *model.Suggestion
	UserMap                    map[model.SpaceMemberID]*model.User
	CanUpdateSuggestion        bool
	CanUpdateSuggestionComment bool
}

// Execute は編集提案編集フォームに必要なデータを取得する
func (uc *GetSuggestionEditUsecase) Execute(ctx context.Context, input GetSuggestionEditInput) (*GetSuggestionEditOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
		return nil, nil
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, nil
	}

	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, suggestion.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, nil
	}

	topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	// 権限チェック
	authorizer := newAuthorizer(spaceMember, topicMember)
	if !authorizer.CanShowTopic(topic) {
		return nil, nil
	}

	// 作成者のユーザー情報を取得
	userMap, err := buildUserMapBySpaceMemberIDs(ctx, uc.spaceMemberRepo, uc.userRepo, []model.SpaceMemberID{suggestion.CreatedSpaceMemberID}, space.ID)
	if err != nil {
		return nil, fmt.Errorf("ユーザー情報の取得に失敗: %w", err)
	}

	// 認可チェック
	canUpdateSuggestion := authorizer.CanUpdateSuggestion(suggestion)
	canUpdateSuggestionComment := authorizer.CanUpdateSuggestionComment(suggestion)

	return &GetSuggestionEditOutput{
		Space:                      space,
		Topic:                      topic,
		Suggestion:                 suggestion,
		UserMap:                    userMap,
		CanUpdateSuggestion:        canUpdateSuggestion,
		CanUpdateSuggestionComment: canUpdateSuggestionComment,
	}, nil
}
