package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// saveDraftPageContentInputはDraftPageの内容保存に必要な共通パラメータ
type saveDraftPageContentInput struct {
	SpaceID                   model.SpaceID
	PageID                    model.PageID
	SpaceMemberID             model.SpaceMemberID
	TopicID                   model.TopicID
	Title                     *string
	Body                      string
	FeaturedImageAttachmentID *model.AttachmentID
	CurrentTopicName          string
	// BaseTitleとBaseBodyは、下書きが無く作るときに起点として書くタイトルと本文。
	// 呼び出し元がすでに読んでいるページの内容を渡す
	BaseTitle *string
	BaseBody  string
	// ExpectedDraftPageDigestは、保存の前提にした下書きの版 (`DraftPage.ContentDigest()` の値)。
	// 空文字列は「下書きが無い」ことを前提にした保存を表し、nilのときは照合しない
	ExpectedDraftPageDigest *string
}

// DraftPageConflictは、保存の前提にした下書きの版が今の版と食い違ったことを表す。
// `*model.AppError` (AppErrCodeConflict) のInternalに入れて返し、Handlerが今の版を利用者に返すのに使う
type DraftPageConflict struct {
	// CurrentDigestは下書きの今の版。下書きが無いときは空文字列になる
	CurrentDigest string
}

func (e *DraftPageConflict) Error() string {
	return fmt.Sprintf("下書きの版が前提と異なります: current=%q", e.CurrentDigest)
}

// draftPageConflictErrorは、下書きが保存の前提にした版の後に変わっていたことを表すエラーを返す
func draftPageConflictError(ctx context.Context, currentDigest string) error {
	return &model.AppError{
		Code:     model.AppErrCodeConflict,
		UserMsg:  i18n.T(ctx, "error_draft_page_modified"),
		Internal: &DraftPageConflict{CurrentDigest: currentDigest},
	}
}

// saveDraftPageContentはDraftPageのfind_or_create・Wikiリンクの解決・更新を行う共通ロジック
func saveDraftPageContent(
	ctx context.Context,
	input saveDraftPageContentInput,
	now time.Time,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
) (*model.DraftPage, error) {
	// 1. DraftPageをfind_or_createで取得・作成
	draftPage, created, err := findOrCreateDraftPage(ctx, draftPageRepo, input, now)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得・作成に失敗しました: %w", err)
	}

	// 2. 本文のWikiリンクを解決し、リンク先ページIDを集める。下書きは本文HTMLを保存せず、
	// 表示するときにレンダリングするため、ここで必要なのは画面が読む先だけである。
	// 解決はこのトランザクション内で存在しないリンク先ページを自動作成する。
	linkedPageIDs, err := resolveLinkedPageIDs(ctx, resolveLinkedPageIDsInput{
		Body:             input.Body,
		CurrentTopicName: input.CurrentTopicName,
		SpaceID:          input.SpaceID,
		SpaceMemberID:    input.SpaceMemberID,
	}, topicRepo, pageRepo, pageEditorRepo)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの解決に失敗しました: %w", err)
	}

	// 3. 下書きに書くトピックは、作成時に書いた値ではなく、ロックして読んだページのトピックにする
	topicID, err := lockTopicIDForDraftPage(ctx, pageRepo, input.PageID, input.SpaceID)
	if err != nil {
		return nil, err
	}

	// 4. 前提にした版を照合する。リンク先ページの解決とページのロックの後に下書きの行をロックするのは、
	// 同じ下書きを更新するAPIのトランザクションと同じ順にして、循環待ちにならないようにするため。
	// ロックしてから照合するので、照合から更新までの間に別の場所の書き込みが割り込むことはない
	if err := checkDraftPageDigest(ctx, draftPageRepo, input, draftPage, created); err != nil {
		return nil, err
	}

	// 5. DraftPageを更新
	updatedDraftPage, err := draftPageRepo.Update(ctx, repository.UpdateDraftPageInput{
		ID:                        draftPage.ID,
		SpaceID:                   input.SpaceID,
		TopicID:                   topicID,
		Title:                     input.Title,
		Body:                      input.Body,
		LinkedPageIDs:             linkedPageIDs,
		FeaturedImageAttachmentID: input.FeaturedImageAttachmentID,
		ModifiedAt:                now,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きページの更新に失敗しました: %w", err)
	}

	return updatedDraftPage, nil
}

// checkDraftPageDigestは、下書きの行をロックしてから、保存の前提にした版と今の版を照合する。
// 下書きが無い前提の保存では、このトランザクションで下書きを作ったときだけ一致とみなす。
// 下書きがある前提の保存では、このトランザクションで作り直した下書き (前提の下書きが公開や削除で消えていた) は不一致とする
func checkDraftPageDigest(
	ctx context.Context,
	draftPageRepo *repository.DraftPageRepository,
	input saveDraftPageContentInput,
	draftPage *model.DraftPage,
	created bool,
) error {
	if input.ExpectedDraftPageDigest == nil {
		return nil
	}
	expected := *input.ExpectedDraftPageDigest

	locked, err := draftPageRepo.FindByIDForUpdate(ctx, draftPage.ID, input.SpaceID)
	if err != nil {
		return fmt.Errorf("下書きのロックに失敗しました: %w", err)
	}
	if locked == nil {
		return draftPageConflictError(ctx, "")
	}

	if expected == "" {
		if created {
			return nil
		}
		return draftPageConflictError(ctx, locked.ContentDigest())
	}
	if created {
		// 前提にした下書きはもう無い。作った下書きはこの保存のロールバックで消えるため、今は下書きが無い
		return draftPageConflictError(ctx, "")
	}
	if locked.ContentDigest() != expected {
		return draftPageConflictError(ctx, locked.ContentDigest())
	}
	return nil
}

// lockTopicIDForDraftPageは、下書きに書くトピックとしてページのトピックを返す。
// 下書きのトピックはページのトピックに従い、ページの移動 (MovePageUsecase) が下書きごと書き換える。
// トランザクションの外で読んだトピックを書くと、その後にコミットされた移動を古いトピックで上書きするため、
// ページの行を共有ロックして読み、このトランザクションが終わるまで移動を待たせる。
// 公開 (PublishPageUsecase) と同じく、リンク先ページを解決してからページをロックし、下書きを書く順にする
func lockTopicIDForDraftPage(ctx context.Context, pageRepo *repository.PageRepository, pageID model.PageID, spaceID model.SpaceID) (model.TopicID, error) {
	page, err := pageRepo.FindByIDForShare(ctx, pageID, spaceID)
	if err != nil {
		return "", fmt.Errorf("ページのロックに失敗しました: %w", err)
	}
	if page == nil {
		return "", fmt.Errorf("下書きのページが見つかりません: page_id=%s", pageID)
	}
	return page.TopicID, nil
}

// findOrCreateDraftPageはDraftPageを取得するか、存在しなければ作成する。
// 呼び出し元のトランザクション内で一意制約 (space_member_id + page_id) に違反すると、
// PostgreSQLがトランザクションを中断して以降の取得も失敗するため、競合を無視して作成し、
// 同時に作成されて作成できなかった場合は取得し直す。
func findOrCreateDraftPage(
	ctx context.Context,
	repo *repository.DraftPageRepository,
	input saveDraftPageContentInput,
	now time.Time,
) (draftPage *model.DraftPage, created bool, err error) {
	draftPage, err = repo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, false, err
	}
	if draftPage != nil {
		return draftPage, false, nil
	}

	draftPage, err = repo.CreateIfNotExists(ctx, repository.CreateDraftPageInput{
		SpaceID:       input.SpaceID,
		PageID:        input.PageID,
		SpaceMemberID: input.SpaceMemberID,
		TopicID:       input.TopicID,
		Title:         input.Title,
		Body:          "",
		BaseTitle:     input.BaseTitle,
		BaseBody:      input.BaseBody,
		LinkedPageIDs: nil,
		ModifiedAt:    now,
	})
	if err != nil {
		return nil, false, err
	}
	if draftPage != nil {
		return draftPage, true, nil
	}

	// 競合した下書きは他のトランザクションがコミット済みで、READ COMMITTEDでは次の文から見える
	draftPage, err = repo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, false, err
	}
	if draftPage == nil {
		return nil, false, fmt.Errorf("同時に作成された下書きページが見つかりませんでした")
	}

	return draftPage, false, nil
}
