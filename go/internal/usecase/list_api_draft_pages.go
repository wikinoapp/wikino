package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ListAPIDraftPagesUsecaseは、公開APIでトークンの持ち主がトークンの属するスペースに持つ下書きの一覧を取得するユースケース
type ListAPIDraftPagesUsecase struct {
	draftPageRepo   *repository.DraftPageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewListAPIDraftPagesUsecaseはListAPIDraftPagesUsecaseを生成する
func NewListAPIDraftPagesUsecase(draftPageRepo *repository.DraftPageRepository, topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *ListAPIDraftPagesUsecase {
	return &ListAPIDraftPagesUsecase{
		draftPageRepo:   draftPageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// APIDraftPageCursorは、下書きの一覧で前のページの最後の下書きの位置を表す
type APIDraftPageCursor struct {
	ModifiedAt  time.Time
	DraftPageID model.DraftPageID
}

// ListAPIDraftPagesInputは下書きの一覧の取得に必要な入力パラメータ
type ListAPIDraftPagesInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	// Afterは前のページの最後の下書きの位置。この位置より後ろの下書きを返す。nilなら最初のページを返す
	After *APIDraftPageCursor
	// Limitは1ページの件数 (1以上)
	Limit int32
}

// ListAPIDraftPagesOutputは下書きの一覧の取得結果
type ListAPIDraftPagesOutput struct {
	// DraftPagesはトークンの持ち主の下書きを更新日時の新しい順に並べたもの。各下書きのPageはIDと番号だけを持つ
	DraftPages []*model.DraftPage
	// TopicsはDraftPagesの各下書きが属するトピック
	Topics map[model.TopicID]*model.Topic
	// HasNextは次のページがあるかどうか
	HasNext bool
}

// Executeはパスのスペースにトークンが属していれば、トークンの持ち主の下書きを更新日時の新しい順に返す。
// 他のメンバーの下書きは、所有者をクエリで限定して含めない。
// ゴミ箱のページの下書きは、ページの取得 (getPage) と揃えて含めない。
// 下書きを読めるか (draft_page:read) と非公開トピックを開けるかは、トピックごとに
// メンバーのスコープとトークンのスコープの論理積で判定する
func (uc *ListAPIDraftPagesUsecase) Execute(ctx context.Context, input ListAPIDraftPagesInput) (*ListAPIDraftPagesOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	empty := &ListAPIDraftPagesOutput{DraftPages: []*model.DraftPage{}, Topics: map[model.TopicID]*model.Topic{}}
	spaceMember := input.Principal.SpaceMember
	if spaceMember == nil {
		return empty, nil
	}

	access, err := fetchAPITopicAccess(ctx, uc.topicRepo, uc.topicMemberRepo, input.Principal)
	if err != nil {
		return nil, err
	}

	var topicIDs []model.TopicID
	for _, topic := range access.visibleTopics() {
		if access.authorizer(topic.ID).CanShowDraftPage() {
			topicIDs = append(topicIDs, topic.ID)
		}
	}
	if len(topicIDs) == 0 {
		return empty, nil
	}

	listInput := repository.ListDraftPagesByModifiedAtInput{
		SpaceID:       space.ID,
		SpaceMemberID: spaceMember.ID,
		TopicIDs:      topicIDs,
		// 次のページがあるかを知るため、1件多く取得する
		Limit: input.Limit + 1,
	}
	if input.After != nil {
		listInput.AfterModifiedAt = &input.After.ModifiedAt
		listInput.AfterID = &input.After.DraftPageID
	}
	drafts, err := uc.draftPageRepo.ListByModifiedAt(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("下書き一覧の取得に失敗: %w", err)
	}

	hasNext := len(drafts) > int(input.Limit)
	if hasNext {
		drafts = drafts[:input.Limit]
	}

	topics := make(map[model.TopicID]*model.Topic)
	for _, draft := range drafts {
		if topic, ok := access.topicByID[draft.TopicID]; ok {
			topics[draft.TopicID] = topic
		}
	}

	return &ListAPIDraftPagesOutput{
		DraftPages: drafts,
		Topics:     topics,
		HasNext:    hasNext,
	}, nil
}
