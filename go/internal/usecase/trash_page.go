package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// TrashPageUsecase moves a page into the trash (POST /s/:space_identifier/pages/:page_number/trash).
//
// The trash is the UI-level one backed by pages.trashed_at, not the discarded_at logical deletion
// the batch jobs use. A trashed page keeps its title and body, and the Rails trash screen restores
// it from the same column, so this UseCase only stamps the timestamp.
//
// [Ja] TrashPageUsecase はページをゴミ箱へ入れる (POST /s/:space_identifier/pages/:page_number/trash)。
//
// ここでのゴミ箱は pages.trashed_at による UI 上のゴミ箱で、バッチ処理が使う discarded_at の
// 論理削除とは別物である。ゴミ箱に入ったページはタイトルと本文を保持し、Rails 版のゴミ箱画面が
// 同じカラムを見て復元するため、本 UseCase は時刻を打刻するだけでよい。
type TrashPageUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewTrashPageUsecase creates a TrashPageUsecase.
//
// [Ja] NewTrashPageUsecase は TrashPageUsecase を生成する。
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

// TrashPageInput holds the input parameters for moving a page into the trash.
// Unlike the page detail screen, this operation requires a signed-in user, so UserID is not a pointer.
//
// [Ja] TrashPageInput はページをゴミ箱へ入れる操作の入力パラメータ。
// ページ表示画面と違い本操作はログインが必須なので、UserID はポインタにしない。
type TrashPageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID
}

// TrashPageOutput is the output of moving a page into the trash. It carries the space and the topic
// the page belonged to, so that the handler can build the redirect target from the stored
// identifier and number instead of re-reading the request URL.
//
// [Ja] TrashPageOutput はページをゴミ箱へ入れた結果。ページが属していたスペースとトピックを返し、
// ハンドラーがリクエスト URL を読み直すのではなく保存済みの識別子と番号から遷移先を組み立てられる
// ようにする。
type TrashPageOutput struct {
	Space *model.Space
	Topic *model.Topic
}

// Execute moves the page into the trash.
//
// It is a write UseCase without a transaction: the whole operation is the single UPDATE that stamps
// trashed_at, so there is nothing to roll back alongside it.
//
// [Ja] Execute はページをゴミ箱へ入れる。
//
// トランザクションを開かない書き込み UseCase である。操作の実体は trashed_at を打刻する 1 回の
// UPDATE だけで、併せてロールバックすべき処理が無いためである。
func (uc *TrashPageUsecase) Execute(ctx context.Context, input TrashPageInput) (*TrashPageOutput, error) {
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	authorizer := newAuthorizer(data.spaceMember, data.topicMember)

	// A page in a topic the member cannot open stays not-found, the same way the page detail screen
	// hides it. page:trash is a space-wide scope, so without this gate a member holding it could
	// trash pages in a private topic they are not part of and cannot even read.
	//
	// [Ja] 開けないトピックのページはページ表示画面と同じく「存在しない」扱いにする。page:trash は
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
		Topic: data.topic,
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
