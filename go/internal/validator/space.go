package validator

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

const spaceNameMaxLength = 30

// スペースの識別子の形式 (半角英数字とハイフンのみ)
var spaceIdentifierRegex = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// spaceIdentifierReservedWordsはスペースの識別子に使えない語。識別子は大文字と小文字を
// 区別せずに一意になるため、予約語との比較も大文字と小文字を区別しない。
var spaceIdentifierReservedWords = []string{"www"}

// SpaceCreateValidatorはスペース作成のバリデーションを行う
type SpaceCreateValidator struct {
	spaceRepo *repository.SpaceRepository
}

// NewSpaceCreateValidatorはSpaceCreateValidatorを生成する
func NewSpaceCreateValidator(spaceRepo *repository.SpaceRepository) *SpaceCreateValidator {
	return &SpaceCreateValidator{spaceRepo: spaceRepo}
}

// SpaceCreateValidatorInputはバリデーションの入力パラメータ
type SpaceCreateValidatorInput struct {
	Identifier string
	Name       string
}

// Validateはバリデーションを行う
func (v *SpaceCreateValidator) Validate(ctx context.Context, input SpaceCreateValidatorInput) error {
	ve := model.NewValidationError()

	validateSpaceIdentifier(ctx, ve, input.Identifier)
	validateSpaceName(ctx, ve, input.Name)

	if ve.HasErrors() {
		return ve
	}

	// 識別子の一意性チェック (DB検証)
	exists, err := v.spaceRepo.ExistsByIdentifier(ctx, model.SpaceIdentifier(input.Identifier))
	if err != nil {
		return fmt.Errorf("スペースの識別子の一意性チェックに失敗: %w", err)
	}
	if exists {
		ve.AddField("identifier", i18n.T(ctx, "validation_space_identifier_uniqueness"))
		return ve
	}

	return nil
}

// validateSpaceIdentifierはスペースの識別子の形式を検証し、見つかったものをveに積む。
func validateSpaceIdentifier(ctx context.Context, ve *model.ValidationError, identifier string) {
	if identifier == "" {
		ve.AddField("identifier", i18n.T(ctx, "validation_space_identifier_required"))
		return
	}

	if utf8.RuneCountInString(identifier) > model.SpaceIdentifierMaxLength {
		ve.AddField("identifier", i18n.T(ctx, "validation_space_identifier_too_long"))
	}

	if !spaceIdentifierRegex.MatchString(identifier) {
		ve.AddField("identifier", i18n.T(ctx, "validation_space_identifier_invalid_format"))
	}

	if slices.ContainsFunc(spaceIdentifierReservedWords, func(word string) bool {
		return strings.EqualFold(identifier, word)
	}) {
		ve.AddField("identifier", i18n.T(ctx, "validation_space_identifier_reserved"))
	}
}

// validateSpaceNameはスペースの名前を検証し、見つかったものをveに積む。
func validateSpaceName(ctx context.Context, ve *model.ValidationError, name string) {
	// 空白だけの名前は、一覧やパンくずで名前の無いスペースに見えるため、未入力として扱う
	if strings.TrimSpace(name) == "" {
		ve.AddField("name", i18n.T(ctx, "validation_space_name_required"))
		return
	}

	if utf8.RuneCountInString(name) > spaceNameMaxLength {
		ve.AddField("name", i18n.T(ctx, "validation_space_name_too_long"))
	}
}
