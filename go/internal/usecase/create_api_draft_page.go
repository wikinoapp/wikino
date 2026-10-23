package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateAPIDraftPageUsecaseは、公開APIでトークンの持ち主の下書きを作成するユースケース。
// 既存のページの下書きと、新しいページ (Web画面の作成と同じく空の未公開のページ) の下書きを作る。
// 作成はWeb画面の手動保存と同じく、下書きの編集履歴 (リビジョン) を伴う
type CreateAPIDraftPageUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	pageRepo              *repository.PageRepository
	pageEditorRepo        *repository.PageEditorRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
	createValidator       *validator.APIDraftPageCreateValidator
}

const createAPIDraftPageRetryLimit = 3

// NewCreateAPIDraftPageUsecaseはCreateAPIDraftPageUsecaseを生成する
func NewCreateAPIDraftPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	createValidator *validator.APIDraftPageCreateValidator,
) *CreateAPIDraftPageUsecase {
	return &CreateAPIDraftPageUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		pageRepo:              pageRepo,
		pageEditorRepo:        pageEditorRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
		createValidator:       createValidator,
	}
}

// CreateAPIDraftPageInputは下書きの作成に必要な入力パラメータ。
// PageNumber (既存のページ) とTopicNumber (新しいページ) のどちらか一方を指定する。
// TitleとBodyは省略できる (nil)
type CreateAPIDraftPageInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      *int32
	TopicNumber     *int32
	Title           *string
	Body            *string
}

// CreateAPIDraftPageOutputは下書きの作成結果
type CreateAPIDraftPageOutput struct {
	// DraftPageは作成した下書き。Pageに下書きのページを持つ
	DraftPage *model.DraftPage
	// Topicは下書きが属するトピック
	Topic *model.Topic
}

// createAPIDraftPageTargetは下書きを作る先として解決したデータ。
// pageは既存のページの場合だけ持ち、新しいページの場合は永続化の中で作る
type createAPIDraftPageTarget struct {
	space       *model.Space
	spaceMember *model.SpaceMember
	topic       *model.Topic
	page        *model.Page
	// canCreateLinkedPageInはWikiリンク先のページを作れるトピックかを返す (トークンで見られるトピックだけ)
	canCreateLinkedPageIn func(topic *model.Topic) bool
}

// Executeはトークンが属するスペースに、トークンの持ち主の下書きを作成する。
// 存在しない・ゴミ箱に入っている・見られないページやトピックは、存在を秘匿するため区別せず、
// リクエスト本文の `page_number` / `topic_number` の問題として扱う。
// ページやトピックを見られるが下書きを作成できない場合は、メンバー権限の不足として未存在を返す。
// 既存のページにトークンの持ち主の下書きがすでにあれば、上書きせずにAppErrCodeAlreadyExistsを返す
func (uc *CreateAPIDraftPageUsecase) Execute(ctx context.Context, input CreateAPIDraftPageInput) (*CreateAPIDraftPageOutput, error) {
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

	// どちらを指定したかで取得するデータが変わるため、作る先の指定はデータ取得より前に検証する
	if err := uc.createValidator.Validate(ctx, validator.APIDraftPageCreateValidatorInput{
		PageNumber:  input.PageNumber,
		TopicNumber: input.TopicNumber,
	}); err != nil {
		return nil, err
	}

	// 1. データ取得と2. 認可チェック
	target, err := uc.resolveTarget(ctx, space, spaceMember, input)
	if err != nil {
		return nil, err
	}
	target.canCreateLinkedPageIn, err = apiLinkedPageTopicFilter(ctx, uc.topicRepo, uc.topicMemberRepo, input.Principal)
	if err != nil {
		return nil, err
	}

	// 3. 状態のチェック: トランザクションの前に既存の下書きを見て、多くの場合の409を早く返す。
	// 同時に作られた下書きは、永続化の中の作成で検出する
	if target.page != nil {
		existing, err := uc.draftPageRepo.FindByPageAndMember(ctx, target.page.ID, spaceMember.ID, space.ID)
		if err != nil {
			return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
		}
		if existing != nil {
			return nil, draftPageAlreadyExistsError(ctx)
		}
	}

	// 4. ビジネスロジックと5. 永続化
	title, body := draftPageContent(target.page, input.Title, input.Body)
	draft, err := uc.createDraftPage(ctx, target, title, body)
	if err != nil {
		return nil, err
	}
	return &CreateAPIDraftPageOutput{DraftPage: draft, Topic: target.topic}, nil
}

// resolveTargetは下書きを作る先のページかトピックを取得し、下書きの作成を認可する。
// 新しいページの下書きで作るのは本文の無い未公開のページで、内容が見えるようになるのは
// page:writeを持つ人が公開したときだけのため、既存のページと同じく下書きの保存の権限だけを求める
func (uc *CreateAPIDraftPageUsecase) resolveTarget(
	ctx context.Context,
	space *model.Space,
	spaceMember *model.SpaceMember,
	input CreateAPIDraftPageInput,
) (*createAPIDraftPageTarget, error) {
	target := &createAPIDraftPageTarget{space: space, spaceMember: spaceMember}
	field, message := "topic_number", "api_error_topic_not_found"

	if input.PageNumber != nil {
		field, message = "page_number", "api_error_page_not_found"
		page, err := uc.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(*input.PageNumber))
		if err != nil {
			return nil, fmt.Errorf("ページの取得に失敗: %w", err)
		}
		if isAPIVisiblePage(page) {
			target.page = page
			target.topic, err = uc.topicRepo.FindBySpaceAndID(ctx, space.ID, page.TopicID)
			if err != nil {
				return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
			}
		}
	} else {
		topic, err := uc.topicRepo.FindBySpaceAndNumber(ctx, space.ID, *input.TopicNumber)
		if err != nil {
			return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
		}
		target.topic = topic
	}

	var topicMember *model.TopicMember
	if target.topic != nil {
		var err error
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, target.topic.ID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	authorizer := newAPIAuthorizer(input.Principal, topicMember)
	if target.topic == nil || !authorizer.CanShowTopic(target.topic) {
		ve := model.NewValidationError()
		ve.AddField(field, i18n.T(ctx, message))
		return nil, ve
	}
	if !authorizer.CanSaveDraftPage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return target, nil
}

// createDraftPageは下書きを作成し、同時に作られたページと一意制約で競合したときはやり直す。
// APIの作成どうしはスペースのロックで直列化されるが、Web画面の作成・下書き保存はロックを取らずに
// ページ (Wikiリンク先を含む) を作るため、それとの競合に備える (CreateAPIPageUsecaseと同じ)
func (uc *CreateAPIDraftPageUsecase) createDraftPage(
	ctx context.Context,
	target *createAPIDraftPageTarget,
	title *string,
	body string,
) (*model.DraftPage, error) {
	// トランザクション前: 下書きの保存と同じく、アイキャッチ画像を抽出する
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, body, target.space.ID, uc.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の抽出に失敗しました: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < createAPIDraftPageRetryLimit; attempt++ {
		draft, err := uc.createDraftPageOnce(ctx, target, title, body, featuredImageAttachmentID)
		if err == nil {
			return draft, nil
		}
		if !isUniqueViolationOn(err, "index_pages_on_space_id_and_number") &&
			!isUniqueViolationOn(err, "index_pages_on_topic_id_and_title") {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("下書き作成の競合が%d回続きました: %w", createAPIDraftPageRetryLimit, lastErr)
}

// createDraftPageOnceは、新しいページの作成・下書きの作成・Wikiリンク先の解決・リビジョンの作成を
// 1つのトランザクションで行う。新しいページはCreatePageUsecaseと同じく、タイトルの無い空のページを作り、
// 作成者を編集者として登録する
func (uc *CreateAPIDraftPageUsecase) createDraftPageOnce(
	ctx context.Context,
	target *createAPIDraftPageTarget,
	title *string,
	body string,
	featuredImageAttachmentID *model.AttachmentID,
) (*model.DraftPage, error) {
	now := time.Now()
	spaceID := target.space.ID
	spaceMemberID := target.spaceMember.ID

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 次の番号を読む前にスペースをロックする。新しいページとWikiリンクで自動作成するリンク先が
	// 番号を取るため、CreateAPIPageUsecaseと同じく同時に作られたページと番号が重ならないようにする
	locked, err := uc.spaceRepo.WithTx(tx).LockByID(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("スペースのロックに失敗しました: %w", err)
	}
	if !locked {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	pageRepo := uc.pageRepo.WithTx(tx)
	pageEditorRepo := uc.pageEditorRepo.WithTx(tx)
	draftPageRepo := uc.draftPageRepo.WithTx(tx)

	page := target.page
	if page == nil {
		nextNumber, err := pageRepo.NextPageNumber(ctx, spaceID)
		if err != nil {
			return nil, fmt.Errorf("次のページ番号の取得に失敗しました: %w", err)
		}
		page, err = pageRepo.CreateBlankPage(ctx, repository.CreateBlankPageInput{
			SpaceID: spaceID,
			TopicID: target.topic.ID,
			Number:  nextNumber,
		})
		if err != nil {
			return nil, fmt.Errorf("ページの作成に失敗しました: %w", err)
		}
		_, err = pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             page.ID,
			SpaceMemberID:      spaceMemberID,
			LastPageModifiedAt: page.ModifiedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("ページの編集者登録に失敗しました: %w", err)
		}
	}

	// Web保存とロックの取得順を揃え、下書きとリンク先ページの一意制約で循環待ちにならないよう、
	// リンク先ページより先に下書きを作る。競合した下書きは上書きせず409にする
	draft, err := draftPageRepo.CreateIfNotExists(ctx, repository.CreateDraftPageInput{
		SpaceID:                   spaceID,
		PageID:                    page.ID,
		SpaceMemberID:             spaceMemberID,
		TopicID:                   page.TopicID,
		Title:                     title,
		Body:                      body,
		LinkedPageIDs:             nil,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		ModifiedAt:                now,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きの作成に失敗しました: %w", err)
	}
	if draft == nil {
		return nil, draftPageAlreadyExistsError(ctx)
	}

	// 本文のWikiリンクが指すページを解決する。存在しないリンク先ページは、トークンで見られるトピックにだけ自動作成される
	linkedPageIDs, err := resolveLinkedPageIDs(ctx, resolveLinkedPageIDsInput{
		Body:                  body,
		CurrentTopicName:      target.topic.Name,
		SpaceID:               spaceID,
		SpaceMemberID:         spaceMemberID,
		CanCreateLinkedPageIn: target.canCreateLinkedPageIn,
	}, uc.topicRepo.WithTx(tx), pageRepo, pageEditorRepo)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの解決に失敗しました: %w", err)
	}

	topicID, err := lockTopicIDForDraftPage(ctx, pageRepo, page.ID, spaceID)
	if err != nil {
		return nil, err
	}
	draft, err = draftPageRepo.Update(ctx, repository.UpdateDraftPageInput{
		ID:                        draft.ID,
		SpaceID:                   spaceID,
		TopicID:                   topicID,
		Title:                     title,
		Body:                      body,
		LinkedPageIDs:             linkedPageIDs,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		ModifiedAt:                now,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きの更新に失敗しました: %w", err)
	}

	// 手動保存と同じく、タイトルの無い下書きのリビジョンは空のタイトルで記録する
	revisionTitle := ""
	if title != nil {
		revisionTitle = *title
	}
	_, err = uc.draftPageRevisionRepo.WithTx(tx).Create(ctx, repository.CreateDraftPageRevisionInput{
		DraftPageID:   draft.ID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         revisionTitle,
		Body:          body,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きページリビジョンの作成に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	draft.Page = page
	return draft, nil
}

// draftPageContentは作成する下書きのタイトルと本文を返す。
// 省略された内容は、既存のページ (pageがnilでない) では今の内容、新しいページではタイトル無し・空の本文にする
func draftPageContent(page *model.Page, title, body *string) (*string, string) {
	if page != nil {
		if title == nil {
			title = page.Title
		}
		if body == nil {
			body = &page.Body
		}
	}
	if body == nil {
		return title, ""
	}
	return title, *body
}

// draftPageAlreadyExistsErrorは、ページにトークンの持ち主の下書きがすでにあることを表すエラーを返す
func draftPageAlreadyExistsError(ctx context.Context) error {
	return &model.AppError{
		Code:    model.AppErrCodeAlreadyExists,
		UserMsg: i18n.T(ctx, "api_error_draft_page_already_exists"),
	}
}
