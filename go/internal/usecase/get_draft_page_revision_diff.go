package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetDraftPageRevisionDiffUsecase fetches the data for the revision diff modal on the page
// editor: the target revision and the revision immediately preceding it (the diff comparison
// target). The diff itself is computed in the presentation layer (viewmodel).
//
// [Ja] GetDraftPageRevisionDiffUsecase はページ編集画面のリビジョン差分モーダル用データを取得する。
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

// NewGetDraftPageRevisionDiffUsecase creates a new GetDraftPageRevisionDiffUsecase.
// [Ja] NewGetDraftPageRevisionDiffUsecase は GetDraftPageRevisionDiffUsecase を生成する。
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

// GetDraftPageRevisionDiffInput is the input parameters for fetching the revision diff.
// [Ja] GetDraftPageRevisionDiffInput はリビジョン差分取得の入力パラメータ。
type GetDraftPageRevisionDiffInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	RevisionID      model.DraftPageRevisionID
	UserID          model.UserID
}

// GetDraftPageRevisionDiffOutput is the output of fetching the revision diff.
// [Ja] GetDraftPageRevisionDiffOutput はリビジョン差分取得の出力。
type GetDraftPageRevisionDiffOutput struct {
	// Revision is the revision selected in the edit history column.
	// [Ja] Revision は編集履歴カラムで選択されたリビジョン。
	Revision *model.DraftPageRevision

	// PreviousRevision is the revision immediately preceding Revision (the diff comparison
	// target). It is nil when Revision is the oldest one, in which case the diff is rendered
	// as a full addition.
	//
	// [Ja] PreviousRevision は Revision の直前のリビジョン (差分の比較対象)。Revision が最古の
	// 場合は nil となり、差分は全文追加として表示する。
	PreviousRevision *model.DraftPageRevision
}

// Execute fetches the target revision and the one immediately preceding it.
// [Ja] Execute は対象リビジョンと直前リビジョンを取得する。
func (uc *GetDraftPageRevisionDiffUsecase) Execute(ctx context.Context, input GetDraftPageRevisionDiffInput) (*GetDraftPageRevisionDiffOutput, error) {
	// 1. Fetch data + 2. authorization check (using the same shared helper as manual save).
	// [Ja] 1. データ取得 + 2. 認可チェック (手動保存と同じ共通ヘルパーを使う)。
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}
	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// Drafts are personal to a space member, so resolve the requesting member's own draft for
	// this page and accept only revisions belonging to it. This both validates the page number
	// in the URL and keeps other members' draft revisions hidden (404, not 403).
	//
	// [Ja] 下書きはスペースメンバーごとの個人データのため、リクエストしたメンバー自身の
	// このページの下書きを解決し、それに属するリビジョンのみ受け付ける。これにより URL の
	// ページ番号との整合を検証しつつ、他メンバーの下書きリビジョンを隠す (403 ではなく 404)。
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

	return &GetDraftPageRevisionDiffOutput{
		Revision:         revision,
		PreviousRevision: previous,
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
