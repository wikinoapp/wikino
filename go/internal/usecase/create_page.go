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

// CreatePageUsecaseはページ新規作成の入口からページを作成するユースケース。常に空ページを
// 作成し、タイトルまたは本文が事前入力されているときは同じトランザクションでそれらを下書きとして
// 保存することで、編集画面がその値の入った状態で開くようにする。
type CreatePageUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	pageRepo        *repository.PageRepository
	pageEditorRepo  *repository.PageEditorRepository
	draftPageRepo   *repository.DraftPageRepository
	attachmentRepo  *repository.AttachmentRepository
}

// NewCreatePageUsecaseはCreatePageUsecaseを生成する。
func NewCreatePageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	draftPageRepo *repository.DraftPageRepository,
	attachmentRepo *repository.AttachmentRepository,
) *CreatePageUsecase {
	return &CreatePageUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		pageRepo:        pageRepo,
		pageEditorRepo:  pageEditorRepo,
		draftPageRepo:   draftPageRepo,
		attachmentRepo:  attachmentRepo,
	}
}

// CreatePageInputはページ作成の入力パラメータ。TitleとBodyは事前入力された値で、
// どちらも空のときは下書きを作らない。
type CreatePageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          model.UserID
	Title           string
	Body            string
}

// CreatePageOutputは作成されたページを保持する。
type CreatePageOutput struct {
	Page *model.Page
}

// createPageAccessDataはページ作成の認可と永続化に必要な解決済みのデータを保持する。
type createPageAccessData struct {
	space       *model.Space
	spaceMember *model.SpaceMember
	topic       *model.Topic
	topicMember *model.TopicMember
}

// Executeはページを作成する。
func (uc *CreatePageUsecase) Execute(ctx context.Context, input CreatePageInput) (*CreatePageOutput, error) {
	// 1. 必要なデータを取得する。
	data, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 操作を認可する。
	if err := authorizeCreatePage(ctx, data); err != nil {
		return nil, err
	}

	// 3. ページを永続化する。
	return uc.createPage(ctx, data, input)
}

func (uc *CreatePageUsecase) fetchData(ctx context.Context, input CreatePageInput) (*createPageAccessData, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	topic, err := uc.topicRepo.FindBySpaceAndNumber(ctx, space.ID, input.TopicNumber)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	var topicMember *model.TopicMember
	if spaceMember != nil {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	return &createPageAccessData{
		space:       space,
		spaceMember: spaceMember,
		topic:       topic,
		topicMember: topicMember,
	}, nil
}

// authorizeCreatePageは対象トピックでのページ作成を認可する。
func authorizeCreatePage(ctx context.Context, data *createPageAccessData) error {
	if data.spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	if !newAuthorizer(data.spaceMember, data.topicMember).CanCreatePage() {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

// createPageは空ページの作成・編集者の登録・事前入力値の下書き保存を1つの
// トランザクションで行う。下書きの書き込みに失敗したページの編集画面へ遷移してしまうことを
// 防ぐため。
func (uc *CreatePageUsecase) createPage(ctx context.Context, data *createPageAccessData, input CreatePageInput) (*CreatePageOutput, error) {
	now := time.Now()

	// 自動保存の経路と同じ順序で、トランザクション前にアイキャッチ画像を抽出する。本文HTMLの
	// レンダリングとWikiリンクの解決はsaveDraftPageContentがトランザクション内で行う。
	var featuredImageAttachmentID *model.AttachmentID
	if hasPrefilledContent(input) {
		var err error
		featuredImageAttachmentID, err = extractFeaturedImageAttachmentID(ctx, input.Body, data.space.ID, uc.attachmentRepo)
		if err != nil {
			return nil, err
		}
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	pageRepo := uc.pageRepo.WithTx(tx)
	pageEditorRepo := uc.pageEditorRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)
	topicRepo := uc.topicRepo.WithTx(tx)
	attachmentRepo := uc.attachmentRepo.WithTx(tx)

	// 1. 空ページを作成する。
	nextNumber, err := pageRepo.NextPageNumber(ctx, data.space.ID)
	if err != nil {
		return nil, fmt.Errorf("次のページ番号の取得に失敗しました: %w", err)
	}

	page, err := pageRepo.CreateBlankPage(ctx, repository.CreateBlankPageInput{
		SpaceID: data.space.ID,
		TopicID: data.topic.ID,
		Number:  nextNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの作成に失敗しました: %w", err)
	}

	// 2. 作成者を編集者として登録する。
	_, err = pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            data.space.ID,
		PageID:             page.ID,
		SpaceMemberID:      data.spaceMember.ID,
		LastPageModifiedAt: page.ModifiedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの編集者登録に失敗しました: %w", err)
	}

	// 3. 事前入力された値を下書きとして保存する。
	if hasPrefilledContent(input) {
		// 本文だけが事前入力されたときは、基になる空ページと同様に下書きのタイトルを
		// 未設定のままにする。
		var draftTitle *string
		if input.Title != "" {
			draftTitle = &input.Title
		}

		_, err = saveDraftPageContent(ctx, saveDraftPageContentInput{
			SpaceID:                   data.space.ID,
			PageID:                    page.ID,
			SpaceMemberID:             data.spaceMember.ID,
			TopicID:                   data.topic.ID,
			Title:                     draftTitle,
			Body:                      input.Body,
			FeaturedImageAttachmentID: featuredImageAttachmentID,
			SpaceIdentifier:           input.SpaceIdentifier,
			CurrentTopicName:          data.topic.Name,
		}, now,
			draftPageRepo,
			pageRepo,
			pageEditorRepo,
			topicRepo,
			attachmentRepo,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &CreatePageOutput{
		Page: page,
	}, nil
}

// hasPrefilledContentは入口が下書きとして残すべき値を渡されたかを返す。
func hasPrefilledContent(input CreatePageInput) bool {
	return input.Title != "" || input.Body != ""
}
