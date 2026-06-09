package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/markup"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPagePreviewUsecase is a read usecase that generates the preview for the page edit screen.
// It converts the in-progress (including unsaved) title and body into HTML through the same
// Markdown rendering path as the page detail screen. It performs no persistence such as saving
// drafts or auto-creating linked pages.
//
// [Ja] GetPagePreviewUsecase はページ編集画面のプレビューを生成する読み取りユースケース。
// 編集中の (未保存を含む) タイトルと本文を、ページ詳細画面と同じ Markdown レンダリング経路で
// HTML に変換する。下書きの保存やリンク先ページの自動作成などの永続化は一切行わない。
type GetPagePreviewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	attachmentRepo  *repository.AttachmentRepository
}

// NewGetPagePreviewUsecase は GetPagePreviewUsecase を生成する
func NewGetPagePreviewUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
) *GetPagePreviewUsecase {
	return &GetPagePreviewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		attachmentRepo:  attachmentRepo,
	}
}

// GetPagePreviewInput はプレビュー生成の入力パラメータ
type GetPagePreviewInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID

	// Title and Body are the current form values (including unsaved content).
	// [Ja] Title と Body はフォームの現在値 (未保存を含む)。
	Title string
	Body  string
}

// GetPagePreviewOutput はプレビュー生成の出力
type GetPagePreviewOutput struct {
	// Title is the title shown in the preview (plain text).
	// [Ja] Title はプレビューに表示するタイトル (プレーンテキスト)。
	Title string

	// BodyHTML is the rendered and sanitized body HTML.
	// [Ja] BodyHTML はレンダリング・サニタイズ済みの本文 HTML。
	BodyHTML string
}

// Execute はプレビュー用の HTML を生成する。
// 認可はページ編集画面と同じ (CanUpdatePage) で、非メンバーは拒否する。
func (uc *GetPagePreviewUsecase) Execute(ctx context.Context, input GetPagePreviewInput) (*GetPagePreviewOutput, error) {
	// 1. データ取得
	data, err := fetchPageAccessData(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック (編集画面と同じ認可)
	if err := authorizePageUpdate(ctx, data); err != nil {
		return nil, err
	}

	// 3. Render the body to HTML through the same path as the page detail screen.
	// Wiki links resolve to existing pages only; link targets are never created or persisted.
	//
	// [Ja] ページ詳細画面と同じレンダリング経路で本文を HTML 化する。
	// Wiki リンクは既存ページの解決のみ行い、リンク先の作成・保存などの永続化は一切行わない。
	resolver := &previewPageLocationResolver{topicRepo: uc.topicRepo, pageRepo: uc.pageRepo}
	bodyHTML, err := markup.RenderHTML(ctx, input.Body, data.topic.Name, data.space.ID, data.space.Identifier, resolver, uc.attachmentRepo)
	if err != nil {
		return nil, fmt.Errorf("プレビューのレンダリングに失敗: %w", err)
	}

	return &GetPagePreviewOutput{
		Title:    input.Title,
		BodyHTML: bodyHTML,
	}, nil
}

func (uc *GetPagePreviewUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

// previewPageLocationResolver is a markup.PageLocationResolver that resolves wiki link keys to existing pages only.
// Unlike resolveAndCreateLinkedPages, it never auto-creates missing pages because preview performs no persistence.
//
// [Ja] previewPageLocationResolver は Wiki リンクキーを既存ページにのみ解決する markup.PageLocationResolver。
// resolveAndCreateLinkedPages と異なり、存在しないページの自動作成は行わない (プレビューは永続化しないため)。
type previewPageLocationResolver struct {
	topicRepo *repository.TopicRepository
	pageRepo  *repository.PageRepository
}

// ResolveByKeys は Wiki リンクキーを既存ページの位置情報に解決する。存在しないページはスキップする。
func (r *previewPageLocationResolver) ResolveByKeys(ctx context.Context, keys []markup.WikilinkKey, spaceID model.SpaceID) ([]markup.PageLocation, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	topics, err := r.topicRepo.FindBySpaceAndNames(ctx, spaceID, uniqueTopicNames(keys))
	if err != nil {
		return nil, err
	}
	topicMap := make(map[string]*model.Topic, len(topics))
	for _, t := range topics {
		topicMap[t.Name] = t
	}

	var locations []markup.PageLocation
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		lookupKey := key.TopicName + "/" + key.PageTitle
		if seen[lookupKey] {
			continue
		}
		seen[lookupKey] = true

		topic := topicMap[key.TopicName]
		if topic == nil {
			continue
		}

		page, err := r.pageRepo.FindByTopicAndTitle(ctx, topic.ID, key.PageTitle, spaceID)
		if err != nil {
			return nil, err
		}
		if page == nil {
			// Resolve existing pages only; do not create missing link targets in preview.
			// [Ja] プレビューでは既存ページのみ解決し、存在しないリンク先は作成しない。
			continue
		}

		pageTitle := key.PageTitle
		if page.Title != nil {
			pageTitle = *page.Title
		}
		locations = append(locations, markup.PageLocation{
			Key:        key,
			TopicName:  topic.Name,
			PageID:     page.ID,
			PageNumber: int(page.Number),
			PageTitle:  pageTitle,
		})
	}

	return locations, nil
}
