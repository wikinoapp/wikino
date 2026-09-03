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

// RestoreDraftPageRevisionUsecase restores the draft to the content of a selected revision.
// The draft is updated with the revision's title and body through the same save path as the
// manual save (re-rendering the body HTML and re-resolving wiki links), and the restored state
// is recorded as a new revision so the history is never lost.
//
// [Ja] RestoreDraftPageRevisionUsecase は選択されたリビジョンの内容に下書きを復元する。
// 下書きはリビジョンのタイトル・本文を使い、手動保存と同じ保存経路 (本文 HTML の再レンダリングと
// Wiki リンクの再解決) で更新する。復元後の状態は新しいリビジョンとして記録し、履歴は失わない。
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

// NewRestoreDraftPageRevisionUsecase creates a new RestoreDraftPageRevisionUsecase.
// [Ja] NewRestoreDraftPageRevisionUsecase は RestoreDraftPageRevisionUsecase を生成する。
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

// RestoreDraftPageRevisionInput is the input parameters for restoring a draft page revision.
// [Ja] RestoreDraftPageRevisionInput は下書きページリビジョン復元の入力パラメータ。
type RestoreDraftPageRevisionInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	RevisionID      model.DraftPageRevisionID
	UserID          model.UserID
}

// RestoreDraftPageRevisionOutput is the output of restoring a draft page revision.
// [Ja] RestoreDraftPageRevisionOutput は下書きページリビジョン復元の出力。
type RestoreDraftPageRevisionOutput struct {
	// DraftPage is the draft updated with the restored content.
	// [Ja] DraftPage は復元後の内容で更新された下書き。
	DraftPage *model.DraftPage

	// DraftPageRevision is the new revision recording the restored state.
	// [Ja] DraftPageRevision は復元後の状態を記録した新しいリビジョン。
	DraftPageRevision *model.DraftPageRevision
}

// Execute restores the draft to the selected revision's content and records a new revision.
// [Ja] Execute は選択されたリビジョンの内容に下書きを復元し、新しいリビジョンを記録する。
func (uc *RestoreDraftPageRevisionUsecase) Execute(ctx context.Context, input RestoreDraftPageRevisionInput) (*RestoreDraftPageRevisionOutput, error) {
	// 1. Fetch data + 2. authorization check (using the same shared helper as manual save).
	// [Ja] 1. データ取得 + 2. 認可チェック (手動保存と同じ共通ヘルパーを使う)。
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}
	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// Drafts are personal to a space member, so resolve the requesting member's own draft for
	// this page and accept only revisions belonging to it (404, not 403, mirroring the diff
	// fetch in GetDraftPageRevisionDiffUsecase).
	//
	// [Ja] 下書きはスペースメンバーごとの個人データのため、リクエストしたメンバー自身の
	// このページの下書きを解決し、それに属するリビジョンのみ受け付ける (403 ではなく 404。
	// 差分取得の GetDraftPageRevisionDiffUsecase と同じ方針)。
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

	// 3. Persistence: restore the draft content and record the new revision.
	// [Ja] 3. 永続化: 下書きの内容を復元し、新しいリビジョンを記録する。
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

// restore updates the draft with the revision's title/body and creates the new revision in one
// transaction. The body HTML is re-rendered through saveDraftPageContent (the same path as the
// manual save) instead of reusing the revision's stored body_html, so linked page IDs and
// attachment references stay consistent with the restored body.
//
// [Ja] restore は 1 トランザクションで下書きをリビジョンのタイトル・本文で更新し、新しい
// リビジョンを作成する。本文 HTML はリビジョンに保存済みの body_html を使い回さず、手動保存と
// 同じ経路 saveDraftPageContent で再レンダリングする。これによりリンク先ページ ID や添付参照が
// 復元後の本文と整合する。
func (uc *RestoreDraftPageRevisionUsecase) restore(ctx context.Context, data *pageAccessData, input RestoreDraftPageRevisionInput, revision *model.DraftPageRevision) (*RestoreDraftPageRevisionOutput, error) {
	now := time.Now()

	var titlePtr *string
	if revision.Title != "" {
		title := revision.Title
		titlePtr = &title
	}

	// Before the transaction: extract the featured image only (same as the manual save).
	// [Ja] トランザクション前: アイキャッチ画像のみ抽出する (手動保存と同様)。
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
