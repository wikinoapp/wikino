package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetDraftPageRevisionDiffUsecaseはページ編集画面のリビジョン差分モーダル用データを取得する。
// 対象リビジョンと、その直前のリビジョン (差分の比較対象) を返す。差分の計算自体は
// プレゼンテーション層 (viewmodel) で行う。
type GetDraftPageRevisionDiffUsecase struct {
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	pageRepo              *repository.PageRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
}

// NewGetDraftPageRevisionDiffUsecaseはGetDraftPageRevisionDiffUsecaseを生成する。
func NewGetDraftPageRevisionDiffUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
) *GetDraftPageRevisionDiffUsecase {
	return &GetDraftPageRevisionDiffUsecase{
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		pageRepo:              pageRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
	}
}

// GetDraftPageRevisionDiffInputはリビジョン差分取得の入力パラメータ。
type GetDraftPageRevisionDiffInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	RevisionID      model.DraftPageRevisionID
	UserID          model.UserID
}

// GetDraftPageRevisionDiffOutputはリビジョン差分取得の出力。
type GetDraftPageRevisionDiffOutput struct {
	// Revisionは編集履歴カラムで選択されたリビジョン。
	Revision *model.DraftPageRevision

	// PreviousRevisionはRevisionの直前のリビジョン (差分の比較対象)。Revisionが最古の
	// 場合はnilとなり、差分は全文追加として表示する。
	PreviousRevision *model.DraftPageRevision

	// IsCurrentはRevisionが下書きの最新リビジョン (編集履歴で「現在」バッジが付くもの)
	// かどうかを表す。現在の状態への復元は重複リビジョンを積むだけのno-opのため、差分モーダルは
	// このとき復元ボタンを隠す。
	IsCurrent bool
}

// Executeは対象リビジョンと直前リビジョンを取得する。
func (uc *GetDraftPageRevisionDiffUsecase) Execute(ctx context.Context, input GetDraftPageRevisionDiffInput) (*GetDraftPageRevisionDiffOutput, error) {
	// 1. データ取得 + 2. 認可チェック (手動保存と同じ共通ヘルパーを使う)。
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}
	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// 下書きはスペースメンバーごとの個人データのため、リクエストしたメンバー自身の
	// このページの下書きを解決し、それに属するリビジョンのみ受け付ける。これによりURLの
	// ページ番号との整合を検証しつつ、他メンバーの下書きリビジョンを隠す (403ではなく404)。
	draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, data.page.ID, data.spaceMember.ID, data.space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
	}
	if draftPage == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	revision, err := uc.draftPageRevisionRepo.FindByID(ctx, input.RevisionID, data.space.ID)
	if err != nil {
		return nil, fmt.Errorf("リビジョンの取得に失敗: %w", err)
	}
	if revision == nil || revision.DraftPageID != draftPage.ID {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	previous, err := uc.draftPageRevisionRepo.FindPrevious(ctx, revision)
	if err != nil {
		return nil, fmt.Errorf("直前リビジョンの取得に失敗: %w", err)
	}

	// 選択されたリビジョンが最新かどうかを判定する。最新リビジョンを1件 (新しい順・limit 1)
	// 取得してIDを比較する。手動保存が最新リビジョンを判定するのと同じ方法で、既存の
	// ListByDraftPageIDを流用するため新しいクエリ定義は不要。
	latest, err := uc.draftPageRevisionRepo.ListByDraftPageID(ctx, draftPage.ID, data.space.ID, 1)
	if err != nil {
		return nil, fmt.Errorf("最新リビジョンの取得に失敗: %w", err)
	}
	isCurrent := len(latest) > 0 && latest[0].ID == revision.ID

	return &GetDraftPageRevisionDiffOutput{
		Revision:         revision,
		PreviousRevision: previous,
		IsCurrent:        isCurrent,
	}, nil
}

func (uc *GetDraftPageRevisionDiffUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}
