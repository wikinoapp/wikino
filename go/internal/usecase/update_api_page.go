package usecase

import (
	"context"
	"database/sql"
	"slices"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdateAPIPageUsecaseは、公開APIでトークンが属するスペースのページを更新するユースケース。
// Web画面の公開 (PublishPageUsecase) と違い、更新の前提にした版 (`If-Match`) を今の版と照合してから
// 更新し、トークンの持ち主の下書きには触れない
type UpdateAPIPageUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	draftPageRepo         *repository.DraftPageRepository
	suggestionPageRepo    *repository.SuggestionPageRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
	updateValidator       *validator.APIPageUpdateValidator
}

// NewUpdateAPIPageUsecaseはUpdateAPIPageUsecaseを生成する
func NewUpdateAPIPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageEditorRepo *repository.PageEditorRepository,
	draftPageRepo *repository.DraftPageRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
	updateValidator *validator.APIPageUpdateValidator,
) *UpdateAPIPageUsecase {
	return &UpdateAPIPageUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		pageRepo:              pageRepo,
		pageRevisionRepo:      pageRevisionRepo,
		pageEditorRepo:        pageEditorRepo,
		draftPageRepo:         draftPageRepo,
		suggestionPageRepo:    suggestionPageRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
		updateValidator:       updateValidator,
	}
}

// UpdateAPIPageInputはページの更新に必要な入力パラメータ
type UpdateAPIPageInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// IfMatchDigestsは更新の前提にするページの版 (`Page.ContentDigest()` の値) の候補。
	// 今の版がいずれかと一致する場合だけ更新する
	IfMatchDigests []string
	// IfMatchAnyは、ページが存在すれば版を問わず更新することを表す (`If-Match: *`)
	IfMatchAny bool
	// Titleは新しいタイトル。nilならタイトルを変えない
	Title *string
	// Bodyは新しい本文。nilなら本文を変えない
	Body *string
}

// UpdateAPIPageOutputはページの更新結果
type UpdateAPIPageOutput struct {
	Page *model.Page
	// Topicはページが属するトピック
	Topic *model.Topic
}

// Executeはトークンが属するスペースのページのタイトルと本文を更新する。
// 未公開のページは、Web画面の編集と同じく更新して公開する。
// 取得 (GetAPIPageUsecase) と同じく、ゴミ箱のページと閲覧者が開けない非公開トピックの
// ページは未存在として扱い、ページを開けるが更新できない場合も存在を秘匿して未存在を返す。
// 今の版が前提の版と一致しなければ、AppErrCodePreconditionFailedのエラーを返す。
// 公開済みのページでタイトルと本文が今と同じ場合は、何も書き込まずに今のページを返す
func (uc *UpdateAPIPageUsecase) Execute(ctx context.Context, input UpdateAPIPageInput) (*UpdateAPIPageOutput, error) {
	return retryOnDeadlock(ctx, func() (*UpdateAPIPageOutput, error) {
		return uc.executeOnce(ctx, input)
	})
}

// executeOnceは取得・認可・検証から永続化までを1回実行する
func (uc *UpdateAPIPageUsecase) executeOnce(ctx context.Context, input UpdateAPIPageInput) (*UpdateAPIPageOutput, error) {
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

	return retryAPIPageUpdate(func() (*UpdateAPIPageOutput, error) {
		return uc.updateOnce(ctx, space, spaceMember, input)
	})
}

// updateOnceはページを取得して認可・前提の照合・検証を行い、1つのトランザクションで更新する
func (uc *UpdateAPIPageUsecase) updateOnce(
	ctx context.Context,
	space *model.Space,
	spaceMember *model.SpaceMember,
	input UpdateAPIPageInput,
) (*UpdateAPIPageOutput, error) {
	repos := uc.apiPageWriteRepos()

	// 1. データ取得・2. 認可チェック
	page, topic, err := findAPIPageToUpdate(ctx, repos, input.Principal, space, spaceMember, input.PageNumber)
	if err != nil {
		return nil, err
	}

	// 前提の照合は内容の検証より先に行う。古い版を元にした更新は、内容が正しくても受け付けない
	if !input.matches(page) {
		return nil, &model.AppError{
			Code:    model.AppErrCodePreconditionFailed,
			UserMsg: i18n.T(ctx, "api_error_page_modified"),
		}
	}

	// 3. バリデーション・4. ビジネスロジック
	return saveAPIPageContent(ctx, uc.db, repos, uc.updateValidator, saveAPIPageContentInput{
		principal:   input.Principal,
		space:       space,
		spaceMember: spaceMember,
		topic:       topic,
		page:        page,
		title:       derefOr(input.Title, derefOr(page.Title, "")),
		body:        derefOr(input.Body, page.Body),
	})
}

func (uc *UpdateAPIPageUsecase) apiPageWriteRepos() apiPageWriteRepos {
	return apiPageWriteRepos{
		spaceRepo:             uc.spaceRepo,
		pageRepo:              uc.pageRepo,
		pageRevisionRepo:      uc.pageRevisionRepo,
		pageEditorRepo:        uc.pageEditorRepo,
		draftPageRepo:         uc.draftPageRepo,
		suggestionPageRepo:    uc.suggestionPageRepo,
		topicRepo:             uc.topicRepo,
		topicMemberRepo:       uc.topicMemberRepo,
		attachmentRepo:        uc.attachmentRepo,
		pageAttachmentRefRepo: uc.pageAttachmentRefRepo,
	}
}

// matchesはページの今の版が、更新の前提にした版と一致するかを返す
func (input UpdateAPIPageInput) matches(page *model.Page) bool {
	return input.IfMatchAny || slices.Contains(input.IfMatchDigests, page.ContentDigest())
}
