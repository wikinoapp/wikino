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

// RemoveSuggestionPageUsecase は編集提案ページ削除ユースケース
type RemoveSuggestionPageUsecase struct {
	db                         *sql.DB
	spaceRepo                  *repository.SpaceRepository
	spaceMemberRepo            *repository.SpaceMemberRepository
	topicMemberRepo            *repository.TopicMemberRepository
	suggestionRepo             *repository.SuggestionRepository
	suggestionPageRepo         *repository.SuggestionPageRepository
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
	draftPageRepo              *repository.DraftPageRepository
}

// NewRemoveSuggestionPageUsecase は RemoveSuggestionPageUsecase を生成する
func NewRemoveSuggestionPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	draftPageRepo *repository.DraftPageRepository,
) *RemoveSuggestionPageUsecase {
	return &RemoveSuggestionPageUsecase{
		db:                         db,
		spaceRepo:                  spaceRepo,
		spaceMemberRepo:            spaceMemberRepo,
		topicMemberRepo:            topicMemberRepo,
		suggestionRepo:             suggestionRepo,
		suggestionPageRepo:         suggestionPageRepo,
		suggestionPageRevisionRepo: suggestionPageRevisionRepo,
		draftPageRepo:              draftPageRepo,
	}
}

// RemoveSuggestionPageInput は編集提案ページ削除の入力パラメータ
type RemoveSuggestionPageInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	SuggestionPageID model.SuggestionPageID
	UserID           model.UserID
}

// RemoveSuggestionPageOutput は編集提案ページ削除の出力パラメータ
type RemoveSuggestionPageOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案からページを削除する
func (uc *RemoveSuggestionPageUsecase) Execute(ctx context.Context, input RemoveSuggestionPageInput) (*RemoveSuggestionPageOutput, error) {
	// 1. データ取得
	space, spaceMember, suggestion, suggestionPage, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, space, spaceMember, suggestion); err != nil {
		return nil, err
	}

	// 3. 残りページ数チェック
	if err := uc.checkRemainingPages(ctx, suggestion, space.ID); err != nil {
		return nil, err
	}

	// 4. 永続化（トランザクション）
	return uc.removeSuggestionPage(ctx, space.ID, suggestion, suggestionPage)
}

func (uc *RemoveSuggestionPageUsecase) fetchData(ctx context.Context, input RemoveSuggestionPageInput) (*model.Space, *model.SpaceMember, *model.Suggestion, *model.SuggestionPage, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
		return nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	suggestionPage, err := uc.suggestionPageRepo.FindByID(ctx, input.SuggestionPageID, space.ID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("編集提案ページの取得に失敗: %w", err)
	}
	if suggestionPage == nil || suggestionPage.SuggestionID != suggestion.ID {
		return nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	return space, spaceMember, suggestion, suggestionPage, nil
}

func (uc *RemoveSuggestionPageUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, suggestion *model.Suggestion) error {
	if spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	var topicMember *model.TopicMember
	if spaceMember.Role != model.SpaceMemberRoleOwner {
		var err error
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, suggestion.TopicID)
		if err != nil {
			return fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
	if !topicPolicy.CanRemoveSuggestionPage(suggestion) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

func (uc *RemoveSuggestionPageUsecase) checkRemainingPages(ctx context.Context, suggestion *model.Suggestion, spaceID model.SpaceID) error {
	pages, err := uc.suggestionPageRepo.ListBySuggestionID(ctx, suggestion.ID, spaceID)
	if err != nil {
		return fmt.Errorf("編集提案ページ一覧の取得に失敗: %w", err)
	}

	if len(pages) <= 1 {
		return &model.AppError{
			Code:    model.AppErrCodeConflict,
			UserMsg: i18n.T(ctx, "error_suggestion_page_remove_last_page"),
		}
	}

	return nil
}

func (uc *RemoveSuggestionPageUsecase) removeSuggestionPage(
	ctx context.Context,
	spaceID model.SpaceID,
	suggestion *model.Suggestion,
	suggestionPage *model.SuggestionPage,
) (*RemoveSuggestionPageOutput, error) {
	// DraftPageの取得はトランザクション前に行う
	draftPage, err := uc.draftPageRepo.FindBySuggestionPageID(ctx, suggestionPage.ID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得に失敗: %w", err)
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	draftPageRepo := uc.draftPageRepo.WithTx(tx)
	suggestionPageRevisionRepo := uc.suggestionPageRevisionRepo.WithTx(tx)
	suggestionPageRepo := uc.suggestionPageRepo.WithTx(tx)

	// DraftPageのsuggestion_page_idをクリア
	if draftPage != nil {
		_, err = draftPageRepo.UpdateSuggestionPageID(ctx, draftPage.ID, spaceID, nil)
		if err != nil {
			return nil, fmt.Errorf("下書きページの編集提案ページIDクリアに失敗しました: %w", err)
		}
	}

	// SuggestionPageRevisionを削除
	err = suggestionPageRevisionRepo.DeleteBySuggestionPageID(ctx, suggestionPage.ID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("編集提案ページリビジョンの削除に失敗しました: %w", err)
	}

	// SuggestionPageを削除
	err = suggestionPageRepo.Delete(ctx, suggestionPage.ID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("編集提案ページの削除に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &RemoveSuggestionPageOutput{
		Suggestion: suggestion,
	}, nil
}
