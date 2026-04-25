package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdateSuggestionPageUsecase は編集提案ページ更新ユースケース
type UpdateSuggestionPageUsecase struct {
	db                         *sql.DB
	spaceRepo                  *repository.SpaceRepository
	spaceMemberRepo            *repository.SpaceMemberRepository
	topicMemberRepo            *repository.TopicMemberRepository
	suggestionRepo             *repository.SuggestionRepository
	suggestionPageRepo         *repository.SuggestionPageRepository
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
	updateValidator            *validator.SuggestionPageUpdateValidator
}

// NewUpdateSuggestionPageUsecase は UpdateSuggestionPageUsecase を生成する
func NewUpdateSuggestionPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	updateValidator *validator.SuggestionPageUpdateValidator,
) *UpdateSuggestionPageUsecase {
	return &UpdateSuggestionPageUsecase{
		db:                         db,
		spaceRepo:                  spaceRepo,
		spaceMemberRepo:            spaceMemberRepo,
		topicMemberRepo:            topicMemberRepo,
		suggestionRepo:             suggestionRepo,
		suggestionPageRepo:         suggestionPageRepo,
		suggestionPageRevisionRepo: suggestionPageRevisionRepo,
		updateValidator:            updateValidator,
	}
}

// UpdateSuggestionPageInput は編集提案ページ更新の入力パラメータ
type UpdateSuggestionPageInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	SuggestionPageID model.SuggestionPageID
	UserID           model.UserID
}

// UpdateSuggestionPageOutput は編集提案ページ更新の出力パラメータ
type UpdateSuggestionPageOutput struct {
	SuggestionPage *model.SuggestionPage
}

// Execute は編集提案ページを更新する
func (uc *UpdateSuggestionPageUsecase) Execute(ctx context.Context, input UpdateSuggestionPageInput) (*UpdateSuggestionPageOutput, error) {
	// 1. データ取得
	space, spaceMember, suggestion, suggestionPage, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, space, spaceMember, suggestion); err != nil {
		return nil, err
	}

	// 3. バリデーション
	draftPage, err := uc.updateValidator.Validate(ctx, validator.SuggestionPageUpdateValidatorInput{
		SuggestionPageID: input.SuggestionPageID,
		PageID:           suggestionPage.PageID,
		SpaceMemberID:    spaceMember.ID,
		SpaceID:          space.ID,
	})
	if err != nil {
		if errors.Is(err, validator.ErrDraftPageNotFound) || errors.Is(err, validator.ErrDraftPageNotLinked) {
			return nil, &model.AppError{
				Code:     model.AppErrCodeConflict,
				UserMsg:  i18n.T(ctx, "error_suggestion_page_update_conflict"),
				Internal: err,
			}
		}
		return nil, err
	}

	// 4. 永続化（トランザクション）
	return uc.updateSuggestionPage(ctx, space.ID, spaceMember.ID, input.SuggestionPageID, draftPage)
}

func (uc *UpdateSuggestionPageUsecase) fetchData(ctx context.Context, input UpdateSuggestionPageInput) (*model.Space, *model.SpaceMember, *model.Suggestion, *model.SuggestionPage, error) {
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

	// SuggestionPageの取得と所属チェック
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

func (uc *UpdateSuggestionPageUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, suggestion *model.Suggestion) error {
	if spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	var topicMember *model.TopicMember
	if !model.HasScope(spaceMember.Scopes, model.ScopeSpaceAdmin) {
		var err error
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, suggestion.TopicID)
		if err != nil {
			return fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	authorizer := newAuthorizer(spaceMember, topicMember)
	if !authorizer.CanUpdateSuggestion(suggestion) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

func (uc *UpdateSuggestionPageUsecase) updateSuggestionPage(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, suggestionPageID model.SuggestionPageID, draftPage *model.DraftPage) (*UpdateSuggestionPageOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionPageRepo := uc.suggestionPageRepo.WithTx(tx)
	suggestionPageRevisionRepo := uc.suggestionPageRevisionRepo.WithTx(tx)

	// SuggestionPageのコンテンツを更新
	updatedSP, err := suggestionPageRepo.UpdateContent(ctx, repository.UpdateSuggestionPageContentInput{
		ID:                        suggestionPageID,
		SpaceID:                   spaceID,
		Title:                     draftPage.Title,
		Body:                      draftPage.Body,
		BodyHTML:                  draftPage.BodyHTML,
		LinkedPageIDs:             draftPage.LinkedPageIDs,
		FeaturedImageAttachmentID: draftPage.FeaturedImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案ページの更新に失敗しました: %w", err)
	}

	// SuggestionPageRevisionを作成（スナップショット）
	_, err = suggestionPageRevisionRepo.Create(ctx, repository.CreateSuggestionPageRevisionInput{
		SpaceID:             spaceID,
		SuggestionPageID:    suggestionPageID,
		EditorSpaceMemberID: spaceMemberID,
		Title:               draftPage.Title,
		Body:                draftPage.Body,
		BodyHTML:            draftPage.BodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案ページリビジョンの作成に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &UpdateSuggestionPageOutput{
		SuggestionPage: updatedSP,
	}, nil
}
