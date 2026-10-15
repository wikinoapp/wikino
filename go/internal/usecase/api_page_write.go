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

// apiPageWriteReposは公開APIでページのタイトルと本文を書き込むのに必要なリポジトリ群
type apiPageWriteRepos struct {
	spaceRepo             *repository.SpaceRepository
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
}

// writeAPIPageInputは公開APIでページのタイトルと本文を書き込む入力パラメータ
type writeAPIPageInput struct {
	principal   *model.APIPrincipal
	space       *model.Space
	spaceMember *model.SpaceMember
	topic       *model.Topic
	// pageは前提の照合と検証に使ったページ
	page  *model.Page
	title string
	body  string
	// unpublishedConflictingPageIDは、タイトルが重なるため論理削除して置き換える中身の無い未公開のページ
	unpublishedConflictingPageID *model.PageID
	// republishは公開済みのページも今の日時で公開し直すかを表す。
	// falseなら公開済みのページの公開日時を変えず (内容を変える操作で、公開し直す操作ではないため)、
	// 未公開のページだけをWeb画面の公開と同じく今の日時で公開する
	republish bool
	// beforeCommitがnilでなければ、ページを書き込んだ後に同じトランザクションで呼ぶ。
	// エラーを返せば、ページの書き込みごと取り消してそのエラーを返す
	beforeCommit func(tx *sql.Tx) error
}

// writeAPIPageはページの更新・リビジョンと編集者の記録を1つのトランザクションで行い、ページを公開する。
// 手順はPublishPageUsecaseの公開に揃え、下書きには触れない (下書きを扱う呼び出し元はbeforeCommitで扱う)。
// 検証に使ったページ (input.page) がロックした時点で変わっていれば、errAPIPageChangedを返す
func writeAPIPage(ctx context.Context, db *sql.DB, repos apiPageWriteRepos, input writeAPIPageInput) (*model.Page, error) {
	now := time.Now()

	// トランザクション前: PublishPageUsecaseと同じく、アイキャッチ画像を抽出する
	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, input.body, input.space.ID, repos.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の抽出に失敗しました: %w", err)
	}
	canCreateLinkedPageIn, err := apiLinkedPageTopicFilter(ctx, repos.topicRepo, repos.topicMemberRepo, input.principal)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Wikiリンク先のページの採番をAPIのページ作成と重ねないよう、CreateAPIPageUsecaseと同じく
	// スペースをロックする。ページの行より先にロックし、ロックの順序を作成と揃える
	locked, err := repos.spaceRepo.WithTx(tx).LockByID(ctx, input.space.ID)
	if err != nil {
		return nil, fmt.Errorf("スペースのロックに失敗しました: %w", err)
	}
	if !locked {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	pageRepo := repos.pageRepo.WithTx(tx)
	pageRevisionRepo := repos.pageRevisionRepo.WithTx(tx)
	pageEditorRepo := repos.pageEditorRepo.WithTx(tx)
	topicMemberRepo := repos.topicMemberRepo.WithTx(tx)
	pageAttachmentRefRepo := repos.pageAttachmentRefRepo.WithTx(tx)

	// 前提の照合から更新までの間に、Web画面の公開や別のAPIの更新が割り込まないよう、ページの行を
	// ロックしてから、照合に使ったページと同じ内容であることを確かめる。
	// ゴミ箱への移動は版 (ContentDigest) を変えないため、状態も確かめる
	lockedPage, err := pageRepo.FindByIDForUpdate(ctx, input.page.ID, input.space.ID)
	if err != nil {
		return nil, fmt.Errorf("ページのロックに失敗しました: %w", err)
	}
	if !isAPIVisiblePage(lockedPage) || lockedPage.ContentDigest() != input.page.ContentDigest() {
		return nil, errAPIPageChanged
	}

	// 検証後にWeb画面で公開されていた場合は削除せず、検証からやり直す
	if input.unpublishedConflictingPageID != nil {
		discarded, err := pageRepo.DiscardEmptyUnpublishedByID(ctx, *input.unpublishedConflictingPageID, input.space.ID, input.topic.ID, input.title, now)
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
	if input.title != derefOr(lockedPage.Title, "") {
		_, err := pageRepo.Update(ctx, repository.UpdatePageInput{
			ID:                        lockedPage.ID,
			SpaceID:                   input.space.ID,
			TopicID:                   lockedPage.TopicID,
			Title:                     &input.title,
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

	// 本文のWikiリンクが指すページを解決する。存在しないリンク先ページは、トークンで見られるトピックにだけ自動作成される
	linkedPageIDs, err := resolveLinkedPageIDs(ctx, resolveLinkedPageIDsInput{
		Body:                  input.body,
		CurrentTopicName:      input.topic.Name,
		SpaceID:               input.space.ID,
		SpaceMemberID:         input.spaceMember.ID,
		CanCreateLinkedPageIn: canCreateLinkedPageIn,
	}, repos.topicRepo.WithTx(tx), pageRepo, pageEditorRepo)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの解決に失敗しました: %w", err)
	}

	attachmentRefsToAdd, attachmentRefsToRemove, err := calculateAttachmentRefDiff(ctx, input.body, input.page.ID, input.space.ID, repos.attachmentRepo, pageAttachmentRefRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイル参照の差分計算に失敗しました: %w", err)
	}
	if err := applyAttachmentRefChanges(ctx, input.page.ID, input.space.ID, attachmentRefsToAdd, attachmentRefsToRemove, pageAttachmentRefRepo); err != nil {
		return nil, fmt.Errorf("添付ファイル参照の同期に失敗しました: %w", err)
	}

	publishedAt := lockedPage.PublishedAt
	if publishedAt == nil || input.republish {
		publishedAt = &now
	}
	updatedPage, err := pageRepo.Update(ctx, repository.UpdatePageInput{
		ID:                        input.page.ID,
		SpaceID:                   input.space.ID,
		TopicID:                   input.topic.ID,
		Title:                     &input.title,
		Body:                      input.body,
		LinkedPageIDs:             linkedPageIDs,
		ModifiedAt:                now,
		PublishedAt:               publishedAt,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの更新に失敗しました: %w", err)
	}

	_, err = pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
		SpaceID:       input.space.ID,
		SpaceMemberID: input.spaceMember.ID,
		PageID:        input.page.ID,
		Title:         input.title,
		Body:          input.body,
	})
	if err != nil {
		return nil, fmt.Errorf("ページリビジョンの作成に失敗しました: %w", err)
	}

	pageEditor, err := pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            input.space.ID,
		PageID:             input.page.ID,
		SpaceMemberID:      input.spaceMember.ID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の追加に失敗しました: %w", err)
	}

	_, err = pageEditorRepo.UpdateLastPageModifiedAt(ctx, repository.UpdateLastPageModifiedAtInput{
		ID:                 pageEditor.ID,
		SpaceID:            input.space.ID,
		LastPageModifiedAt: now,
	})
	if err != nil {
		return nil, fmt.Errorf("ページ編集者の更新に失敗しました: %w", err)
	}

	err = topicMemberRepo.UpdateLastPageModifiedAt(ctx, input.space.ID, input.topic.ID, input.spaceMember.ID, now)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの更新に失敗しました: %w", err)
	}

	if input.beforeCommit != nil {
		if err := input.beforeCommit(tx); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return updatedPage, nil
}

const updateAPIPageRetryLimit = 3

// errAPIPageChangedは、検証に使ったページが、ロックした時点では別の内容に変わっていたことを表す。
// 検証からやり直し、前提の照合で412にするか、変わった内容で検証し直す
var errAPIPageChanged = errors.New("更新対象のページが変更されました")

// retryAPIPageUpdateはページの更新 (update) を、ページが検証の後に変わっていた場合と、同時に作られた
// ページと一意制約で競合した場合に、ページの取得からやり直す。Web画面の公開・下書き保存はスペースを
// ロックせずにページ (Wikiリンク先を含む) を作るため、CreateAPIPageUsecaseと同じく競合に備える
func retryAPIPageUpdate(update func() (*UpdateAPIPageOutput, error)) (*UpdateAPIPageOutput, error) {
	var lastErr error
	for attempt := 0; attempt < updateAPIPageRetryLimit; attempt++ {
		output, err := update()
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

// findAPIPageToUpdateは公開APIで更新するページと所属トピックを取得し、トークンの持ち主が更新できるかを判定する。
// 取得 (GetAPIPageUsecase) と同じく、ゴミ箱のページと閲覧者が開けない非公開トピックのページは
// 未存在として扱い、ページを開けるが更新できない場合はAppErrCodeForbiddenを返す
func findAPIPageToUpdate(
	ctx context.Context,
	repos apiPageWriteRepos,
	principal *model.APIPrincipal,
	space *model.Space,
	spaceMember *model.SpaceMember,
	pageNumber int32,
) (*model.Page, *model.Topic, error) {
	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

	page, err := repos.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(pageNumber))
	if err != nil {
		return nil, nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if !isAPIVisiblePage(page) {
		return nil, nil, notFound
	}

	topic, err := repos.topicRepo.FindBySpaceAndID(ctx, space.ID, page.TopicID)
	if err != nil {
		return nil, nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, nil, notFound
	}

	topicMember, err := repos.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	authorizer := newAPIAuthorizer(principal, topicMember)
	if !authorizer.CanShowTopic(topic) {
		return nil, nil, notFound
	}
	if !authorizer.CanUpdatePage() {
		return nil, nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return page, topic, nil
}

// saveAPIPageContentInputは公開APIで、取得したページを新しいタイトルと本文で保存する入力パラメータ
type saveAPIPageContentInput struct {
	principal   *model.APIPrincipal
	space       *model.Space
	spaceMember *model.SpaceMember
	topic       *model.Topic
	// pageは認可と前提の照合に使ったページ
	page  *model.Page
	title string
	body  string
}

// saveAPIPageContentは取得したページを新しいタイトルと本文で保存し、未公開のページは公開する。
// 公開済みのページでタイトルと本文が今と同じ場合は、何も書き込まずに今のページを返す
func saveAPIPageContent(
	ctx context.Context,
	db *sql.DB,
	repos apiPageWriteRepos,
	updateValidator *validator.APIPageUpdateValidator,
	input saveAPIPageContentInput,
) (*UpdateAPIPageOutput, error) {
	page := input.page
	if page.IsPublished() && input.title == derefOr(page.Title, "") && input.body == page.Body {
		return &UpdateAPIPageOutput{Page: page, Topic: input.topic}, nil
	}

	// タイトルを変えない場合は検証しない。タイトルの規則ができる前に付けられたタイトルのページでも、
	// 本文だけは更新できるようにするため。
	// タイトルの無い未公開のページは、タイトルを指定しなければ必須の検証で拒否する
	var unpublishedConflictingPageID *model.PageID
	if input.title != derefOr(page.Title, "") || input.title == "" {
		var err error
		unpublishedConflictingPageID, err = updateValidator.Validate(ctx, validator.APIPageUpdateValidatorInput{
			Title:   input.title,
			PageID:  page.ID,
			TopicID: input.topic.ID,
			SpaceID: input.space.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	updatedPage, err := writeAPIPage(ctx, db, repos, writeAPIPageInput{
		principal:                    input.principal,
		space:                        input.space,
		spaceMember:                  input.spaceMember,
		topic:                        input.topic,
		page:                         page,
		title:                        input.title,
		body:                         input.body,
		unpublishedConflictingPageID: unpublishedConflictingPageID,
	})
	if err != nil {
		return nil, err
	}

	return &UpdateAPIPageOutput{Page: updatedPage, Topic: input.topic}, nil
}

// isAPIVisiblePageは、ページが公開APIで取得・更新できる (ゴミ箱に入っていない) ページかを返す
func isAPIVisiblePage(page *model.Page) bool {
	return page != nil && page.TrashedAt == nil
}

// derefOrはpがnilでなければその値を、nilならfallbackを返す
func derefOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
