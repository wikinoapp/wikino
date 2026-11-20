package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

// CleanupAttachmentObjectsUsecaseは、行を消した添付ファイルの実体を回収する。
// 削除した行に依存せず、ジョブに保存したキーを使って再試行できる。
type CleanupAttachmentObjectsUsecase struct {
	dispatcher    *dispatcher.Dispatcher
	objectStorage storage.ObjectStorage
}

// NewCleanupAttachmentObjectsUsecaseはCleanupAttachmentObjectsUsecaseを生成する。
func NewCleanupAttachmentObjectsUsecase(d *dispatcher.Dispatcher, objectStorage storage.ObjectStorage) *CleanupAttachmentObjectsUsecase {
	return &CleanupAttachmentObjectsUsecase{dispatcher: d, objectStorage: objectStorage}
}

// CleanupAttachmentObjectsInputは、同じトランザクションで削除した添付ファイルとその実体のキー。
// スペースの物理削除のあとも使うため、スペースや添付ファイルの行は引き直さない。
type CleanupAttachmentObjectsInput struct {
	SpaceID       string
	AttachmentIDs []string
	Keys          []string
}

// Executeは、書き戻す可能性がある処理ジョブの終了後に実体を消す。
// 戻り値が0より大きければ、その時間だけ先送りして、再試行の回数を使わずに待つ。
func (uc *CleanupAttachmentObjectsUsecase) Execute(ctx context.Context, input CleanupAttachmentObjectsInput) (time.Duration, error) {
	active, err := uc.dispatcher.HasActiveAttachmentProcessing(ctx, input.SpaceID, input.AttachmentIDs)
	if err != nil {
		return 0, fmt.Errorf("添付ファイルの処理ジョブの確認に失敗: %w", err)
	}
	if active {
		return 10 * time.Second, nil
	}
	// スペースの物理削除が登録するジョブは、スペースの添付ファイルのキーをまとめて持つため、一括で消す
	if err := uc.objectStorage.DeleteMany(ctx, input.Keys); err != nil {
		return 0, fmt.Errorf("添付ファイルの実体の後始末に失敗: %w", err)
	}
	return 0, nil
}
