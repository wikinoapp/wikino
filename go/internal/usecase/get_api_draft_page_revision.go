package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAPIDraftPageRevisionUsecaseは、公開APIでページについてトークンの持ち主の下書きのリビジョンを1件取得するユースケース
type GetAPIDraftPageRevisionUsecase struct {
	pageRepo              *repository.PageRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
}

// NewGetAPIDraftPageRevisionUsecaseはGetAPIDraftPageRevisionUsecaseを生成する
func NewGetAPIDraftPageRevisionUsecase(
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *GetAPIDraftPageRevisionUsecase {
	return &GetAPIDraftPageRevisionUsecase{
		pageRepo:              pageRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
	}
}

// GetAPIDraftPageRevisionInputは下書きのリビジョンの取得に必要な入力パラメータ
type GetAPIDraftPageRevisionInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	RevisionID      model.DraftPageRevisionID
}

// GetAPIDraftPageRevisionOutputは下書きのリビジョンの取得結果
type GetAPIDraftPageRevisionOutput struct {
	Revision *model.DraftPageRevision
}

// Executeはトークンが属するスペースのページについて、トークンの持ち主の下書きのリビジョンを返す。
// 下書きの解決と未存在の扱いは下書きの取得 (getDraftPage) と同じ。
// リビジョンはパスのページの、トークンの持ち主の下書きに属するものだけを返し、
// 別のページや他のメンバーの下書きのリビジョンは未存在として扱う
func (uc *GetAPIDraftPageRevisionUsecase) Execute(ctx context.Context, input GetAPIDraftPageRevisionInput) (*GetAPIDraftPageRevisionOutput, error) {
	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

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
		return nil, notFound
	}

	revision, err := uc.draftPageRevisionRepo.FindByID(ctx, input.RevisionID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きのリビジョンの取得に失敗: %w", err)
	}
	if revision == nil || revision.DraftPageID != access.draft.ID {
		return nil, notFound
	}
	return &GetAPIDraftPageRevisionOutput{Revision: revision}, nil
}
