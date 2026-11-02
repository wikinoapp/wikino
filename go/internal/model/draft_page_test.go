package model_test

import (
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestDraftPage_ContentDigest(t *testing.T) {
	t.Parallel()

	title := "タイトル"
	base := model.DraftPage{ID: "draft-1", PageID: "page-1", TopicID: "topic-1", Title: &title, Body: "本文", ModifiedAt: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}

	otherTitle := "別のタイトル"
	tests := []struct {
		name     string
		draft    model.DraftPage
		wantSame bool
	}{
		{
			name: "内容に含めない項目 (IDなど) だけが違えば同じ値",
			draft: func() model.DraftPage {
				d := base
				d.ID = "draft-2"
				d.Page = &model.Page{Number: 2}
				d.UpdatedAt = d.ModifiedAt.Add(time.Hour)
				return d
			}(),
			wantSame: true,
		},
		{
			name: "起点だけが違えば同じ値",
			draft: func() model.DraftPage {
				d := base
				d.BaseTitle = &otherTitle
				baseBody := "起点の本文"
				d.BaseBody = &baseBody
				return d
			}(),
			wantSame: true,
		},
		{
			name: "タイトルが違えば別の値",
			draft: func() model.DraftPage {
				d := base
				d.Title = &otherTitle
				return d
			}(),
		},
		{
			name: "タイトルの有無が違えば別の値",
			draft: func() model.DraftPage {
				d := base
				d.Title = nil
				return d
			}(),
		},
		{
			name: "本文が違えば別の値",
			draft: func() model.DraftPage {
				d := base
				d.Body = "別の本文"
				return d
			}(),
		},
		{
			name: "所属トピックが違えば別の値",
			draft: func() model.DraftPage {
				d := base
				d.TopicID = "topic-2"
				return d
			}(),
		},
		{
			name: "内容が同じでも更新日時が違えば別の値",
			draft: func() model.DraftPage {
				d := base
				d.ModifiedAt = d.ModifiedAt.Add(time.Second)
				return d
			}(),
		},
		{
			name: "項目の境界をずらしても同じ値にならない",
			draft: func() model.DraftPage {
				d := base
				t := "タイトル本"
				d.Title = &t
				d.Body = "文"
				return d
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.draft.ContentDigest() == base.ContentDigest(); got != tt.wantSame {
				t.Errorf("同じ値か = %v、期待値 = %v", got, tt.wantSame)
			}
		})
	}
}

func TestDraftPage_PrecedingContent(t *testing.T) {
	t.Parallel()

	baseTitle := "起点のタイトル"
	baseBody := "起点の本文"
	emptyBody := ""
	previous := &model.DraftPageRevision{Title: "直前のタイトル", Body: "直前の本文"}

	tests := []struct {
		name     string
		draft    model.DraftPage
		previous *model.DraftPageRevision
		want     *model.DraftPageContent
	}{
		{
			name:     "直前のリビジョンがあればその内容",
			draft:    model.DraftPage{BaseTitle: &baseTitle, BaseBody: &baseBody},
			previous: previous,
			want:     &model.DraftPageContent{Title: "直前のタイトル", Body: "直前の本文"},
		},
		{
			name:  "直前のリビジョンが無ければ起点",
			draft: model.DraftPage{BaseTitle: &baseTitle, BaseBody: &baseBody},
			want:  &model.DraftPageContent{Title: "起点のタイトル", Body: "起点の本文"},
		},
		{
			name:  "起点のタイトルが無ければ空のタイトル",
			draft: model.DraftPage{BaseBody: &emptyBody},
			want:  &model.DraftPageContent{Title: "", Body: ""},
		},
		{
			name:  "起点が分からなければ直前の状態は無い",
			draft: model.DraftPage{},
			want:  nil,
		},
		{
			name:     "起点が分からなくても直前のリビジョンがあればその内容",
			draft:    model.DraftPage{},
			previous: previous,
			want:     &model.DraftPageContent{Title: "直前のタイトル", Body: "直前の本文"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.draft.PrecedingContent(tt.previous)
			if tt.want == nil {
				if got != nil {
					t.Errorf("PrecedingContent() = %+v、期待値 = nil", got)
				}
				return
			}
			if got == nil || *got != *tt.want {
				t.Errorf("PrecedingContent() = %+v、期待値 = %+v", got, tt.want)
			}
		})
	}
}
