package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdateSuggestionCommentUsecase は編集提案コメント更新ユースケース
type UpdateSuggestionCommentUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	suggestionRepo        *repository.SuggestionRepository
	suggestionCommentRepo *repository.SuggestionCommentRepository
	updateValidator       *validator.SuggestionCommentUpdateValidator
}

// NewUpdateSuggestionCommentUsecase は UpdateSuggestionCommentUsecase を生成する
func NewUpdateSuggestionCommentUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionCommentRepo *repository.SuggestionCommentRepository,
	updateValidator *validator.SuggestionCommentUpdateValidator,
) *UpdateSuggestionCommentUsecase {
	return &UpdateSuggestionCommentUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		suggestionRepo:        suggestionRepo,
		suggestionCommentRepo: suggestionCommentRepo,
		updateValidator:       updateValidator,
	}
}

// UpdateSuggestionCommentInput は編集提案コメント更新の入力パラメータ
type UpdateSuggestionCommentInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	CommentNumber    model.SuggestionCommentNumber
	UserID           model.UserID
	Body             string
}

// UpdateSuggestionCommentOutput は編集提案コメント更新の出力パラメータ
type UpdateSuggestionCommentOutput struct {
	Comment *model.SuggestionComment
}

// Execute は編集提案コメントを更新する
func (uc *UpdateSuggestionCommentUsecase) Execute(ctx context.Context, input UpdateSuggestionCommentInput) (*UpdateSuggestionCommentOutput, error) {
	// 1. データ取得
	space, spaceMember, topic, suggestion, comment, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, space, spaceMember, topic, suggestion); err != nil {
		return nil, err
	}

	// 3. バリデーション
	if err := uc.updateValidator.Validate(ctx, validator.SuggestionCommentUpdateValidatorInput{
		Body: input.Body,
	}); err != nil {
		return nil, err
	}

	// 4. ビジネスロジック（トランザクション前）
	bodyHTML := markup.RenderMarkdown(input.Body)

	// 5. 永続化（トランザクション）
	return uc.updateComment(ctx, comment.ID, space.ID, input.Body, bodyHTML)
}

func (uc *UpdateSuggestionCommentUsecase) fetchData(ctx context.Context, input UpdateSuggestionCommentInput) (*model.Space, *model.SpaceMember, *model.Topic, *model.Suggestion, *model.SuggestionComment, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
		return nil, nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, suggestion.TopicID)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	comment, err := uc.suggestionCommentRepo.FindByNumber(ctx, suggestion.ID, input.CommentNumber, space.ID)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("編集提案コメントの取得に失敗: %w", err)
	}
	if comment == nil {
		return nil, nil, nil, nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	return space, spaceMember, topic, suggestion, comment, nil
}

func (uc *UpdateSuggestionCommentUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, topic *model.Topic, suggestion *model.Suggestion) error {
	if spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	var topicMember *model.TopicMember
	if spaceMember.Role != model.SpaceMemberRoleOwner {
		var err error
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
		if err != nil {
			return fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
	if !topicPolicy.CanUpdateSuggestionComment(suggestion) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

func (uc *UpdateSuggestionCommentUsecase) updateComment(ctx context.Context, commentID model.SuggestionCommentID, spaceID model.SpaceID, body, bodyHTML string) (*UpdateSuggestionCommentOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionCommentRepo := uc.suggestionCommentRepo.WithTx(tx)

	comment, err := suggestionCommentRepo.Update(ctx, repository.UpdateSuggestionCommentInput{
		ID:       commentID,
		SpaceID:  spaceID,
		Body:     body,
		BodyHTML: bodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案コメントの更新に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &UpdateSuggestionCommentOutput{
		Comment: comment,
	}, nil
}
