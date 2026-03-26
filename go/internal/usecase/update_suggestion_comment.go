package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// UpdateSuggestionCommentUsecase は編集提案コメント更新ユースケース
type UpdateSuggestionCommentUsecase struct {
	db                    *sql.DB
	suggestionCommentRepo *repository.SuggestionCommentRepository
}

// NewUpdateSuggestionCommentUsecase は UpdateSuggestionCommentUsecase を生成する
func NewUpdateSuggestionCommentUsecase(
	db *sql.DB,
	suggestionCommentRepo *repository.SuggestionCommentRepository,
) *UpdateSuggestionCommentUsecase {
	return &UpdateSuggestionCommentUsecase{
		db:                    db,
		suggestionCommentRepo: suggestionCommentRepo,
	}
}

// UpdateSuggestionCommentInput は編集提案コメント更新の入力パラメータ
type UpdateSuggestionCommentInput struct {
	CommentID model.SuggestionCommentID
	SpaceID   model.SpaceID
	Body      string
}

// UpdateSuggestionCommentOutput は編集提案コメント更新の出力パラメータ
type UpdateSuggestionCommentOutput struct {
	Comment *model.SuggestionComment
}

// Execute は編集提案コメントを更新する
func (uc *UpdateSuggestionCommentUsecase) Execute(ctx context.Context, input UpdateSuggestionCommentInput) (*UpdateSuggestionCommentOutput, error) {
	bodyHTML := markup.RenderMarkdown(input.Body)

	return uc.updateComment(ctx, input, bodyHTML)
}

func (uc *UpdateSuggestionCommentUsecase) updateComment(ctx context.Context, input UpdateSuggestionCommentInput, bodyHTML string) (*UpdateSuggestionCommentOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionCommentRepo := uc.suggestionCommentRepo.WithTx(tx)

	comment, err := suggestionCommentRepo.Update(ctx, repository.UpdateSuggestionCommentInput{
		ID:       input.CommentID,
		SpaceID:  input.SpaceID,
		Body:     input.Body,
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
