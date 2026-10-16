package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestCopyableInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
		wants  []string
	}{
		{
			name:   "日本語",
			locale: "ja",
			wants: []string{
				`aria-label="トークンをコピー"`,
				`data-success-message="トークンをコピーしました"`,
				`data-failure-message="コピーできませんでした。入力欄の値を選んでコピーしてください"`,
			},
		},
		{
			name:   "英語",
			locale: "en",
			wants: []string{
				`aria-label="Copy トークン"`,
				`data-success-message="Copied トークン to the clipboard"`,
				`data-failure-message="Couldn&#39;t copy. Select the value in the field and copy it instead"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), tt.locale)

			var buf bytes.Buffer
			err := components.CopyableInput(components.CopyableInputData{
				ID:    "token_value",
				Label: "トークン",
				Value: "wkp_abc-def",
			}).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("Render()のエラー = %v", err)
			}

			html := buf.String()
			wants := append([]string{
				// web/copyable-input.tsが振る舞いを付ける目印と、読み取り専用の欄
				`<div class="grid gap-2" data-copyable-input>`,
				`<label class="label" for="token_value">トークン</label>`,
				`data-copyable-input-value id="token_value" readonly spellcheck="false" type="text" value="wkp_abc-def"`,
				`<button type="button" class="btn shrink-0" hidden`,
				`data-copyable-input-button`,
				// 読み上げは描画時からあるステータス領域で伝える
				`<p class="sr-only" role="status" data-copyable-input-status></p>`,
			}, tt.wants...)
			for _, want := range wants {
				if !strings.Contains(html, want) {
					t.Errorf("出力に %qが含まれていない\n%s", want, html)
				}
			}
			// コピーボタンのアイコンは装飾で、名前はaria-labelが持つ
			if !strings.Contains(html, `<svg aria-hidden="true" focusable="false"`) {
				t.Error("コピーボタンのアイコンが支援技術から隠されていない")
			}
		})
	}
}
