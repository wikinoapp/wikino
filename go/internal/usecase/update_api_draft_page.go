package usecase

import (
	"context"
	"database/sql"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// UpdateAPIDraftPageUsecaseは、公開APIでページについてトークンの持ち主の下書きを更新するユースケース。
// Web画面の手動保存 (ManualSaveDraftPageUsecase) と同じく、下書きを書き換えて最新のリビジョンと異なる内容なら
// リビジョンを作る。加えて、更新の前提にした版 (`If-Match`) を今の版と照合してから更新する
type UpdateAPIDraftPageUsecase struct {
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

// NewUpdateAPIDraftPageUsecaseはUpdateAPIDraftPageUsecaseを生成する
func NewUpdateAPIDraftPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
) *UpdateAPIDraftPageUsecase {
	return &UpdateAPIDraftPageUsecase{
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

// UpdateAPIDraftPageInputは下書きの更新に必要な入力パラメータ
type UpdateAPIDraftPageInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	// IfMatchDigestsは更新の前提にする下書きの版 (`DraftPage.ContentDigest()` の値) の候補。
	// 今の版がいずれかと一致する場合だけ更新する
	IfMatchDigests []string
	// IfMatchAnyは、下書きが存在すれば版を問わず更新することを表す (`If-Match: *`)
	IfMatchAny bool
	// UpdateTitleはタイトルを変えるかを表す。falseならTitleを使わずタイトルを変えない
	UpdateTitle bool
	// Titleは新しいタイトル。UpdateTitleがtrueでnilなら、タイトルを外す
	Title *string
	// Bodyは新しい本文。nilなら本文を変えない
	Body *string
}

// UpdateAPIDraftPageOutputは下書きの更新結果
type UpdateAPIDraftPageOutput struct {
	// DraftPageは更新した下書き。Pageに下書きのページを持つ
	DraftPage *model.DraftPage
	// Topicは下書きが属するトピック
	Topic *model.Topic
}

// Executeはトークンが属するスペースのページについて、トークンの持ち主の下書きのタイトルと本文を更新する。
// 取得 (GetAPIDraftPageUsecase) と同じく、他のメンバーの下書き・ゴミ箱のページの下書き・閲覧者が開けない
// 非公開トピックの下書きは未存在として扱い、下書きを保存できない場合も存在を秘匿して未存在を返す。
// 今の版が前提の版と一致しなければ、AppErrCodePreconditionFailedのエラーを返す。
// タイトルと本文が今と同じ場合は下書きを書き換えず (版を変えず)、最新のリビジョンと異なる場合だけリビジョンを作る
func (uc *UpdateAPIDraftPageUsecase) Execute(ctx context.Context, input UpdateAPIDraftPageInput) (*UpdateAPIDraftPageOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}

	// 照合の後に下書きが変わっていた場合と、同時に作られたページと一意制約で競合した場合は、
	// 下書きの取得からやり直す
	return retryAPIDraftPageUpdate(func() (*UpdateAPIDraftPageOutput, error) {
		return uc.updateOnce(ctx, space, input)
	})
}

// updateOnceは下書きを取得して認可・前提の照合を行い、1つのトランザクションで更新する
func (uc *UpdateAPIDraftPageUsecase) updateOnce(ctx context.Context, space *model.Space, input UpdateAPIDraftPageInput) (*UpdateAPIDraftPageOutput, error) {
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

	// 前提の照合は書き込みより先に行う。古い版を元にした更新は受け付けない
	if !apiDraftPageMatches(access.draft, input.IfMatchDigests, input.IfMatchAny) {
		return nil, apiDraftPageModifiedError(ctx)
	}

	// 3. ビジネスロジックと4. 永続化
	title := access.draft.Title
	if input.UpdateTitle {
		title = input.Title
	}
	draft, err := saveAPIDraftPageContent(ctx, uc.db, repos, saveAPIDraftPageContentInput{
		principal: input.Principal,
		space:     space,
		access:    access,
		title:     title,
		body:      derefOr(input.Body, access.draft.Body),
	})
	if err != nil {
		return nil, err
	}
	return &UpdateAPIDraftPageOutput{DraftPage: draft, Topic: access.topic}, nil
}

func (uc *UpdateAPIDraftPageUsecase) apiDraftPageWriteRepos() apiDraftPageWriteRepos {
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
