package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ListAPITopicsUsecaseは、公開APIでトークンを束縛したスペースのトピックの一覧を取得するユースケース
type ListAPITopicsUsecase struct {
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewListAPITopicsUsecaseはListAPITopicsUsecaseを生成する
func NewListAPITopicsUsecase(topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *ListAPITopicsUsecase {
	return &ListAPITopicsUsecase{
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// ListAPITopicsInputはトピックの一覧の取得に必要な入力パラメータ
type ListAPITopicsInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	// AfterNumberは前のページの最後のトピックの番号。この番号より大きいトピックを返す。nilなら最初のページを返す
	AfterNumber *int32
	// Limitは1ページの件数 (1以上)
	Limit int32
}

// ListAPITopicsOutputはトピックの一覧の取得結果
type ListAPITopicsOutput struct {
	// Topicsは閲覧者が開けるトピックを番号の小さい順に並べたもの
	Topics []*model.Topic
	// HasNextは次のページがあるかどうか
	HasNext bool
}

// Executeはパスのスペースがトークンの束縛先であれば、閲覧者が開けるトピックをトピック番号の順に返す。
// 非公開トピックを開けるかは、メンバーのスコープとトークンのスコープの論理積で判定する。
//
// 閲覧者が開けるトピックの判定には、スペースのトピックと閲覧者のトピックメンバーをすべて取得する
// 必要がある (Web画面のページ一覧と同じ)。そのため一覧のカーソルもSQLではなく、取得済みのトピックに
// 対して適用する。トピック番号はスペース内で一意なため、番号だけで並び順とカーソルの位置が決まる
func (uc *ListAPITopicsUsecase) Execute(ctx context.Context, input ListAPITopicsInput) (*ListAPITopicsOutput, error) {
	if _, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier); err != nil {
		return nil, err
	}

	access, err := fetchAPITopicAccess(ctx, uc.topicRepo, uc.topicMemberRepo, input.Principal)
	if err != nil {
		return nil, err
	}

	topics := access.visibleTopics()
	if input.AfterNumber != nil {
		start := len(topics)
		for i, topic := range topics {
			if topic.Number > *input.AfterNumber {
				start = i
				break
			}
		}
		topics = topics[start:]
	}

	hasNext := len(topics) > int(input.Limit)
	if hasNext {
		topics = topics[:input.Limit]
	}

	return &ListAPITopicsOutput{Topics: topics, HasNext: hasNext}, nil
}
