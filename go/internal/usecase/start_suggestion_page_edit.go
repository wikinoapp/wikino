package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// StartSuggestionPageEditUsecase は編集提案ページの編集開始ユースケース
type StartSuggestionPageEditUsecase struct {
	db                 *sql.DB
	suggestionPageRepo *repository.SuggestionPageRepository
	draftPageRepo      *repository.DraftPageRepository
	pageRepo           *repository.PageRepository
}

// NewStartSuggestionPageEditUsecase は StartSuggestionPageEditUsecase を生成する
func NewStartSuggestionPageEditUsecase(
	db *sql.DB,
	suggestionPageRepo *repository.SuggestionPageRepository,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
) *StartSuggestionPageEditUsecase {
	return &StartSuggestionPageEditUsecase{
		db:                 db,
		suggestionPageRepo: suggestionPageRepo,
		draftPageRepo:      draftPageRepo,
		pageRepo:           pageRepo,
	}
}

// StartSuggestionPageEditStatus は編集開始の結果ステータス
type StartSuggestionPageEditStatus int

const (
	// StartSuggestionPageEditRedirect はページ編集画面にリダイレクト可能な状態
	StartSuggestionPageEditRedirect StartSuggestionPageEditStatus = iota
	// StartSuggestionPageEditConflict は既存の下書きが存在する状態
	StartSuggestionPageEditConflict
)

// ConflictDraftKind はコンフリクト時の下書きの種類
type ConflictDraftKind int

const (
	// ConflictDraftKindNormal は通常の下書き（suggestion_page_idがNULL）
	ConflictDraftKindNormal ConflictDraftKind = iota
	// ConflictDraftKindOtherSuggestion は別の編集提案にリンクされた下書き
	ConflictDraftKindOtherSuggestion
)

// StartSuggestionPageEditInput は編集開始の入力パラメータ
type StartSuggestionPageEditInput struct {
	SpaceID          model.SpaceID
	SpaceMemberID    model.SpaceMemberID
	SuggestionPageID model.SuggestionPageID
	Force            bool
}

// StartSuggestionPageEditOutput は編集開始の出力
type StartSuggestionPageEditOutput struct {
	Status            StartSuggestionPageEditStatus
	PageNumber        model.PageNumber
	ConflictDraftKind ConflictDraftKind
}

// Execute は編集提案ページの編集を開始する
func (uc *StartSuggestionPageEditUsecase) Execute(ctx context.Context, input StartSuggestionPageEditInput) (*StartSuggestionPageEditOutput, error) {
	// 編集提案ページを取得
	sp, err := uc.suggestionPageRepo.FindByID(ctx, input.SuggestionPageID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("編集提案ページの取得に失敗: %w", err)
	}
	if sp == nil {
		return nil, fmt.Errorf("編集提案ページが見つかりません: %s", input.SuggestionPageID)
	}

	// ページを取得（番号の取得とトピックID確認用）
	pages, err := uc.pageRepo.FindByIDs(ctx, []model.PageID{sp.PageID}, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("ページが見つかりません: %s", sp.PageID)
	}
	page := pages[0]

	// 既存の下書きを確認
	draft, err := uc.draftPageRepo.FindByPageAndMember(ctx, sp.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
	}

	// 下書きが存在し、同じ編集提案ページにリンク済みの場合はそのままリダイレクト
	if draft != nil && draft.SuggestionPageID != nil && *draft.SuggestionPageID == input.SuggestionPageID {
		return &StartSuggestionPageEditOutput{
			Status:     StartSuggestionPageEditRedirect,
			PageNumber: page.Number,
		}, nil
	}

	// 下書きが存在し、Force=falseの場合はコンフリクト
	if draft != nil && !input.Force {
		draftKind := ConflictDraftKindNormal
		if draft.SuggestionPageID != nil {
			draftKind = ConflictDraftKindOtherSuggestion
		}
		return &StartSuggestionPageEditOutput{
			Status:            StartSuggestionPageEditConflict,
			PageNumber:        page.Number,
			ConflictDraftKind: draftKind,
		}, nil
	}

	// トランザクションを開始
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	draftPageRepo := uc.draftPageRepo.WithTx(tx)
	now := time.Now()

	if draft != nil {
		// Force=true: 既存の下書きを編集提案の内容で上書き
		_, err = draftPageRepo.Update(ctx, repository.UpdateDraftPageInput{
			ID:                        draft.ID,
			SpaceID:                   input.SpaceID,
			TopicID:                   page.TopicID,
			Title:                     sp.Title,
			Body:                      sp.Body,
			BodyHTML:                  sp.BodyHTML,
			LinkedPageIDs:             sp.LinkedPageIDs,
			FeaturedImageAttachmentID: sp.FeaturedImageAttachmentID,
			ModifiedAt:                now,
		})
		if err != nil {
			return nil, fmt.Errorf("下書きの更新に失敗: %w", err)
		}

		// suggestion_page_idを設定
		_, err = draftPageRepo.UpdateSuggestionPageID(ctx, draft.ID, input.SpaceID, &input.SuggestionPageID)
		if err != nil {
			return nil, fmt.Errorf("下書きのsuggestion_page_idの更新に失敗: %w", err)
		}
	} else {
		// 下書きが存在しない場合は新規作成
		_, err = draftPageRepo.Create(ctx, repository.CreateDraftPageInput{
			SpaceID:                   input.SpaceID,
			PageID:                    sp.PageID,
			SpaceMemberID:             input.SpaceMemberID,
			TopicID:                   page.TopicID,
			SuggestionPageID:          &input.SuggestionPageID,
			Title:                     sp.Title,
			Body:                      sp.Body,
			BodyHTML:                  sp.BodyHTML,
			LinkedPageIDs:             sp.LinkedPageIDs,
			FeaturedImageAttachmentID: sp.FeaturedImageAttachmentID,
			ModifiedAt:                now,
		})
		if err != nil {
			return nil, fmt.Errorf("下書きの作成に失敗: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return &StartSuggestionPageEditOutput{
		Status:     StartSuggestionPageEditRedirect,
		PageNumber: page.Number,
	}, nil
}
