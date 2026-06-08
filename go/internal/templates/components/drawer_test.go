package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestDrawer_Side(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		side      string
		wantClass string
	}{
		{
			name:      "左から開くドロワー",
			side:      "left",
			wantClass: "left-0",
		},
		{
			name:      "右から開くドロワー",
			side:      "right",
			wantClass: "right-0",
		},
		{
			name:      "未知の side は左端を既定とする",
			side:      "",
			wantClass: "left-0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content := components.Drawer(components.DrawerData{
				ID:      "draft-list-drawer",
				Side:    tt.side,
				Content: templ.Raw("ドロワーの中身"),
			})

			var buf bytes.Buffer
			if err := content.Render(t.Context(), &buf); err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()

			// The drawer is hidden by default and carries the data attributes the JS keys off.
			// [Ja] ドロワーは既定で非表示で、JS が手がかりにする data 属性を持つ。
			if !strings.Contains(html, `id="draft-list-drawer"`) {
				t.Error("ドロワーの id が含まれていない")
			}
			if !strings.Contains(html, "data-drawer") {
				t.Error("data-drawer 属性が含まれていない")
			}
			if !strings.Contains(html, "data-drawer-close") {
				t.Error("背景クリックで閉じるための data-drawer-close が含まれていない")
			}
			if !strings.Contains(html, "z-40 hidden") {
				t.Error("既定で非表示にするための hidden クラスが含まれていない")
			}
			if !strings.Contains(html, "ドロワーの中身") {
				t.Error("渡したコンテンツが表示されていない")
			}
			if !strings.Contains(html, tt.wantClass) {
				t.Errorf("パネルを %q に寄せるクラスが含まれていない", tt.wantClass)
			}
		})
	}
}

func TestDrawerOpenButton(t *testing.T) {
	t.Parallel()

	content := components.DrawerOpenButton(components.DrawerOpenData{
		DrawerID: "draft-list-drawer",
		Label:    "下書き一覧",
		IconName: "list-regular",
	})

	var buf bytes.Buffer
	if err := content.Render(t.Context(), &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// The button targets the paired drawer, shows the provided label, and starts collapsed.
	// [Ja] ボタンは対応するドロワーを指し、渡されたラベルを表示し、初期状態は閉じている。
	if !strings.Contains(html, `data-drawer-open="draft-list-drawer"`) {
		t.Error("開く対象を指す data-drawer-open が含まれていない")
	}
	if !strings.Contains(html, `aria-controls="draft-list-drawer"`) {
		t.Error("aria-controls が含まれていない")
	}
	if !strings.Contains(html, `aria-expanded="false"`) {
		t.Error("初期状態を示す aria-expanded=\"false\" が含まれていない")
	}
	if !strings.Contains(html, "下書き一覧") {
		t.Error("ボタンのラベルが含まれていない")
	}
}
