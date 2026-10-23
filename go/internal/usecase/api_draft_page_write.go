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
)

const updateAPIDraftPageRetryLimit = 3

// errAPIDraftPageChangedは、照合に使った下書きが、ロックした時点では別の内容に変わっていたか消えていたことを表す。
// 下書きの取得からやり直し、前提の照合で412にするか、未存在として404にする
var errAPIDraftPageChanged = errors.New("更新対象の下書きが変更されました")

// apiDraftPageWriteReposは公開APIで下書きのタイトルと本文を保存するのに必要なリポジトリ群
type apiDraftPageWriteRepos struct {
	spaceRepo             *repository.SpaceRepository
	pageRepo              *repository.PageRepository
	pageEditorRepo        *repository.PageEditorRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	attachmentRepo        *repository.AttachmentRepository
}

func (repos apiDraftPageWriteRepos) accessRepos() apiDraftPageAccessRepos {
	return apiDraftPageAccessRepos{
		pageRepo:        repos.pageRepo,
		draftPageRepo:   repos.draftPageRepo,
		topicRepo:       repos.topicRepo,
		topicMemberRepo: repos.topicMemberRepo,
	}
}

// retryAPIDraftPageUpdateは下書きの更新 (update) を、照合の後に下書きが変わっていた場合と、同時に作られた
// ページと一意制約で競合した場合に、下書きの取得からやり直す。Web画面の作成・下書き保存はスペースをロックせずに
// ページ (Wikiリンク先を含む) を作るため、CreateAPIDraftPageUsecaseと同じく競合に備える
func retryAPIDraftPageUpdate(update func() (*UpdateAPIDraftPageOutput, error)) (*UpdateAPIDraftPageOutput, error) {
	var lastErr error
	for attempt := 0; attempt < updateAPIDraftPageRetryLimit; attempt++ {
		output, err := update()
		if err == nil {
			return output, nil
		}
		if !errors.Is(err, errAPIDraftPageChanged) &&
			!isUniqueViolationOn(err, "index_pages_on_space_id_and_number") &&
			!isUniqueViolationOn(err, "index_pages_on_topic_id_and_title") {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("下書き更新の競合が%d回続きました: %w", updateAPIDraftPageRetryLimit, lastErr)
}

// saveAPIDraftPageContentInputは公開APIで下書きのタイトルと本文を保存する入力パラメータ
type saveAPIDraftPageContentInput struct {
	principal *model.APIPrincipal
	space     *model.Space
	// accessは前提の照合に使った下書きと所属トピック
	access *apiDraftPageAccess
	// titleは新しいタイトル。nilならタイトルの無い下書きにする
	title *string
	body  string
}

// saveAPIDraftPageContentは、Web画面の手動保存と同じく、下書きの更新とリビジョンの作成を1つのトランザクションで行う。
// タイトルと本文が今と同じ場合は下書きを書き換えず (版を変えず)、最新のリビジョンと異なる場合だけリビジョンを作る。
// 照合に使った下書き (input.access.draft) がロックした時点で変わっていれば、errAPIDraftPageChangedを返す
func saveAPIDraftPageContent(
	ctx context.Context,
	db *sql.DB,
	repos apiDraftPageWriteRepos,
	input saveAPIDraftPageContentInput,
) (*model.DraftPage, error) {
	now := time.Now()
	space := input.space
	current := input.access.draft
	title := input.title
	body := input.body
	// タイトルの無い下書きと空のタイトルの下書きは版が異なるため、nilかどうかも比べる
	titleChanged := (title == nil) != (current.Title == nil) || (title != nil && *title != *current.Title)
	changed := titleChanged || body != current.Body

	// トランザクション前: 下書きの保存と同じく、アイキャッチ画像を抽出する
	var featuredImageAttachmentID *model.AttachmentID
	var canCreateLinkedPageIn func(topic *model.Topic) bool
	if changed {
		var err error
		featuredImageAttachmentID, err = extractFeaturedImageAttachmentID(ctx, body, space.ID, repos.attachmentRepo)
		if err != nil {
			return nil, fmt.Errorf("アイキャッチ画像の抽出に失敗しました: %w", err)
		}
		canCreateLinkedPageIn, err = apiLinkedPageTopicFilter(ctx, repos.topicRepo, repos.topicMemberRepo, input.principal)
		if err != nil {
			return nil, err
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	draftPageRepo := repos.draftPageRepo.WithTx(tx)

	var linkedPageIDs []model.PageID
	if changed {
		// Wikiリンク先のページの採番をAPIのページ作成と重ねないよう、CreateAPIDraftPageUsecaseと同じく
		// スペースをロックする
		locked, err := repos.spaceRepo.WithTx(tx).LockByID(ctx, space.ID)
		if err != nil {
			return nil, fmt.Errorf("スペースのロックに失敗しました: %w", err)
		}
		if !locked {
			return nil, &model.AppError{
				Code:    model.AppErrCodeResourceNotFound,
				UserMsg: i18n.T(ctx, "error_not_found_message"),
			}
		}

		// 本文のWikiリンクが指すページを解決する。存在しないリンク先ページは自動作成される。
		// Web画面の保存と同じく、リンク先ページを作ってから下書きの行をロックする順にし、
		// 同じ下書きを保存するWeb画面のトランザクションと循環待ちにならないようにする
		linkedPageIDs, err = resolveLinkedPageIDs(ctx, resolveLinkedPageIDsInput{
			Body:                  body,
			CurrentTopicName:      input.access.topic.Name,
			SpaceID:               space.ID,
			SpaceMemberID:         current.SpaceMemberID,
			CanCreateLinkedPageIn: canCreateLinkedPageIn,
		}, repos.topicRepo.WithTx(tx), repos.pageRepo.WithTx(tx), repos.pageEditorRepo.WithTx(tx))
		if err != nil {
			return nil, fmt.Errorf("リンク先ページの解決に失敗しました: %w", err)
		}
	}

	// 照合から更新までの間にWeb画面の自動保存や別のAPIの更新が割り込まないよう、下書きの行を
	// ロックしてから、照合に使った下書きと同じ内容であることを確かめる
	draft, err := draftPageRepo.FindByIDForUpdate(ctx, current.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きのロックに失敗しました: %w", err)
	}
	if draft == nil || draft.ContentDigest() != current.ContentDigest() {
		return nil, errAPIDraftPageChanged
	}

	if changed {
		draft, err = draftPageRepo.Update(ctx, repository.UpdateDraftPageInput{
			ID:                        current.ID,
			SpaceID:                   space.ID,
			TopicID:                   current.TopicID,
			Title:                     title,
			Body:                      body,
			LinkedPageIDs:             linkedPageIDs,
			FeaturedImageAttachmentID: featuredImageAttachmentID,
			ModifiedAt:                now,
		})
		if err != nil {
			return nil, fmt.Errorf("下書きの更新に失敗しました: %w", err)
		}
	}

	// 手動保存と同じく、最新のリビジョンと同じ内容ならリビジョンを作らない。
	// タイトルの無い下書きのリビジョンは空のタイトルで記録する
	draftPageRevisionRepo := repos.draftPageRevisionRepo.WithTx(tx)
	latest, err := draftPageRevisionRepo.ListByDraftPageID(ctx, current.ID, space.ID, 1)
	if err != nil {
		return nil, fmt.Errorf("最新リビジョンの取得に失敗しました: %w", err)
	}
	revisionTitle := derefOr(title, "")
	if len(latest) == 0 || latest[0].Title != revisionTitle || latest[0].Body != body {
		_, err = draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
			DraftPageID:   current.ID,
			SpaceID:       space.ID,
			SpaceMemberID: current.SpaceMemberID,
			Title:         revisionTitle,
			Body:          body,
		})
		if err != nil {
			return nil, fmt.Errorf("下書きページリビジョンの作成に失敗しました: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	draft.Page = current.Page
	return draft, nil
}
