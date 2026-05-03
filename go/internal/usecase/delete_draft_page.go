package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// DeleteDraftPageUsecase は下書きページの削除ユースケース
type DeleteDraftPageUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	pageRepo              *repository.PageRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
}

// NewDeleteDraftPageUsecase は DeleteDraftPageUsecase を生成する
func NewDeleteDraftPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *DeleteDraftPageUsecase {
	return &DeleteDraftPageUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		pageRepo:              pageRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
	}
}

// DeleteDraftPageInput は下書き削除の入力パラメータ
type DeleteDraftPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
}

// Execute は呼び出しユーザー本人の下書きを削除する。
// admin が他メンバーの下書きを操作する経路は別 UseCase で実装する想定。
func (uc *DeleteDraftPageUsecase) Execute(ctx context.Context, input DeleteDraftPageInput) error {
	// 1. データ取得
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return err
	}

	if data.spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 2. 削除対象の下書きを取得 (本人の下書きに限定)
	draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, data.page.ID, data.spaceMember.ID, data.space.ID)
	if err != nil {
		return fmt.Errorf("下書きの取得に失敗: %w", err)
	}
	if draftPage == nil {
		return &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// 3. 認可チェック
	authorizer := newAuthorizer(data.spaceMember, data.topicMember)
	if !authorizer.CanDeleteDraftPage() {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 4. 永続化（トランザクション内で revision → draft の順で削除）
	return uc.deleteDraftPage(ctx, draftPage.ID, data.space.ID)
}

func (uc *DeleteDraftPageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

func (uc *DeleteDraftPageUsecase) deleteDraftPage(ctx context.Context, draftPageID model.DraftPageID, spaceID model.SpaceID) error {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// draft_page_revisions の FK 制約があるためリビジョンを先に削除する
	if err := uc.draftPageRevisionRepo.WithTx(tx).DeleteByDraftPageID(ctx, draftPageID, spaceID); err != nil {
		return fmt.Errorf("下書きリビジョンの削除に失敗しました: %w", err)
	}

	if err := uc.draftPageRepo.WithTx(tx).Delete(ctx, draftPageID, spaceID); err != nil {
		return fmt.Errorf("下書きの削除に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return nil
}
