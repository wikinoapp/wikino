package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSuggestionListUsecase は編集提案一覧取得ユースケース
type GetSuggestionListUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	suggestionRepo  *repository.SuggestionRepository
	userRepo        *repository.UserRepository
}

// NewGetSuggestionListUsecase は GetSuggestionListUsecase を生成する
func NewGetSuggestionListUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	userRepo *repository.UserRepository,
) *GetSuggestionListUsecase {
	return &GetSuggestionListUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		suggestionRepo:  suggestionRepo,
		userRepo:        userRepo,
	}
}

// GetSuggestionListInput は編集提案一覧取得の入力パラメータ
type GetSuggestionListInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          *model.UserID
	Statuses        []model.SuggestionStatus
}

// GetSuggestionListOutput は編集提案一覧取得の出力
type GetSuggestionListOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	Topic       *model.Topic
	TopicMember *model.TopicMember
	Suggestions []*model.Suggestion
	UserMap     map[model.SpaceMemberID]*model.User
	OpenCount   int64
	ClosedCount int64
}

// Execute は編集提案一覧を取得する
func (uc *GetSuggestionListUsecase) Execute(ctx context.Context, input GetSuggestionListInput) (*GetSuggestionListOutput, error) {
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

	// 指定ステータスの編集提案を取得
	suggestions, err := uc.suggestionRepo.ListByTopicAndStatuses(ctx, topic.ID, space.ID, input.Statuses)
	if err != nil {
		return nil, fmt.Errorf("編集提案一覧の取得に失敗: %w", err)
	}

	// オープン件数を取得（下書き・オープン）
	openCount, err := uc.suggestionRepo.CountByTopicAndStatuses(ctx, topic.ID, space.ID, []model.SuggestionStatus{
		model.SuggestionStatusDraft,
		model.SuggestionStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("オープン件数の取得に失敗: %w", err)
	}

	// クローズ件数を取得（反映済み・クローズ）
	closedCount, err := uc.suggestionRepo.CountByTopicAndStatuses(ctx, topic.ID, space.ID, []model.SuggestionStatus{
		model.SuggestionStatusApplied,
		model.SuggestionStatusClosed,
	})
	if err != nil {
		return nil, fmt.Errorf("クローズ件数の取得に失敗: %w", err)
	}

	// 作成者情報を取得
	userMap, err := uc.buildUserMap(ctx, suggestions, space.ID)
	if err != nil {
		return nil, fmt.Errorf("作成者情報の取得に失敗: %w", err)
	}

	return &GetSuggestionListOutput{
		Space:       space,
		SpaceMember: spaceMember,
		Topic:       topic,
		TopicMember: topicMember,
		Suggestions: suggestions,
		UserMap:     userMap,
		OpenCount:   openCount,
		ClosedCount: closedCount,
	}, nil
}

// buildUserMap は編集提案の作成者のユーザー情報をマップで返す
func (uc *GetSuggestionListUsecase) buildUserMap(ctx context.Context, suggestions []*model.Suggestion, spaceID model.SpaceID) (map[model.SpaceMemberID]*model.User, error) {
	if len(suggestions) == 0 {
		return map[model.SpaceMemberID]*model.User{}, nil
	}

	// SpaceMemberIDを収集（重複排除）
	memberIDSet := make(map[model.SpaceMemberID]struct{})
	for _, s := range suggestions {
		memberIDSet[s.CreatedSpaceMemberID] = struct{}{}
	}
	memberIDs := make([]model.SpaceMemberID, 0, len(memberIDSet))
	for id := range memberIDSet {
		memberIDs = append(memberIDs, id)
	}

	// SpaceMemberを一括取得
	members, err := uc.spaceMemberRepo.FindByIDs(ctx, memberIDs, spaceID)
	if err != nil {
		return nil, err
	}

	// UserIDを収集
	userIDs := make([]model.UserID, 0, len(members))
	memberToUser := make(map[model.SpaceMemberID]model.UserID, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
		memberToUser[m.ID] = m.UserID
	}

	// Userを一括取得
	users, err := uc.userRepo.FindByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	userByID := make(map[model.UserID]*model.User, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}

	// SpaceMemberID → User のマップを構築
	result := make(map[model.SpaceMemberID]*model.User, len(memberIDs))
	for memberID, userID := range memberToUser {
		if u, ok := userByID[userID]; ok {
			result[memberID] = u
		}
	}

	return result, nil
}
