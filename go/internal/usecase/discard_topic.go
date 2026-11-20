package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// discardTopicCleanupTimeoutは、ジョブの投入に失敗したときに論理削除を取り消す処理に与える時間。
// リクエストが打ち切られたあとも取り消しを終えられるよう、リクエストのものとは別に持つ。
const discardTopicCleanupTimeout = 10 * time.Second

// DiscardTopicUsecaseはトピックを削除する。トピックを論理削除して直ちに隠し、関連レコードと
// ともに物理削除するジョブを投入する (Rails版の `Topics::SoftDestroyService` と同じ)。
type DiscardTopicUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	dispatcher      *dispatcher.Dispatcher
	validator       *validator.TopicDeletionCreateValidator
}

// NewDiscardTopicUsecaseはDiscardTopicUsecaseを生成する。
func NewDiscardTopicUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	d *dispatcher.Dispatcher,
	v *validator.TopicDeletionCreateValidator,
) *DiscardTopicUsecase {
	return &DiscardTopicUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		dispatcher:      d,
		validator:       v,
	}
}

// DiscardTopicInputはトピックを削除するための入力パラメータ。
type DiscardTopicInput struct {
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	UserID          model.UserID
	// TopicNameは確認のために閲覧者が入力したトピックの名前
	TopicName string
}

// DiscardTopicOutputは削除したトピックとそのスペースを返す。入力の検証に失敗したときも、
// 確認画面を描画し直せるよう、エラーと一緒に返す。
type DiscardTopicOutput struct {
	Space *model.Space
	Topic *model.Topic
}

// Executeは、閲覧者がトピックを削除でき、入力した名前がトピックの名前と一致するときに、
// トピックを論理削除してから物理削除のジョブを投入する。
func (uc *DiscardTopicUsecase) Execute(ctx context.Context, input DiscardTopicInput) (*DiscardTopicOutput, error) {
	target, err := fetchTopicDeleteAccess(
		ctx,
		uc.spaceRepo,
		uc.spaceMemberRepo,
		uc.topicRepo,
		uc.topicMemberRepo,
		input.SpaceIdentifier,
		input.TopicNumber,
		input.UserID,
	)
	if err != nil {
		return nil, err
	}

	output := &DiscardTopicOutput{Space: target.Space, Topic: target.Topic}

	if err := uc.validator.Validate(ctx, validator.TopicDeletionCreateValidatorInput{
		TopicName: input.TopicName,
		Topic:     target.Topic,
	}); err != nil {
		return output, err
	}

	// 取り消しで自分の論理削除だけを戻せるよう、DBに保存できる精度にそろえた時刻を使う。
	discardedAt := time.Now().Truncate(time.Microsecond)
	discarded, err := uc.topicRepo.Discard(ctx, target.Topic.ID, target.Space.ID, discardedAt)
	if err != nil {
		return nil, fmt.Errorf("トピックの論理削除に失敗: %w", err)
	}
	if !discarded {
		// 確認してから論理削除するまでの間に、別のリクエストが同じトピックを削除した。
		return nil, notFoundError(ctx)
	}

	if err := uc.dispatcher.EnqueueDeleteTopic(ctx, string(target.Space.ID), string(target.Topic.ID)); err != nil {
		uc.undiscard(ctx, target, discardedAt)
		return nil, fmt.Errorf("トピックの物理削除のジョブの投入に失敗: %w", err)
	}

	return output, nil
}

// undiscardは、物理削除のジョブを投入できなかったトピックの論理削除を取り消す。論理削除した
// ままにすると、誰からも見えないのに消えないトピックが残り、閲覧者は削除をやり直せない。
func (uc *DiscardTopicUsecase) undiscard(ctx context.Context, target *topicSettingsTarget, discardedAt time.Time) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), discardTopicCleanupTimeout)
	defer cancel()

	if err := uc.topicRepo.Undiscard(cleanupCtx, target.Topic.ID, target.Space.ID, discardedAt, time.Now()); err != nil {
		slog.ErrorContext(ctx, "ジョブの投入に失敗したトピックの論理削除の取り消しに失敗しました",
			"topic_id", string(target.Topic.ID),
			"error", err,
		)
	}
}
