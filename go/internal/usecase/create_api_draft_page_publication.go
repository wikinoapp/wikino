package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateAPIDraftPagePublicationUsecaseは、公開APIでページについてトークンの持ち主の下書きを公開するユースケース。
// Web画面の公開 (PublishPageUsecase) と同じく、下書きのタイトルと本文でページを公開して下書きを編集履歴ごと削除する。
// 加えて、公開の前提にした下書きの版 (`If-Match`) を今の版と照合してから公開する
type CreateAPIDraftPagePublicationUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
	updateValidator       *validator.APIPageUpdateValidator
}

const createAPIDraftPagePublicationRetryLimit = 3

// NewCreateAPIDraftPagePublicationUsecaseはCreateAPIDraftPagePublicationUsecaseを生成する
func NewCreateAPIDraftPagePublicationUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageEditorRepo *repository.PageEditorRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
	updateValidator *validator.APIPageUpdateValidator,
) *CreateAPIDraftPagePublicationUsecase {
	return &CreateAPIDraftPagePublicationUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		pageRepo:              pageRepo,
		pageRevisionRepo:      pageRevisionRepo,
		pageEditorRepo:        pageEditorRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
		updateValidator:       updateValidator,
	}
}

// CreateAPIDraftPagePublicationInputは下書きの公開に必要な入力パラメータ
type CreateAPIDraftPagePublicationInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// IfMatchDigestsは公開の前提にする下書きの版 (`DraftPage.ContentDigest()` の値) の候補。
	// 今の版がいずれかと一致する場合だけ公開する
	IfMatchDigests []string
	// IfMatchAnyは、下書きが存在すれば版を問わず公開することを表す (`If-Match: *`)
	IfMatchAny bool
}

// CreateAPIDraftPagePublicationOutputは下書きの公開結果
type CreateAPIDraftPagePublicationOutput struct {
	// Pageは公開したページ
	Page *model.Page
	// Topicはページが属するトピック
	Topic *model.Topic
}

// Executeはトークンが属するスペースのページについて、トークンの持ち主の下書きのタイトルと本文でページを公開する。
// 取得 (GetAPIDraftPageUsecase) と同じく、他のメンバーの下書き・ゴミ箱のページの下書き・閲覧者が開けない
// 非公開トピックの下書きは未存在として扱う。取得できる下書きでも、保存・公開の権限が無ければ
// AppErrCodeForbiddenのエラーを返す。今の版が前提の版と一致しなければ、AppErrCodePreconditionFailedのエラーを返す。
// タイトルの無い下書きや、ページのタイトルの規則に合わない下書きは公開せず、ValidationErrorを返す
func (uc *CreateAPIDraftPagePublicationUsecase) Execute(ctx context.Context, input CreateAPIDraftPagePublicationInput) (*CreateAPIDraftPagePublicationOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	// 照合・検証の後に下書きやページが変わっていた場合と、同時に作られたページと一意制約で競合した場合は、
	// 下書きの取得からやり直す。Web画面の公開・下書き保存はスペースをロックせずにページ
	// (Wikiリンク先を含む) を作るため、UpdateAPIPageUsecaseと同じく競合に備える
	var lastErr error
	for attempt := 0; attempt < createAPIDraftPagePublicationRetryLimit; attempt++ {
		output, err := uc.publishOnce(ctx, space, input)
		if err == nil {
			return output, nil
		}
		if !errors.Is(err, errAPIDraftPageChanged) &&
			!errors.Is(err, errAPIPageChanged) &&
			!errors.Is(err, errAPIPageReplacementChanged) &&
			!isUniqueViolationOn(err, "index_pages_on_space_id_and_number") &&
			!isUniqueViolationOn(err, "index_pages_on_topic_id_and_title") {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("下書きの公開の競合が%d回続きました: %w", createAPIDraftPagePublicationRetryLimit, lastErr)
}

// publishOnceは下書きを取得して認可・前提の照合・検証を行い、1つのトランザクションで公開する
func (uc *CreateAPIDraftPagePublicationUsecase) publishOnce(ctx context.Context, space *model.Space, input CreateAPIDraftPagePublicationInput) (*CreateAPIDraftPagePublicationOutput, error) {
	// 1. データ取得
	access, err := findAPIDraftPage(ctx, apiDraftPageAccessRepos{
		pageRepo:        uc.pageRepo,
		draftPageRepo:   uc.draftPageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}, input.Principal, space, input.PageNumber)
	if err != nil {
		return nil, err
	}
	draft := access.draft
	page := draft.Page
	// ページの移動は下書きの所属トピックも同じトランザクションで移すが、ページと下書きは別々に取得するため、
	// その間に移動がコミットされると一致しないことがある。一致しない下書きを公開するとページが下書きの
	// トピックへ黙って移るため公開せず、取得からやり直す。やり直しても一致しなければ、上限に達して失敗する
	if draft.TopicID != page.TopicID {
		return nil, fmt.Errorf("下書き %s の所属トピックがページ %s の所属トピックと異なります: %w", draft.ID, page.ID, errAPIDraftPageChanged)
	}

	// 2. 認可チェック
	if !access.authorizer.CanSaveDraftPage() || !access.authorizer.CanUpdatePage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 前提の照合は内容の検証より先に行う。利用者が確かめていない版は、内容が正しくても公開しない
	if !apiDraftPageMatches(draft, input.IfMatchDigests, input.IfMatchAny) {
		return nil, apiDraftPageModifiedError(ctx)
	}

	// 3. バリデーション
	// UpdateAPIPageUsecaseと同じく、ページのタイトルを変えない場合は検証しない。
	// タイトルの無い下書きは、必須の検証で拒否する
	title := derefOr(draft.Title, "")
	var unpublishedConflictingPageID *model.PageID
	if title != derefOr(page.Title, "") || title == "" {
		unpublishedConflictingPageID, err = uc.updateValidator.Validate(ctx, validator.APIPageUpdateValidatorInput{
			Title:   title,
			PageID:  page.ID,
			TopicID: access.topic.ID,
			SpaceID: space.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	// 4. ビジネスロジックと永続化
	// Web画面の公開と同じく、公開済みのページも今の日時で公開し直し、同じトランザクションで下書きを削除する
	publishedPage, err := writeAPIPage(ctx, uc.db, uc.apiPageWriteRepos(), writeAPIPageInput{
		principal:                    input.Principal,
		space:                        space,
		spaceMember:                  input.Principal.SpaceMember,
		topic:                        access.topic,
		page:                         page,
		title:                        title,
		body:                         draft.Body,
		unpublishedConflictingPageID: unpublishedConflictingPageID,
		republish:                    true,
		beforeCommit: func(tx *sql.Tx) error {
			return uc.deleteDraftPage(ctx, tx, draft)
		},
	})
	if err != nil {
		return nil, err
	}

	return &CreateAPIDraftPagePublicationOutput{Page: publishedPage, Topic: access.topic}, nil
}

// deleteDraftPageは公開した下書きを編集履歴ごと削除する。
// 照合から公開までの間にWeb画面の自動保存などが割り込まないよう、下書きの行をロックしてから、
// 公開した内容と同じ下書きであることを確かめる。変わっていたか消えていれば、errAPIDraftPageChangedを返す
func (uc *CreateAPIDraftPagePublicationUsecase) deleteDraftPage(ctx context.Context, tx *sql.Tx, draft *model.DraftPage) error {
	deleted, err := deleteDraftPageWithRevisions(ctx, uc.draftPageRepo.WithTx(tx), uc.draftPageRevisionRepo.WithTx(tx), draft.ID, draft.SpaceID,
		func(locked *model.DraftPage) error {
			if locked.ContentDigest() != draft.ContentDigest() {
				return errAPIDraftPageChanged
			}
			return nil
		})
	if err != nil {
		return err
	}
	if !deleted {
		return errAPIDraftPageChanged
	}
	return nil
}

func (uc *CreateAPIDraftPagePublicationUsecase) apiPageWriteRepos() apiPageWriteRepos {
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
