package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ListAPIPagesUsecaseは、公開APIでトークンを束縛したスペースのページの一覧を取得するユースケース
type ListAPIPagesUsecase struct {
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewListAPIPagesUsecaseはListAPIPagesUsecaseを生成する
func NewListAPIPagesUsecase(pageRepo *repository.PageRepository, topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *ListAPIPagesUsecase {
	return &ListAPIPagesUsecase{
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// APIPageCursorは、ページの一覧で前のページの最後のページの位置を表す
type APIPageCursor struct {
	ModifiedAt time.Time
	PageID     model.PageID
}

// ListAPIPagesInputはページの一覧の取得に必要な入力パラメータ
type ListAPIPagesInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	// TopicNumberはこの番号のトピックのページに絞る。nilなら絞らない
	TopicNumber *int32
	// ModifiedSinceはこの日時以降に更新されたページに絞る。nilなら絞らない
	ModifiedSince *time.Time
	// Afterは前のページの最後のページの位置。この位置より後ろのページを返す。nilなら最初のページを返す
	After *APIPageCursor
	// Limitは1ページの件数 (1以上)
	Limit int32
}

// ListAPIPagesOutputはページの一覧の取得結果
type ListAPIPagesOutput struct {
	// Pagesは閲覧者が開けるページを更新日時の新しい順に並べたもの
	Pages []*model.Page
	// TopicsはPagesの各ページが属するトピック
	Topics map[model.TopicID]*model.Topic
	// HasNextは次のページがあるかどうか
	HasNext bool
}

// Executeはパスのスペースがトークンの束縛先であれば、閲覧者が開けるトピックの公開済みのページを
// 更新日時の新しい順に返す。下書きのまま公開していないページとゴミ箱のページは含めない。
// 非公開トピックを開けるかは、メンバーのスコープとトークンのスコープの論理積で判定する。
//
// ページの数はトピックと違って伸び続けるため、閲覧者が開けるトピックだけをこのユースケースで
// 解決し、絞り込みとカーソルはSQLで適用する
func (uc *ListAPIPagesUsecase) Execute(ctx context.Context, input ListAPIPagesInput) (*ListAPIPagesOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	access, err := fetchAPITopicAccess(ctx, uc.topicRepo, uc.topicMemberRepo, input.Principal)
	if err != nil {
		return nil, err
	}

	var visibleTopicIDs []model.TopicID
	for _, topic := range access.visibleTopics() {
		if input.TopicNumber != nil && topic.Number != *input.TopicNumber {
			continue
		}
		visibleTopicIDs = append(visibleTopicIDs, topic.ID)
	}
	// 存在しないトピックや開けないトピックで絞り込んだ場合は、トピックの存在を明かさないよう
	// 404ではなく空の一覧を返す
	if len(visibleTopicIDs) == 0 {
		return &ListAPIPagesOutput{Pages: []*model.Page{}, Topics: map[model.TopicID]*model.Topic{}}, nil
	}

	listInput := repository.ListPublishedByModifiedAtInput{
		SpaceID:         space.ID,
		VisibleTopicIDs: visibleTopicIDs,
		ModifiedSince:   input.ModifiedSince,
		// 次のページがあるかを知るため、1件多く取得する
		Limit: input.Limit + 1,
	}
	if input.After != nil {
		listInput.AfterModifiedAt = &input.After.ModifiedAt
		listInput.AfterID = &input.After.PageID
	}
	pages, err := uc.pageRepo.ListPublishedByModifiedAt(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("ページ一覧の取得に失敗: %w", err)
	}

	hasNext := len(pages) > int(input.Limit)
	if hasNext {
		pages = pages[:input.Limit]
	}

	return &ListAPIPagesOutput{
		Pages:   pages,
		Topics:  access.topicMapForPages(pages),
		HasNext: hasNext,
	}, nil
}
