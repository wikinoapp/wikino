package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// UpdateSuggestionPageUsecase は編集提案ページ更新ユースケース
type UpdateSuggestionPageUsecase struct {
	db                         *sql.DB
	suggestionPageRepo         *repository.SuggestionPageRepository
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
	draftPageRepo              *repository.DraftPageRepository
}

// NewUpdateSuggestionPageUsecase は UpdateSuggestionPageUsecase を生成する
func NewUpdateSuggestionPageUsecase(
	db *sql.DB,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	draftPageRepo *repository.DraftPageRepository,
) *UpdateSuggestionPageUsecase {
	return &UpdateSuggestionPageUsecase{
		db:                         db,
		suggestionPageRepo:         suggestionPageRepo,
		suggestionPageRevisionRepo: suggestionPageRevisionRepo,
		draftPageRepo:              draftPageRepo,
	}
}

// UpdateSuggestionPageInput は編集提案ページ更新の入力パラメータ
type UpdateSuggestionPageInput struct {
	SpaceID          model.SpaceID
	SpaceMemberID    model.SpaceMemberID
	SuggestionPageID model.SuggestionPageID
}

// UpdateSuggestionPageOutput は編集提案ページ更新の出力パラメータ
type UpdateSuggestionPageOutput struct {
	SuggestionPage *model.SuggestionPage
}

// Execute は編集提案ページを更新する
func (uc *UpdateSuggestionPageUsecase) Execute(ctx context.Context, input UpdateSuggestionPageInput) (*UpdateSuggestionPageOutput, error) {
	// 1. SuggestionPageを取得（PageIDの取得用）
	sp, err := uc.suggestionPageRepo.FindByID(ctx, input.SuggestionPageID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("編集提案ページの取得に失敗しました: %w", err)
	}
	if sp == nil {
		return nil, fmt.Errorf("編集提案ページが見つかりません: %s", input.SuggestionPageID)
	}

	// 2. DraftPageを取得（SuggestionPageにリンクされている下書き）
	dp, err := uc.draftPageRepo.FindByPageAndMember(ctx, sp.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得に失敗しました: %w", err)
	}
	if dp == nil {
		return nil, fmt.Errorf("下書きページが見つかりません")
	}
	if dp.SuggestionPageID == nil || *dp.SuggestionPageID != input.SuggestionPageID {
		return nil, fmt.Errorf("下書きページが編集提案ページにリンクされていません")
	}

	// 3. トランザクション開始
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionPageRepo := uc.suggestionPageRepo.WithTx(tx)
	suggestionPageRevisionRepo := uc.suggestionPageRevisionRepo.WithTx(tx)

	// 4. SuggestionPageのコンテンツを更新
	updatedSP, err := suggestionPageRepo.UpdateContent(ctx, repository.UpdateSuggestionPageContentInput{
		ID:                        input.SuggestionPageID,
		SpaceID:                   input.SpaceID,
		Title:                     dp.Title,
		Body:                      dp.Body,
		BodyHTML:                  dp.BodyHTML,
		LinkedPageIDs:             dp.LinkedPageIDs,
		FeaturedImageAttachmentID: dp.FeaturedImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案ページの更新に失敗しました: %w", err)
	}

	// 5. SuggestionPageRevisionを作成（スナップショット）
	_, err = suggestionPageRevisionRepo.Create(ctx, repository.CreateSuggestionPageRevisionInput{
		SpaceID:             input.SpaceID,
		SuggestionPageID:    input.SuggestionPageID,
		EditorSpaceMemberID: input.SpaceMemberID,
		Title:               dp.Title,
		Body:                dp.Body,
		BodyHTML:            dp.BodyHTML,
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
