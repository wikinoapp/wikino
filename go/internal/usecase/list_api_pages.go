package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ListAPIPagesUsecaseは、公開APIでトークンが属するスペースのページの一覧を取得するユースケース
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

// APIPageStatusFilterは、ページの一覧に含めるページの公開状態
type APIPageStatusFilter string

const (
	// APIPageStatusFilterPublishedは公開済みのページだけを含める
	APIPageStatusFilterPublished APIPageStatusFilter = "published"
	// APIPageStatusFilterUnpublishedは未公開のページだけを含める
	APIPageStatusFilterUnpublished APIPageStatusFilter = "unpublished"
	// APIPageStatusFilterAllは公開済みと未公開の両方のページを含める
	APIPageStatusFilterAll APIPageStatusFilter = "all"
)

// ListAPIPagesInputはページの一覧の取得に必要な入力パラメータ
type ListAPIPagesInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	// TopicNumberはこの番号のトピックのページに絞る。nilなら絞らない
	TopicNumber *int32
	// Statusは含めるページの公開状態。空なら公開済みのページだけを含める
	Status APIPageStatusFilter
	// Titleはこのタイトルのページに絞る (大文字小文字を区別しない)。nilなら絞らない
	Title *string
	// Keywordsはすべてのキーワードをタイトルか本文に含むページに絞る (大文字小文字を区別しない部分一致)。
	// 空なら絞らない
	Keywords []string
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

// Executeはパスのスペースにトークンが属していれば、閲覧者が開けるトピックのページを
// 更新日時の新しい順に返す。ゴミ箱のページは含めない。
// 未公開のページはStatusで指定したときだけ含め、Wikiリンクの入力補完と同じく、開けるトピックの
// 公開済みのページかトークンの持ち主の下書きからリンクされているものに限る。
// 本文を書くべきリンク先を探すための一覧であり、リンクされずに残ったページは含めない。
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

	var visibleTopicIDs, topicIDs []model.TopicID
	for _, topic := range access.visibleTopics() {
		visibleTopicIDs = append(visibleTopicIDs, topic.ID)
		if input.TopicNumber != nil && topic.Number != *input.TopicNumber {
			continue
		}
		topicIDs = append(topicIDs, topic.ID)
	}
	// 存在しないトピックや開けないトピックで絞り込んだ場合は、トピックの存在を明かさないよう
	// 404ではなく空の一覧を返す
	if len(topicIDs) == 0 {
		return &ListAPIPagesOutput{Pages: []*model.Page{}, Topics: map[model.TopicID]*model.Topic{}}, nil
	}

	listInput := repository.ListByModifiedAtInput{
		SpaceID:            space.ID,
		TopicIDs:           topicIDs,
		VisibleTopicIDs:    visibleTopicIDs,
		SpaceMemberID:      input.Principal.SpaceMember.ID,
		IncludePublished:   input.Status != APIPageStatusFilterUnpublished,
		IncludeUnpublished: input.Status == APIPageStatusFilterUnpublished || input.Status == APIPageStatusFilterAll,
		Title:              input.Title,
		Keywords:           input.Keywords,
		ModifiedSince:      input.ModifiedSince,
		// 次のページがあるかを知るため、1件多く取得する
		Limit: input.Limit + 1,
	}
	if input.After != nil {
		listInput.AfterModifiedAt = &input.After.ModifiedAt
		listInput.AfterID = &input.After.PageID
	}
	pages, err := uc.pageRepo.ListByModifiedAt(ctx, listInput)
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
