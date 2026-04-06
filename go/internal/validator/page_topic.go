package validator

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// validatePageTopicConsistency はページの現在のトピックが期待するトピックと一致するか検証する。
// ページ移動後にDraftPageのトピックとPageのトピックが不整合になるケースを防止する。
func validatePageTopicConsistency(ctx context.Context, pageRepo *repository.PageRepository, draftPages []*model.DraftPage, topicID model.TopicID, spaceID model.SpaceID, fieldName string, msgKey string) error {
	pageIDs := make([]model.PageID, len(draftPages))
	for i, dp := range draftPages {
		pageIDs[i] = dp.PageID
	}
	pages, err := pageRepo.FindByIDs(ctx, pageIDs, spaceID)
	if err != nil {
		return err
	}
	pageTopicByID := make(map[model.PageID]model.TopicID, len(pages))
	for _, p := range pages {
		pageTopicByID[p.ID] = p.TopicID
	}

	ve := model.NewValidationError()
	for _, dp := range draftPages {
		pageTopicID, ok := pageTopicByID[dp.PageID]
		if !ok {
			ve.AddField(fieldName, i18n.T(ctx, msgKey))
			return ve
		}
		if pageTopicID != topicID {
			ve.AddField(fieldName, i18n.T(ctx, msgKey))
			return ve
		}
	}

	return nil
}
