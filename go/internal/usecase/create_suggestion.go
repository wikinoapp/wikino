package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateSuggestionUsecase は編集提案作成ユースケース
type CreateSuggestionUsecase struct {
	db                         *sql.DB
	suggestionRepo             *repository.SuggestionRepository
	suggestionPageRepo         *repository.SuggestionPageRepository
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
	draftPageRepo              *repository.DraftPageRepository
}

// NewCreateSuggestionUsecase は CreateSuggestionUsecase を生成する
func NewCreateSuggestionUsecase(
	db *sql.DB,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	draftPageRepo *repository.DraftPageRepository,
) *CreateSuggestionUsecase {
	return &CreateSuggestionUsecase{
		db:                         db,
		suggestionRepo:             suggestionRepo,
		suggestionPageRepo:         suggestionPageRepo,
		suggestionPageRevisionRepo: suggestionPageRevisionRepo,
		draftPageRepo:              draftPageRepo,
	}
}

// CreateSuggestionInput は編集提案作成の入力パラメータ
type CreateSuggestionInput struct {
	SpaceID       model.SpaceID
	TopicID       model.TopicID
	SpaceMemberID model.SpaceMemberID
	Title         string
	Body          string
	BodyHTML      string
	DraftPages    []*model.DraftPage
	PageRevisions map[model.PageID]*model.PageRevision
}

// CreateSuggestionOutput は編集提案作成の出力パラメータ
type CreateSuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案を作成する
func (uc *CreateSuggestionUsecase) Execute(ctx context.Context, input CreateSuggestionInput) (*CreateSuggestionOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionRepo := uc.suggestionRepo.WithTx(tx)
	suggestionPageRepo := uc.suggestionPageRepo.WithTx(tx)
	suggestionPageRevisionRepo := uc.suggestionPageRevisionRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)

	// 1. スペース内の次の編集提案番号を取得
	nextNumber, err := suggestionRepo.GetNextNumber(ctx, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("次の編集提案番号の取得に失敗しました: %w", err)
	}

	// 2. 編集提案を作成
	suggestion, err := suggestionRepo.Create(ctx, repository.CreateSuggestionInput{
		SpaceID:              input.SpaceID,
		TopicID:              input.TopicID,
		CreatedSpaceMemberID: input.SpaceMemberID,
		Number:               nextNumber,
		Title:                input.Title,
		Body:                 input.Body,
		BodyHTML:             input.BodyHTML,
		Status:               model.SuggestionStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案の作成に失敗しました: %w", err)
	}

	// 3. 各下書きページからSuggestionPageとSuggestionPageRevisionを作成
	for _, draftPage := range input.DraftPages {
		latestRevision := input.PageRevisions[draftPage.PageID]

		// SuggestionPageを作成
		suggestionPage, err := suggestionPageRepo.Create(ctx, repository.CreateSuggestionPageInput{
			SpaceID:                   input.SpaceID,
			SuggestionID:              suggestion.ID,
			PageID:                    draftPage.PageID,
			PageRevisionID:            latestRevision.ID,
			Title:                     draftPage.Title,
			Body:                      draftPage.Body,
			BodyHTML:                  draftPage.BodyHTML,
			LinkedPageIDs:             draftPage.LinkedPageIDs,
			FeaturedImageAttachmentID: draftPage.FeaturedImageAttachmentID,
		})
		if err != nil {
			return nil, fmt.Errorf("編集提案ページの作成に失敗しました: %w", err)
		}

		// SuggestionPageRevisionを作成
		_, err = suggestionPageRevisionRepo.Create(ctx, repository.CreateSuggestionPageRevisionInput{
			SpaceID:             input.SpaceID,
			SuggestionPageID:    suggestionPage.ID,
			EditorSpaceMemberID: input.SpaceMemberID,
			Title:               draftPage.Title,
			Body:                draftPage.Body,
			BodyHTML:            draftPage.BodyHTML,
		})
		if err != nil {
			return nil, fmt.Errorf("編集提案ページリビジョンの作成に失敗しました: %w", err)
		}

		// DraftPageのsuggestion_page_idを設定し、編集提案モードにリンクする
		if draftPage.ID != "" {
			_, err = draftPageRepo.UpdateSuggestionPageID(ctx, draftPage.ID, input.SpaceID, &suggestionPage.ID)
			if err != nil {
				return nil, fmt.Errorf("下書きページのsuggestion_page_id設定に失敗しました: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &CreateSuggestionOutput{
		Suggestion: suggestion,
	}, nil
}
