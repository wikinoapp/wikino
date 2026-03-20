package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ApplySuggestionUsecase は編集提案反映ユースケース
type ApplySuggestionUsecase struct {
	db                    *sql.DB
	suggestionRepo        *repository.SuggestionRepository
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
}

// NewApplySuggestionUsecase は ApplySuggestionUsecase を生成する
func NewApplySuggestionUsecase(
	db *sql.DB,
	suggestionRepo *repository.SuggestionRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
) *ApplySuggestionUsecase {
	return &ApplySuggestionUsecase{
		db:                    db,
		suggestionRepo:        suggestionRepo,
		pageRepo:              pageRepo,
		pageRevisionRepo:      pageRevisionRepo,
		pageEditorRepo:        pageEditorRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
	}
}

// ApplySuggestionInput は編集提案反映の入力パラメータ
type ApplySuggestionInput struct {
	Suggestion      *model.Suggestion
	SuggestionPages []*model.SuggestionPage
	Pages           []*model.Page
	SpaceMemberID   model.SpaceMemberID
}

// ApplySuggestionOutput は編集提案反映の出力パラメータ
type ApplySuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Execute は編集提案をトピックに反映する
func (uc *ApplySuggestionUsecase) Execute(ctx context.Context, input ApplySuggestionInput) (*ApplySuggestionOutput, error) {
	now := time.Now()
	spaceID := input.Suggestion.SpaceID

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionRepo := uc.suggestionRepo.WithTx(tx)
	pageRepo := uc.pageRepo.WithTx(tx)
	pageRevisionRepo := uc.pageRevisionRepo.WithTx(tx)
	pageEditorRepo := uc.pageEditorRepo.WithTx(tx)
	topicMemberRepo := uc.topicMemberRepo.WithTx(tx)
	attachmentRepo := uc.attachmentRepo.WithTx(tx)
	pageAttachmentRefRepo := uc.pageAttachmentRefRepo.WithTx(tx)

	// ページをマップに変換
	pageMap := make(map[model.PageID]*model.Page, len(input.Pages))
	for _, p := range input.Pages {
		pageMap[p.ID] = p
	}

	// 各SuggestionPageの内容をPageに反映
	for _, sp := range input.SuggestionPages {
		page := pageMap[sp.PageID]
		if page == nil {
			return nil, fmt.Errorf("ページが見つかりません: %s", sp.PageID)
		}

		// Pageを更新
		_, err = pageRepo.Update(ctx, repository.UpdatePageInput{
			ID:                        sp.PageID,
			SpaceID:                   spaceID,
			TopicID:                   page.TopicID,
			Title:                     sp.Title,
			Body:                      sp.Body,
			BodyHTML:                  sp.BodyHTML,
			LinkedPageIDs:             sp.LinkedPageIDs,
			FeaturedImageAttachmentID: sp.FeaturedImageAttachmentID,
			ModifiedAt:                now,
			PublishedAt:               &now,
		})
		if err != nil {
			return nil, fmt.Errorf("ページの更新に失敗しました: %w", err)
		}

		// 添付ファイル参照の同期
		if err := syncAttachmentReferences(ctx, sp.BodyHTML, sp.PageID, spaceID, attachmentRepo, pageAttachmentRefRepo); err != nil {
			return nil, fmt.Errorf("添付ファイル参照の同期に失敗しました: %w", err)
		}

		// PageRevisionを作成（スナップショット）
		var title string
		if sp.Title != nil {
			title = *sp.Title
		}
		_, err = pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
			SpaceID:       spaceID,
			SpaceMemberID: input.SpaceMemberID,
			PageID:        sp.PageID,
			Title:         title,
			Body:          sp.Body,
			BodyHTML:      sp.BodyHTML,
		})
		if err != nil {
			return nil, fmt.Errorf("ページリビジョンの作成に失敗しました: %w", err)
		}

		// PageEditorを追加・更新
		pageEditor, err := pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             sp.PageID,
			SpaceMemberID:      input.SpaceMemberID,
			LastPageModifiedAt: now,
		})
		if err != nil {
			return nil, fmt.Errorf("ページ編集者の追加に失敗しました: %w", err)
		}

		_, err = pageEditorRepo.UpdateLastPageModifiedAt(ctx, repository.UpdateLastPageModifiedAtInput{
			ID:                 pageEditor.ID,
			SpaceID:            spaceID,
			LastPageModifiedAt: now,
		})
		if err != nil {
			return nil, fmt.Errorf("ページ編集者の更新に失敗しました: %w", err)
		}

		// TopicMemberのlast_page_modified_atを更新
		err = topicMemberRepo.UpdateLastPageModifiedAt(ctx, spaceID, page.TopicID, input.SpaceMemberID, now)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの更新に失敗しました: %w", err)
		}
	}

	// 編集提案のステータスを「反映済み」に変更
	updatedSuggestion, err := suggestionRepo.UpdateStatus(ctx, repository.UpdateStatusInput{
		ID:        input.Suggestion.ID,
		SpaceID:   spaceID,
		Status:    model.SuggestionStatusApplied,
		AppliedAt: &now,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案のステータス更新に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &ApplySuggestionOutput{
		Suggestion: updatedSuggestion,
	}, nil
}
