package validator

import (
	"context"
	"strconv"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

// PageMoveCreateValidator はページ移動のバリデーションを行う
type PageMoveCreateValidator struct {
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// NewPageMoveCreateValidator は PageMoveCreateValidator を生成する
func NewPageMoveCreateValidator(
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
) *PageMoveCreateValidator {
	return &PageMoveCreateValidator{
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
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

// PageMoveCreateValidatorResult はバリデーションの結果
type PageMoveCreateValidatorResult struct {
	DestTopic  *model.Topic
	FormErrors *session.FormErrors
	Err        error
}

// Validate はバリデーションを行う
func (v *PageMoveCreateValidator) Validate(ctx context.Context, input PageMoveCreateValidatorInput) *PageMoveCreateValidatorResult {
	formErrors := session.NewFormErrors()

	// 形式バリデーション: 移動先トピックが選択されていること
	if input.DestTopicNumber == "" {
		formErrors.AddField("dest_topic", i18n.T(ctx, "page_move_error_topic_required"))
		return &PageMoveCreateValidatorResult{FormErrors: formErrors}
	}

	// 移動先トピック番号をパース
	parsed, err := strconv.ParseInt(input.DestTopicNumber, 10, 32)
	if err != nil {
		formErrors.AddField("dest_topic", i18n.T(ctx, "page_move_error_topic_required"))
		return &PageMoveCreateValidatorResult{FormErrors: formErrors}
	}
	destTopicNumber := int32(parsed)

	// 状態バリデーション: 移動先トピックが同一スペース内に存在すること
	destTopic, err := v.topicRepo.FindBySpaceAndNumber(ctx, input.SpaceID, destTopicNumber)
	if err != nil {
		return &PageMoveCreateValidatorResult{Err: err}
	}
	if destTopic == nil {
		formErrors.AddField("dest_topic", i18n.T(ctx, "page_move_error_topic_required"))
		return &PageMoveCreateValidatorResult{FormErrors: formErrors}
	}

	// 移動先トピックが現在のトピックと異なること
	if destTopic.ID == input.CurrentTopicID {
		formErrors.AddField("dest_topic", i18n.T(ctx, "page_move_error_same_topic"))
		return &PageMoveCreateValidatorResult{FormErrors: formErrors}
	}

	// 移動先トピックにページ作成権限があること
	topicMember, err := v.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, input.SpaceID, input.SpaceMember.ID, destTopic.ID)
	if err != nil {
		return &PageMoveCreateValidatorResult{Err: err}
	}

	topicPolicy := policy.NewTopicPolicy(input.SpaceMember, topicMember)
	if !topicPolicy.CanCreatePage(destTopic) {
		formErrors.AddField("dest_topic", i18n.T(ctx, "page_move_error_no_permission"))
		return &PageMoveCreateValidatorResult{FormErrors: formErrors}
	}

	// 移動先トピックに同名のページが存在しないこと
	if input.PageTitle != "" {
		existingPage, err := v.pageRepo.FindByTopicAndTitle(ctx, destTopic.ID, input.PageTitle, input.SpaceID)
		if err != nil {
			return &PageMoveCreateValidatorResult{Err: err}
		}
		if existingPage != nil && existingPage.ID != input.PageID {
			formErrors.AddField("dest_topic", i18n.T(ctx, "page_move_error_title_exists"))
			return &PageMoveCreateValidatorResult{FormErrors: formErrors}
		}
	}

	return &PageMoveCreateValidatorResult{DestTopic: destTopic}
}
