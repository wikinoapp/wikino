package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

// renderMainTitleはcをjaロケールで描画し、HTMLを返す。
func renderMainTitle(t *testing.T, c templ.Component) string {
	t.Helper()

	ctx := i18n.SetLocale(context.Background(), "ja")

	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}
	return buf.String()
}

// TestMainTitle_PinsActionsToTheFirstLineOfTheTitleは、このコンポーネントが取りうる中で最も
// 高い左の列 (タイトル・サブタイトル・その下の追加コンテンツ) で見出し行の配置を確認する。
func TestMainTitle_PinsActionsToTheFirstLineOfTheTitle(t *testing.T) {
	t.Parallel()

	html := renderMainTitle(t, components.MainTitle(components.MainTitleData{
		Title:    "テストトピック",
		Subtitle: components.SubtitleText("トピックの説明"),
		Content:  templ.Raw(`<span>追加のコンテンツ</span>`),
		Actions:  templ.Raw(`<a class="btn rounded-full" data-size="sm">編集する</a>`),
	}))

	// 見出し行は子を上端に揃え、タイトル側の列がどれだけ高くなっても操作領域がタイトルの1行目
	// の横に留まる。
	//
	// class属性は部分文字列ではなく値を丸ごと照合する。items-centerはこの行の中に入れ子になった
	// 操作領域にも付いており、部分文字列では両者を区別できず、見出し行が中央へ戻る変更を捕まえられない
	// ためである。
	if !strings.Contains(html, `class="flex flex-wrap md:flex-nowrap items-start justify-between gap-2"`) {
		t.Error("見出し行が子を上端に揃えていない")
	}
	if strings.Contains(html, `class="flex flex-wrap md:flex-nowrap items-center justify-between gap-2"`) {
		t.Error("見出し行が子を中央に揃えている")
	}

	// 操作領域はタイトルの1行目の高さを持ち、中のボタンが行ボックスの上端に張り付かず
	// タイトルに対して中央に来る。
	if !strings.Contains(html, `class="flex w-full flex-none items-center justify-end gap-2 md:w-auto md:min-h-9"`) {
		t.Error("操作領域がタイトルの1行目の高さを持っていない")
	}
}
