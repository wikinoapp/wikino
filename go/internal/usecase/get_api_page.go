package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAPIPageUsecaseは、公開APIでトークンが属するスペースのページを取得するユースケース
type GetAPIPageUsecase struct {
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewGetAPIPageUsecaseはGetAPIPageUsecaseを生成する
func NewGetAPIPageUsecase(pageRepo *repository.PageRepository, topicRepo *repository.TopicRepository, topicMemberRepo *repository.TopicMemberRepository) *GetAPIPageUsecase {
	return &GetAPIPageUsecase{
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
	}
}

// GetAPIPageInputはページの取得に必要な入力パラメータ
type GetAPIPageInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
}

// GetAPIPageOutputはページの取得結果
type GetAPIPageOutput struct {
	Page *model.Page
	// Topicはページが属するトピック
	Topic *model.Topic
}

// Executeはトークンが属するスペースのページを返す。
// Web画面のページ画面と同じく、未公開のページ (Wikiリンクで自動作成されたページなど) も返す。
// ページが存在しない場合に加え、ゴミ箱のページ・廃棄済みのトピックのページ・閲覧者が開けない
// 非公開トピックのページも、存在を秘匿するため未存在として扱う。
// Web画面と違い、ゴミ箱を開ける権限を持つメンバーにもゴミ箱のページを返さない (一覧と揃えるため)
func (uc *GetAPIPageUsecase) Execute(ctx context.Context, input GetAPIPageInput) (*GetAPIPageOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

	page, err := uc.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(input.PageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if !isAPIVisiblePage(page) {
		return nil, notFound
	}

	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, page.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, notFound
	}

	var topicMember *model.TopicMember
	if spaceMember := input.Principal.SpaceMember; spaceMember != nil {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	if !newAPIAuthorizer(input.Principal, topicMember).CanShowTopic(topic) {
		return nil, notFound
	}

	return &GetAPIPageOutput{Page: page, Topic: topic}, nil
}
