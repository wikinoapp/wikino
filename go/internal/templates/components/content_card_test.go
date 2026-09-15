package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

// TestContentCard_DoesNotClipItsChildrenは、カードの内側で開いたドロップダウンや
// ポップオーバーがカードの外へ出られることを確認する。
//
// class属性は部分文字列ではなく値を丸ごと照合する。これにより、後から同じ語で終わる別の
// クラスが増えてもoverflow-visibleが落ちたことを捕まえられる。切り抜きを持つのはカードだけ
// で、内側のsectionはもともと持っていない。
func TestContentCard_DoesNotClipItsChildren(t *testing.T) {
	t.Parallel()

	ctx := templ.WithChildren(context.Background(), templ.Raw(`<span>中身</span>`))

	var buf bytes.Buffer
	if err := components.ContentCard().Render(ctx, &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `class="card py-4 rounded-none md:rounded-xl mx-0 md:mx-4 overflow-visible"`) {
		t.Error("カードが切り抜きを外していない")
	}
}
