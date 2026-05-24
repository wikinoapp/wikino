package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSpaceShowUsecase aggregates the data shown on the space detail page (GET /s/:identifier):
// the space itself, the pinned and regular pages within it, and the data the empty state needs.
//
// [Ja] GetSpaceShowUsecase はスペース詳細画面 (GET /s/:identifier) に表示するデータを集約する
// 読み取り UseCase。スペース本体、スペース内のピン留めページと通常ページ、空状態に必要なデータを取得する。
type GetSpaceShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
}

// NewGetSpaceShowUsecase creates a GetSpaceShowUsecase.
// [Ja] NewGetSpaceShowUsecase は GetSpaceShowUsecase を生成する。
func NewGetSpaceShowUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
) *GetSpaceShowUsecase {
	return &GetSpaceShowUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
	}
}

// GetSpaceShowInput holds the input parameters for fetching the space detail.
// UserID is nil when the user is not logged in (the space detail is viewable even
// without logging in as long as the topic is public).
//
// [Ja] GetSpaceShowInput はスペース詳細取得の入力パラメータ。
// UserID は未ログイン時に nil になる (スペース詳細は非ログインでも公開トピックなら閲覧できる)。
type GetSpaceShowInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          *model.UserID
	Page            int32
	PageLimit       int32
}

// GetSpaceShowOutput is the output of fetching the space detail.
// [Ja] GetSpaceShowOutput はスペース詳細取得の出力。
type GetSpaceShowOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	PinnedPages []*model.Page
	Pages       []*model.Page
	TotalCount  int64

	// FirstJoinedTopic is the member's joined topic with the smallest id (nil for guests or for
	// members who have not joined any topic). Used by the empty-state "create a new page" link.
	//
	// [Ja] FirstJoinedTopic はメンバーが参加しているトピックのうち id が最小のもの (ゲスト、
	// またはどのトピックにも参加していないメンバーでは nil)。空状態の「新しいページを作る」導線で使う。
	FirstJoinedTopic *model.Topic

	JoinedSpace    bool
	CanCreateTopic bool
}

// Execute fetches the data shown on the space detail page. It returns (nil, nil)
// when the space is not found.
//
// [Ja] Execute はスペース詳細画面に表示するデータを取得する。スペースが見つからない場合は (nil, nil) を返す。
func (uc *GetSpaceShowUsecase) Execute(ctx context.Context, input GetSpaceShowInput) (*GetSpaceShowOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	// Fetch the logged-in user's space membership (nil when not logged in).
	// [Ja] ログインユーザーのスペースメンバーを取得 (未ログインなら nil)。
	var spaceMember *model.SpaceMember
	if input.UserID != nil {
		spaceMember, err = uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, *input.UserID)
		if err != nil {
			return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
		}
	}

	// Members see every active page; non-members (guests included) see only public-topic pages.
	// [Ja] メンバーは全アクティブページを閲覧でき、非メンバー (ゲスト含む) は公開トピックのページのみ閲覧できる。
	joinedSpace := spaceMember != nil
	publicOnly := !joinedSpace

	authorizer := newAuthorizer(spaceMember, nil)

	pinnedPages, err := uc.pageRepo.FindPinnedBySpace(ctx, space.ID, publicOnly)
	if err != nil {
		return nil, fmt.Errorf("ピン留めページの取得に失敗: %w", err)
	}

	paginatedResult, err := uc.pageRepo.FindRegularBySpacePaginated(ctx, space.ID, publicOnly, input.Page, input.PageLimit)
	if err != nil {
		return nil, fmt.Errorf("通常ページの取得に失敗: %w", err)
	}

	// Fetch the first joined topic for every member so the empty state can offer a "new page"
	// link. It is consumed only when no pages are shown, but the fetch is not gated on emptiness:
	// that keeps this in step with the Rails version and avoids tying the fetch condition to the
	// template's empty-state check (the cost is one indexed LIMIT 1 lookup).
	//
	// [Ja] 空状態で「新しいページを作る」導線を出せるよう、メンバーには常に最初の参加トピックを取得する。
	// 使うのはページが 0 件のときだけだが、取得を空状態判定でゲートしていない。Rails 版と挙動を揃え、
	// 取得条件をテンプレートの空状態判定と結合させないため (コストはインデックス済みの LIMIT 1 クエリ 1 回)。
	var firstJoinedTopic *model.Topic
	if spaceMember != nil {
		firstJoinedTopic, err = uc.topicRepo.FindFirstJoinedBySpaceMember(ctx, spaceMember.ID, space.ID)
		if err != nil {
			return nil, fmt.Errorf("最初の参加トピックの取得に失敗: %w", err)
		}
	}

	return &GetSpaceShowOutput{
		Space:            space,
		SpaceMember:      spaceMember,
		PinnedPages:      pinnedPages,
		Pages:            paginatedResult.Pages,
		TotalCount:       paginatedResult.TotalCount,
		FirstJoinedTopic: firstJoinedTopic,
		JoinedSpace:      joinedSpace,
		CanCreateTopic:   authorizer.CanCreateTopic(),
	}, nil
}
