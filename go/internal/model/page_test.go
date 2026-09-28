package model_test

import (
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestPage_ContentDigest(t *testing.T) {
	t.Parallel()

	title := "タイトル"
	base := model.Page{ID: "page-1", TopicID: "topic-1", Title: &title, Body: "本文", ModifiedAt: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}

	otherTitle := "別のタイトル"
	tests := []struct {
		name     string
		page     model.Page
		wantSame bool
	}{
		{
			name: "内容に含めない項目 (IDなど) だけが違えば同じ値",
			page: func() model.Page {
				p := base
				p.ID = "page-2"
				p.Number = 2
				return p
			}(),
			wantSame: true,
		},
		{
			name: "タイトルが違えば別の値",
			page: func() model.Page {
				p := base
				p.Title = &otherTitle
				return p
			}(),
		},
		{
			name: "タイトルの有無が違えば別の値",
			page: func() model.Page {
				p := base
				p.Title = nil
				return p
			}(),
		},
		{
			name: "本文が違えば別の値",
			page: func() model.Page {
				p := base
				p.Body = "別の本文"
				return p
			}(),
		},
		{
			name: "所属トピックが違えば別の値",
			page: func() model.Page {
				p := base
				p.TopicID = "topic-2"
				return p
			}(),
		},
		{
			name: "内容が同じでも更新日時が違えば別の値",
			page: func() model.Page {
				p := base
				p.ModifiedAt = p.ModifiedAt.Add(time.Second)
				return p
			}(),
		},
		{
			name: "項目の境界をずらしても同じ値にならない",
			page: func() model.Page {
				p := base
				t := "タイトル本"
				p.Title = &t
				p.Body = "文"
				return p
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.page.ContentDigest() == base.ContentDigest(); got != tt.wantSame {
				t.Errorf("同じ値か = %v、期待値 = %v", got, tt.wantSame)
			}
		})
	}
}
