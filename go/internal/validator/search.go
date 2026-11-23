package validator

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

const (
	searchQueryMinLength = 2
	searchQueryMaxLength = 100
)

// searchSpaceFilterPatternは、検索のキーワードのうち、検索するスペースを指定する `space:識別子` に一致する。
var searchSpaceFilterPattern = regexp.MustCompile(`space:(\S+)`)

// SearchShowValidatorは、検索の画面に送られたキーワードを検証し、キーワードとスペースの指定に分ける。
type SearchShowValidator struct{}

// NewSearchShowValidatorはSearchShowValidatorを生成する。
func NewSearchShowValidator() *SearchShowValidator {
	return &SearchShowValidator{}
}

// SearchShowValidatorInputは検証の入力。
type SearchShowValidatorInput struct {
	// Queryは前後の空白を除いたキーワード。空なら検索しない
	Query string
}

// SearchShowValidateOutputは、キーワードを検索の条件に分けた結果。
type SearchShowValidateOutput struct {
	// Keywordsは、スペースの指定を除いて空白で分けたキーワード。空なら検索しない
	Keywords []string
	// SpaceIdentifiersは `space:識別子` で指定されたスペース
	SpaceIdentifiers []model.SpaceIdentifier
}

// Validateは、キーワードの長さ (2文字以上100文字以内) と使えない文字 (`<` と `>`) を確かめ、
// キーワードとスペースの指定に分ける。キーワードが空なら、検証せずに空の結果を返す。
func (v *SearchShowValidator) Validate(ctx context.Context, input SearchShowValidatorInput) (*SearchShowValidateOutput, error) {
	if input.Query == "" {
		return &SearchShowValidateOutput{}, nil
	}

	ve := model.NewValidationError()
	length := utf8.RuneCountInString(input.Query)
	if length < searchQueryMinLength {
		ve.AddField("q", i18n.T(ctx, "validation_search_query_too_short", map[string]any{"Count": searchQueryMinLength}))
	}
	if length > searchQueryMaxLength {
		ve.AddField("q", i18n.T(ctx, "validation_search_query_too_long", map[string]any{"Count": searchQueryMaxLength}))
	}
	if strings.ContainsAny(input.Query, "<>") {
		ve.AddField("q", i18n.T(ctx, "validation_search_query_invalid_characters"))
	}
	if ve.HasErrors() {
		return nil, ve
	}

	var spaceIdentifiers []model.SpaceIdentifier
	for _, match := range searchSpaceFilterPattern.FindAllStringSubmatch(input.Query, -1) {
		spaceIdentifiers = append(spaceIdentifiers, model.SpaceIdentifier(match[1]))
	}

	return &SearchShowValidateOutput{
		Keywords:         strings.Fields(searchSpaceFilterPattern.ReplaceAllString(input.Query, "")),
		SpaceIdentifiers: spaceIdentifiers,
	}, nil
}
