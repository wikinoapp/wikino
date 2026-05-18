package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// homeJoinedTopicsLimit is the upper bound on the number of joined topics surfaced on the
// home page. Kept aligned with viewmodel.SidebarJoinedTopicsLimit (= 10) so home and the
// sidebar show the same "topics the user has been active in" set; the value is duplicated
// here rather than imported because Application layer cannot depend on Presentation layer.
//
// [Ja] homeJoinedTopicsLimit はホーム画面に表示する参加中トピック数の上限。
// viewmodel.SidebarJoinedTopicsLimit (= 10) と揃えてあり、ホームとサイドバーで
// 「ユーザーが直近で動いたトピック」の同じセットを表示できるようにしている。
// Application 層は Presentation 層に依存できないため、import せずに値を二重定義している。
const homeJoinedTopicsLimit = 10

// GetHomeShowUsecase はホーム画面表示用ユースケース
type GetHomeShowUsecase struct {
	spaceRepo *repository.SpaceRepository
	topicRepo *repository.TopicRepository
}

// NewGetHomeShowUsecase は GetHomeShowUsecase を生成する
func NewGetHomeShowUsecase(
	spaceRepo *repository.SpaceRepository,
	topicRepo *repository.TopicRepository,
) *GetHomeShowUsecase {
	return &GetHomeShowUsecase{
		spaceRepo: spaceRepo,
		topicRepo: topicRepo,
	}
}

// GetHomeShowInput はホーム画面表示の入力パラメータ
type GetHomeShowInput struct {
	UserID model.UserID
}

// GetHomeShowOutput contains the data shown on the home page: the user's active spaces
// and the topics the user is joined to (with the published page count for each topic).
//
// JoinedTopics is intentionally typed as []repository.JoinedTopicWithStats so the UseCase
// can hand the Repository result through unchanged. The Handler iterates over this slice
// to build ViewModel cards; ViewModel itself stays decoupled from Infrastructure types via
// viewmodel.NewJoinedTopicCard, which takes (*model.Topic, count int32).
//
// [Ja] GetHomeShowOutput はホーム画面に表示するデータ。ユーザーが参加中のスペース一覧と、
// 参加中のトピック一覧 (各トピックの公開中ページ数付き) を保持する。
//
// JoinedTopics は意図的に []repository.JoinedTopicWithStats のままにしている。
// UseCase は Repository 戻り値をそのまま渡し、ViewModel カードへの変換は Handler 側で
// viewmodel.NewJoinedTopicCard((*model.Topic, count int32)) を介して行うことで、
// ViewModel が Infrastructure 層に依存しない設計を維持している。
type GetHomeShowOutput struct {
	ActiveSpaces []*model.Space
	JoinedTopics []repository.JoinedTopicWithStats
}

// Execute はホーム画面に表示する参加中スペースと参加中トピックを取得する。
func (uc *GetHomeShowUsecase) Execute(ctx context.Context, input GetHomeShowInput) (*GetHomeShowOutput, error) {
	spaces, err := uc.spaceRepo.ListActiveByUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("参加中スペース一覧の取得に失敗: %w", err)
	}

	joinedTopics, err := uc.topicRepo.ListJoinedWithStatsByUser(ctx, input.UserID, homeJoinedTopicsLimit)
	if err != nil {
		return nil, fmt.Errorf("参加中トピック一覧の取得に失敗: %w", err)
	}

	return &GetHomeShowOutput{
		ActiveSpaces: spaces,
		JoinedTopics: joinedTopics,
	}, nil
}
