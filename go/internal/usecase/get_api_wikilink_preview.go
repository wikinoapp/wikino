package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAPIWikilinkPreviewUsecaseは、公開APIで本文を書き込んだときに各Wikiリンクがどう解決されるか
// (既存のページ / 作成されるページ / トピックが無い) を、書き込まずに求めるユースケース
type GetAPIWikilinkPreviewUsecase struct {
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetAPIWikilinkPreviewUsecaseはGetAPIWikilinkPreviewUsecaseを生成する
func NewGetAPIWikilinkPreviewUsecase(pageRepo *repository.PageRepository, topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *GetAPIWikilinkPreviewUsecase {
	return &GetAPIWikilinkPreviewUsecase{
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetAPIWikilinkPreviewInputはWikiリンクのプレビューに必要な入力パラメータ
type GetAPIWikilinkPreviewInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	// TopicNumberは本文を書き込むページのトピックの番号。トピックを指定しないWikiリンクの解決に使う
	TopicNumber int32
	Body        string
}

// WikilinkResolutionはWikiリンクの解決の結果
type WikilinkResolution string

const (
	// WikilinkResolutionExistingPageは、リンク先のページが既にあることを表す
	WikilinkResolutionExistingPage WikilinkResolution = "existing_page"
	// WikilinkResolutionNewPageは、書き込むとリンク先のページが未公開のページとして作られることを表す
	WikilinkResolutionNewPage WikilinkResolution = "new_page"
	// WikilinkResolutionTopicNotFoundは、リンク先のトピックが無いか見られず、リンクにならないことを表す
	WikilinkResolutionTopicNotFound WikilinkResolution = "topic_not_found"
)

// WikilinkPreviewLinkは本文の1つのWikiリンクの解決の結果
type WikilinkPreviewLink struct {
	Key        markup.WikilinkKey
	Resolution WikilinkResolution
	// Pageはリンク先の既存のページ。既存のページでも、ゴミ箱に入っているなどでトークンでは開けないページではnil
	Page *model.Page
}

// GetAPIWikilinkPreviewOutputはWikiリンクのプレビューの結果
type GetAPIWikilinkPreviewOutput struct {
	// Linksは本文のWikiリンクを、同じリンク先の重複を除いて現れる順に並べたもの
	Links []WikilinkPreviewLink
	// TopicsはLinksの各ページが属するトピック
	Topics map[model.TopicID]*model.Topic
}

// Executeは本文のWikiリンクを、ページの作成・更新と同じ解釈とリンク先の一括取得 (findWikilinkTargets) で解決する。
// 書き込みと同じく、トークンで見られないトピックにはページを作らない扱いにし、そのトピックへのリンクは
// ページの有無を明かさないよう、トピックが無いリンクと区別しない。
// 存在しないトピックと見られないトピックを `topic_number` に指定した場合は、ページの作成と同じく区別せず、
// リクエスト本文の `topic_number` の問題として扱う
func (uc *GetAPIWikilinkPreviewUsecase) Execute(ctx context.Context, input GetAPIWikilinkPreviewInput) (*GetAPIWikilinkPreviewOutput, error) {
	// 1. データ取得
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	access, err := fetchAPITopicAccess(ctx, uc.topicRepo, uc.topicMemberRepo, input.Principal)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	var currentTopic *model.Topic
	for _, topic := range access.topics {
		if topic.Number == input.TopicNumber {
			currentTopic = topic
			break
		}
	}
	if currentTopic == nil || !access.canShowTopic(currentTopic) {
		ve := model.NewValidationError()
		ve.AddField("topic_number", i18n.T(ctx, "api_error_topic_not_found"))
		return nil, ve
	}

	// 3. ビジネスロジック
	keys := markup.ScanWikilinks(input.Body, currentTopic.Name)
	targets, err := findWikilinkTargets(ctx, keys, space.ID, uc.topicRepo, uc.pageRepo)
	if err != nil {
		return nil, fmt.Errorf("リンク先のトピックとページの解決に失敗: %w", err)
	}

	showablePages, err := uc.findShowablePages(ctx, targets, space, access)
	if err != nil {
		return nil, err
	}

	links := make([]WikilinkPreviewLink, 0, len(targets))
	var pages []*model.Page
	for _, target := range targets {
		link := WikilinkPreviewLink{Key: target.key}
		switch {
		case target.topic == nil || !access.canShowTopic(target.topic):
			link.Resolution = WikilinkResolutionTopicNotFound
		case target.page != nil:
			link.Resolution = WikilinkResolutionExistingPage
			link.Page = showablePages[target.page.ID]
			if link.Page != nil {
				pages = append(pages, link.Page)
			}
		default:
			link.Resolution = WikilinkResolutionNewPage
		}
		links = append(links, link)
	}

	return &GetAPIWikilinkPreviewOutput{
		Links:  links,
		Topics: access.topicMapForPages(pages),
	}, nil
}

// findShowablePagesは見られるトピックの既存のリンク先のうち、ページの取得 (getPage) で開けるページをIDで引けるように返す。
// ゴミ箱のページと廃棄済みのページは、リンク先が既にある (作られない) ことだけを示し、ページの内容は返さない
func (uc *GetAPIWikilinkPreviewUsecase) findShowablePages(ctx context.Context, targets []wikilinkTarget, space *model.Space, access *topicAccess) (map[model.PageID]*model.Page, error) {
	var ids []model.PageID
	for _, target := range targets {
		if target.page != nil && access.canShowTopic(target.topic) {
			ids = append(ids, target.page.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	pages, err := uc.pageRepo.FindMetadataByIDs(ctx, ids, space.ID)
	if err != nil {
		return nil, fmt.Errorf("リンク先のページの取得に失敗: %w", err)
	}
	showable := make(map[model.PageID]*model.Page, len(pages))
	for _, page := range pages {
		if isAPIVisiblePage(page) && access.canShowPage(page) {
			showable[page.ID] = page
		}
	}
	return showable, nil
}
