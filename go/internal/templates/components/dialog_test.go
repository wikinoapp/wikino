package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestDialog(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(t.Context(), "ja")

	content := components.Dialog(components.DialogData{
		ID:            "test-dialog",
		Title:         "テストダイアログ",
		MaxWidthClass: "sm:max-w-2xl",
		Content:       templ.Raw("ダイアログの中身"),
	})

	var buf bytes.Buffer
	if err := content.Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `<dialog id="test-dialog" class="dialog"`) {
		t.Error("dialog要素とidが含まれていない")
	}
	if !strings.Contains(html, `aria-labelledby="test-dialog-title"`) {
		t.Error("aria-labelledbyが含まれていない")
	}
	if !strings.Contains(html, "テストダイアログ") {
		t.Error("タイトルが含まれていない")
	}
	if !strings.Contains(html, "ダイアログの中身") {
		t.Error("コンテンツが含まれていない")
	}
	// パネルには素のz-50ではなくCSS変数ベースのz-index (z-dialog) を使い、
	// 任意の幅クラスを受け付けること。
	if !strings.Contains(html, "z-dialog") {
		t.Error("z-dialogクラスが含まれていない")
	}
	if !strings.Contains(html, "sm:max-w-2xl") {
		t.Error("MaxWidthClassが含まれていない")
	}
	// 閉じる手段: 閉じるボタンのラベルとバックドロップクリックのハンドラー。
	if !strings.Contains(html, `aria-label="閉じる"`) {
		t.Error("閉じるボタンのaria-labelが含まれていない")
	}
	if !strings.Contains(html, "if (event.target === this) this.close()") {
		t.Error("バックドロップクリックで閉じるハンドラーが含まれていない")
	}
}
