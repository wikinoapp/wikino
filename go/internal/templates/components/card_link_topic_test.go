package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestCardLinkTopic_SpaceName(t *testing.T) {
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

			topic := viewmodel.CardLinkTopic{
				Name:            "トピック名",
				Number:          3,
				SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
				SpaceName:       tt.spaceName,
				TopicIconName:   "globe-regular",
			}

			ctx := i18n.SetLocale(context.Background(), "ja")

			var buf bytes.Buffer
			if err := components.CardLinkTopic(topic).Render(ctx, &buf); err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()

			// The topic detail link is always present.
			// [Ja] トピック詳細へのリンクは常に存在する。
			if !strings.Contains(html, "/s/my-space/topics/3") {
				t.Error("トピック詳細へのリンクが含まれていない")
			}

			// The topic name is always rendered.
			// [Ja] トピック名は常に表示される。
			if !strings.Contains(html, "トピック名") {
				t.Error("トピック名が含まれていない")
			}

			gotSpaceName := strings.Contains(html, "マイスペース")
			if gotSpaceName != tt.wantSpaceName {
				t.Errorf("スペース名の表示 = %v, want %v", gotSpaceName, tt.wantSpaceName)
			}
		})
	}
}

func TestCardLinkTopic_CanCreatePage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		canCreatePage   bool
		wantNewPageLink bool
	}{
		{
			name:            "作成権限あり: 新規ページリンクを表示する",
			canCreatePage:   true,
			wantNewPageLink: true,
		},
		{
			name:            "作成権限なし: 新規ページリンクを表示しない",
			canCreatePage:   false,
			wantNewPageLink: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			topic := viewmodel.CardLinkTopic{
				Name:            "トピック名",
				Number:          3,
				SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
				SpaceName:       "マイスペース",
				TopicIconName:   "globe-regular",
				CanCreatePage:   tt.canCreatePage,
			}

			ctx := i18n.SetLocale(context.Background(), "ja")

			var buf bytes.Buffer
			if err := components.CardLinkTopic(topic).Render(ctx, &buf); err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()

			gotNewPageLink := strings.Contains(html, "/s/my-space/topics/3/pages/new")
			if gotNewPageLink != tt.wantNewPageLink {
				t.Errorf("新規ページリンクの表示 = %v, want %v", gotNewPageLink, tt.wantNewPageLink)
			}

			gotTooltip := strings.Contains(html, "新規ページ")
			if gotTooltip != tt.wantNewPageLink {
				t.Errorf("新規ページツールチップの表示 = %v, want %v", gotTooltip, tt.wantNewPageLink)
			}
		})
	}
}
