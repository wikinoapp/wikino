package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// discardSpaceCleanupTimeoutは、ジョブの投入に失敗したときに論理削除を取り消す処理に与える時間。
// リクエストが打ち切られたあとも取り消しを終えられるよう、リクエストのものとは別に持つ。
const discardSpaceCleanupTimeout = 10 * time.Second

// DiscardSpaceUsecaseはスペースを削除する。スペースとそのトピックを論理削除して直ちに隠し、
// 関連レコードとともに物理削除するジョブを投入する。
type DiscardSpaceUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	dispatcher      *dispatcher.Dispatcher
	validator       *validator.SpaceDeletionCreateValidator
}

// NewDiscardSpaceUsecaseはDiscardSpaceUsecaseを生成する。
func NewDiscardSpaceUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	d *dispatcher.Dispatcher,
	v *validator.SpaceDeletionCreateValidator,
) *DiscardSpaceUsecase {
	return &DiscardSpaceUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		dispatcher:      d,
		validator:       v,
	}
}

// DiscardSpaceInputはスペースを削除するための入力パラメータ。
type DiscardSpaceInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	// SpaceNameは確認のために閲覧者が入力したスペースの名前
	SpaceName string
}

// DiscardSpaceOutputは削除したスペースを返す。入力の検証に失敗したときも、確認画面を
// 描画し直せるよう、エラーと一緒に返す。
type DiscardSpaceOutput struct {
	Space *model.Space
}

// Executeは、閲覧者がスペースを削除でき、入力した名前がスペースの名前と一致するときに、
// スペースとそのトピックを論理削除してから物理削除のジョブを投入する。
func (uc *DiscardSpaceUsecase) Execute(ctx context.Context, input DiscardSpaceInput) (*DiscardSpaceOutput, error) {
	space, err := fetchSpaceDeleteAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	output := &DiscardSpaceOutput{Space: space}

	if err := uc.validator.Validate(ctx, validator.SpaceDeletionCreateValidatorInput{
		SpaceName: input.SpaceName,
		Space:     space,
	}); err != nil {
		return output, err
	}

	// 取り消しで自分の論理削除だけを戻せるよう、DBに保存できる精度にそろえた時刻を使う。
	discardedAt := time.Now().Truncate(time.Microsecond)
	if err := uc.discard(ctx, space.ID, discardedAt); err != nil {
		return nil, err
	}

	if err := uc.dispatcher.EnqueueDeleteSpace(ctx, string(space.ID)); err != nil {
		uc.undiscard(ctx, space.ID, discardedAt)
		return nil, fmt.Errorf("スペースの物理削除のジョブの投入に失敗: %w", err)
	}

	return output, nil
}

// discardは、スペースとその論理削除していないトピックを、同じ時刻で1つのトランザクションで論理削除する。
func (uc *DiscardSpaceUsecase) discard(ctx context.Context, spaceID model.SpaceID, discardedAt time.Time) error {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// スペースを先に更新し、その行のロックで同じスペースを削除する別のリクエストを待たせる。
	discarded, err := uc.spaceRepo.WithTx(tx).Discard(ctx, spaceID, discardedAt)
	if err != nil {
		return fmt.Errorf("スペースの論理削除に失敗: %w", err)
	}
	if !discarded {
		// 確認してから論理削除するまでの間に、別のリクエストが同じスペースを削除した。
		return notFoundError(ctx)
	}

	if err := uc.topicRepo.WithTx(tx).DiscardBySpace(ctx, spaceID, discardedAt); err != nil {
		return fmt.Errorf("スペースのトピックの論理削除に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return nil
}

// undiscardは、物理削除のジョブを投入できなかったスペースと、同じ時刻で論理削除したトピックの
// 論理削除を取り消す。論理削除したままにすると、誰からも見えないのに消えないスペースが残り、
// 閲覧者は削除をやり直せない。
func (uc *DiscardSpaceUsecase) undiscard(ctx context.Context, spaceID model.SpaceID, discardedAt time.Time) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), discardSpaceCleanupTimeout)
	defer cancel()

	if err := uc.undiscardTx(cleanupCtx, spaceID, discardedAt); err != nil {
		slog.ErrorContext(ctx, "ジョブの投入に失敗したスペースの論理削除の取り消しに失敗しました",
			"space_id", string(spaceID),
			"error", err,
		)
	}
}

// undiscardTxは、スペースとトピックの論理削除の取り消しを1つのトランザクションで行う。
func (uc *DiscardSpaceUsecase) undiscardTx(ctx context.Context, spaceID model.SpaceID, discardedAt time.Time) error {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	if err := uc.topicRepo.WithTx(tx).UndiscardBySpace(ctx, spaceID, discardedAt, now); err != nil {
		return fmt.Errorf("スペースのトピックの論理削除の取り消しに失敗: %w", err)
	}
	if err := uc.spaceRepo.WithTx(tx).Undiscard(ctx, spaceID, discardedAt, now); err != nil {
		return fmt.Errorf("スペースの論理削除の取り消しに失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return nil
}
