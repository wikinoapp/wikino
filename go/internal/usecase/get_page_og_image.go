package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageOgImageUsecaseはページのog:image用カード画像に描く内容を取得する読み取りUseCase
//
// カード画像はSNSのクローラーに配信するため、閲覧者ではなくゲストとして見せてよいページ
// (公開トピック・ゴミ箱に入っていない・タイトルあり) だけを返す。公開トピックの判定には
// ページ表示画面 (`GetPageShowUsecase` の `IsPublic`) と同じゲスト用Policyを使い、
// 両者の判定が食い違わないようにする。
type GetPageOgImageUsecase struct {
	spaceRepo *repository.SpaceRepository
	pageRepo  *repository.PageRepository
	topicRepo *repository.TopicRepository
}

// NewGetPageOgImageUsecaseはGetPageOgImageUsecaseを生成する
func NewGetPageOgImageUsecase(
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
) *GetPageOgImageUsecase {
	return &GetPageOgImageUsecase{
		spaceRepo: spaceRepo,
		pageRepo:  pageRepo,
		topicRepo: topicRepo,
	}
}

// GetPageOgImageInputはUseCaseの入力
type GetPageOgImageInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
}

// GetPageOgImageOutputはUseCaseの出力
//
// Spaceは正規URLの組み立て (保存済みのスペース識別子) にも使う。
// Page.Titleは非nilかつ空でないことが保証される。
type GetPageOgImageOutput struct {
	Space *model.Space
	Topic *model.Topic
	Page  *model.Page
}

// Executeはカード画像に描く内容を取得する。ゲストに見せられないページは、存在しない
// ページと同じAppErrCodeResourceNotFoundの *model.AppErrorを返し、レスポンス上で
// 「隠している」と「存在しない」を区別しないようにする。
func (uc *GetPageOgImageUsecase) Execute(ctx context.Context, input GetPageOgImageInput) (*GetPageOgImageOutput, error) {
	// 閲覧者を渡さないため、メンバー情報の取得 (spaceMemberRepo / topicMemberRepo) は行われない
	repos := pageAccessRepos{
		spaceRepo: uc.spaceRepo,
		pageRepo:  uc.pageRepo,
		topicRepo: uc.topicRepo,
	}
	data, err := fetchPageAccessDataAllowingGuest(ctx, repos, input.SpaceIdentifier, input.PageNumber, nil)
	if err != nil {
		return nil, err
	}

	isPublic := policy.NewGuestPolicy().CanShowTopic(data.topic) && data.page.TrashedAt == nil
	// タイトルの無いページは既定のOGP画像を使うため、カード画像を作らない
	hasTitle := data.page.Title != nil && *data.page.Title != ""
	if !isPublic || !hasTitle {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	return &GetPageOgImageOutput{
		Space: data.space,
		Topic: data.topic,
		Page:  data.page,
	}, nil
}
