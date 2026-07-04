package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestCardLinkDraftPage_SpaceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		spaceName     string
		wantSpaceName bool
	}{
		{
			name:          "スペース名あり: ラベルとして表示される",
			spaceName:     "マイスペース",
			wantSpaceName: true,
		},
		{
			name:          "スペース名が空: ラベルを表示しない",
			spaceName:     "",
			wantSpaceName: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			draft := viewmodel.NewCardLinkDraftPage(&model.DraftPage{
				Title: strPtr("下書きタイトル"),
				Page:  &model.Page{Number: 5, Title: strPtr("公開ページタイトル")},
				Topic: &model.Topic{
					Name:       "トピック名",
					Visibility: model.TopicVisibilityPublic,
					Space:      &model.Space{Identifier: "my-space", Name: tt.spaceName},
				},
			})

			ctx := i18n.SetLocale(t.Context(), "ja")

			var buf bytes.Buffer
			if err := components.CardLinkDraftPage(draft).Render(ctx, &buf); err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()

			// The page editor link is always present.
			// [Ja] ページ編集画面へのリンクは常に存在する。
			if !strings.Contains(html, "/s/my-space/pages/5/edit") {
				t.Error("ページ編集画面へのリンクが含まれていない")
			}

			// The topic name and draft title are always rendered.
			// [Ja] トピック名と下書きタイトルは常に表示される。
			if !strings.Contains(html, "トピック名") {
				t.Error("トピック名が含まれていない")
			}
			if !strings.Contains(html, "下書きタイトル") {
				t.Error("下書きタイトルが含まれていない")
			}

			gotSpaceName := strings.Contains(html, "マイスペース")
			if gotSpaceName != tt.wantSpaceName {
				t.Errorf("スペース名の表示 = %v, want %v", gotSpaceName, tt.wantSpaceName)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
