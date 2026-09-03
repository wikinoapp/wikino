package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/timezone"
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
			name:         "タイムゾーン未設定の場合はUTCで表示される",
			timeZone:     "",
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
			if tt.timeZone != "" {
				ctx = timezone.ToContext(ctx, tt.timeZone)
			}

			data := components.DraftPageShowResponseData{
				HasDraft:   true,
				ModifiedAt: modifiedAt,
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

	// OOBスワップ用の 3 つの関連ページセクションは含まれる
	if !strings.Contains(html, `id="page-link-list"`) {
		t.Error("リンク一覧のOOBスワップ要素が含まれていない")
	}
	if !strings.Contains(html, `id="page-related-link-list"`) {
		t.Error("関連リンク一覧のOOBスワップ要素が含まれていない")
	}
	if !strings.Contains(html, `id="page-backlink-list"`) {
		t.Error("バックリンク一覧のOOBスワップ要素が含まれていない")
	}
}

func TestDraftPageShowResponse_OOBスワップ属性(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")
	ctx = timezone.ToContext(ctx, "Asia/Tokyo")

	data := components.DraftPageShowResponseData{
		HasDraft:     true,
		ModifiedAt:   time.Date(2025, 1, 15, 5, 30, 0, 0, time.UTC),
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

	// 3 つの関連ページセクションは innerHTML で置換
	if !strings.Contains(html, `id="page-link-list" hx-swap-oob="innerHTML"`) {
		t.Error("リンク一覧のOOBスワップ属性 innerHTML が含まれていない")
	}
	if !strings.Contains(html, `id="page-related-link-list" hx-swap-oob="innerHTML"`) {
		t.Error("関連リンク一覧のOOBスワップ属性 innerHTML が含まれていない")
	}
	if !strings.Contains(html, `id="page-backlink-list" hx-swap-oob="innerHTML"`) {
		t.Error("バックリンク一覧のOOBスワップ属性 innerHTML が含まれていない")
	}
}

func TestDraftPageShowResponse_見出しを含まずリストだけを再描画する(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	// The heading (h2) was moved out of the OOB swap target on the edit.templ side, so this
	// response (the re-render target) renders only the lists and never the heading.
	//
	// [Ja] 見出し (h2) は edit.templ 側の OOB スワップ対象外に置いたため、
	// 本レスポンス (再描画対象) には見出しを含めず、リストだけを描画する。
	data := components.DraftPageShowResponseData{
		LinkList: viewmodel.LinkList{
			Items: []viewmodel.LinkListItem{
				{CardLinkPage: viewmodel.CardLinkPage{Title: "ページA", Number: 2}},
			},
			SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
			PageNumber:      1,
		},
		BacklinkList: viewmodel.BacklinkList{
			Items: []viewmodel.BacklinkListItem{
				{CardLinkPage: viewmodel.CardLinkPage{Title: "ページB", Number: 3}},
			},
			SpaceIdentifier: viewmodel.SpaceIdentifier("my-space"),
			PageNumber:      1,
		},
	}

	var buf bytes.Buffer
	if err := components.DraftPageShowResponse(data).Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	html := buf.String()

	// The list contents must be rendered.
	//
	// [Ja] リストの内容は描画されること。
	if !strings.Contains(html, "ページA") {
		t.Error("リンク一覧のカードが描画されていない")
	}
	if !strings.Contains(html, "ページB") {
		t.Error("バックリンク一覧のカードが描画されていない")
	}

	// The heading (h2) must not be included in the re-render target.
	//
	// [Ja] 見出し (h2) は再描画対象に含めないこと。
	if strings.Contains(html, "<h2") {
		t.Errorf("再描画レスポンスに見出し (h2) が含まれてはいけない: got %q", html)
	}
	if strings.Contains(html, "バックリンク") {
		t.Error("再描画レスポンスにバックリンクの見出しが含まれてはいけない")
	}
}
