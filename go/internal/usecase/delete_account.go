package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ErrAccountHasSpaceMembersは、物理削除するユーザーのスペースメンバーの行が残っていて、
// まだユーザーを消せないことを表す。行は削除済みのスペースの物理削除が消すため、時間をおいて
// やり直せば消せる。
var ErrAccountHasSpaceMembers = errors.New("スペースメンバーの行が残っているため、ユーザーを物理削除できない")

// DeleteAccountUsecaseは、論理削除したユーザーを関連レコードとともに物理削除する。
// アカウントの削除の画面から投入したジョブがWorkerから呼ぶ。
type DeleteAccountUsecase struct {
	db                    *sql.DB
	userRepo              *repository.UserRepository
	userSessionRepo       *repository.UserSessionRepository
	userPasswordRepo      *repository.UserPasswordRepository
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
}

// NewDeleteAccountUsecaseはDeleteAccountUsecaseを生成する。
func NewDeleteAccountUsecase(
	db *sql.DB,
	userRepo *repository.UserRepository,
	userSessionRepo *repository.UserSessionRepository,
	userPasswordRepo *repository.UserPasswordRepository,
	userTwoFactorAuthRepo *repository.UserTwoFactorAuthRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
) *DeleteAccountUsecase {
	return &DeleteAccountUsecase{
		db:                    db,
		userRepo:              userRepo,
		userSessionRepo:       userSessionRepo,
		userPasswordRepo:      userPasswordRepo,
		userTwoFactorAuthRepo: userTwoFactorAuthRepo,
		spaceMemberRepo:       spaceMemberRepo,
	}
}

// DeleteAccountInputは物理削除するユーザーを指す。
type DeleteAccountInput struct {
	UserID model.UserID
}

// Executeは、論理削除したユーザーのセッション・パスワード・二要素認証の設定とユーザーを、
// 1つのトランザクションで物理削除する。フィーチャーフラグ・パスワードリセットのトークン・
// ユーザーが開始したメールアドレス変更の確認は、外部キーのON DELETE CASCADEでユーザーとともに消える。
//
// スペースメンバーの行が残っている間は何も消さず、ErrAccountHasSpaceMembersを返す。
// 論理削除していないユーザーと、既に物理削除したユーザーには何もしない。再試行で同じジョブが
// 2度動いても結果が変わらないようにする。
func (uc *DeleteAccountUsecase) Execute(ctx context.Context, input DeleteAccountInput) error {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userRepo := uc.userRepo.WithTx(tx)

	// 論理削除したユーザーをロックしてから、残っている行を確かめる。ロックの前に確かめると、
	// その間に論理削除を取り消されたユーザーまで消してしまう。
	locked, err := userRepo.LockDiscarded(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("論理削除したユーザーのロックに失敗: %w", err)
	}
	if !locked {
		return nil
	}

	hasSpaceMembers, err := uc.spaceMemberRepo.WithTx(tx).ExistsByUser(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("スペースメンバーの行の確認に失敗: %w", err)
	}
	if hasSpaceMembers {
		return ErrAccountHasSpaceMembers
	}

	// 論理削除のときに消したあとでも、そのあとに作られたセッションが残りうるため消し直す。
	if err := uc.userSessionRepo.WithTx(tx).DeleteByUserID(ctx, input.UserID); err != nil {
		return fmt.Errorf("ユーザーのセッションの削除に失敗: %w", err)
	}
	if err := uc.userPasswordRepo.WithTx(tx).DeleteByUserID(ctx, input.UserID); err != nil {
		return fmt.Errorf("ユーザーのパスワードの削除に失敗: %w", err)
	}
	if err := uc.userTwoFactorAuthRepo.WithTx(tx).DeleteByUserID(ctx, input.UserID); err != nil {
		return fmt.Errorf("ユーザーの二要素認証の設定の削除に失敗: %w", err)
	}
	if _, err := userRepo.DeleteDiscarded(ctx, input.UserID); err != nil {
		return fmt.Errorf("ユーザーの削除に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return nil
}
