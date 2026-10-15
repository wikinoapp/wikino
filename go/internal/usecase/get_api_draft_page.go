package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAPIDraftPageUsecaseは、公開APIでページについてトークンの持ち主の下書きを取得するユースケース
type GetAPIDraftPageUsecase struct {
	pageRepo        *repository.PageRepository
	draftPageRepo   *repository.DraftPageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetAPIDraftPageUsecaseはGetAPIDraftPageUsecaseを生成する
func NewGetAPIDraftPageUsecase(pageRepo *repository.PageRepository, draftPageRepo *repository.DraftPageRepository, topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *GetAPIDraftPageUsecase {
	return &GetAPIDraftPageUsecase{
		pageRepo:        pageRepo,
		draftPageRepo:   draftPageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetAPIDraftPageInputは下書きの取得に必要な入力パラメータ
type GetAPIDraftPageInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
}

// GetAPIDraftPageOutputは下書きの取得結果
type GetAPIDraftPageOutput struct {
	// DraftPageはトークンの持ち主の下書き。Pageに下書きのページを持つ
	DraftPage *model.DraftPage
	// Topicは下書きが属するトピック
	Topic *model.Topic
}

// Executeはトークンが属するスペースのページについて、トークンの持ち主の下書きを返す。
// 他のメンバーの下書きは、所有者をクエリで限定して返さない (存在を秘匿するため未存在として扱う)。
// ページや下書きが存在しない場合に加え、ゴミ箱のページの下書き・廃棄済みのトピックの下書き・
// 閲覧者が開けない非公開トピックの下書き・draft_page:readを持たないトピックの下書きも未存在として扱う
func (uc *GetAPIDraftPageUsecase) Execute(ctx context.Context, input GetAPIDraftPageInput) (*GetAPIDraftPageOutput, error) {
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

	return &GetAPIDraftPageOutput{DraftPage: access.draft, Topic: access.topic}, nil
}
