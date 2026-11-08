package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// PinPageUsecaseはページにピンを打つ (POST /s/:space_identifier/pages/:page_number/pin)。
//
// ピンを打ったページはトピック画面とスペース画面の一覧の上部に並ぶ。公開状態とゴミ箱の状態では
// 拒否しない。公開前やゴミ箱のページにピンの状態があっても一覧には出ないだけで、ゴミ箱から戻した
// ときにピンを保てるためである。
type PinPageUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewPinPageUsecaseはPinPageUsecaseを生成する。
func NewPinPageUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *PinPageUsecase {
	return &PinPageUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// PinPageInputはページにピンを打つ操作の入力パラメータ。
type PinPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
}

// PinPageOutputはページにピンを打った結果。ハンドラーが保存済みの識別子から遷移先を
// 組み立てられるよう、ページが属するスペースを返す。
type PinPageOutput struct {
	Space *model.Space
}

// Executeはページにピンを打つ。
//
// pinned_atを書く1回のUPDATEで完結し、併せてロールバックすべき処理が無いため、
// トランザクションを開かない。
func (uc *PinPageUsecase) Execute(ctx context.Context, input PinPageInput) (*PinPageOutput, error) {
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	authorizer := newAuthorizer(data.spaceMember, data.topicMember)

	// page_pin:writeはスペース単位でも持てるため、開けないトピックのページは「存在しない」扱いにする。
	if !authorizer.CanShowTopic(data.topic) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	if !authorizer.CanPinPage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	if err := uc.pageRepo.PinByID(ctx, data.page.ID, data.space.ID, time.Now()); err != nil {
		return nil, fmt.Errorf("ページのピン留めに失敗: %w", err)
	}

	return &PinPageOutput{
		Space: data.space,
	}, nil
}

func (uc *PinPageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}
