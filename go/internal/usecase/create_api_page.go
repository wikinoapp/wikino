package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateAPIPageUsecaseは、公開APIでトークンを束縛したスペースにページを作成して公開するユースケース。
// Web画面は空のページを作って編集画面へ送り、公開は別の操作 (PublishPageUsecase) で行うが、
// APIは作成と公開を1つのトランザクションで行う
type CreateAPIPageUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
	createValidator       *validator.PageCreateValidator
}

const createAPIPageRetryLimit = 3

// errAPIPageReplacementChangedは、検証で置き換えの対象としたページが、論理削除の時点では
// 同じタイトルの空の未公開ページでなくなっていた (Web画面で公開されたなど) ことを表す
var errAPIPageReplacementChanged = errors.New("置き換え対象のページが変更されました")

// NewCreateAPIPageUsecaseはCreateAPIPageUsecaseを生成する
func NewCreateAPIPageUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
	createValidator *validator.PageCreateValidator,
) *CreateAPIPageUsecase {
	return &CreateAPIPageUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		pageRepo:              pageRepo,
		pageRevisionRepo:      pageRevisionRepo,
		pageEditorRepo:        pageEditorRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
		createValidator:       createValidator,
	}
}

// CreateAPIPageInputはページの作成に必要な入力パラメータ
type CreateAPIPageInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
	TopicNumber     int32
	Title           string
	Body            string
}

// CreateAPIPageOutputはページの作成結果
type CreateAPIPageOutput struct {
	Page *model.Page
	// Topicはページが属するトピック
	Topic *model.Topic
}

// Executeはトークンを束縛したスペースのトピックにページを作成し、公開する。
// 存在しないトピックと閲覧者が開けない非公開トピックは、存在を秘匿するため区別せず、
// リクエスト本文の `topic_number` の問題として扱う。
// トピックを開けるがページを作成できない場合は、メンバー権限の不足として未存在を返す
func (uc *CreateAPIPageUsecase) Execute(ctx context.Context, input CreateAPIPageInput) (*CreateAPIPageOutput, error) {
	// 1. データ取得
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

	topic, err := uc.topicRepo.FindBySpaceAndNumber(ctx, space.ID, input.TopicNumber)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}

	var topicMember *model.TopicMember
	if topic != nil {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	// 2. 認可チェック
	authorizer := newAPIAuthorizer(input.Principal, topicMember)
	if topic == nil || !authorizer.CanShowTopic(topic) {
		ve := model.NewValidationError()
		ve.AddField("topic_number", i18n.T(ctx, "api_error_topic_not_found"))
		return nil, ve
	}
	if !authorizer.CanCreatePage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 3. バリデーションと4. ビジネスロジック
	// 一意制約の競合からやり直すときにタイトルの重複を確かめ直すため、バリデーションは作成の
	// 試行ごとに行う (createPage)
	page, err := uc.createPage(ctx, space, spaceMember, topic, input)
	if err != nil {
		return nil, err
	}

	return &CreateAPIPageOutput{Page: page, Topic: topic}, nil
}

// createPageはタイトルを検証してからページを作成し、同時に作られたページと一意制約で競合したときは
// 検証からやり直す。APIの作成どうしはスペースのロックで直列化されるが、Web画面の作成・下書き保存は
// ロックを取らずにページ (Wikiリンク先を含む) を作るため、それとの競合に備える。
// 番号の競合は番号を取り直せば済み、Wikiリンク先のタイトルの競合は既存のページとして見つかる。
// 作成するページ自身のタイトルの競合は、検証し直して重複 (422) か未公開ページの置き換えになる。
// 置き換える未公開ページが検証後にWeb画面で公開された場合も、検証し直して重複 (422) になる
func (uc *CreateAPIPageUsecase) createPage(
	ctx context.Context,
	space *model.Space,
	spaceMember *model.SpaceMember,
	topic *model.Topic,
	input CreateAPIPageInput,
) (*model.Page, error) {
	var lastErr error
	for attempt := 0; attempt < createAPIPageRetryLimit; attempt++ {
		unpublishedConflictingPageID, err := uc.createValidator.Validate(ctx, validator.PageCreateValidatorInput{
			Title:   input.Title,
			TopicID: topic.ID,
			SpaceID: space.ID,
		})
		if err != nil {
			return nil, err
		}

		page, err := uc.createPageOnce(ctx, space, spaceMember, topic, input, unpublishedConflictingPageID)
		if err == nil {
			return page, nil
		}
		if errors.Is(err, errAPIPageReplacementChanged) {
			lastErr = err
			continue
		}
		if !isUniqueViolationOn(err, "index_pages_on_space_id_and_number") &&
			!isUniqueViolationOn(err, "index_pages_on_topic_id_and_title") {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("ページ作成の競合が%d回続きました: %w", createAPIPageRetryLimit, lastErr)
}

// createPageOnceはページの作成・公開・リビジョンと編集者の記録を1つのトランザクションで行う。
// 手順はPublishPageUsecaseの公開に揃え、下書きには触れない (作成したばかりのページに下書きは無い)
func (uc *CreateAPIPageUsecase) createPageOnce(
	ctx context.Context,
	space *model.Space,
	spaceMember *model.SpaceMember,
	topic *model.Topic,
	input CreateAPIPageInput,
	unpublishedConflictingPageID *model.PageID,
) (*model.Page, error) {
	now := time.Now()

	// トランザクション前: PublishPageUsecaseと同じく、アイキャッチ画像を抽出する
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, input.Body, space.ID, uc.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の抽出に失敗しました: %w", err)
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 次の番号を読む前にスペースをロックする。pages(space_id, number) は一意であり、ロック
	// しなければ同時に作られたページ (Wikiリンクで自動作成されるリンク先を含む) が同じ番号を取り、
	// 後からINSERTした側が失敗する
	locked, err := uc.spaceRepo.WithTx(tx).LockByID(ctx, space.ID)
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
	pageRevisionRepo := uc.pageRevisionRepo.WithTx(tx)
	pageEditorRepo := uc.pageEditorRepo.WithTx(tx)
	topicMemberRepo := uc.topicMemberRepo.WithTx(tx)
	pageAttachmentRefRepo := uc.pageAttachmentRefRepo.WithTx(tx)

	// 検証後にWeb画面で公開されていた場合は削除せず、検証からやり直す。
	if unpublishedConflictingPageID != nil {
		discarded, err := pageRepo.DiscardEmptyUnpublishedByID(ctx, *unpublishedConflictingPageID, space.ID, topic.ID, input.Title, now)
		if err != nil {
			return nil, fmt.Errorf("競合する未公開ページの論理削除に失敗しました: %w", err)
		}
		if !discarded {
			return nil, errAPIPageReplacementChanged
		}
	}

	// Wikiリンクを解決する前に、タイトルを持つ未公開のページとして作る。
	// 本文がこのページ自身へのWikiリンクを含むとき、リンクの解決が同じタイトルのページを
	// 別に自動作成しないようにするため (Wikiリンクの自動作成と同じ形のページになる)
	nextNumber, err := pageRepo.NextPageNumber(ctx, space.ID)
	if err != nil {
		return nil, fmt.Errorf("次のページ番号の取得に失敗しました: %w", err)
	}
	page, err := pageRepo.CreateLinkedPage(ctx, repository.CreateLinkedPageInput{
		SpaceID: space.ID,
		TopicID: topic.ID,
		Number:  nextNumber,
		Title:   input.Title,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの作成に失敗しました: %w", err)
	}

	// 本文のWikiリンクが指すページを解決する。存在しないリンク先ページは自動作成される
	linkedPageIDs, err := resolveLinkedPageIDs(ctx, resolveLinkedPageIDsInput{
		Body:             input.Body,
		CurrentTopicName: topic.Name,
		SpaceID:          space.ID,
		SpaceMemberID:    spaceMember.ID,
	}, uc.topicRepo.WithTx(tx), pageRepo, pageEditorRepo)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの解決に失敗しました: %w", err)
	}

	// 作成したばかりのページには添付ファイル参照が無いため、差分は追加だけになる
	attachmentRefsToAdd, _, err := calculateAttachmentRefDiff(ctx, input.Body, page.ID, space.ID, uc.attachmentRepo, pageAttachmentRefRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイル参照の差分計算に失敗しました: %w", err)
	}
	if err := applyAttachmentRefChanges(ctx, page.ID, space.ID, attachmentRefsToAdd, nil, pageAttachmentRefRepo); err != nil {
		return nil, fmt.Errorf("添付ファイル参照の同期に失敗しました: %w", err)
	}

	title := input.Title
	publishedPage, err := pageRepo.Update(ctx, repository.UpdatePageInput{
		ID:                        page.ID,
		SpaceID:                   space.ID,
		TopicID:                   topic.ID,
		Title:                     &title,
		Body:                      input.Body,
		LinkedPageIDs:             linkedPageIDs,
		ModifiedAt:                now,
		PublishedAt:               &now,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの公開に失敗しました: %w", err)
	}

	_, err = pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
		SpaceID:       space.ID,
		SpaceMemberID: spaceMember.ID,
		PageID:        page.ID,
		Title:         input.Title,
		Body:          input.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("ページリビジョンの作成に失敗しました: %w", err)
	}

	_, err = pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            space.ID,
		PageID:             page.ID,
		SpaceMemberID:      spaceMember.ID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の追加に失敗しました: %w", err)
	}

	err = topicMemberRepo.UpdateLastPageModifiedAt(ctx, space.ID, topic.ID, spaceMember.ID, now)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの更新に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return publishedPage, nil
}
