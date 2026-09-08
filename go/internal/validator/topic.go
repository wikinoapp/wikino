package validator

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

const (
	topicNameMaxLength        = 30
	topicDescriptionMaxLength = 150
)

// topicNameForbiddenChars holds the characters a topic name may not carry: the ones a file name
// cannot hold on one of the operating systems the export is opened on.
//
// [Ja] topicNameForbiddenChars はトピック名が持てない文字。エクスポートを開く OS のいずれかで
// ファイル名に含められない文字である。
const topicNameForbiddenChars = `/\:*?"<>|`

// topicNameReservedNames holds the device names Windows keeps reserved. A file of one of these
// names cannot be created there, whatever the extension is.
//
// [Ja] topicNameReservedNames は Windows が予約しているデバイス名。これらの名前のファイルは、
// 拡張子が何であれ Windows では作成できない。
var topicNameReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// TopicCreateValidator はトピック作成のバリデーションを行う
type TopicCreateValidator struct {
	topicRepo *repository.TopicRepository
}

// NewTopicCreateValidator は TopicCreateValidator を生成する
func NewTopicCreateValidator(topicRepo *repository.TopicRepository) *TopicCreateValidator {
	return &TopicCreateValidator{topicRepo: topicRepo}
}

// TopicCreateValidatorInput はバリデーションの入力パラメータ
type TopicCreateValidatorInput struct {
	Name        string
	Description string
	Visibility  string
	SpaceID     model.SpaceID
}

// Validate はバリデーションを行い、フォームが送信した公開範囲を変換して返す。
func (v *TopicCreateValidator) Validate(ctx context.Context, input TopicCreateValidatorInput) (model.TopicVisibility, error) {
	ve := model.NewValidationError()

	validateTopicName(ctx, ve, input.Name)

	if utf8.RuneCountInString(input.Description) > topicDescriptionMaxLength {
		ve.AddField("description", i18n.T(ctx, "validation_topic_description_too_long"))
	}

	visibility, ok := model.ParseTopicVisibility(input.Visibility)
	if !ok {
		ve.AddField("visibility", i18n.T(ctx, "validation_topic_visibility_required"))
	}

	if ve.HasErrors() {
		return visibility, ve
	}

	// 名前の一意性チェック (DB検証)
	exists, err := v.topicRepo.ExistsBySpaceAndName(ctx, input.SpaceID, input.Name)
	if err != nil {
		return visibility, fmt.Errorf("トピック名の一意性チェックに失敗: %w", err)
	}
	if exists {
		ve.AddField("name", i18n.T(ctx, "validation_topic_name_uniqueness"))
		return visibility, ve
	}

	return visibility, nil
}

// validateTopicName checks the format of a topic name and adds what it finds to ve.
//
// [Ja] validateTopicName はトピック名の形式を検証し、見つかったものを ve に積む。
func validateTopicName(ctx context.Context, ve *model.ValidationError, name string) {
	if name == "" {
		ve.AddField("name", i18n.T(ctx, "validation_topic_name_required"))
		return
	}

	if utf8.RuneCountInString(name) > topicNameMaxLength {
		ve.AddField("name", i18n.T(ctx, "validation_topic_name_too_long"))
	}

	if strings.ContainsAny(name, topicNameForbiddenChars) {
		ve.AddField("name", i18n.T(ctx, "validation_topic_name_invalid_chars"))
	}

	if strings.HasPrefix(name, " ") || strings.HasPrefix(name, ".") ||
		strings.HasSuffix(name, " ") || strings.HasSuffix(name, ".") {
		ve.AddField("name", i18n.T(ctx, "validation_topic_name_invalid_format"))
	}

	if topicNameReservedNames[strings.ToUpper(name)] {
		ve.AddField("name", i18n.T(ctx, "validation_topic_name_reserved"))
	}
}
