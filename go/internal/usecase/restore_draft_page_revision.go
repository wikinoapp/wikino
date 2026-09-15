package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// RestoreDraftPageRevisionUsecaseは選択されたリビジョンの内容に下書きを復元する。
// 下書きはリビジョンのタイトル・本文を使い、手動保存と同じ保存経路 (本文HTMLの再レンダリングと
// Wikiリンクの再解決) で更新する。復元後の状態は新しいリビジョンとして記録し、履歴は失わない。
type RestoreDraftPageRevisionUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	pageRepo              *repository.PageRepository
	pageEditorRepo        *repository.PageEditorRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	attachmentRepo        *repository.AttachmentRepository
}

// NewRestoreDraftPageRevisionUsecaseはRestoreDraftPageRevisionUsecaseを生成する。
func NewRestoreDraftPageRevisionUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	attachmentRepo *repository.AttachmentRepository,
) *RestoreDraftPageRevisionUsecase {
	return &RestoreDraftPageRevisionUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		pageRepo:              pageRepo,
		pageEditorRepo:        pageEditorRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		attachmentRepo:        attachmentRepo,
	}
}

// RestoreDraftPageRevisionInputは下書きページリビジョン復元の入力パラメータ。
type RestoreDraftPageRevisionInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	RevisionID      model.DraftPageRevisionID
	UserID          model.UserID
}

// RestoreDraftPageRevisionOutputは下書きページリビジョン復元の出力。
type RestoreDraftPageRevisionOutput struct {
	// DraftPageは復元後の内容で更新された下書き。
	DraftPage *model.DraftPage

	// DraftPageRevisionは復元後の状態を記録した新しいリビジョン。
	DraftPageRevision *model.DraftPageRevision
}

// Executeは選択されたリビジョンの内容に下書きを復元し、新しいリビジョンを記録する。
func (uc *RestoreDraftPageRevisionUsecase) Execute(ctx context.Context, input RestoreDraftPageRevisionInput) (*RestoreDraftPageRevisionOutput, error) {
	// 1. データ取得 + 2. 認可チェック (手動保存と同じ共通ヘルパーを使う)。
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}
	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// 下書きはスペースメンバーごとの個人データのため、リクエストしたメンバー自身の
	// このページの下書きを解決し、それに属するリビジョンのみ受け付ける (403ではなく404。
	// 差分取得のGetDraftPageRevisionDiffUsecaseと同じ方針)。
	draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, data.page.ID, data.spaceMember.ID, data.space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
	}
	if draftPage == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	revision, err := uc.draftPageRevisionRepo.FindByID(ctx, input.RevisionID, data.space.ID)
	if err != nil {
		return nil, fmt.Errorf("リビジョンの取得に失敗: %w", err)
	}
	if revision == nil || revision.DraftPageID != draftPage.ID {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// 3. 永続化: 下書きの内容を復元し、新しいリビジョンを記録する。
	return uc.restore(ctx, data, input, revision)
}

func (uc *RestoreDraftPageRevisionUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

// restoreは1トランザクションで下書きをリビジョンのタイトル・本文で更新し、新しい
// リビジョンを作成する。本文HTMLはリビジョンに保存済みのbody_htmlを使い回さず、手動保存と
// 同じ経路saveDraftPageContentで再レンダリングする。これによりリンク先ページIDや添付参照が
// 復元後の本文と整合する。
func (uc *RestoreDraftPageRevisionUsecase) restore(ctx context.Context, data *pageAccessData, input RestoreDraftPageRevisionInput, revision *model.DraftPageRevision) (*RestoreDraftPageRevisionOutput, error) {
	now := time.Now()

	var titlePtr *string
	if revision.Title != "" {
		title := revision.Title
		titlePtr = &title
	}

	// トランザクション前: アイキャッチ画像のみ抽出する (手動保存と同様)。
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, revision.Body, data.space.ID, uc.attachmentRepo)
	if err != nil {
		return nil, err
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	contentInput := saveDraftPageContentInput{
		SpaceID:                   data.space.ID,
		PageID:                    data.page.ID,
		SpaceMemberID:             data.spaceMember.ID,
		TopicID:                   data.page.TopicID,
		Title:                     titlePtr,
		Body:                      revision.Body,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		SpaceIdentifier:           input.SpaceIdentifier,
		CurrentTopicName:          data.topic.Name,
	}

	result, err := saveDraftPageContent(ctx, contentInput, now,
		uc.draftPageRepo.WithTx(tx),
		uc.pageRepo.WithTx(tx),
		uc.pageEditorRepo.WithTx(tx),
		uc.topicRepo.WithTx(tx),
		uc.attachmentRepo.WithTx(tx),
	)
	if err != nil {
		return nil, err
	}

	newRevision, err := uc.draftPageRevisionRepo.WithTx(tx).Create(ctx, repository.CreateDraftPageRevisionInput{
		DraftPageID:   result.DraftPage.ID,
		SpaceID:       data.space.ID,
		SpaceMemberID: data.spaceMember.ID,
		Title:         revision.Title,
		Body:          revision.Body,
		BodyHTML:      result.BodyHTML,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きページリビジョンの作成に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &RestoreDraftPageRevisionOutput{
		DraftPage:         result.DraftPage,
		DraftPageRevision: newRevision,
	}, nil
}
