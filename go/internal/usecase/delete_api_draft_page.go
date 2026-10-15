package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// DeleteAPIDraftPageUsecaseは、公開APIでページについてトークンの持ち主の下書きを編集履歴ごと削除するユースケース。
// Web画面の削除 (DeleteDraftPageUsecase) と同じく下書きと編集履歴を削除し、加えて削除の前提にした版
// (`If-Match`) を今の版と照合する
type DeleteAPIDraftPageUsecase struct {
	db                    *sql.DB
	pageRepo              *repository.PageRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
}

// NewDeleteAPIDraftPageUsecaseはDeleteAPIDraftPageUsecaseを生成する
func NewDeleteAPIDraftPageUsecase(
	db *sql.DB,
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *DeleteAPIDraftPageUsecase {
	return &DeleteAPIDraftPageUsecase{
		db:                    db,
		pageRepo:              pageRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
	}
}

// DeleteAPIDraftPageInputは下書きの削除に必要な入力パラメータ
type DeleteAPIDraftPageInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// IfMatchDigestsは削除の前提にする下書きの版 (`DraftPage.ContentDigest()` の値) の候補。
	// 今の版がいずれかと一致する場合だけ削除する
	IfMatchDigests []string
	// IfMatchAnyは、下書きが存在すれば版を問わず削除することを表す (`If-Match: *`)
	IfMatchAny bool
}

// Executeはトークンが属するスペースのページについて、トークンの持ち主の下書きを編集履歴ごと削除する。
// 取得 (GetAPIDraftPageUsecase) と同じく、他のメンバーの下書き・ゴミ箱のページの下書き・閲覧者が開けない
// 非公開トピックの下書きは未存在として扱い、下書きを削除できない場合も存在を秘匿して未存在を返す。
// 今の版が前提の版と一致しなければ、AppErrCodePreconditionFailedのエラーを返す
func (uc *DeleteAPIDraftPageUsecase) Execute(ctx context.Context, input DeleteAPIDraftPageInput) error {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return err
	}

	// 1. データ取得
	access, err := findAPIDraftPage(ctx, apiDraftPageAccessRepos{
		pageRepo:        uc.pageRepo,
		draftPageRepo:   uc.draftPageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}, input.Principal, space, input.PageNumber)
	if err != nil {
		return err
	}

	// 2. 認可チェック
	if !access.authorizer.CanDeleteDraftPage() {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 3. 前提の照合: 多くの場合の412をトランザクションの前に返す。
	// 照合の後に割り込んだ更新は、永続化の中で行をロックしてから照合し直して検出する
	if !apiDraftPageMatches(access.draft, input.IfMatchDigests, input.IfMatchAny) {
		return apiDraftPageModifiedError(ctx)
	}

	// 4. 永続化
	return uc.deleteDraftPage(ctx, access.draft, input)
}

// deleteDraftPageは下書きの行をロックして前提の版と照合し直してから、編集履歴ごと削除する
func (uc *DeleteAPIDraftPageUsecase) deleteDraftPage(ctx context.Context, draft *model.DraftPage, input DeleteAPIDraftPageInput) error {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 照合から削除までの間にWeb画面の自動保存などが割り込まないよう、行をロックしてから照合し直す
	deleted, err := deleteDraftPageWithRevisions(ctx, uc.draftPageRepo.WithTx(tx), uc.draftPageRevisionRepo.WithTx(tx), draft.ID, draft.SpaceID,
		func(locked *model.DraftPage) error {
			if !apiDraftPageMatches(locked, input.IfMatchDigests, input.IfMatchAny) {
				return apiDraftPageModifiedError(ctx)
			}
			return nil
		})
	if err != nil {
		return err
	}
	// 同時に削除・公開されていれば、取得と同じく未存在として扱う
	if !deleted {
		return &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}
	return nil
}
