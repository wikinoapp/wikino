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
	BodyHTML                  string
	FeaturedImageAttachmentID *model.AttachmentID
	WikilinkKeys              []markup.WikilinkKey
	TopicMap                  map[string]*model.Topic
	SpaceIdentifier           model.SpaceIdentifier
	CurrentTopicName          string
}

// saveDraftPageContentOutput はDraftPageの内容保存の結果
type saveDraftPageContentOutput struct {
	DraftPage *model.DraftPage
	BodyHTML  string
}

// saveDraftPageContent はDraftPageのfind_or_create・リンク解決・更新を行う共通ロジック
func saveDraftPageContent(
	ctx context.Context,
	input saveDraftPageContentInput,
	now time.Time,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
	pageEditorRepo *repository.PageEditorRepository,
) (*saveDraftPageContentOutput, error) {
	// 1. DraftPageをfind_or_createで取得・作成
	draftPage, err := findOrCreateDraftPage(ctx, draftPageRepo, input, now)
	if err != nil {
		return nil, fmt.Errorf("下書きページの取得・作成に失敗しました: %w", err)
	}

	// 2. Wikiリンクのリンク先ページ自動作成（事前にスキャン・トピック検索済みのデータを使用）
	linkedPageIDs, pageLocations, err := resolveAndCreateLinkedPages(
		ctx, input.WikilinkKeys, input.TopicMap, input.SpaceID, input.SpaceMemberID, pageRepo, pageEditorRepo,
	)
	if err != nil {
		return nil, fmt.Errorf("wikiリンクの解析に失敗しました: %w", err)
	}

	// 3. bodyHTML内のWikiリンクを<a>タグに変換
	bodyHTML := input.BodyHTML
	if len(pageLocations) > 0 {
		bodyHTML = markup.ReplaceWikilinks(bodyHTML, input.CurrentTopicName, input.SpaceIdentifier, pageLocations)
	}

	// 4. DraftPageを更新
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

// draftPageSaveData は下書き保存の事前計算データ
type draftPageSaveData struct {
	bodyHTML                  string
	featuredImageAttachmentID *model.AttachmentID
	wikilinkKeys              []markup.WikilinkKey
	topicMap                  map[string]*model.Topic
}

// calculateDraftPageSaveData はMarkdownレンダリング・Wikiリンクスキャン・トピック検索・アイキャッチ画像抽出・添付ファイルフィルター・画像ラッピングを行う
func calculateDraftPageSaveData(
	ctx context.Context,
	body string,
	currentTopicName string,
	spaceID model.SpaceID,
	topicRepo *repository.TopicRepository,
	attachmentRepo *repository.AttachmentRepository,
) (*draftPageSaveData, error) {
	bodyHTML := markup.RenderMarkdown(body)

	wikilinkKeys, topicMap, err := scanAndLookupWikilinks(ctx, body, currentTopicName, spaceID, topicRepo)
	if err != nil {
		return nil, fmt.Errorf("wikiリンクのスキャンに失敗しました: %w", err)
	}

	featuredImageAttachmentID, err := extractFeaturedImageAttachmentID(ctx, body, spaceID, attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の抽出に失敗しました: %w", err)
	}

	bodyHTML, err = markup.FilterAttachments(ctx, bodyHTML, spaceID, attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルのフィルター処理に失敗しました: %w", err)
	}

	bodyHTML = markup.WrapStandaloneImageLinks(bodyHTML)

	return &draftPageSaveData{
		bodyHTML:                  bodyHTML,
		featuredImageAttachmentID: featuredImageAttachmentID,
		wikilinkKeys:              wikilinkKeys,
		topicMap:                  topicMap,
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
