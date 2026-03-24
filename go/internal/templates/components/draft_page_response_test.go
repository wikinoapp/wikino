package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestDraftPageShowResponse_タイムゾーン変換(t *testing.T) {
	t.Parallel()

	// UTC 05:30 を基準とする
	modifiedAt := time.Date(2025, 1, 15, 5, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		timeZone     string
		expectedTime string
	}{
		{
			name:         "Asia/Tokyoの場合はJST(UTC+9)で表示される",
			timeZone:     "Asia/Tokyo",
			expectedTime: "14:30",
		},
		{
			name:         "空文字の場合はUTCで表示される",
			timeZone:     "",
			expectedTime: "05:30",
		},
		{
			name:         "不正なタイムゾーンの場合はUTCで表示される",
			timeZone:     "Invalid/TimeZone",
			expectedTime: "05:30",
		},
		{
			name:         "America/New_Yorkの場合はEST(UTC-5)で表示される",
			timeZone:     "America/New_York",
			expectedTime: "00:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			data := components.DraftPageShowResponseData{
				HasDraft:   true,
				ModifiedAt: modifiedAt,
				TimeZone:   tt.timeZone,
			}

			var buf bytes.Buffer
			err := components.DraftPageShowResponse(data).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()
			if !strings.Contains(html, tt.expectedTime) {
				t.Errorf("出力に期待する時刻 %q が含まれていない\n出力: %s", tt.expectedTime, html)
			}
		})
	}
}

func TestDraftPageShowResponse_下書きなしの場合に保存時刻が含まれない(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	data := components.DraftPageShowResponseData{
		HasDraft: false,
	}

	var buf bytes.Buffer
	err := components.DraftPageShowResponse(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if strings.Contains(html, `id="page-draft-saved-at"`) {
		t.Error("下書きなしの場合、保存時刻要素が含まれるべきではない")
	}

	// OOBスワップ用のリンク一覧・バックリンク一覧は含まれる
	if !strings.Contains(html, `id="page-link-list"`) {
		t.Error("リンク一覧のOOBスワップ要素が含まれていない")
	}
	if !strings.Contains(html, `id="page-backlink-list"`) {
		t.Error("バックリンク一覧のOOBスワップ要素が含まれていない")
	}
}

func TestDraftPageShowResponse_OOBスワップ属性(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	data := components.DraftPageShowResponseData{
		HasDraft:     true,
		ModifiedAt:   time.Date(2025, 1, 15, 5, 30, 0, 0, time.UTC),
		TimeZone:     "Asia/Tokyo",
		LinkList:     viewmodel.LinkList{},
		BacklinkList: viewmodel.BacklinkList{},
	}

	var buf bytes.Buffer
	err := components.DraftPageShowResponse(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// 保存時刻はouterHTMLで置換
	if !strings.Contains(html, `hx-swap-oob="outerHTML"`) {
		t.Error("保存時刻のOOBスワップ属性 outerHTML が含まれていない")
	}

	// リンク一覧・バックリンク一覧はinnerHTMLで置換
	if !strings.Contains(html, `id="page-link-list" hx-swap-oob="innerHTML"`) {
		t.Error("リンク一覧のOOBスワップ属性 innerHTML が含まれていない")
	}
	if !strings.Contains(html, `id="page-backlink-list" hx-swap-oob="innerHTML"`) {
		t.Error("バックリンク一覧のOOBスワップ属性 innerHTML が含まれていない")
	}
}
