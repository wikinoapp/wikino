package validator

import (
	"context"
	"strconv"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// PageMoveCreateValidator はページ移動のバリデーションを行う
type PageMoveCreateValidator struct {
	pageRepo           *repository.PageRepository
	topicRepo          *repository.TopicRepository
	topicMemberRepo    *repository.TopicMemberRepository
	suggestionPageRepo *repository.SuggestionPageRepository
}

// NewPageMoveCreateValidator は PageMoveCreateValidator を生成する
func NewPageMoveCreateValidator(
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
) *PageMoveCreateValidator {
	return &PageMoveCreateValidator{
		pageRepo:           pageRepo,
		topicRepo:          topicRepo,
		topicMemberRepo:    topicMemberRepo,
		suggestionPageRepo: suggestionPageRepo,
	}
}

// PageMoveCreateValidatorInput はバリデーションの入力パラメータ
type PageMoveCreateValidatorInput struct {
	DestTopicNumber string
	PageID          model.PageID
	PageTitle       string
	CurrentTopicID  model.TopicID
	SpaceID         model.SpaceID
	SpaceMember     *model.SpaceMember
}

// Validate はバリデーションを行う。
// 成功時は移動先トピックを返す。バリデーションエラー時は *model.ValidationError を返す。
func (v *PageMoveCreateValidator) Validate(ctx context.Context, input PageMoveCreateValidatorInput) (*model.Topic, error) {
	ve := model.NewValidationError()

	// 形式バリデーション: 移動先トピックが選択されていること
	if input.DestTopicNumber == "" {
		ve.AddField("dest_topic", i18n.T(ctx, "page_move_error_topic_required"))
		return nil, ve
	}

	// 移動先トピック番号をパース
	parsed, err := strconv.ParseInt(input.DestTopicNumber, 10, 32)
	if err != nil {
		ve.AddField("dest_topic", i18n.T(ctx, "page_move_error_topic_required"))
		return nil, ve
	}
	destTopicNumber := int32(parsed)

	// 状態バリデーション: 対象ページにオープンな編集提案が存在しないこと
	hasOpenSuggestion, err := v.suggestionPageRepo.ExistsByPageIDAndOpenStatus(ctx, input.PageID, input.SpaceID)
	if err != nil {
		return nil, err
	}
	if hasOpenSuggestion {
		ve.AddField("dest_topic", i18n.T(ctx, "page_move_error_open_suggestion_exists"))
		return nil, ve
	}

	// 状態バリデーション: 移動先トピックが同一スペース内に存在すること
	destTopic, err := v.topicRepo.FindBySpaceAndNumber(ctx, input.SpaceID, destTopicNumber)
	if err != nil {
		return nil, err
	}
	if destTopic == nil {
		ve.AddField("dest_topic", i18n.T(ctx, "page_move_error_topic_required"))
		return nil, ve
	}

	// 移動先トピックが現在のトピックと異なること
	if destTopic.ID == input.CurrentTopicID {
		ve.AddField("dest_topic", i18n.T(ctx, "page_move_error_same_topic"))
		return nil, ve
	}

	// 移動先トピックにページ作成権限があること
	topicMember, err := v.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, input.SpaceID, input.SpaceMember.ID, destTopic.ID)
	if err != nil {
		return nil, err
	}

	var topicScopes []model.Scope
	if topicMember != nil {
		topicScopes = topicMember.Scopes
	}
	authorizer := policy.NewMemberPolicy(input.SpaceMember.Scopes, topicScopes)
	if !authorizer.CanCreatePage() {
		ve.AddField("dest_topic", i18n.T(ctx, "page_move_error_no_permission"))
		return nil, ve
	}

	// 移動先トピックに同名のページが存在しないこと
	if input.PageTitle != "" {
		existingPage, err := v.pageRepo.FindByTopicAndTitle(ctx, destTopic.ID, input.PageTitle, input.SpaceID)
		if err != nil {
			return nil, err
		}
		if existingPage != nil && existingPage.ID != input.PageID {
			ve.AddField("dest_topic", i18n.T(ctx, "page_move_error_title_exists"))
			return nil, ve
		}
	}

	return destTopic, nil
}
