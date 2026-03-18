package viewmodel

import (
	"context"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// SuggestionForList は編集提案一覧用の表示データです
type SuggestionForList struct {
	Number      int32
	Title       string
	Status      model.SuggestionStatus
	CreatorName string
	CreatedAt   time.Time
}

// NewSuggestionForListInput はNewSuggestionForListの入力パラメータです
type NewSuggestionForListInput struct {
	Suggestions []*model.Suggestion
	UserMap     map[model.SpaceMemberID]*model.User
}

// NewSuggestionsForList は編集提案モデルのスライスから一覧用ViewModelのスライスを生成します
func NewSuggestionsForList(input NewSuggestionForListInput) []SuggestionForList {
	items := make([]SuggestionForList, len(input.Suggestions))
	for i, s := range input.Suggestions {
		var creatorName string
		if user, ok := input.UserMap[s.CreatedSpaceMemberID]; ok {
			creatorName = userDisplayName(user)
		}
		items[i] = SuggestionForList{
			Number:      int32(s.Number),
			Title:       s.Title,
			Status:      s.Status,
			CreatorName: creatorName,
			CreatedAt:   s.CreatedAt,
		}
	}
	return items
}

// userDisplayName はユーザーの表示名を返す（名前があれば名前、なければアットネーム）
func userDisplayName(user *model.User) string {
	if user.Name != "" {
		return user.Name
	}
	return user.Atname
}

// SuggestionForDetail は編集提案詳細画面用の表示データです
type SuggestionForDetail struct {
	Number      int32
	Title       string
	Body        string
	BodyHTML    string
	Status      model.SuggestionStatus
	CreatorName string
	CreatedAt   time.Time
}

// NewSuggestionForDetailInput はNewSuggestionForDetailの入力パラメータです
type NewSuggestionForDetailInput struct {
	Suggestion *model.Suggestion
	UserMap    map[model.SpaceMemberID]*model.User
}

// NewSuggestionForDetail は編集提案モデルから詳細画面用ViewModelを生成します
func NewSuggestionForDetail(input NewSuggestionForDetailInput) SuggestionForDetail {
	var creatorName string
	if user, ok := input.UserMap[input.Suggestion.CreatedSpaceMemberID]; ok {
		creatorName = userDisplayName(user)
	}
	return SuggestionForDetail{
		Number:      int32(input.Suggestion.Number),
		Title:       input.Suggestion.Title,
		Body:        input.Suggestion.Body,
		BodyHTML:    input.Suggestion.BodyHTML,
		Status:      input.Suggestion.Status,
		CreatorName: creatorName,
		CreatedAt:   input.Suggestion.CreatedAt,
	}
}

// SuggestionCommentForList は編集提案コメント一覧用の表示データです
type SuggestionCommentForList struct {
	CreatorName string
	BodyHTML    string
	CreatedAt   time.Time
}

// NewSuggestionCommentsForListInput はNewSuggestionCommentsForListの入力パラメータです
type NewSuggestionCommentsForListInput struct {
	Comments []*model.SuggestionComment
	UserMap  map[model.SpaceMemberID]*model.User
}

// NewSuggestionCommentsForList はコメントモデルのスライスから一覧用ViewModelのスライスを生成します
func NewSuggestionCommentsForList(input NewSuggestionCommentsForListInput) []SuggestionCommentForList {
	items := make([]SuggestionCommentForList, len(input.Comments))
	for i, c := range input.Comments {
		var creatorName string
		if user, ok := input.UserMap[c.CreatedSpaceMemberID]; ok {
			creatorName = userDisplayName(user)
		}
		items[i] = SuggestionCommentForList{
			CreatorName: creatorName,
			BodyHTML:    c.BodyHTML,
			CreatedAt:   c.CreatedAt,
		}
	}
	return items
}

// SuggestionPageForList は編集提案ページ一覧用の表示データです
type SuggestionPageForList struct {
	Title string
}

// NewSuggestionPagesForList は編集提案ページモデルのスライスから一覧用ViewModelのスライスを生成します
func NewSuggestionPagesForList(pages []*model.SuggestionPage) []SuggestionPageForList {
	items := make([]SuggestionPageForList, len(pages))
	for i, p := range pages {
		var title string
		if p.Title != nil {
			title = *p.Title
		}
		items[i] = SuggestionPageForList{
			Title: title,
		}
	}
	return items
}

// SuggestionPageDiff は編集提案ページの差分表示データです
type SuggestionPageDiff struct {
	PageTitle      string
	OldTitle       string
	NewTitle       string
	HasTitleChange bool
	BodyBlocks     []DiffBlock
}

// NewSuggestionPageDiffsInput はNewSuggestionPageDiffsの入力パラメータです
type NewSuggestionPageDiffsInput struct {
	SuggestionPages []*model.SuggestionPage
	BaseRevisions   map[model.SuggestionPageID]*model.PageRevision
}

// NewSuggestionPageDiffs は編集提案ページとベースリビジョンから差分表示用ViewModelのスライスを生成します
func NewSuggestionPageDiffs(input NewSuggestionPageDiffsInput) []SuggestionPageDiff {
	diffs := make([]SuggestionPageDiff, len(input.SuggestionPages))
	for i, sp := range input.SuggestionPages {
		var oldTitle, oldBody string
		baseRev := input.BaseRevisions[sp.ID]
		if baseRev != nil {
			oldTitle = baseRev.Title
			oldBody = baseRev.Body
		}

		newTitle := oldTitle
		if sp.Title != nil {
			newTitle = *sp.Title
		}

		pageTitle := newTitle
		if pageTitle == "" {
			pageTitle = oldTitle
		}

		diffs[i] = SuggestionPageDiff{
			PageTitle:      pageTitle,
			OldTitle:       oldTitle,
			NewTitle:       newTitle,
			HasTitleChange: oldTitle != newTitle,
			BodyBlocks:     ComputeDiffBlocks(oldBody, sp.Body, 3),
		}
	}
	return diffs
}

// DraftPageForSuggestionNew は編集提案作成画面で選択可能な下書きページの表示データです
type DraftPageForSuggestionNew struct {
	ID         model.DraftPageID
	Title      string
	PageNumber int32
}

// DisplayTitle は表示用タイトルを返します。タイトルが未設定の場合は「無題」を返します。
func (d DraftPageForSuggestionNew) DisplayTitle(ctx context.Context) string {
	if d.Title != "" {
		return d.Title
	}
	return i18n.T(ctx, "draft_page_index_untitled")
}

// NewDraftPagesForSuggestionNew は下書きページモデルのスライスから編集提案作成画面用のViewModelスライスを生成します
func NewDraftPagesForSuggestionNew(drafts []*model.DraftPage) []DraftPageForSuggestionNew {
	items := make([]DraftPageForSuggestionNew, len(drafts))
	for i, d := range drafts {
		var title string
		if d.Title != nil {
			title = *d.Title
		} else if d.Page != nil && d.Page.Title != nil {
			title = *d.Page.Title
		}
		var pageNumber int32
		if d.Page != nil {
			pageNumber = int32(d.Page.Number)
		}
		items[i] = DraftPageForSuggestionNew{
			ID:         d.ID,
			Title:      title,
			PageNumber: pageNumber,
		}
	}
	return items
}
