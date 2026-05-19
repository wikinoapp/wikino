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

// homeDraftPagesLimit is the upper bound on the number of draft pages surfaced on the home
// page. Kept aligned with viewmodel.SidebarDraftPagesLimit (= 5) so home and the sidebar
// show the same set of recent drafts; duplicated here rather than imported because the
// Application layer cannot depend on the Presentation layer.
//
// [Ja] homeDraftPagesLimit はホーム画面に表示する下書きページ数の上限。
// viewmodel.SidebarDraftPagesLimit (= 5) と揃えてあり、ホームとサイドバーで同じ
// 「直近の下書き」を表示できるようにしている。Application 層は Presentation 層に依存できないため、
// import せずに値を二重定義している。
const homeDraftPagesLimit = 5

// GetHomeShowUsecase はホーム画面表示用ユースケース
type GetHomeShowUsecase struct {
	spaceRepo     *repository.SpaceRepository
	topicRepo     *repository.TopicRepository
	draftPageRepo *repository.DraftPageRepository
}

// NewGetHomeShowUsecase は GetHomeShowUsecase を生成する
func NewGetHomeShowUsecase(
	spaceRepo *repository.SpaceRepository,
	topicRepo *repository.TopicRepository,
	draftPageRepo *repository.DraftPageRepository,
) *GetHomeShowUsecase {
	return &GetHomeShowUsecase{
		spaceRepo:     spaceRepo,
		topicRepo:     topicRepo,
		draftPageRepo: draftPageRepo,
	}
}

// GetHomeShowInput はホーム画面表示の入力パラメータ
type GetHomeShowInput struct {
	UserID model.UserID
}

// GetHomeShowOutput contains the data shown on the home page: the user's active spaces,
// the topics the user is joined to, and the draft pages the user is working on.
//
// [Ja] GetHomeShowOutput はホーム画面に表示するデータ。ユーザーが参加中のスペース一覧、
// 参加中のトピック一覧、ユーザーが作業中の下書きページを保持する。
type GetHomeShowOutput struct {
	ActiveSpaces []*model.Space
	JoinedTopics []*model.Topic
	DraftPages   []*model.DraftPage
}

// Execute はホーム画面に表示する参加中スペース・参加中トピック・下書きページを取得する。
func (uc *GetHomeShowUsecase) Execute(ctx context.Context, input GetHomeShowInput) (*GetHomeShowOutput, error) {
	spaces, err := uc.spaceRepo.ListActiveByUser(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("参加中スペース一覧の取得に失敗: %w", err)
	}

	joinedTopics, err := uc.topicRepo.ListJoinedByUser(ctx, input.UserID, homeJoinedTopicsLimit)
	if err != nil {
		return nil, fmt.Errorf("参加中トピック一覧の取得に失敗: %w", err)
	}

	drafts, err := uc.draftPageRepo.ListByUser(ctx, input.UserID, homeDraftPagesLimit)
	if err != nil {
		return nil, fmt.Errorf("下書きページ一覧の取得に失敗: %w", err)
	}

	return &GetHomeShowOutput{
		ActiveSpaces: spaces,
		JoinedTopics: joinedTopics,
		DraftPages:   drafts,
	}, nil
}
