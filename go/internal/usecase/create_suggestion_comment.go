package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateSuggestionCommentUsecase は編集提案コメント作成ユースケース
type CreateSuggestionCommentUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	topicMemberRepo       *repository.TopicMemberRepository
	suggestionRepo        *repository.SuggestionRepository
	suggestionCommentRepo *repository.SuggestionCommentRepository
	createValidator       *validator.SuggestionCommentCreateValidator
}

// NewCreateSuggestionCommentUsecase は CreateSuggestionCommentUsecase を生成する
func NewCreateSuggestionCommentUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionCommentRepo *repository.SuggestionCommentRepository,
	createValidator *validator.SuggestionCommentCreateValidator,
) *CreateSuggestionCommentUsecase {
	return &CreateSuggestionCommentUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		topicMemberRepo:       topicMemberRepo,
		suggestionRepo:        suggestionRepo,
		suggestionCommentRepo: suggestionCommentRepo,
		createValidator:       createValidator,
	}
}

// CreateSuggestionCommentInput は編集提案コメント作成の入力パラメータ
type CreateSuggestionCommentInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	UserID           model.UserID
	Body             string
}

// CreateSuggestionCommentOutput は編集提案コメント作成の出力パラメータ
type CreateSuggestionCommentOutput struct {
	Comment *model.SuggestionComment
}

// Execute は編集提案コメントを作成する
func (uc *CreateSuggestionCommentUsecase) Execute(ctx context.Context, input CreateSuggestionCommentInput) (*CreateSuggestionCommentOutput, error) {
	// 1. データ取得
	space, spaceMember, suggestion, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, space, spaceMember, suggestion); err != nil {
		return nil, err
	}

	// 3. バリデーション
	if err := uc.createValidator.Validate(ctx, validator.SuggestionCommentCreateValidatorInput{
		Body: input.Body,
	}); err != nil {
		return nil, err
	}

	// 4. 永続化
	return uc.createComment(ctx, space.ID, suggestion.ID, spaceMember.ID, input.Body)
}

func (uc *CreateSuggestionCommentUsecase) fetchData(ctx context.Context, input CreateSuggestionCommentInput) (*model.Space, *model.SpaceMember, *model.Suggestion, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
		return nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	return space, spaceMember, suggestion, nil
}

func (uc *CreateSuggestionCommentUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, suggestion *model.Suggestion) error {
	if spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, suggestion.TopicID)
	if err != nil {
		return fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
	if !topicPolicy.CanCreateSuggestionComment(suggestion) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

func (uc *CreateSuggestionCommentUsecase) createComment(ctx context.Context, spaceID model.SpaceID, suggestionID model.SuggestionID, spaceMemberID model.SpaceMemberID, body string) (*CreateSuggestionCommentOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionCommentRepo := uc.suggestionCommentRepo.WithTx(tx)

	nextNumber, err := suggestionCommentRepo.GetNextNumber(ctx, suggestionID)
	if err != nil {
		return nil, fmt.Errorf("次のコメント番号の取得に失敗しました: %w", err)
	}

	comment, err := suggestionCommentRepo.Create(ctx, repository.CreateSuggestionCommentInput{
		SpaceID:              spaceID,
		SuggestionID:         suggestionID,
		CreatedSpaceMemberID: spaceMemberID,
		Number:               nextNumber,
		Body:                 body,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案コメントの作成に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &CreateSuggestionCommentOutput{
		Comment: comment,
	}, nil
}
