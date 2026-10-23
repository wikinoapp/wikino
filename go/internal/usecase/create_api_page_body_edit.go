package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateAPIPageBodyEditUsecaseは、公開APIでトークンが属するスペースのページの本文を検索置換するユースケース。
// 置換した本文は、ページの更新 (UpdateAPIPageUsecase) と同じ処理で保存する
type CreateAPIPageBodyEditUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
	updateValidator       *validator.APIPageUpdateValidator
}

// NewCreateAPIPageBodyEditUsecaseはCreateAPIPageBodyEditUsecaseを生成する
func NewCreateAPIPageBodyEditUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
	updateValidator *validator.APIPageUpdateValidator,
) *CreateAPIPageBodyEditUsecase {
	return &CreateAPIPageBodyEditUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		pageRepo:              pageRepo,
		pageRevisionRepo:      pageRevisionRepo,
		pageEditorRepo:        pageEditorRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
		updateValidator:       updateValidator,
	}
}

// CreateAPIPageBodyEditInputはページの本文の検索置換に必要な入力パラメータ
type CreateAPIPageBodyEditInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// HasIfMatchは `If-Match` が送られたかを表す。falseなら版を照合しない
	HasIfMatch bool
	// IfMatchDigestsは置換の前提にするページの版 (`Page.ContentDigest()` の値) の候補。
	// HasIfMatchがtrueなら、今の版がいずれかと一致する場合だけ置換する
	IfMatchDigests []string
	// IfMatchAnyは、ページが存在すれば版を問わず置換することを表す (`If-Match: *`)
	IfMatchAny bool
	// Editsは配列の順に適用する編集
	Edits []model.BodyEdit
}

// Executeはトークンが属するスペースのページの本文に編集を順に適用し、置換後の本文でページを更新する。
// 認可・未公開のページの公開・Wikiリンク先の作成・何も変わらない場合の扱いは、UpdateAPIPageUsecaseと同じ。
// 前提の版が指定されていて今の版と一致しなければ、AppErrCodePreconditionFailedのエラーを返す。
// 編集を1件でも適用できなければ、どの編集も適用せずに、最初に失敗した編集についての
// *model.ValidationErrorを返す
func (uc *CreateAPIPageBodyEditUsecase) Execute(ctx context.Context, input CreateAPIPageBodyEditInput) (*UpdateAPIPageOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}
	spaceMember := input.Principal.SpaceMember
	if spaceMember == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 取得からロックまでの間にページが変わっていた場合は、変わった本文に編集を適用し直す
	return retryAPIPageUpdate(func() (*UpdateAPIPageOutput, error) {
		return uc.editOnce(ctx, space, spaceMember, input)
	})
}

// editOnceはページを取得して認可・前提の照合を行い、本文に編集を適用して、1つのトランザクションで更新する
func (uc *CreateAPIPageBodyEditUsecase) editOnce(
	ctx context.Context,
	space *model.Space,
	spaceMember *model.SpaceMember,
	input CreateAPIPageBodyEditInput,
) (*UpdateAPIPageOutput, error) {
	repos := uc.apiPageWriteRepos()

	// 1. データ取得・2. 認可チェック
	page, topic, err := findAPIPageToUpdate(ctx, repos, input.Principal, space, spaceMember, input.PageNumber)
	if err != nil {
		return nil, err
	}

	// 前提の照合は編集の照合より先に行う。古い版を元にした編集は、今の本文に適用できても受け付けない
	if !input.matches(page) {
		return nil, &model.AppError{
			Code:    model.AppErrCodePreconditionFailed,
			UserMsg: i18n.T(ctx, "api_error_page_modified"),
		}
	}

	// 3. バリデーション・4. ビジネスロジック
	body, failure := model.ApplyBodyEdits(page.Body, input.Edits)
	if failure != nil {
		return nil, bodyEditFailureError(ctx, failure)
	}

	return saveAPIPageContent(ctx, uc.db, repos, uc.updateValidator, saveAPIPageContentInput{
		principal:   input.Principal,
		space:       space,
		spaceMember: spaceMember,
		topic:       topic,
		page:        page,
		title:       derefOr(page.Title, ""),
		body:        body,
	})
}

func (uc *CreateAPIPageBodyEditUsecase) apiPageWriteRepos() apiPageWriteRepos {
	return apiPageWriteRepos{
		spaceRepo:             uc.spaceRepo,
		pageRepo:              uc.pageRepo,
		pageRevisionRepo:      uc.pageRevisionRepo,
		pageEditorRepo:        uc.pageEditorRepo,
		topicRepo:             uc.topicRepo,
		topicMemberRepo:       uc.topicMemberRepo,
		attachmentRepo:        uc.attachmentRepo,
		pageAttachmentRefRepo: uc.pageAttachmentRefRepo,
	}
}

// matchesはページの今の版が、置換の前提にした版と一致するかを返す。`If-Match` が無ければ照合しない
func (input CreateAPIPageBodyEditInput) matches(page *model.Page) bool {
	if !input.HasIfMatch {
		return true
	}
	return input.IfMatchAny || slices.Contains(input.IfMatchDigests, page.ContentDigest())
}

// bodyEditFailureErrorは適用できなかった編集を、照合失敗ならold_str、サイズ超過ならnew_strのエラーにする。
// 公開APIではフィールド名をリクエスト本文の中の位置 (JSON Pointer) として返す
func bodyEditFailureError(ctx context.Context, failure *model.BodyEditFailure) *model.ValidationError {
	ve := model.NewValidationError()
	switch {
	case failure.SizeExceeded:
		ve.AddField(fmt.Sprintf("edits/%d/new_str", failure.Index),
			i18n.T(ctx, "api_error_body_edit_too_large", map[string]any{"Max": model.BodyEditMaxBodyBytes}))
	case failure.MatchCount > 1:
		ve.AddField(fmt.Sprintf("edits/%d/old_str", failure.Index),
			i18n.T(ctx, "api_error_body_edit_multiple_matches", map[string]any{"Count": failure.MatchCount}))
	default:
		ve.AddField(fmt.Sprintf("edits/%d/old_str", failure.Index), i18n.T(ctx, "api_error_body_edit_no_match"))
	}
	return ve
}
