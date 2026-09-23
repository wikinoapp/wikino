package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSuggestionCommentUsecaseは編集提案コメント取得ユースケース
type GetSuggestionCommentUsecase struct {
	suggestionCommentRepo *repository.SuggestionCommentRepository
}

// NewGetSuggestionCommentUsecaseはGetSuggestionCommentUsecaseを生成する
func NewGetSuggestionCommentUsecase(
	suggestionCommentRepo *repository.SuggestionCommentRepository,
) *GetSuggestionCommentUsecase {
	return &GetSuggestionCommentUsecase{
		suggestionCommentRepo: suggestionCommentRepo,
	}
}

// GetSuggestionCommentInputは編集提案コメント取得の入力パラメータ
type GetSuggestionCommentInput struct {
	SuggestionID  model.SuggestionID
	CommentNumber model.SuggestionCommentNumber
	SpaceID       model.SpaceID
}

// GetSuggestionCommentOutputは編集提案コメント取得の出力
type GetSuggestionCommentOutput struct {
	Comment *model.SuggestionComment
}

// Executeは編集提案コメントを番号で取得する
func (uc *GetSuggestionCommentUsecase) Execute(ctx context.Context, input GetSuggestionCommentInput) (*GetSuggestionCommentOutput, error) {
	comment, err := uc.suggestionCommentRepo.FindByNumber(ctx, input.SuggestionID, input.CommentNumber, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("編集提案コメントの取得に失敗: %w", err)
	}

	return &GetSuggestionCommentOutput{
		Comment: comment,
	}, nil
}
