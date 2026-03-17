package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateSuggestionCommentUsecase は編集提案コメント作成ユースケース
type CreateSuggestionCommentUsecase struct {
	suggestionCommentRepo *repository.SuggestionCommentRepository
}

// NewCreateSuggestionCommentUsecase は CreateSuggestionCommentUsecase を生成する
func NewCreateSuggestionCommentUsecase(
	suggestionCommentRepo *repository.SuggestionCommentRepository,
) *CreateSuggestionCommentUsecase {
	return &CreateSuggestionCommentUsecase{
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

	comment, err := uc.suggestionCommentRepo.Create(ctx, repository.CreateSuggestionCommentInput{
		SpaceID:              input.SpaceID,
		SuggestionID:         input.SuggestionID,
		CreatedSpaceMemberID: input.SpaceMemberID,
		Body:                 input.Body,
		BodyHTML:             bodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案コメントの作成に失敗しました: %w", err)
	}

	return &CreateSuggestionCommentOutput{
		Comment: comment,
	}, nil
}
