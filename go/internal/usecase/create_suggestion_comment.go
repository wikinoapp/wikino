package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateSuggestionCommentUsecase は編集提案コメント作成ユースケース
type CreateSuggestionCommentUsecase struct {
	db                    *sql.DB
	suggestionCommentRepo *repository.SuggestionCommentRepository
}

// NewCreateSuggestionCommentUsecase は CreateSuggestionCommentUsecase を生成する
func NewCreateSuggestionCommentUsecase(
	db *sql.DB,
	suggestionCommentRepo *repository.SuggestionCommentRepository,
) *CreateSuggestionCommentUsecase {
	return &CreateSuggestionCommentUsecase{
		db:                    db,
		suggestionCommentRepo: suggestionCommentRepo,
	}
}

// CreateSuggestionCommentInput は編集提案コメント作成の入力パラメータ
type CreateSuggestionCommentInput struct {
	SpaceID       model.SpaceID
	SuggestionID  model.SuggestionID
	SpaceMemberID model.SpaceMemberID
	Body          string
}

// CreateSuggestionCommentOutput は編集提案コメント作成の出力パラメータ
type CreateSuggestionCommentOutput struct {
	Comment *model.SuggestionComment
}

// Execute は編集提案コメントを作成する
func (uc *CreateSuggestionCommentUsecase) Execute(ctx context.Context, input CreateSuggestionCommentInput) (*CreateSuggestionCommentOutput, error) {
	bodyHTML := markup.RenderMarkdown(input.Body)

	return uc.createComment(ctx, input, bodyHTML)
}

func (uc *CreateSuggestionCommentUsecase) createComment(ctx context.Context, input CreateSuggestionCommentInput, bodyHTML string) (*CreateSuggestionCommentOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionCommentRepo := uc.suggestionCommentRepo.WithTx(tx)

	nextNumber, err := suggestionCommentRepo.GetNextNumber(ctx, input.SuggestionID)
	if err != nil {
		return nil, fmt.Errorf("次のコメント番号の取得に失敗しました: %w", err)
	}

	comment, err := suggestionCommentRepo.Create(ctx, repository.CreateSuggestionCommentInput{
		SpaceID:              input.SpaceID,
		SuggestionID:         input.SuggestionID,
		CreatedSpaceMemberID: input.SpaceMemberID,
		Number:               nextNumber,
		Body:                 input.Body,
		BodyHTML:             bodyHTML,
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
