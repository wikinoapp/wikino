package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdateAPIPageUsecaseは、公開APIでトークンを束縛したスペースの公開済みのページを更新するユースケース。
// Web画面の公開 (PublishPageUsecase) と違い、更新の前提にした版 (`If-Match`) を今の版と照合してから
// 更新し、トークンの持ち主の下書きには触れない
type UpdateAPIPageUsecase struct {
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

const updateAPIPageRetryLimit = 3

// errAPIPageChangedは、検証に使ったページが、ロックした時点では別の内容に変わっていたことを表す。
// 検証からやり直し、前提の照合で412にするか、変わった内容で検証し直す
var errAPIPageChanged = errors.New("更新対象のページが変更されました")

// NewUpdateAPIPageUsecaseはUpdateAPIPageUsecaseを生成する
func NewUpdateAPIPageUsecase(
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
) *UpdateAPIPageUsecase {
	return &UpdateAPIPageUsecase{
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

// Executeはトークンを束縛したスペースの公開済みのページのタイトルと本文を更新する。
// 取得 (GetAPIPageUsecase) と同じく、未公開・ゴミ箱のページと閲覧者が開けない非公開トピックの
// ページは未存在として扱い、ページを開けるが更新できない場合も存在を秘匿して未存在を返す。
// 今の版が前提の版と一致しなければ、AppErrCodePreconditionFailedのエラーを返す。
// タイトルと本文が今と同じ場合は、何も書き込まずに今のページを返す
func (uc *UpdateAPIPageUsecase) Execute(ctx context.Context, input UpdateAPIPageInput) (*UpdateAPIPageOutput, error) {
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

	// ページが検証の後に変わっていた場合と、同時に作られたページと一意制約で競合した場合は、
	// ページの取得からやり直す。Web画面の公開・下書き保存はスペースをロックせずにページ
	// (Wikiリンク先を含む) を作るため、CreateAPIPageUsecaseと同じく競合に備える
	var lastErr error
	for attempt := 0; attempt < updateAPIPageRetryLimit; attempt++ {
		output, err := uc.updateOnce(ctx, space, spaceMember, input)
		if err == nil {
			return output, nil
		}
		if !errors.Is(err, errAPIPageChanged) &&
			!errors.Is(err, errAPIPageReplacementChanged) &&
			!isUniqueViolationOn(err, "index_pages_on_space_id_and_number") &&
			!isUniqueViolationOn(err, "index_pages_on_topic_id_and_title") {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("ページ更新の競合が%d回続きました: %w", updateAPIPageRetryLimit, lastErr)
}

// updateOnceはページを取得して認可・前提の照合・検証を行い、1つのトランザクションで更新する
func (uc *UpdateAPIPageUsecase) updateOnce(
	ctx context.Context,
	space *model.Space,
	spaceMember *model.SpaceMember,
	input UpdateAPIPageInput,
) (*UpdateAPIPageOutput, error) {
	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

	// 1. データ取得
	page, err := uc.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(input.PageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if !isAPIVisiblePage(page) {
		return nil, notFound
	}

	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, page.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, notFound
	}

	topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	// 2. 認可チェック
	authorizer := newAPIAuthorizer(input.Principal, topicMember)
	if !authorizer.CanShowTopic(topic) {
		return nil, notFound
	}
	if !authorizer.CanUpdatePage() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	// 前提の照合は内容の検証より先に行う。古い版を元にした更新は、内容が正しくても受け付けない
	if !input.matches(page) {
		return nil, &model.AppError{
			Code:    model.AppErrCodePreconditionFailed,
			UserMsg: i18n.T(ctx, "api_error_page_modified"),
		}
	}

	title := derefOr(input.Title, derefOr(page.Title, ""))
	body := derefOr(input.Body, page.Body)
	if title == derefOr(page.Title, "") && body == page.Body {
		return &UpdateAPIPageOutput{Page: page, Topic: topic}, nil
	}

	// 3. バリデーション
	// タイトルを変えない場合は検証しない。タイトルの規則ができる前に付けられたタイトルのページでも、
	// 本文だけは更新できるようにするため
	var unpublishedConflictingPageID *model.PageID
	if title != derefOr(page.Title, "") {
		unpublishedConflictingPageID, err = uc.updateValidator.Validate(ctx, validator.APIPageUpdateValidatorInput{
			Title:   title,
			PageID:  page.ID,
			TopicID: topic.ID,
			SpaceID: space.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	// 4. ビジネスロジック
	updatedPage, err := uc.updatePage(ctx, space, spaceMember, topic, page, title, body, unpublishedConflictingPageID)
	if err != nil {
		return nil, err
	}

	return &UpdateAPIPageOutput{Page: updatedPage, Topic: topic}, nil
}

// updatePageはページの更新・リビジョンと編集者の記録を1つのトランザクションで行う。
// 手順はPublishPageUsecaseの公開に揃え、下書きには触れない。
// 検証に使ったページ (page) がロックした時点で変わっていれば、errAPIPageChangedを返す
func (uc *UpdateAPIPageUsecase) updatePage(
	ctx context.Context,
	space *model.Space,
	spaceMember *model.SpaceMember,
	topic *model.Topic,
	page *model.Page,
	title string,
	body string,
	unpublishedConflictingPageID *model.PageID,
) (*model.Page, error) {
	now := time.Now()

	// トランザクション前: PublishPageUsecaseと同じく、アイキャッチ画像を抽出する
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, body, space.ID, uc.attachmentRepo)
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

	// Wikiリンク先のページの採番をAPIのページ作成と重ねないよう、CreateAPIPageUsecaseと同じく
	// スペースをロックする。ページの行より先にロックし、ロックの順序を作成と揃える
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

	// 前提の照合から更新までの間に、Web画面の公開や別のAPIの更新が割り込まないよう、ページの行を
	// ロックしてから、照合に使ったページと同じ内容であることを確かめる。
	// ゴミ箱への移動は版 (ContentDigest) を変えないため、状態も確かめる
	lockedPage, err := pageRepo.FindByIDForUpdate(ctx, page.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("ページのロックに失敗しました: %w", err)
	}
	if !isAPIVisiblePage(lockedPage) || lockedPage.ContentDigest() != page.ContentDigest() {
		return nil, errAPIPageChanged
	}

	// 検証後にWeb画面で公開されていた場合は削除せず、検証からやり直す
	if unpublishedConflictingPageID != nil {
		discarded, err := pageRepo.DiscardEmptyUnpublishedByID(ctx, *unpublishedConflictingPageID, space.ID, topic.ID, title, now)
		if err != nil {
			return nil, fmt.Errorf("競合する未公開ページの論理削除に失敗しました: %w", err)
		}
		if !discarded {
			return nil, errAPIPageReplacementChanged
		}
	}

	// タイトルを変える場合は、Wikiリンクを解決する前にタイトルだけを先に書き換える。
	// 本文が新しいタイトルで自身を指すWikiリンクを含むとき、リンクの解決が同じタイトルのページを
	// 別に自動作成しないようにするため (CreateAPIPageUsecaseでタイトルを持つページを先に作るのと同じ理由)
	if title != derefOr(lockedPage.Title, "") {
		_, err := pageRepo.Update(ctx, repository.UpdatePageInput{
			ID:                        lockedPage.ID,
			SpaceID:                   space.ID,
			TopicID:                   lockedPage.TopicID,
			Title:                     &title,
			Body:                      lockedPage.Body,
			LinkedPageIDs:             lockedPage.LinkedPageIDs,
			ModifiedAt:                lockedPage.ModifiedAt,
			PublishedAt:               lockedPage.PublishedAt,
			FeaturedImageAttachmentID: lockedPage.FeaturedImageAttachmentID,
		})
		if err != nil {
			return nil, fmt.Errorf("ページのタイトルの更新に失敗しました: %w", err)
		}
	}

	// 本文のWikiリンクが指すページを解決する。存在しないリンク先ページは自動作成される
	linkedPageIDs, err := resolveLinkedPageIDs(ctx, resolveLinkedPageIDsInput{
		Body:             body,
		CurrentTopicName: topic.Name,
		SpaceID:          space.ID,
		SpaceMemberID:    spaceMember.ID,
	}, uc.topicRepo.WithTx(tx), pageRepo, pageEditorRepo)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの解決に失敗しました: %w", err)
	}

	attachmentRefsToAdd, attachmentRefsToRemove, err := calculateAttachmentRefDiff(ctx, body, page.ID, space.ID, uc.attachmentRepo, pageAttachmentRefRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイル参照の差分計算に失敗しました: %w", err)
	}
	if err := applyAttachmentRefChanges(ctx, page.ID, space.ID, attachmentRefsToAdd, attachmentRefsToRemove, pageAttachmentRefRepo); err != nil {
		return nil, fmt.Errorf("添付ファイル参照の同期に失敗しました: %w", err)
	}

	// 公開日時は変えない。APIの更新は公開済みのページの内容を変える操作で、公開し直す操作ではないため
	updatedPage, err := pageRepo.Update(ctx, repository.UpdatePageInput{
		ID:                        page.ID,
		SpaceID:                   space.ID,
		TopicID:                   topic.ID,
		Title:                     &title,
		Body:                      body,
		LinkedPageIDs:             linkedPageIDs,
		ModifiedAt:                now,
		PublishedAt:               lockedPage.PublishedAt,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの更新に失敗しました: %w", err)
	}

	_, err = pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
		SpaceID:       space.ID,
		SpaceMemberID: spaceMember.ID,
		PageID:        page.ID,
		Title:         title,
		Body:          body,
	})
	if err != nil {
		return nil, fmt.Errorf("ページリビジョンの作成に失敗しました: %w", err)
	}

	pageEditor, err := pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            space.ID,
		PageID:             page.ID,
		SpaceMemberID:      spaceMember.ID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の追加に失敗しました: %w", err)
	}

	_, err = pageEditorRepo.UpdateLastPageModifiedAt(ctx, repository.UpdateLastPageModifiedAtInput{
		ID:                 pageEditor.ID,
		SpaceID:            space.ID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の更新に失敗しました: %w", err)
	}

	err = topicMemberRepo.UpdateLastPageModifiedAt(ctx, space.ID, topic.ID, spaceMember.ID, now)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの更新に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return updatedPage, nil
}

// matchesはページの今の版が、更新の前提にした版と一致するかを返す
func (input UpdateAPIPageInput) matches(page *model.Page) bool {
	return input.IfMatchAny || slices.Contains(input.IfMatchDigests, page.ContentDigest())
}

// isAPIVisiblePageは、ページが公開APIで扱える (公開済みでゴミ箱に入っていない) ページかを返す
func isAPIVisiblePage(page *model.Page) bool {
	return page != nil && page.PublishedAt != nil && page.TrashedAt == nil
}

// derefOrはpがnilでなければその値を、nilならfallbackを返す
func derefOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
