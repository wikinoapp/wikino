package validator

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// BulkRestoredPageCreateValidatorは、ゴミ箱で選んだページをまとめて復元する送信を検証する。
type BulkRestoredPageCreateValidator struct {
	pageRepo *repository.PageRepository
}

// NewBulkRestoredPageCreateValidatorはBulkRestoredPageCreateValidatorを生成する。
func NewBulkRestoredPageCreateValidator(pageRepo *repository.PageRepository) *BulkRestoredPageCreateValidator {
	return &BulkRestoredPageCreateValidator{pageRepo: pageRepo}
}

// BulkRestoredPageCreateValidatorInputは検証の入力。
type BulkRestoredPageCreateValidatorInput struct {
	// PageIDsはフォームで選ばれたページのID。利用者が送った値のため、IDの形式も確かめる
	PageIDs []string
	SpaceID model.SpaceID

	// RestorableTopicIDsは、閲覧者がページを復元できるトピック
	RestorableTopicIDs map[model.TopicID]bool
}

// Validateは、選ばれたページがすべて復元できるかを確かめ、そのうちゴミ箱に入っているページを返す。
//
// 1つでも復元できないページ (形式の誤ったID、別のスペースや廃棄したページ、復元できないトピックの
// ページ) が含まれていれば、何も復元しないようにエラーを返す。既にゴミ箱から戻したページは、
// 二重の送信でも失敗しないよう、エラーにせずに返り値から除く。
func (v *BulkRestoredPageCreateValidator) Validate(ctx context.Context, input BulkRestoredPageCreateValidatorInput) ([]*model.Page, error) {
	ve := model.NewValidationError()

	if len(input.PageIDs) == 0 {
		ve.AddField("page_ids", i18n.T(ctx, "validation_bulk_restored_page_ids_required"))
		return nil, ve
	}

	pageIDs := make([]model.PageID, 0, len(input.PageIDs))
	seen := make(map[model.PageID]bool, len(input.PageIDs))
	for _, raw := range input.PageIDs {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			ve.AddField("page_ids", i18n.T(ctx, "validation_bulk_restored_page_ids_not_restorable"))
			return nil, ve
		}
		// uuid.Parseは、DBが受け付けない形式や大文字のIDも解釈する。DBが返すIDと同じ正規の形式に
		// 直してから、重複の除去とクエリに使う。
		id := model.PageID(parsed.String())
		if !seen[id] {
			seen[id] = true
			pageIDs = append(pageIDs, id)
		}
	}

	pages, err := v.pageRepo.FindMetadataByIDs(ctx, pageIDs, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("選ばれたページの取得に失敗: %w", err)
	}
	if len(pages) != len(pageIDs) {
		ve.AddField("page_ids", i18n.T(ctx, "validation_bulk_restored_page_ids_not_restorable"))
		return nil, ve
	}

	trashedPages := make([]*model.Page, 0, len(pages))
	for _, pg := range pages {
		if !input.RestorableTopicIDs[pg.TopicID] {
			ve.AddField("page_ids", i18n.T(ctx, "validation_bulk_restored_page_ids_not_restorable"))
			return nil, ve
		}
		if pg.TrashedAt != nil {
			trashedPages = append(trashedPages, pg)
		}
	}

	return trashedPages, nil
}
