package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetTopicFeedUsecaseはトピックのフィード (GET /s/:space_identifier/topics/:topic_number/atom) に
// 載せる内容を取得する読み取りUseCase
//
// フィードリーダーはログインの状態を持たずに取得するため、閲覧者を受け取らず、公開トピックのフィードだけを返す。
// 公開トピックの判定には、ページのカード画像 (`GetPageOgImageUsecase`) と同じゲスト用Policyを使う。
type GetTopicFeedUsecase struct {
	cfg            *config.Config
	spaceRepo      *repository.SpaceRepository
	pageRepo       *repository.PageRepository
	topicRepo      *repository.TopicRepository
	attachmentRepo *repository.AttachmentRepository
}

// NewGetTopicFeedUsecaseはGetTopicFeedUsecaseを生成する
func NewGetTopicFeedUsecase(
	cfg *config.Config,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	attachmentRepo *repository.AttachmentRepository,
) *GetTopicFeedUsecase {
	return &GetTopicFeedUsecase{
		cfg:            cfg,
		spaceRepo:      spaceRepo,
		pageRepo:       pageRepo,
		topicRepo:      topicRepo,
		attachmentRepo: attachmentRepo,
	}
}

// GetTopicFeedInputはUseCaseの入力
type GetTopicFeedInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
}

// GetTopicFeedOutputはUseCaseの出力
//
// Pagesは新しく公開された順に並ぶ。ContentHTMLByPageIDは、Pagesの各ページの本文を
// 絶対URLで描画した全文のHTMLである。
type GetTopicFeedOutput struct {
	Space               *model.Space
	Topic               *model.Topic
	Pages               []*model.Page
	ContentHTMLByPageID map[model.PageID]string
}

// Executeはトピックのフィードに載せる内容を取得する。
// スペースかトピックが無い (論理削除されたものを含む) ときと、トピックが非公開のときは、
// 同じAppErrCodeResourceNotFoundの *model.AppErrorを返し、「隠している」と「存在しない」を区別しない。
func (uc *GetTopicFeedUsecase) Execute(ctx context.Context, input GetTopicFeedInput) (*GetTopicFeedOutput, error) {
	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, notFound
	}

	// FindBySpaceAndNumberは論理削除されたトピックを返さない
	topic, err := uc.topicRepo.FindBySpaceAndNumber(ctx, space.ID, input.TopicNumber)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil || !policy.NewGuestPolicy().CanShowTopic(topic) {
		return nil, notFound
	}

	pages, err := uc.pageRepo.FindRecentPublishedByTopic(ctx, space.ID, topic.ID, feedEntryLimit)
	if err != nil {
		return nil, fmt.Errorf("フィードに載せるページの取得に失敗: %w", err)
	}

	repos := feedContentRepos{topicRepo: uc.topicRepo, pageRepo: uc.pageRepo, attachmentRepo: uc.attachmentRepo}
	contents, err := renderFeedContents(ctx, repos, uc.cfg.AppURL(), space, pages)
	if err != nil {
		return nil, err
	}

	return &GetTopicFeedOutput{
		Space:               space,
		Topic:               topic,
		Pages:               pages,
		ContentHTMLByPageID: contents,
	}, nil
}
