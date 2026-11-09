package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSpaceFeedUsecaseはスペースのフィード (GET /s/:space_identifier/atom) に載せる内容を取得する読み取りUseCase
//
// フィードリーダーはログインの状態を持たずに取得するため、閲覧者を受け取らず、
// 誰が取得しても公開トピックのページだけを返す。
type GetSpaceFeedUsecase struct {
	cfg            *config.Config
	spaceRepo      *repository.SpaceRepository
	pageRepo       *repository.PageRepository
	topicRepo      *repository.TopicRepository
	attachmentRepo *repository.AttachmentRepository
}

// NewGetSpaceFeedUsecaseはGetSpaceFeedUsecaseを生成する
func NewGetSpaceFeedUsecase(
	cfg *config.Config,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	attachmentRepo *repository.AttachmentRepository,
) *GetSpaceFeedUsecase {
	return &GetSpaceFeedUsecase{
		cfg:            cfg,
		spaceRepo:      spaceRepo,
		pageRepo:       pageRepo,
		topicRepo:      topicRepo,
		attachmentRepo: attachmentRepo,
	}
}

// GetSpaceFeedInputはUseCaseの入力
type GetSpaceFeedInput struct {
	SpaceIdentifier model.SpaceIdentifier
}

// GetSpaceFeedOutputはUseCaseの出力
//
// Pagesは新しく公開された順に並ぶ。ContentHTMLByPageIDは、Pagesの各ページの本文を
// 絶対URLで描画した全文のHTMLである。
type GetSpaceFeedOutput struct {
	Space               *model.Space
	Pages               []*model.Page
	ContentHTMLByPageID map[model.PageID]string
}

// Executeはスペースのフィードに載せる内容を取得する。
// スペースが無いときはAppErrCodeResourceNotFoundの *model.AppErrorを返す。
func (uc *GetSpaceFeedUsecase) Execute(ctx context.Context, input GetSpaceFeedInput) (*GetSpaceFeedOutput, error) {
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

	pages, err := uc.pageRepo.FindRecentPublishedBySpace(ctx, space.ID, feedEntryLimit)
	if err != nil {
		return nil, fmt.Errorf("フィードに載せるページの取得に失敗: %w", err)
	}

	repos := feedContentRepos{topicRepo: uc.topicRepo, pageRepo: uc.pageRepo, attachmentRepo: uc.attachmentRepo}
	contents, err := renderFeedContents(ctx, repos, uc.cfg.AppURL(), space, pages)
	if err != nil {
		return nil, err
	}

	return &GetSpaceFeedOutput{
		Space:               space,
		Pages:               pages,
		ContentHTMLByPageID: contents,
	}, nil
}
