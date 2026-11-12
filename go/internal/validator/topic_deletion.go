package validator

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicDeletionCreateValidatorはトピックの削除の確認のバリデーションを行う。
// 削除するトピックの名前を入力させ、取り違えたトピックを消さないようにする。
type TopicDeletionCreateValidator struct{}

// NewTopicDeletionCreateValidatorはTopicDeletionCreateValidatorを生成する
func NewTopicDeletionCreateValidator() *TopicDeletionCreateValidator {
	return &TopicDeletionCreateValidator{}
}

// TopicDeletionCreateValidatorInputはバリデーションの入力パラメータ
type TopicDeletionCreateValidatorInput struct {
	// TopicNameは閲覧者が入力したトピックの名前
	TopicName string
	// Topicは削除しようとしているトピック
	Topic *model.Topic
}

// Validateは、入力したトピックの名前が削除するトピックの名前と一致するかを確かめる。
// 一致は前後の空白も含めた完全一致で判定する (Rails版の `Topics::DestroyConfirmationForm` と同じ)。
func (v *TopicDeletionCreateValidator) Validate(ctx context.Context, input TopicDeletionCreateValidatorInput) error {
	ve := model.NewValidationError()

	if input.TopicName == "" {
		ve.AddField("topic_name", i18n.T(ctx, "validation_required"))
		return ve
	}

	if input.TopicName != input.Topic.Name {
		ve.AddField("topic_name", i18n.T(ctx, "validation_topic_deletion_name_mismatch"))
		return ve
	}

	return nil
}
