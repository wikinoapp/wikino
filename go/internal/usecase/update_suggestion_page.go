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
	pageRepo                   *repository.PageRepository
}

// NewUpdateSuggestionPageUsecase は UpdateSuggestionPageUsecase を生成する
func NewUpdateSuggestionPageUsecase(
	db *sql.DB,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
) *UpdateSuggestionPageUsecase {
	return &UpdateSuggestionPageUsecase{
		db:                         db,
		suggestionPageRepo:         suggestionPageRepo,
		suggestionPageRevisionRepo: suggestionPageRevisionRepo,
		draftPageRepo:              draftPageRepo,
		pageRepo:                   pageRepo,
	}
}

// UpdateSuggestionPageInput は編集提案ページ更新の入力パラメータ
type UpdateSuggestionPageInput struct {
	SpaceID         model.SpaceID
	SpaceMemberID   model.SpaceMemberID
	SuggestionID    model.SuggestionID
	PageNumber      int32
	SuggestionPages []*model.SuggestionPage
}

// UpdateSuggestionPageOutput は編集提案ページ更新の出力パラメータ
type UpdateSuggestionPageOutput struct {
	SuggestionPage *model.SuggestionPage
}

// Execute は編集提案ページを更新する
func (uc *UpdateSuggestionPageUsecase) Execute(ctx context.Context, input UpdateSuggestionPageInput) (*UpdateSuggestionPageOutput, error) {
	// 1. ページ番号からページを取得
	pg, err := uc.pageRepo.FindBySpaceAndNumber(ctx, input.SpaceID, model.PageNumber(input.PageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗しました: %w", err)
	}
	if pg == nil {
		return nil, fmt.Errorf("ページが見つかりません: number=%d", input.PageNumber)
	}

	// 2. 該当するSuggestionPageを特定
	var targetSP *model.SuggestionPage
	for _, sp := range input.SuggestionPages {
		if sp.PageID == pg.ID {
			targetSP = sp
			break
		}
	}
	if targetSP == nil {
		return nil, fmt.Errorf("編集提案に含まれないページです: pageID=%s", pg.ID)
	}

	// 3. DraftPageを取得（SuggestionPageにリンクされている下書き）
	dp, err := uc.draftPageRepo.FindByPageAndMember(ctx, pg.ID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得に失敗しました: %w", err)
	}
	if dp == nil {
		return nil, fmt.Errorf("下書きページが見つかりません")
	}
	if dp.SuggestionPageID == nil || *dp.SuggestionPageID != targetSP.ID {
		return nil, fmt.Errorf("下書きページが編集提案ページにリンクされていません")
	}

	// 4. トランザクション開始
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionPageRepo := uc.suggestionPageRepo.WithTx(tx)
	suggestionPageRevisionRepo := uc.suggestionPageRevisionRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)

	// 5. SuggestionPageのコンテンツを更新
	updatedSP, err := suggestionPageRepo.UpdateContent(ctx, repository.UpdateSuggestionPageContentInput{
		ID:                        targetSP.ID,
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

	// 6. SuggestionPageRevisionを作成（スナップショット）
	_, err = suggestionPageRevisionRepo.Create(ctx, repository.CreateSuggestionPageRevisionInput{
		SpaceID:             input.SpaceID,
		SuggestionPageID:    targetSP.ID,
		EditorSpaceMemberID: input.SpaceMemberID,
		Title:               dp.Title,
		Body:                dp.Body,
		BodyHTML:            dp.BodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案ページリビジョンの作成に失敗しました: %w", err)
	}

	// 7. DraftPageのsuggestion_page_idをクリア
	_, err = draftPageRepo.UpdateSuggestionPageID(ctx, dp.ID, input.SpaceID, nil)
	if err != nil {
		return nil, fmt.Errorf("下書きページのsuggestion_page_idクリアに失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &UpdateSuggestionPageOutput{
		SuggestionPage: updatedSP,
	}, nil
}
