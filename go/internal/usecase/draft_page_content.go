package usecase

import (
	"context"
	"fmt"
	"time"

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
	draftPage, err := findOrCreateDraftPage(ctx, draftPageRepo, input, now)
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

	// 3. DraftPageを更新。トピックは作成時に書いた値ではなく、ロックして読んだページのトピックにする
	topicID, err := lockTopicIDForDraftPage(ctx, pageRepo, input.PageID, input.SpaceID)
	if err != nil {
		return nil, err
	}
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
) (*model.DraftPage, error) {
	draftPage, err := repo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, err
	}
	if draftPage != nil {
		return draftPage, nil
	}

	draftPage, err = repo.CreateIfNotExists(ctx, repository.CreateDraftPageInput{
		SpaceID:       input.SpaceID,
		PageID:        input.PageID,
		SpaceMemberID: input.SpaceMemberID,
		TopicID:       input.TopicID,
		Title:         input.Title,
		Body:          "",
		LinkedPageIDs: nil,
		ModifiedAt:    now,
	})
	if err != nil {
		return nil, err
	}
	if draftPage != nil {
		return draftPage, nil
	}

	// 競合した下書きは他のトランザクションがコミット済みで、READ COMMITTEDでは次の文から見える
	draftPage, err = repo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, err
	}
	if draftPage == nil {
		return nil, fmt.Errorf("同時に作成された下書きページが見つかりませんでした")
	}

	return draftPage, nil
}
