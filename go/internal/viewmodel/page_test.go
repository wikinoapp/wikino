package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewPageForEdit(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name      string
		page      *model.Page
		draftPage *model.DraftPage
		wantTitle string
		wantBody  string
		wantNum   int32
	}{
		{
			name: "公開ページ（タイトルあり、下書きなし）",
			page: &model.Page{
				Number: 1,
				Title:  strPtr("公開タイトル"),
				Body:   "公開本文",
			},
			draftPage: nil,
			wantTitle: "公開タイトル",
			wantBody:  "公開本文",
			wantNum:   1,
		},
		{
			name: "公開ページ（タイトルなし、下書きなし）",
			page: &model.Page{
				Number: 2,
				Title:  nil,
				Body:   "タイトルなし本文",
			},
			draftPage: nil,
			wantTitle: "",
			wantBody:  "タイトルなし本文",
			wantNum:   2,
		},
		{
			name: "下書きあり（下書きのタイトル/本文が使われる）",
			page: &model.Page{
				Number: 3,
				Title:  strPtr("公開タイトル"),
				Body:   "公開本文",
			},
			draftPage: &model.DraftPage{
				Title: strPtr("下書きタイトル"),
				Body:  "下書き本文",
			},
			wantTitle: "下書きタイトル",
			wantBody:  "下書き本文",
			wantNum:   3,
		},
		{
			name: "下書きあり（下書きのタイトルがnil）",
			page: &model.Page{
				Number: 4,
				Title:  strPtr("公開タイトル"),
				Body:   "公開本文",
			},
			draftPage: &model.DraftPage{
				Title: nil,
				Body:  "下書き本文のみ",
			},
			wantTitle: "",
			wantBody:  "下書き本文のみ",
			wantNum:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewPageForEdit(tt.page, tt.draftPage)

			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}

			if got.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", got.Body, tt.wantBody)
			}

			if got.Number != tt.wantNum {
				t.Errorf("Number = %d, want %d", got.Number, tt.wantNum)
			}
		})
	}
}

func TestPage_AutofocusTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		page viewmodel.Page
		want bool
	}{
		{
			name: "タイトルあり → false",
			page: viewmodel.Page{Title: "タイトル"},
			want: false,
		},
		{
			name: "タイトルなし → true",
			page: viewmodel.Page{Title: ""},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.page.AutofocusTitle(); got != tt.want {
				t.Errorf("AutofocusTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}
