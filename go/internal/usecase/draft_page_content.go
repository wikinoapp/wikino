package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// saveDraftPageContentInput はDraftPageの内容保存に必要な共通パラメータ
type saveDraftPageContentInput struct {
	SpaceID                   model.SpaceID
	PageID                    model.PageID
	SpaceMemberID             model.SpaceMemberID
	TopicID                   model.TopicID
	Title                     *string
	Body                      string
	FeaturedImageAttachmentID *model.AttachmentID
	SpaceIdentifier           model.SpaceIdentifier
	CurrentTopicName          string
}

// saveDraftPageContentOutput はDraftPageの内容保存の結果
type saveDraftPageContentOutput struct {
	DraftPage *model.DraftPage
	BodyHTML  string
}

// saveDraftPageContent はDraftPageのfind_or_create・レンダリング・更新を行う共通ロジック
func saveDraftPageContent(
	ctx context.Context,
	input saveDraftPageContentInput,
	now time.Time,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
	topicRepo *repository.TopicRepository,
	attachmentRepo *repository.AttachmentRepository,
) (*saveDraftPageContentOutput, error) {
	// 1. DraftPageをfind_or_createで取得・作成
	draftPage, err := findOrCreateDraftPage(ctx, draftPageRepo, input, now)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得・作成に失敗しました: %w", err)
	}

	// 2. Render the body HTML through markup.RenderHTML, the same unified path used by the
	// preview and page detail screens. The resolver auto-creates missing linked pages
	// within this transaction and records their IDs.
	//
	// [Ja] プレビュー・ページ詳細画面と同じ統合経路 markup.RenderHTML で本文 HTML を
	// レンダリングする。resolver がこのトランザクション内で存在しないリンク先ページを自動作成し、
	// その ID を記録する。
	resolver := &linkCreatingPageLocationResolver{
		spaceMemberID:  input.SpaceMemberID,
		topicRepo:      topicRepo,
		pageRepo:       pageRepo,
		pageEditorRepo: pageEditorRepo,
	}
	bodyHTML, err := markup.RenderHTML(ctx, input.Body, input.CurrentTopicName, input.SpaceID, input.SpaceIdentifier, resolver, attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("本文のレンダリングに失敗しました: %w", err)
	}
	linkedPageIDs := resolver.linkedPageIDs

	// 3. DraftPageを更新
	updatedDraftPage, err := draftPageRepo.Update(ctx, repository.UpdateDraftPageInput{
		ID:                        draftPage.ID,
		SpaceID:                   input.SpaceID,
		TopicID:                   input.TopicID,
		Title:                     input.Title,
		Body:                      input.Body,
		BodyHTML:                  bodyHTML,
		LinkedPageIDs:             linkedPageIDs,
		FeaturedImageAttachmentID: input.FeaturedImageAttachmentID,
		ModifiedAt:                now,
	})
	if err != nil {
		return nil, fmt.Errorf("下書きページの更新に失敗しました: %w", err)
	}

	return &saveDraftPageContentOutput{
		DraftPage: updatedDraftPage,
		BodyHTML:  bodyHTML,
	}, nil
}

// findOrCreateDraftPage はDraftPageを取得するか、存在しなければ作成する。
// ユニーク制約（space_member_id + page_id）違反時はリトライする。
func findOrCreateDraftPage(
	ctx context.Context,
	repo *repository.DraftPageRepository,
	input saveDraftPageContentInput,
	now time.Time,
) (*model.DraftPage, error) {
	for i := 0; i < findOrCreateRetryLimit; i++ {
		draftPage, err := repo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
		if err != nil {
			return nil, err
		}
		if draftPage != nil {
			return draftPage, nil
		}

		draftPage, err = repo.Create(ctx, repository.CreateDraftPageInput{
			SpaceID:       input.SpaceID,
			PageID:        input.PageID,
			SpaceMemberID: input.SpaceMemberID,
			TopicID:       input.TopicID,
			Title:         input.Title,
			Body:          "",
			BodyHTML:      "",
			LinkedPageIDs: nil,
			ModifiedAt:    now,
		})
		if err != nil {
			if isUniqueViolation(err) {
				slog.WarnContext(ctx, "DraftPageのユニーク制約違反によりリトライ", "attempt", i+1)
				continue
			}
			return nil, err
		}

		return draftPage, nil
	}

	return nil, fmt.Errorf("DraftPageの取得・作成が%d回のリトライ後も失敗しました", findOrCreateRetryLimit)
}
