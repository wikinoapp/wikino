package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// エラー定義
var (
	// ErrDraftPageNotFound は下書きページが見つからない場合のエラー
	ErrDraftPageNotFound = errors.New("下書きページが見つかりません")
)

// CreateDraftPageRevisionUsecase は下書きページリビジョン作成ユースケース
type CreateDraftPageRevisionUsecase struct {
	db                    *sql.DB
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
}

// NewCreateDraftPageRevisionUsecase は CreateDraftPageRevisionUsecase を生成する
func NewCreateDraftPageRevisionUsecase(
	db *sql.DB,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
) *CreateDraftPageRevisionUsecase {
	return &CreateDraftPageRevisionUsecase{
		db:                    db,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
	}
}

// CreateDraftPageRevisionInput は下書きページリビジョン作成の入力パラメータ
type CreateDraftPageRevisionInput struct {
	SpaceID       model.SpaceID
	PageID        model.PageID
	SpaceMemberID model.SpaceMemberID
}

// CreateDraftPageRevisionOutput は下書きページリビジョン作成の出力パラメータ
type CreateDraftPageRevisionOutput struct {
	DraftPageRevision *model.DraftPageRevision
}

// Execute は下書きページの現在の内容でリビジョン（スナップショット）を作成する
func (uc *CreateDraftPageRevisionUsecase) Execute(ctx context.Context, input CreateDraftPageRevisionInput) (*CreateDraftPageRevisionOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	draftPageRepo := uc.draftPageRepo.WithTx(tx)
	draftPageRevisionRepo := uc.draftPageRevisionRepo.WithTx(tx)

	// DraftPageを取得
	draftPage, err := draftPageRepo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得に失敗しました: %w", err)
	}
	if draftPage == nil {
		return nil, ErrDraftPageNotFound
	}

	// DraftPageの現在の内容でリビジョンを作成
	var title string
	if draftPage.Title != nil {
		title = *draftPage.Title
	}

	revision, err := draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
		DraftPageID:   draftPage.ID,
		SpaceID:       input.SpaceID,
		SpaceMemberID: input.SpaceMemberID,
		Title:         title,
		Body:          draftPage.Body,
		BodyHTML:      draftPage.BodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きページリビジョンの作成に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &CreateDraftPageRevisionOutput{
		DraftPageRevision: revision,
	}, nil
}
