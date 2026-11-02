package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ManualSaveDraftPageUsecaseは下書きページの手動保存ユースケース
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

// NewManualSaveDraftPageUsecaseはManualSaveDraftPageUsecaseを生成する
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

// ManualSaveDraftPageInputは下書きページの手動保存の入力パラメータ
type ManualSaveDraftPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
	Title           *string
	Body            string
}

// ManualSaveDraftPageOutputは下書きページの手動保存の出力パラメータ
type ManualSaveDraftPageOutput struct {
	DraftPage *model.DraftPage

	// DraftPageRevisionはこの保存で作成されたリビジョン。送信されたタイトル・本文が
	// 直前の状態 (最新のリビジョン、無ければ下書きの起点) と同一でリビジョン作成をスキップした場合はnilになる
	// (保存自体は成功として扱う)。
	DraftPageRevision *model.DraftPageRevision
	TopicNumber       int32
}

// Executeはフォームから受け取った内容でDraftPageを更新し、リビジョンを作成する
func (uc *ManualSaveDraftPageUsecase) Execute(ctx context.Context, input ManualSaveDraftPageInput) (*ManualSaveDraftPageOutput, error) {
	// 1. データ取得
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := authorizeDraftPageSave(ctx, data); err != nil {
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

// needsRevisionは、送信されたタイトル・本文でリビジョンを作るべきかを返す。
// 直前の状態 (最新のリビジョン、無ければ下書きの起点) と同じ内容なら作らない。
// 下書きがまだ無いときは、この保存で作る下書きの起点 (saveDraftPageContentに渡すページの今の内容) と比べる
func (uc *ManualSaveDraftPageUsecase) needsRevision(ctx context.Context, data *pageAccessData, content model.DraftPageContent) (bool, error) {
	draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, data.page.ID, data.spaceMember.ID, data.space.ID)
	if err != nil {
		return false, fmt.Errorf("下書きページの取得に失敗しました: %w", err)
	}
	if draftPage == nil {
		return (&model.DraftPage{BaseTitle: data.page.Title, BaseBody: &data.page.Body}).NeedsRevision(nil, content), nil
	}

	latest, err := uc.draftPageRevisionRepo.ListByDraftPageID(ctx, draftPage.ID, data.space.ID, 1)
	if err != nil {
		return false, fmt.Errorf("最新リビジョンの取得に失敗しました: %w", err)
	}
	var latestRevision *model.DraftPageRevision
	if len(latest) > 0 {
		latestRevision = latest[0]
	}

	return draftPage.NeedsRevision(latestRevision, content), nil
}

func (uc *ManualSaveDraftPageUsecase) saveDraft(ctx context.Context, data *pageAccessData, input ManualSaveDraftPageInput) (*ManualSaveDraftPageOutput, error) {
	now := time.Now()

	// アイキャッチ画像の抽出はトランザクション前に済ませ、トランザクションの保持時間を抑える。
	// Wikiリンクの解決と下書き更新は、リンク先ページの自動作成と一緒にトランザクション内で行う。
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, input.Body, data.space.ID, uc.attachmentRepo)
	if err != nil {
		return nil, err
	}

	var title string
	if input.Title != nil {
		title = *input.Title
	}

	// こちらもトランザクション前: リビジョンを作るか判定する。直前の状態と比べないと、
	// 同一内容の連続保存 (cmd+sの連打など) や書き始めた内容のままの保存で、変化の無いリビジョンが積み上がってしまう。
	createRevision, err := uc.needsRevision(ctx, data, model.DraftPageContent{Title: title, Body: input.Body})
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
		CurrentTopicName:          data.topic.Name,
		BaseTitle:                 data.page.Title,
		BaseBody:                  data.page.Body,
	}

	// DraftPageのfind_or_create・Wikiリンクの解決・更新
	draftPage, err := saveDraftPageContent(ctx, contentInput, now,
		uc.draftPageRepo.WithTx(tx),
		uc.pageRepo.WithTx(tx),
		uc.pageEditorRepo.WithTx(tx),
		uc.topicRepo.WithTx(tx),
	)
	if err != nil {
		return nil, err
	}

	// この保存が直前の状態と同一内容の場合を除き、リビジョンを作成する。
	var revision *model.DraftPageRevision
	if createRevision {
		revision, err = draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
			DraftPageID:   draftPage.ID,
			SpaceID:       data.space.ID,
			SpaceMemberID: data.spaceMember.ID,
			Title:         title,
			Body:          input.Body,
		})
		if err != nil {
			return nil, fmt.Errorf("下書きページリビジョンの作成に失敗しました: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &ManualSaveDraftPageOutput{
		DraftPage:         draftPage,
		DraftPageRevision: revision,
		TopicNumber:       data.topic.Number,
	}, nil
}
