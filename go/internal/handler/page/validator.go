package page

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates"
)

const pageTitleMaxLength = 200

// ファイル名として使用できない文字
var invalidCharsRegex = regexp.MustCompile(`[/\\:*?"<>|]`)

// Windowsの予約デバイス名
var windowsReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// UpdateValidator はページ更新のバリデーションを行う
type UpdateValidator struct {
	pageRepo *repository.PageRepository
}

// NewUpdateValidator は UpdateValidator を生成する
func NewUpdateValidator(pageRepo *repository.PageRepository) *UpdateValidator {
	return &UpdateValidator{
		pageRepo: pageRepo,
	}
}

// UpdateValidatorInput はバリデーションの入力パラメータ
type UpdateValidatorInput struct {
	Title           string
	PageID          model.PageID
	TopicID         model.TopicID
	SpaceID         model.SpaceID
	SpaceIdentifier model.SpaceIdentifier
}

// UpdateValidatorResult はバリデーションの結果
type UpdateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *UpdateValidator) Validate(ctx context.Context, input UpdateValidatorInput) *UpdateValidatorResult {
	formErrors := session.NewFormErrors()

	// 必須チェック
	if input.Title == "" {
		formErrors.AddField("title", i18n.T(ctx, "validation_page_title_required"))
		return &UpdateValidatorResult{FormErrors: formErrors}
	}

	// 文字数チェック
	if utf8.RuneCountInString(input.Title) > pageTitleMaxLength {
		formErrors.AddField("title", i18n.T(ctx, "validation_page_title_too_long"))
	}

	// 禁止文字チェック
	if invalidCharsRegex.MatchString(input.Title) {
		formErrors.AddField("title", i18n.T(ctx, "validation_page_title_invalid_chars"))
	}

	// 先頭・末尾のスペースとドットのチェック
	if strings.HasPrefix(input.Title, " ") || strings.HasSuffix(input.Title, " ") ||
		strings.HasPrefix(input.Title, ".") || strings.HasSuffix(input.Title, ".") {
		formErrors.AddField("title", i18n.T(ctx, "validation_page_title_invalid_format"))
	}

	// Windows予約語チェック
	upperTitle := strings.ToUpper(input.Title)
	if windowsReservedNames[upperTitle] {
		formErrors.AddField("title", i18n.T(ctx, "validation_page_title_reserved"))
	}

	if formErrors.HasErrors() {
		return &UpdateValidatorResult{FormErrors: formErrors}
	}

	// タイトル一意性チェック（DB検証）
	existingPage, err := v.pageRepo.FindByTopicAndTitle(ctx, input.TopicID, input.Title, input.SpaceID)
	if err != nil {
		formErrors.AddField("title", i18n.T(ctx, "validation_system_error"))
		return &UpdateValidatorResult{FormErrors: formErrors}
	}

	if existingPage != nil && existingPage.ID != input.PageID {
		editPath := fmt.Sprintf("/go/s/%s/pages/%d/edit", input.SpaceIdentifier, existingPage.Number)
		errorMsg := templates.T(ctx, "validation_page_title_uniqueness_html")
		formErrors.AddField("title", fmt.Sprintf(errorMsg, editPath))
	}

	return &UpdateValidatorResult{FormErrors: formErrors}
}
