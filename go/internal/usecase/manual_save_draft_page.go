package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ManualSaveDraftPageUsecase は下書きページの手動保存ユースケース
type ManualSaveDraftPageUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	pageRepo              *repository.PageRepository
	pageEditorRepo        *repository.PageEditorRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
}

// NewManualSaveDraftPageUsecase は ManualSaveDraftPageUsecase を生成する
func NewManualSaveDraftPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
) *ManualSaveDraftPageUsecase {
	return &ManualSaveDraftPageUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		pageRepo:              pageRepo,
		pageEditorRepo:        pageEditorRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
	}
}

// ManualSaveDraftPageInput は下書きページの手動保存の入力パラメータ
type ManualSaveDraftPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
	Title           *string
	Body            string
}

// ManualSaveDraftPageOutput は下書きページの手動保存の出力パラメータ
type ManualSaveDraftPageOutput struct {
	DraftPage *model.DraftPage

	// DraftPageRevision is the revision created by this save. It is nil when the submitted
	// title/body are identical to the draft's latest revision and revision creation was skipped
	// (the save itself still succeeds).
	//
	// [Ja] DraftPageRevision はこの保存で作成されたリビジョン。送信されたタイトル・本文が
	// 下書きの最新リビジョンと同一でリビジョン作成をスキップした場合は nil になる
	// (保存自体は成功として扱う)。
	DraftPageRevision *model.DraftPageRevision
	TopicNumber       int32
}

// Execute はフォームから受け取った内容でDraftPageを更新し、リビジョンを作成する
func (uc *ManualSaveDraftPageUsecase) Execute(ctx context.Context, input ManualSaveDraftPageInput) (*ManualSaveDraftPageOutput, error) {
	// 1. データ取得
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// 3. 永続化
	return uc.saveDraft(ctx, data, input)
}

func (uc *ManualSaveDraftPageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

// isSameAsLatestRevision reports whether the submitted title/body are identical to the draft's
// latest revision. When the draft does not exist yet there is no revision to duplicate, so it
// returns false.
//
// [Ja] isSameAsLatestRevision は送信されたタイトル・本文が下書きの最新リビジョンと同一かどうかを
// 返す。下書きがまだ存在しない場合は重複するリビジョンも存在しないため false を返す。
func (uc *ManualSaveDraftPageUsecase) isSameAsLatestRevision(ctx context.Context, data *pageAccessData, title, body string) (bool, error) {
	draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, data.page.ID, data.spaceMember.ID, data.space.ID)
	if err != nil {
		return false, fmt.Errorf("下書きページの取得に失敗しました: %w", err)
	}
	if draftPage == nil {
		return false, nil
	}

	latest, err := uc.draftPageRevisionRepo.ListByDraftPageID(ctx, draftPage.ID, data.space.ID, 1)
	if err != nil {
		return false, fmt.Errorf("最新リビジョンの取得に失敗しました: %w", err)
	}

	return len(latest) > 0 && latest[0].Title == title && latest[0].Body == body, nil
}

func (uc *ManualSaveDraftPageUsecase) saveDraft(ctx context.Context, data *pageAccessData, input ManualSaveDraftPageInput) (*ManualSaveDraftPageOutput, error) {
	now := time.Now()

	// Before the transaction: extract the featured image only. The body HTML
	// rendering and wiki link resolution are unified into saveDraftPageContent,
	// which runs inside the transaction.
	//
	// [Ja] トランザクション前: アイキャッチ画像のみ抽出する。bodyHTML 本体のレンダリングと
	// Wiki リンクの解決はトランザクション内の saveDraftPageContent に一本化した。
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, input.Body, data.space.ID, uc.attachmentRepo)
	if err != nil {
		return nil, err
	}

	var title string
	if input.Title != nil {
		title = *input.Title
	}

	// Also before the transaction: decide whether to skip revision creation. Repeated saves of
	// identical content (e.g. mashing cmd+s) would otherwise pile up duplicate revisions.
	//
	// [Ja] こちらもトランザクション前: リビジョン作成をスキップするか判定する。スキップしないと
	// 同一内容の連続保存 (cmd+s の連打など) で重複リビジョンが積み上がってしまう。
	skipRevision, err := uc.isSameAsLatestRevision(ctx, data, title, input.Body)
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

	draftPageRevisionRepo := uc.draftPageRevisionRepo.WithTx(tx)

	contentInput := saveDraftPageContentInput{
		SpaceID:                   data.space.ID,
		PageID:                    data.page.ID,
		SpaceMemberID:             data.spaceMember.ID,
		TopicID:                   data.page.TopicID,
		Title:                     input.Title,
		Body:                      input.Body,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		SpaceIdentifier:           input.SpaceIdentifier,
		CurrentTopicName:          data.topic.Name,
	}

	// DraftPageのfind_or_create・レンダリング・更新
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

	// Create the revision unless this save is content-identical to the latest one.
	// [Ja] この保存が最新リビジョンと同一内容の場合を除き、リビジョンを作成する。
	var revision *model.DraftPageRevision
	if !skipRevision {
		revision, err = draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
			DraftPageID:   result.DraftPage.ID,
			SpaceID:       data.space.ID,
			SpaceMemberID: data.spaceMember.ID,
			Title:         title,
			Body:          input.Body,
			BodyHTML:      result.BodyHTML,
		})
		if err != nil {
			return nil, fmt.Errorf("下書きページリビジョンの作成に失敗しました: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &ManualSaveDraftPageOutput{
		DraftPage:         result.DraftPage,
		DraftPageRevision: revision,
		TopicNumber:       data.topic.Number,
	}, nil
}
