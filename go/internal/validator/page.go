package validator

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/templates"
)

const pageTitleMaxLength = 200

// pageTitleForbiddenCharsはページタイトルが持てない文字。いずれも名前どうしを分ける。
// "/" はWikiリンクでトピック名とページタイトルを分け、UnixとObsidianのvaultのパス区切り
// でもある。"\" はWindowsのパス区切り、":" はWindowsでドライブレターと代替データストリームを
// 開く。
//
// タイトルはファイル名ではないため、"*" や "|" のようにファイルシステムだけが受け付けない文字は
// ここで拒否せず、ページのエクスポート時に変換する (internal/exportfileを参照)。区切り文字だけは
// 例外である。予定している双方向同期はファイル名からタイトルを読み戻すため、これらを含む
// タイトルは書き出した名前から読み戻せない。
const pageTitleForbiddenChars = `/\:`

// PageUpdateValidatorはページ更新のバリデーションを行う
type PageUpdateValidator struct {
	pageRepo *repository.PageRepository
}

// NewPageUpdateValidatorはPageUpdateValidatorを生成する
func NewPageUpdateValidator(pageRepo *repository.PageRepository) *PageUpdateValidator {
	return &PageUpdateValidator{
		pageRepo: pageRepo,
	}
}

// PageUpdateValidatorInputはバリデーションの入力パラメータ
type PageUpdateValidatorInput struct {
	Title           string
	PageID          model.PageID
	TopicID         model.TopicID
	SpaceID         model.SpaceID
	SpaceIdentifier model.SpaceIdentifier
}

// Validateはバリデーションを行う。
// 戻り値の *model.PageIDは未公開かつ本文が空の競合ページのID (存在する場合)。
func (v *PageUpdateValidator) Validate(ctx context.Context, input PageUpdateValidatorInput) (*model.PageID, error) {
	ve := model.NewValidationError()

	if !validatePageTitleFormat(ctx, ve, input.Title) {
		return nil, ve
	}

	// タイトル一意性チェック (DB検証)
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

// PageCreateValidatorは公開APIでのページ作成のバリデーションを行う。
// タイトルの形式はPageUpdateValidatorと同じ規則で検証する。
// 重複の説明はWeb画面のような編集画面へのリンク (HTML) を含めず、平文にする
type PageCreateValidator struct {
	pageRepo *repository.PageRepository
}

// NewPageCreateValidatorはPageCreateValidatorを生成する
func NewPageCreateValidator(pageRepo *repository.PageRepository) *PageCreateValidator {
	return &PageCreateValidator{
		pageRepo: pageRepo,
	}
}

// PageCreateValidatorInputはバリデーションの入力パラメータ
type PageCreateValidatorInput struct {
	Title   string
	TopicID model.TopicID
	SpaceID model.SpaceID
}

// Validateはバリデーションを行う。
// 戻り値の *model.PageIDは未公開かつ本文が空の競合ページのID (存在する場合)。
// このページはWikiリンクの自動作成などで作られた中身の無いページで、Web画面の公開と同じく
// 呼び出し側で論理削除して置き換える
func (v *PageCreateValidator) Validate(ctx context.Context, input PageCreateValidatorInput) (*model.PageID, error) {
	ve := model.NewValidationError()

	if !validatePageTitleFormat(ctx, ve, input.Title) {
		return nil, ve
	}

	existingPage, err := v.pageRepo.FindByTopicAndTitle(ctx, input.TopicID, input.Title, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("タイトル一意性チェックに失敗: %w", err)
	}
	if existingPage != nil {
		if existingPage.PublishedAt == nil && existingPage.Body == "" {
			return &existingPage.ID, nil
		}

		ve.AddField("title", i18n.T(ctx, "api_error_page_title_taken"))
		return nil, ve
	}

	return nil, nil
}

// APIPageUpdateValidatorは公開APIでのページ更新のタイトルのバリデーションを行う。
// PageCreateValidatorと同じく、形式はWeb画面と同じ規則で検証し、重複の説明は平文にする。
// 更新するページ自身は重複の相手にしない
type APIPageUpdateValidator struct {
	pageRepo *repository.PageRepository
}

// NewAPIPageUpdateValidatorはAPIPageUpdateValidatorを生成する
func NewAPIPageUpdateValidator(pageRepo *repository.PageRepository) *APIPageUpdateValidator {
	return &APIPageUpdateValidator{
		pageRepo: pageRepo,
	}
}

// APIPageUpdateValidatorInputはバリデーションの入力パラメータ
type APIPageUpdateValidatorInput struct {
	Title   string
	PageID  model.PageID
	TopicID model.TopicID
	SpaceID model.SpaceID
}

// Validateはバリデーションを行う。
// 戻り値の *model.PageIDは未公開かつ本文が空の競合ページのID (存在する場合)。
// PageCreateValidatorと同じく、呼び出し側で論理削除して置き換える
func (v *APIPageUpdateValidator) Validate(ctx context.Context, input APIPageUpdateValidatorInput) (*model.PageID, error) {
	ve := model.NewValidationError()

	if !validatePageTitleFormat(ctx, ve, input.Title) {
		return nil, ve
	}

	existingPage, err := v.pageRepo.FindByTopicAndTitle(ctx, input.TopicID, input.Title, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("タイトル一意性チェックに失敗: %w", err)
	}
	if existingPage != nil && existingPage.ID != input.PageID {
		if existingPage.PublishedAt == nil && existingPage.Body == "" {
			return &existingPage.ID, nil
		}

		ve.AddField("title", i18n.T(ctx, "api_error_page_title_taken"))
		return nil, ve
	}

	return nil, nil
}

// validatePageTitleFormatはページタイトルの形式を検証し、問題をveに加える。
// 形式に問題が無ければtrueを返す
func validatePageTitleFormat(ctx context.Context, ve *model.ValidationError, title string) bool {
	// 必須チェック
	if title == "" {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_required"))
		return false
	}

	// 文字数チェック
	if utf8.RuneCountInString(title) > pageTitleMaxLength {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_too_long"))
	}

	// 禁止文字チェック
	if strings.ContainsAny(title, pageTitleForbiddenChars) {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_invalid_chars"))
	}

	// 制御文字も拒否する。タイトルを表示する場所ごとに描画のされ方が異なり、読み手が読める
	// タイトルの一部にもならないためである。
	if strings.ContainsFunc(title, unicode.IsControl) {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_control_chars"))
	}

	// 先頭・末尾の空白を拒否するのはファイル名の都合ではなくタイトル自身の都合である。
	// そこだけが違う2つのタイトルは、一覧に並んだときに同じものに見える。
	if strings.HasPrefix(title, " ") || strings.HasSuffix(title, " ") {
		ve.AddField("title", i18n.T(ctx, "validation_page_title_invalid_format"))
	}

	return !ve.HasErrors()
}
