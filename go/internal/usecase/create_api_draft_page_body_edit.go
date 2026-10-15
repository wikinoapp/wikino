package usecase

import (
	"context"
	"database/sql"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// CreateAPIDraftPageBodyEditUsecaseは、公開APIでページについてトークンの持ち主の下書きの本文を検索置換するユースケース。
// 置換した本文は、下書きの更新 (UpdateAPIDraftPageUsecase) と同じく手動保存と同じ処理で保存する
type CreateAPIDraftPageBodyEditUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	pageRepo              *repository.PageRepository
	pageEditorRepo        *repository.PageEditorRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
}

// NewCreateAPIDraftPageBodyEditUsecaseはCreateAPIDraftPageBodyEditUsecaseを生成する
func NewCreateAPIDraftPageBodyEditUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
) *CreateAPIDraftPageBodyEditUsecase {
	return &CreateAPIDraftPageBodyEditUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		pageRepo:              pageRepo,
		pageEditorRepo:        pageEditorRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
	}
}

// CreateAPIDraftPageBodyEditInputは下書きの本文の検索置換に必要な入力パラメータ
type CreateAPIDraftPageBodyEditInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// HasIfMatchは `If-Match` が送られたかを表す。falseなら版を照合しない
	HasIfMatch bool
	// IfMatchDigestsは置換の前提にする下書きの版 (`DraftPage.ContentDigest()` の値) の候補。
	// HasIfMatchがtrueなら、今の版がいずれかと一致する場合だけ置換する
	IfMatchDigests []string
	// IfMatchAnyは、下書きが存在すれば版を問わず置換することを表す (`If-Match: *`)
	IfMatchAny bool
	// Editsは配列の順に適用する編集
	Edits []model.BodyEdit
}

// Executeはトークンが属するスペースのページについて、トークンの持ち主の下書きの本文に編集を順に適用し、
// 置換後の本文で下書きを更新する。下書きの解決・認可・リビジョンの作成・何も変わらない場合の扱いは、
// UpdateAPIDraftPageUsecaseと同じ。
// 前提の版が指定されていて今の版と一致しなければ、AppErrCodePreconditionFailedのエラーを返す。
// 編集を1件でも適用できなければ、どの編集も適用せずに、最初に失敗した編集についての
// *model.ValidationErrorを返す
func (uc *CreateAPIDraftPageBodyEditUsecase) Execute(ctx context.Context, input CreateAPIDraftPageBodyEditInput) (*UpdateAPIDraftPageOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	// 取得からロックまでの間に下書きが変わっていた場合は、変わった本文に編集を適用し直す
	return retryAPIDraftPageUpdate(func() (*UpdateAPIDraftPageOutput, error) {
		return uc.editOnce(ctx, space, input)
	})
}

// editOnceは下書きを取得して認可・前提の照合を行い、本文に編集を適用して、1つのトランザクションで更新する
func (uc *CreateAPIDraftPageBodyEditUsecase) editOnce(ctx context.Context, space *model.Space, input CreateAPIDraftPageBodyEditInput) (*UpdateAPIDraftPageOutput, error) {
	// 1. データ取得
	repos := uc.apiDraftPageWriteRepos()
	access, err := findAPIDraftPage(ctx, repos.accessRepos(), input.Principal, space, input.PageNumber)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if !access.authorizer.CanSaveDraftPage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 前提の照合は編集の照合より先に行う。古い版を元にした編集は、今の本文に適用できても受け付けない
	if input.HasIfMatch && !apiDraftPageMatches(access.draft, input.IfMatchDigests, input.IfMatchAny) {
		return nil, apiDraftPageModifiedError(ctx)
	}

	// 3. バリデーション・4. ビジネスロジック
	body, failure := model.ApplyBodyEdits(access.draft.Body, input.Edits)
	if failure != nil {
		return nil, bodyEditFailureError(ctx, failure)
	}

	draft, err := saveAPIDraftPageContent(ctx, uc.db, repos, saveAPIDraftPageContentInput{
		principal: input.Principal,
		space:     space,
		access:    access,
		title:     access.draft.Title,
		body:      body,
	})
	if err != nil {
		return nil, err
	}
	return &UpdateAPIDraftPageOutput{DraftPage: draft, Topic: access.topic}, nil
}

func (uc *CreateAPIDraftPageBodyEditUsecase) apiDraftPageWriteRepos() apiDraftPageWriteRepos {
	return apiDraftPageWriteRepos{
		spaceRepo:             uc.spaceRepo,
		pageRepo:              uc.pageRepo,
		pageEditorRepo:        uc.pageEditorRepo,
		draftPageRepo:         uc.draftPageRepo,
		draftPageRevisionRepo: uc.draftPageRevisionRepo,
		topicRepo:             uc.topicRepo,
		topicMemberRepo:       uc.topicMemberRepo,
		attachmentRepo:        uc.attachmentRepo,
	}
}
