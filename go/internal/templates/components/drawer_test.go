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
			name:      "未知のsideは左端を既定とする",
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

			// ドロワーは既定で非表示で、JSが手がかりにするdata属性を持つ。
			if !strings.Contains(html, `id="draft-list-drawer"`) {
				t.Error("ドロワーのidが含まれていない")
			}
			if !strings.Contains(html, "data-drawer") {
				t.Error("data-drawer属性が含まれていない")
			}
			if !strings.Contains(html, "data-drawer-close") {
				t.Error("背景クリックで閉じるためのdata-drawer-closeが含まれていない")
			}
			if !strings.Contains(html, "hidden") {
				t.Error("既定で非表示にするためのhiddenクラスが含まれていない")
			}
			if !strings.Contains(html, "ドロワーの中身") {
				t.Error("渡したコンテンツが表示されていない")
			}
			if !strings.Contains(html, tt.wantClass) {
				t.Errorf("パネルを%qに寄せるクラスが含まれていない", tt.wantClass)
			}
		})
	}
}

func TestDrawer_PanelBackground(t *testing.T) {
	t.Parallel()

	content := components.Drawer(components.DrawerData{
		ID:      "draft-list-drawer",
		Side:    "left",
		Content: templ.Raw("ドロワーの中身"),
	})

	var buf bytes.Buffer
	if err := content.Render(t.Context(), &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// パネルはbg-backgroundを使い、純白ではなく本体 (body) と同じ色に揃える。
	if !strings.Contains(html, "bg-background") {
		t.Error("パネルの背景を本体と揃えるbg-backgroundクラスが含まれていない")
	}
	if strings.Contains(html, "bg-white") {
		t.Error("パネルにbg-whiteが残っている")
	}
}

func TestDrawer_CloseButton(t *testing.T) {
	t.Parallel()

	content := components.Drawer(components.DrawerData{
		ID:      "draft-list-drawer",
		Side:    "left",
		Content: templ.Raw("ドロワーの中身"),
	})

	var buf bytes.Buffer
	if err := content.Render(t.Context(), &buf); err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// パネルは可視の閉じるボタンを持つ。共通ドロワーJSがdata-drawer-closeで閉じ、
	// ボタンにはアクセシブルなラベルが付き、× アイコンが表示される。
	if !strings.Contains(html, "data-drawer-close") {
		t.Error("クリックで閉じるためのdata-drawer-closeが含まれていない")
	}
	if !strings.Contains(html, `aria-label="閉じる"`) {
		t.Error("閉じるボタンのアクセシブルなラベル (aria-label) が含まれていない")
	}
	// x-regularはインラインSVGとして描画されるため、アイコン名ではなくその固有の
	// パスデータ (× の2本の交差ストローク) で検証する。
	if !strings.Contains(html, "M205.66,194.34") {
		t.Error("閉じるボタンのx-regularアイコンが含まれていない")
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

	// ボタンは対応するドロワーを指し、渡されたラベルを表示し、初期状態は閉じている。
	if !strings.Contains(html, `data-drawer-open="draft-list-drawer"`) {
		t.Error("開く対象を指すdata-drawer-openが含まれていない")
	}
	if !strings.Contains(html, `aria-controls="draft-list-drawer"`) {
		t.Error("aria-controlsが含まれていない")
	}
	if !strings.Contains(html, `aria-expanded="false"`) {
		t.Error("初期状態を示すaria-expanded=\"false\" が含まれていない")
	}
	if !strings.Contains(html, "下書き一覧") {
		t.Error("ボタンのラベルが含まれていない")
	}
}
