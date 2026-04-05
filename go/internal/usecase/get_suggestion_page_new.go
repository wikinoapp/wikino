package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSuggestionPageNewUsecase は編集提案ページ追加画面のデータ取得ユースケース
type GetSuggestionPageNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicMemberRepo *repository.TopicMemberRepository
	suggestionRepo  *repository.SuggestionRepository
	topicRepo       *repository.TopicRepository
	draftPageRepo   *repository.DraftPageRepository
}

// NewGetSuggestionPageNewUsecase は GetSuggestionPageNewUsecase を生成する
func NewGetSuggestionPageNewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	topicRepo *repository.TopicRepository,
	draftPageRepo *repository.DraftPageRepository,
) *GetSuggestionPageNewUsecase {
	return &GetSuggestionPageNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicMemberRepo: topicMemberRepo,
		suggestionRepo:  suggestionRepo,
		topicRepo:       topicRepo,
		draftPageRepo:   draftPageRepo,
	}
}

// GetSuggestionPageNewInput は編集提案ページ追加画面のデータ取得の入力パラメータ
type GetSuggestionPageNewInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	UserID           model.UserID
}

// GetSuggestionPageNewOutput は編集提案ページ追加画面のデータ取得の出力パラメータ
type GetSuggestionPageNewOutput struct {
	Space      *model.Space
	Topic      *model.Topic
	Suggestion *model.Suggestion
	DraftPages []*model.DraftPage
}

// Execute は編集提案ページ追加画面用のデータを取得する
func (uc *GetSuggestionPageNewUsecase) Execute(ctx context.Context, input GetSuggestionPageNewInput) (*GetSuggestionPageNewOutput, error) {
	// スペースを取得
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// 編集提案を取得
	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// スペースメンバーを取得
	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 認可チェック
	var topicMember *model.TopicMember
	if spaceMember.Role != model.SpaceMemberRoleOwner {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, suggestion.TopicID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
	if !topicPolicy.CanAddSuggestionPage(suggestion) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// トピックを取得
	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, suggestion.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// トピック内の自分の下書きページ一覧を取得（編集提案にリンクされていないもの）
	draftPages, err := uc.draftPageRepo.ListByMemberAndTopic(ctx, spaceMember.ID, suggestion.TopicID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得に失敗: %w", err)
	}

	return &GetSuggestionPageNewOutput{
		Space:      space,
		Topic:      topic,
		Suggestion: suggestion,
		DraftPages: draftPages,
	}, nil
}
