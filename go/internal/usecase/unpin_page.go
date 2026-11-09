package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// UnpinPageUsecaseはページのピンを外す (DELETE /s/:space_identifier/pages/:page_number/pin)。
//
// ゴミ箱のページでもピンの状態を取り除けるよう、公開状態とゴミ箱の状態では拒否しない。
type UnpinPageUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewUnpinPageUsecaseはUnpinPageUsecaseを生成する。
func NewUnpinPageUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *UnpinPageUsecase {
	return &UnpinPageUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// UnpinPageInputはページのピンを外す操作の入力パラメータ。
type UnpinPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
}

// UnpinPageOutputはページのピンを外した結果。ハンドラーが保存済みの識別子から遷移先を
// 組み立てられるよう、ページが属するスペースを返す。
type UnpinPageOutput struct {
	Space *model.Space
}

// Executeはページのピンを外す。
//
// pinned_atを消す1回のUPDATEで完結するため、トランザクションを開かない。
func (uc *UnpinPageUsecase) Execute(ctx context.Context, input UnpinPageInput) (*UnpinPageOutput, error) {
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	authorizer := newAuthorizer(data.spaceMember, data.topicMember)

	// page_pin:deleteはスペース単位でも持てるため、開けないトピックのページは「存在しない」扱いにする。
	if !authorizer.CanShowTopic(data.topic) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	if !authorizer.CanUnpinPage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	if err := uc.pageRepo.UnpinByID(ctx, data.page.ID, data.space.ID, time.Now()); err != nil {
		return nil, fmt.Errorf("ページのピン留めの解除に失敗: %w", err)
	}

	return &UnpinPageOutput{
		Space: data.space,
	}, nil
}

func (uc *UnpinPageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}
