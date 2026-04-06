package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CloseSuggestionUsecase は編集提案クローズユースケース
type CloseSuggestionUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicMemberRepo *repository.TopicMemberRepository
	suggestionRepo  *repository.SuggestionRepository
	draftPageRepo   *repository.DraftPageRepository
}

// NewCloseSuggestionUsecase は CloseSuggestionUsecase を生成する
func NewCloseSuggestionUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	draftPageRepo *repository.DraftPageRepository,
) *CloseSuggestionUsecase {
	return &CloseSuggestionUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicMemberRepo: topicMemberRepo,
		suggestionRepo:  suggestionRepo,
		draftPageRepo:   draftPageRepo,
	}
}

// CloseSuggestionInput は編集提案クローズの入力パラメータ
type CloseSuggestionInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	UserID           model.UserID
}

// CloseSuggestionOutput は編集提案クローズの出力パラメータ
type CloseSuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案をクローズする
func (uc *CloseSuggestionUsecase) Execute(ctx context.Context, input CloseSuggestionInput) (*CloseSuggestionOutput, error) {
	// 1. データ取得
	data, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, data); err != nil {
		return nil, err
	}

	// 3. ステータスチェック
	if output, err := uc.checkStatusForClose(ctx, data.suggestion); output != nil || err != nil {
		return output, err
	}

	// 4. 永続化（トランザクション）
	return uc.closeSuggestion(ctx, data)
}

// closeSuggestionData はデータ取得結果をまとめた構造体
type closeSuggestionData struct {
	spaceMember *model.SpaceMember
	topicMember *model.TopicMember
	suggestion  *model.Suggestion
}

func (uc *CloseSuggestionUsecase) fetchData(ctx context.Context, input CloseSuggestionInput) (*closeSuggestionData, error) {
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

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

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

	var topicMember *model.TopicMember
	if spaceMember != nil && spaceMember.Role != model.SpaceMemberRoleOwner {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, suggestion.TopicID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	return &closeSuggestionData{
		spaceMember: spaceMember,
		topicMember: topicMember,
		suggestion:  suggestion,
	}, nil
}

func (uc *CloseSuggestionUsecase) authorize(ctx context.Context, data *closeSuggestionData) error {
	if data.spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	topicPolicy := policy.NewTopicPolicy(data.spaceMember, data.topicMember)
	if !topicPolicy.CanCloseSuggestion(data.suggestion) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

// checkStatusForClose はステータスに基づく事前チェックを行う。
// Closed → べき等に成功出力を返す、Open → nil,nil で続行、その他 → Conflict エラー。
func (uc *CloseSuggestionUsecase) checkStatusForClose(ctx context.Context, suggestion *model.Suggestion) (*CloseSuggestionOutput, error) {
	switch suggestion.Status {
	case model.SuggestionStatusClosed:
		return &CloseSuggestionOutput{Suggestion: suggestion}, nil
	case model.SuggestionStatusOpen:
		return nil, nil
	default:
		return nil, &model.AppError{
			Code:    model.AppErrCodeConflict,
			UserMsg: i18n.T(ctx, "suggestion_close_error"),
		}
	}
}

func (uc *CloseSuggestionUsecase) closeSuggestion(ctx context.Context, data *closeSuggestionData) (*CloseSuggestionOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionRepo := uc.suggestionRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)

	updatedSuggestion, err := suggestionRepo.UpdateStatus(ctx, repository.UpdateStatusInput{
		ID:      data.suggestion.ID,
		SpaceID: data.suggestion.SpaceID,
		Status:  model.SuggestionStatusClosed,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案のステータス更新に失敗しました: %w", err)
	}
	if updatedSuggestion == nil {
		return nil, fmt.Errorf("編集提案が見つかりません: %s", data.suggestion.ID)
	}

	if err := draftPageRepo.ClearSuggestionPageIDsBySuggestionID(ctx, data.suggestion.ID, data.suggestion.SpaceID); err != nil {
		return nil, fmt.Errorf("下書きのsuggestion_page_idクリアに失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &CloseSuggestionOutput{
		Suggestion: updatedSuggestion,
	}, nil
}
