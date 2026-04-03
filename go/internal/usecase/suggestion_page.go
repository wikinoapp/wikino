package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// fetchLatestPageRevisions は各下書きページに対応するページの最新リビジョンを取得する
func fetchLatestPageRevisions(ctx context.Context, draftPages []*model.DraftPage, spaceID model.SpaceID, pageRevisionRepo *repository.PageRevisionRepository) (map[model.PageID]*model.PageRevision, error) {
	pageRevisions := make(map[model.PageID]*model.PageRevision, len(draftPages))

	for _, draftPage := range draftPages {
		latestRevision, err := pageRevisionRepo.FindLatestByPageID(ctx, draftPage.PageID, spaceID)
		if err != nil {
			return nil, fmt.Errorf("ページリビジョンの取得に失敗しました: %w", err)
		}
		if latestRevision == nil {
			return nil, fmt.Errorf("ページ %s のリビジョンが見つかりません", draftPage.PageID)
		}

		pageRevisions[draftPage.PageID] = latestRevision
	}

	return pageRevisions, nil
}

// createSuggestionPageInput は編集提案ページ作成の入力パラメータ
type createSuggestionPageInput struct {
	SpaceID        model.SpaceID
	SuggestionID   model.SuggestionID
	SpaceMemberID  model.SpaceMemberID
	DraftPage      *model.DraftPage
	PageRevisionID model.PageRevisionID
}

// createSuggestionPageFromDraftPage は下書きページからSuggestionPage・SuggestionPageRevisionを作成し、DraftPageのsuggestion_page_idを設定する。
// トランザクション内で呼び出すこと。
func createSuggestionPageFromDraftPage(
	ctx context.Context,
	input createSuggestionPageInput,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository,
	draftPageRepo *repository.DraftPageRepository,
) (*model.SuggestionPage, error) {
	dp := input.DraftPage

	// SuggestionPageを作成
	suggestionPage, err := suggestionPageRepo.Create(ctx, repository.CreateSuggestionPageInput{
		SpaceID:                   input.SpaceID,
		SuggestionID:              input.SuggestionID,
		PageID:                    dp.PageID,
		PageRevisionID:            input.PageRevisionID,
		Title:                     dp.Title,
		Body:                      dp.Body,
		BodyHTML:                  dp.BodyHTML,
		LinkedPageIDs:             dp.LinkedPageIDs,
		FeaturedImageAttachmentID: dp.FeaturedImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案ページの作成に失敗しました: %w", err)
	}

	// SuggestionPageRevisionを作成
	_, err = suggestionPageRevisionRepo.Create(ctx, repository.CreateSuggestionPageRevisionInput{
		SpaceID:             input.SpaceID,
		SuggestionPageID:    suggestionPage.ID,
		EditorSpaceMemberID: input.SpaceMemberID,
		Title:               dp.Title,
		Body:                dp.Body,
		BodyHTML:            dp.BodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案ページリビジョンの作成に失敗しました: %w", err)
	}

	// DraftPageのsuggestion_page_idを設定し、編集提案モードにリンクする
	if dp.ID != "" {
		_, err = draftPageRepo.UpdateSuggestionPageID(ctx, dp.ID, input.SpaceID, &suggestionPage.ID)
		if err != nil {
			return nil, fmt.Errorf("下書きページのsuggestion_page_id設定に失敗しました: %w", err)
		}
	}

	return suggestionPage, nil
}
