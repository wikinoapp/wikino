package validator

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
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

// PageUpdateValidator はページ更新のバリデーションを行う
type PageUpdateValidator struct {
	pageRepo *repository.PageRepository
}

// NewPageUpdateValidator は PageUpdateValidator を生成する
func NewPageUpdateValidator(pageRepo *repository.PageRepository) *PageUpdateValidator {
	return &PageUpdateValidator{
		pageRepo: pageRepo,
	}
}

// PageUpdateValidatorInput はバリデーションの入力パラメータ
type PageUpdateValidatorInput struct {
	Title           string
	PageID          model.PageID
	TopicID         model.TopicID
	SpaceID         model.SpaceID
	SpaceIdentifier model.SpaceIdentifier
}

// Validate はバリデーションを行う。
// 戻り値の *model.PageID は未公開かつ本文が空の競合ページのID（存在する場合）。
func (v *PageUpdateValidator) Validate(ctx context.Context, input PageUpdateValidatorInput) (*model.PageID, error) {
	ve := model.NewValidationError()

	// 必須チェック
	if input.Title == "" {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_required"))
		return nil, ve
	}

	// 文字数チェック
	if utf8.RuneCountInString(input.Title) > pageTitleMaxLength {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_too_long"))
	}

	// 禁止文字チェック
	if invalidCharsRegex.MatchString(input.Title) {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_invalid_chars"))
	}

	// 先頭・末尾のスペースとドットのチェック
	if strings.HasPrefix(input.Title, " ") || strings.HasSuffix(input.Title, " ") ||
		strings.HasPrefix(input.Title, ".") || strings.HasSuffix(input.Title, ".") {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_invalid_format"))
	}

	// Windows予約語チェック
	upperTitle := strings.ToUpper(input.Title)
	if windowsReservedNames[upperTitle] {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_reserved"))
	}

	if ve.HasErrors() {
		return nil, ve
	}

	// タイトル一意性チェック（DB検証）
	existingPage, err := v.pageRepo.FindByTopicAndTitle(ctx, input.TopicID, input.Title, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("タイトル一意性チェックに失敗: %w", err)
	}

	if existingPage != nil && existingPage.ID != input.PageID {
		if existingPage.PublishedAt == nil && existingPage.Body == "" {
			// 未公開かつ本文が空のページとの競合 → エラーにせず、競合ページIDを返す
			return &existingPage.ID, nil
		}

		editPath := fmt.Sprintf("/s/%s/pages/%d/edit", input.SpaceIdentifier, existingPage.Number)
		errorMsg := templates.T(ctx, "validation_page_title_uniqueness_html")
		ve.AddField("title", fmt.Sprintf(errorMsg, editPath))
		return nil, ve
	}

	return nil, nil
}
