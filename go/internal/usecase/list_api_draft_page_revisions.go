package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ListAPIDraftPageRevisionsUsecaseは、公開APIでページについてトークンの持ち主の下書きの編集履歴を取得するユースケース
type ListAPIDraftPageRevisionsUsecase struct {
	pageRepo              *repository.PageRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
}

// NewListAPIDraftPageRevisionsUsecaseはListAPIDraftPageRevisionsUsecaseを生成する
func NewListAPIDraftPageRevisionsUsecase(
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *ListAPIDraftPageRevisionsUsecase {
	return &ListAPIDraftPageRevisionsUsecase{
		pageRepo:              pageRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
	}
}

// APIDraftPageRevisionCursorは、編集履歴の一覧で前のページの最後のリビジョンの位置を表す
type APIDraftPageRevisionCursor struct {
	CreatedAt  time.Time
	RevisionID model.DraftPageRevisionID
}

// ListAPIDraftPageRevisionsInputは下書きの編集履歴の取得に必要な入力パラメータ
type ListAPIDraftPageRevisionsInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// Afterは前のページの最後のリビジョンの位置。この位置より後ろのリビジョンを返す。nilなら最初のページを返す
	After *APIDraftPageRevisionCursor
	// Limitは1ページの件数 (1以上)
	Limit int32
}

// ListAPIDraftPageRevisionsOutputは下書きの編集履歴の取得結果
type ListAPIDraftPageRevisionsOutput struct {
	// Revisionsは下書きのリビジョンを作成日時の新しい順に並べたもの
	Revisions []*model.DraftPageRevision
	// HasNextは次のページがあるかどうか
	HasNext bool
}

// Executeはトークンが属するスペースのページについて、トークンの持ち主の下書きのリビジョンを新しい順に返す。
// 下書きの解決と未存在の扱いは下書きの取得 (getDraftPage) と同じで、他のメンバーの下書きのリビジョンは返さない
func (uc *ListAPIDraftPageRevisionsUsecase) Execute(ctx context.Context, input ListAPIDraftPageRevisionsInput) (*ListAPIDraftPageRevisionsOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	access, err := findAPIDraftPage(ctx, apiDraftPageAccessRepos{
		pageRepo:        uc.pageRepo,
		draftPageRepo:   uc.draftPageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}, input.Principal, space, input.PageNumber)
	if err != nil {
		return nil, err
	}
	if !access.authorizer.CanShowDraftPage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	listInput := repository.ListDraftPageRevisionsByCreatedAtInput{
		DraftPageID: access.draft.ID,
		SpaceID:     space.ID,
		// 次のページがあるかを知るため、1件多く取得する
		Limit: input.Limit + 1,
	}
	if input.After != nil {
		listInput.AfterCreatedAt = &input.After.CreatedAt
		listInput.AfterID = &input.After.RevisionID
	}
	revisions, err := uc.draftPageRevisionRepo.ListByCreatedAt(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("下書きのリビジョンの一覧の取得に失敗: %w", err)
	}

	hasNext := len(revisions) > int(input.Limit)
	if hasNext {
		revisions = revisions[:input.Limit]
	}
	return &ListAPIDraftPageRevisionsOutput{Revisions: revisions, HasNext: hasNext}, nil
}
