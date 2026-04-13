package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageDetailUsecase はページ詳細画面のデータ取得ユースケース
type GetPageDetailUsecase struct {
	spaceRepo          *repository.SpaceRepository
	spaceMemberRepo    *repository.SpaceMemberRepository
	pageRepo           *repository.PageRepository
	draftPageRepo      *repository.DraftPageRepository
	topicRepo          *repository.TopicRepository
	topicMemberRepo    *repository.TopicMemberRepository
	suggestionPageRepo *repository.SuggestionPageRepository
	suggestionRepo     *repository.SuggestionRepository
	featureFlagRepo    *repository.FeatureFlagRepository
}

// NewGetPageDetailUsecase は GetPageDetailUsecase を生成する
func NewGetPageDetailUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionRepo *repository.SuggestionRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
) *GetPageDetailUsecase {
	return &GetPageDetailUsecase{
		spaceRepo:          spaceRepo,
		spaceMemberRepo:    spaceMemberRepo,
		pageRepo:           pageRepo,
		draftPageRepo:      draftPageRepo,
		topicRepo:          topicRepo,
		topicMemberRepo:    topicMemberRepo,
		suggestionPageRepo: suggestionPageRepo,
		suggestionRepo:     suggestionRepo,
		featureFlagRepo:    featureFlagRepo,
	}
}

// GetPageDetailInput はページ詳細取得の入力パラメータ
type GetPageDetailInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
}

// GetPageDetailOutput はページ詳細取得の出力
type GetPageDetailOutput struct {
	Space             *model.Space
	SpaceMember       *model.SpaceMember
	Page              *model.Page
	Topic             *model.Topic
	TopicMember       *model.TopicMember
	DraftPage         *model.DraftPage
	Suggestion        *model.Suggestion
	SuggestionEnabled bool
	CanUpdatePage     bool
}

// Execute はページ詳細画面に必要なデータを取得する
func (uc *GetPageDetailUsecase) Execute(ctx context.Context, input GetPageDetailInput) (*GetPageDetailOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, nil
	}

	pg, err := uc.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(input.PageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if pg == nil {
		return nil, nil
	}

	topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, fmt.Errorf("ページのトピックが見つかりません: page_id=%s, topic_id=%s", pg.ID, pg.TopicID)
	}

	draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, pg.ID, spaceMember.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
	}

	// 下書きが編集提案にリンクされている場合、編集提案を取得する
	var suggestion *model.Suggestion
	if draftPage != nil && draftPage.SuggestionPageID != nil {
		sp, err := uc.suggestionPageRepo.FindByID(ctx, *draftPage.SuggestionPageID, space.ID)
		if err != nil {
			return nil, fmt.Errorf("編集提案ページの取得に失敗: %w", err)
		}
		if sp != nil {
			suggestion, err = uc.suggestionRepo.FindByID(ctx, sp.SuggestionID, space.ID)
			if err != nil {
				return nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
			}
		}
	}

	// 編集提案のフィーチャーフラグを確認
	suggestionEnabled, err := uc.featureFlagRepo.IsEnabled(ctx, input.UserID, model.FeatureFlagSuggestion)
	if err != nil {
		return nil, fmt.Errorf("フィーチャーフラグの確認に失敗: %w", err)
	}

	// 認可チェック
	authorizer := newAuthorizer(spaceMember, topicMember)
	canUpdatePage := authorizer.CanUpdatePage()

	return &GetPageDetailOutput{
		Space:             space,
		SpaceMember:       spaceMember,
		Page:              pg,
		Topic:             topic,
		TopicMember:       topicMember,
		DraftPage:         draftPage,
		Suggestion:        suggestion,
		SuggestionEnabled: suggestionEnabled,
		CanUpdatePage:     canUpdatePage,
	}, nil
}
