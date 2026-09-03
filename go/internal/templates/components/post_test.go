package components_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestPost_ActionMenuAccessibility(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
	content := components.Post(components.PostData{
		CreatorAtname: "alice",
		Body:          "Comment body",
		CreatedAt:     time.Now(),
		Actions: []components.PostAction{
			{
				Label: "Edit",
				URL:   "/comments/1/edit",
			},
		},
		ActionsLabel: "コメントの操作",
		DropdownID:   "comment-1",
	})

	var buf bytes.Buffer
	if err := content.Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// Scope the assertions to the trigger element. Matching against the whole output would keep
	// passing if Post gained another decorative icon while the trigger itself regressed.
	//
	// [Ja] アサートはトリガー要素の範囲に限定する。出力全体を対象にすると、Post に別の装飾
	// アイコンが増えたときにトリガー自体が退行しても通過してしまう。
	trigger := elementSegment(html, `id="post-comment-1-dropdown-trigger"`, "</button>")
	if trigger == "" {
		t.Fatal("ドロップダウントリガーの <button> が見つからない")
	}

	if !strings.Contains(trigger, `aria-label="コメントの操作"`) {
		t.Error("ドロップダウントリガーに ActionsLabel が aria-label として出力されていない")
	}
	if !strings.Contains(trigger, `<svg aria-hidden="true" focusable="false"`) {
		t.Error("ドロップダウントリガーの SVG が支援技術とフォーカス順序から除外されていない")
	}
	if !strings.Contains(html, `aria-labelledby="post-comment-1-dropdown-trigger"`) {
		t.Error("メニューが名前付きトリガーを aria-labelledby で参照していない")
	}
}

// A caller that sets Actions without ActionsLabel gets no aria-label attribute at all. Emitting
// an empty one would look like a label was set while the trigger, and the menu pointing at it
// with aria-labelledby, stay unnamed.
//
// [Ja] Actions だけを設定して ActionsLabel を渡さなかった場合、aria-label 属性自体を出力しない。
// 空文字で出力すると、トリガー (と aria-labelledby でそれを参照するメニュー) が無名のままなのに
// ラベルを設定済みに見えてしまう。
func TestPost_ActionMenuOmitsEmptyLabel(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
	content := components.Post(components.PostData{
		CreatorAtname: "alice",
		Body:          "Comment body",
		CreatedAt:     time.Now(),
		Actions: []components.PostAction{
			{
				Label: "Edit",
				URL:   "/comments/1/edit",
			},
		},
		DropdownID: "comment-1",
	})

	var buf bytes.Buffer
	if err := content.Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	trigger := elementSegment(buf.String(), `id="post-comment-1-dropdown-trigger"`, "</button>")
	if trigger == "" {
		t.Fatal("ドロップダウントリガーの <button> が見つからない")
	}

	if strings.Contains(trigger, "aria-label=") {
		t.Error("ActionsLabel が空なのにドロップダウントリガーへ aria-label が出力されている")
	}
}

// elementSegment returns the substring running from the first occurrence of start through the
// first end that follows it. It returns "" when either marker is missing.
//
// [Ja] elementSegment は start が最初に現れる位置から、その後ろで最初に現れる end までの
// 部分文字列を返す。どちらかのマーカーが無い場合は "" を返す。
func elementSegment(html, start, end string) string {
	i := strings.Index(html, start)
	if i < 0 {
		return ""
	}
	j := strings.Index(html[i:], end)
	if j < 0 {
		return ""
	}
	return html[i : i+j+len(end)]
}
