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

// discardAccountCleanupTimeoutは、ジョブの投入に失敗したときに論理削除を取り消す処理に与える時間。
// リクエストが打ち切られたあとも取り消しを終えられるよう、リクエストのものとは別に持つ。
const discardAccountCleanupTimeout = 10 * time.Second

// DiscardAccountUsecaseはアカウントを削除する。ユーザーを論理削除して全セッションを消し、
// 関連レコードとともに物理削除するジョブを投入する (Rails版の `Accounts::SoftDestroyService` に、
// 全セッションの削除を足したもの)。
type DiscardAccountUsecase struct {
	db              *sql.DB
	userRepo        *repository.UserRepository
	userSessionRepo *repository.UserSessionRepository
	dispatcher      *dispatcher.Dispatcher
	validator       *validator.SettingsAccountDeletionCreateValidator
}

// NewDiscardAccountUsecaseはDiscardAccountUsecaseを生成する。
func NewDiscardAccountUsecase(
	db *sql.DB,
	userRepo *repository.UserRepository,
	userSessionRepo *repository.UserSessionRepository,
	d *dispatcher.Dispatcher,
	v *validator.SettingsAccountDeletionCreateValidator,
) *DiscardAccountUsecase {
	return &DiscardAccountUsecase{
		db:              db,
		userRepo:        userRepo,
		userSessionRepo: userSessionRepo,
		dispatcher:      d,
		validator:       v,
	}
}

// DiscardAccountInputはアカウントを削除するための入力パラメータ。
type DiscardAccountInput struct {
	User *model.User
	// Atnameは確認のために閲覧者が入力したアットネーム
	Atname string
}

// Executeは、入力したアットネームが本人のものと一致し、参加しているスペースが無いときに、
// ユーザーを論理削除して全セッションを消してから、物理削除のジョブを投入する。
func (uc *DiscardAccountUsecase) Execute(ctx context.Context, input DiscardAccountInput) error {
	if err := uc.validator.Validate(ctx, validator.SettingsAccountDeletionCreateValidatorInput{
		User:   input.User,
		Atname: input.Atname,
	}); err != nil {
		return err
	}

	// 取り消しで自分の論理削除だけを戻せるよう、DBに保存できる精度にそろえた時刻を使う。
	discardedAt := time.Now().Truncate(time.Microsecond)
	discarded, err := uc.discard(ctx, input.User.ID, discardedAt)
	if err != nil {
		return err
	}
	if !discarded {
		// 確認してから論理削除するまでの間に、別のリクエストが同じアカウントを削除した。
		// 物理削除のジョブはそのリクエストが投入しているため、ここでは投入しない。
		return nil
	}

	if err := uc.dispatcher.EnqueueDeleteAccount(ctx, string(input.User.ID)); err != nil {
		uc.undiscard(ctx, input.User.ID, discardedAt)
		return fmt.Errorf("アカウントの物理削除のジョブの投入に失敗: %w", err)
	}

	return nil
}

// discardは、ユーザーの論理削除と全セッションの削除を1つのトランザクションで行い、論理削除したかを返す。
// 既に論理削除したユーザーでも、残っているセッションは消す。
func (uc *DiscardAccountUsecase) discard(ctx context.Context, userID model.UserID, discardedAt time.Time) (bool, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	discarded, err := uc.userRepo.WithTx(tx).Discard(ctx, userID, discardedAt)
	if err != nil {
		return false, fmt.Errorf("ユーザーの論理削除に失敗: %w", err)
	}
	if err := uc.userSessionRepo.WithTx(tx).DeleteByUserID(ctx, userID); err != nil {
		return false, fmt.Errorf("ユーザーのセッションの削除に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return discarded, nil
}

// undiscardは、物理削除のジョブを投入できなかったユーザーの論理削除を取り消す。論理削除した
// ままにすると、ログインできないのに消えないアカウントが残り、ユーザーは削除をやり直せない。
// 消したセッションは戻らないため、ユーザーはログインし直してから削除をやり直す。
func (uc *DiscardAccountUsecase) undiscard(ctx context.Context, userID model.UserID, discardedAt time.Time) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), discardAccountCleanupTimeout)
	defer cancel()

	if err := uc.userRepo.Undiscard(cleanupCtx, userID, discardedAt, time.Now()); err != nil {
		slog.ErrorContext(ctx, "ジョブの投入に失敗したユーザーの論理削除の取り消しに失敗しました",
			"user_id", string(userID),
			"error", err,
		)
	}
}
