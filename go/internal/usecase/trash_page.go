package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// TrashPageUsecaseはページをゴミ箱へ入れる (POST /s/:space_identifier/pages/:page_number/trash)。
//
// ここでのゴミ箱はpages.trashed_atによるUI上のゴミ箱で、バッチ処理が使うdiscarded_atの
// 論理削除とは別物である。ゴミ箱に入ったページはタイトルと本文を保持し、Rails版のゴミ箱画面が
// 同じカラムを見て復元するため、本UseCaseは時刻を打刻するだけでよい。
type TrashPageUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewTrashPageUsecaseはTrashPageUsecaseを生成する。
func NewTrashPageUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *TrashPageUsecase {
	return &TrashPageUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// TrashPageInputはページをゴミ箱へ入れる操作の入力パラメータ。
// ページ表示画面と違い本操作はログインが必須なので、UserIDはポインタにしない。
type TrashPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
}

// TrashPageOutputはページをゴミ箱へ入れた結果。ページが属していたスペースを返し、
// ハンドラーがリクエストURLを読み直すのではなく保存済みの識別子から遷移先を組み立てられる
// ようにする。
type TrashPageOutput struct {
	Space *model.Space
}

// Executeはページをゴミ箱へ入れる。
//
// トランザクションを開かない書き込みUseCaseである。操作の実体はtrashed_atを打刻する1回の
// UPDATEだけで、併せてロールバックすべき処理が無いためである。
func (uc *TrashPageUsecase) Execute(ctx context.Context, input TrashPageInput) (*TrashPageOutput, error) {
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	authorizer := newAuthorizer(data.spaceMember, data.topicMember)

	// 開けないトピックのページはページ表示画面と同じく「存在しない」扱いにする。page:trashは
	// スペース単位でも持てるスコープのため、このゲートが無いと、参加しておらず読むこともできない
	// 非公開トピックのページをゴミ箱へ入れられてしまう。
	if !authorizer.CanShowTopic(data.topic) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	if !authorizer.CanTrashPage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	if err := uc.pageRepo.TrashByID(ctx, data.page.ID, data.space.ID, time.Now()); err != nil {
		return nil, fmt.Errorf("ページのゴミ箱への移動に失敗: %w", err)
	}

	return &TrashPageOutput{
		Space: data.space,
	}, nil
}

func (uc *TrashPageUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}
