package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSuggestionDetailUsecase は編集提案詳細取得ユースケース
type GetSuggestionDetailUsecase struct {
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	suggestionRepo        *repository.SuggestionRepository
	suggestionPageRepo    *repository.SuggestionPageRepository
	suggestionCommentRepo *repository.SuggestionCommentRepository
	pageRepo              *repository.PageRepository
	userRepo              *repository.UserRepository
}

// NewGetSuggestionDetailUsecase は GetSuggestionDetailUsecase を生成する
func NewGetSuggestionDetailUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionCommentRepo *repository.SuggestionCommentRepository,
	pageRepo *repository.PageRepository,
	userRepo *repository.UserRepository,
) *GetSuggestionDetailUsecase {
	return &GetSuggestionDetailUsecase{
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		suggestionRepo:        suggestionRepo,
		suggestionPageRepo:    suggestionPageRepo,
		suggestionCommentRepo: suggestionCommentRepo,
		pageRepo:              pageRepo,
		userRepo:              userRepo,
	}
}

// GetSuggestionDetailInput は編集提案詳細取得の入力パラメータ
type GetSuggestionDetailInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	UserID           *model.UserID
}

// GetSuggestionDetailOutput は編集提案詳細取得の出力
type GetSuggestionDetailOutput struct {
	Space                      *model.Space
	SpaceMember                *model.SpaceMember
	Topic                      *model.Topic
	TopicMember                *model.TopicMember
	Suggestion                 *model.Suggestion
	SuggestionPages            []*model.SuggestionPage
	Pages                      []*model.Page
	Comments                   []*model.SuggestionComment
	UserMap                    map[model.SpaceMemberID]*model.User
	CanApplySuggestion         bool
	CanCloseSuggestion         bool
	CanUpdateSuggestion        bool
	CanUpdateSuggestionComment bool
}

// Execute は編集提案詳細を取得する
func (uc *GetSuggestionDetailUsecase) Execute(ctx context.Context, input GetSuggestionDetailInput) (*GetSuggestionDetailOutput, error) {
	// スペースを取得
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	// 編集提案を取得（スペースIDと番号で検索）
	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
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

	// トピックを取得（編集提案のTopicIDから逆引き）
	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, suggestion.TopicID)
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

	// 編集提案ページ一覧を取得
	suggestionPages, err := uc.suggestionPageRepo.ListBySuggestionID(ctx, suggestion.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("編集提案ページ一覧の取得に失敗: %w", err)
	}

	// 編集提案ページに対応する元ページを取得
	var pages []*model.Page
	if len(suggestionPages) > 0 {
		pageIDs := make([]model.PageID, len(suggestionPages))
		for i, sp := range suggestionPages {
			pageIDs[i] = sp.PageID
		}
		pages, err = uc.pageRepo.FindByIDs(ctx, pageIDs, space.ID)
		if err != nil {
			return nil, fmt.Errorf("ページの取得に失敗: %w", err)
		}
	}

	// コメント一覧を取得
	comments, err := uc.suggestionCommentRepo.ListBySuggestionID(ctx, suggestion.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("コメント一覧の取得に失敗: %w", err)
	}

	// ユーザー情報を取得（作成者 + コメント投稿者）
	userMap, err := uc.buildUserMap(ctx, suggestion, comments, space.ID)
	if err != nil {
		return nil, fmt.Errorf("ユーザー情報の取得に失敗: %w", err)
	}

	// 認可チェック
	var canApply, canClose, canUpdateSuggestion, canUpdateSuggestionComment bool
	if spaceMember != nil {
		topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
		canApply = topicPolicy.CanApplySuggestion(suggestion)
		canClose = topicPolicy.CanCloseSuggestion(suggestion)
		canUpdateSuggestion = topicPolicy.CanUpdateSuggestion(suggestion)
		canUpdateSuggestionComment = topicPolicy.CanUpdateSuggestionComment(suggestion)
	}

	return &GetSuggestionDetailOutput{
		Space:                      space,
		SpaceMember:                spaceMember,
		Topic:                      topic,
		TopicMember:                topicMember,
		Suggestion:                 suggestion,
		SuggestionPages:            suggestionPages,
		Pages:                      pages,
		Comments:                   comments,
		UserMap:                    userMap,
		CanApplySuggestion:         canApply,
		CanCloseSuggestion:         canClose,
		CanUpdateSuggestion:        canUpdateSuggestion,
		CanUpdateSuggestionComment: canUpdateSuggestionComment,
	}, nil
}

// buildUserMap は編集提案の作成者とコメント投稿者のユーザー情報をマップで返す
func (uc *GetSuggestionDetailUsecase) buildUserMap(ctx context.Context, suggestion *model.Suggestion, comments []*model.SuggestionComment, spaceID model.SpaceID) (map[model.SpaceMemberID]*model.User, error) {
	// SpaceMemberIDを収集（重複排除）
	memberIDSet := make(map[model.SpaceMemberID]struct{})
	memberIDSet[suggestion.CreatedSpaceMemberID] = struct{}{}
	for _, c := range comments {
		memberIDSet[c.CreatedSpaceMemberID] = struct{}{}
	}
	memberIDs := make([]model.SpaceMemberID, 0, len(memberIDSet))
	for id := range memberIDSet {
		memberIDs = append(memberIDs, id)
	}

	return buildUserMapBySpaceMemberIDs(ctx, uc.spaceMemberRepo, uc.userRepo, memberIDs, spaceID)
}
