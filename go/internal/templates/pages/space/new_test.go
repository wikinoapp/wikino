package space_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
)

func TestNew_Autofocus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields []string
		want   string
	}{
		{name: "初回表示は識別子", want: "identifier"},
		{name: "識別子だけが不正", fields: []string{"identifier"}, want: "identifier"},
		{name: "名前だけが不正", fields: []string{"name"}, want: "name"},
		{name: "両方が不正なら最初の識別子", fields: []string{"identifier", "name"}, want: "identifier"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ve *model.ValidationError
			if len(tt.fields) > 0 {
				ve = model.NewValidationError()
				for _, field := range tt.fields {
					ve.AddField(field, "入力を確認してください")
				}
			}
			doc := renderNew(t, i18n.LangJa, spacepages.NewData{FormErrors: ve})
			var focused []string
			for _, input := range elements(doc, "input") {
				if _, ok := attribute(input, "autofocus"); ok {
					id, _ := attribute(input, "id")
					focused = append(focused, id)
				}
			}
			if len(focused) != 1 || focused[0] != tt.want {
				t.Errorf("autofocusを持つ入力欄 = %v、期待値 = [%s]", focused, tt.want)
			}
		})
	}
}

func TestNew_RequiredLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "日本語", locale: i18n.LangJa, want: "必須"},
		{name: "英語", locale: i18n.LangEn, want: "Required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := renderNew(t, tt.locale, spacepages.NewData{})
			for _, field := range []string{"identifier", "name"} {
				var label *html.Node
				for _, node := range elements(doc, "label") {
					if target, _ := attribute(node, "for"); target == field {
						label = node
					}
				}
				if label == nil {
					t.Fatalf("%sに関連付いたラベルが無い", field)
				}
				found := false
				for _, span := range elements(label, "span") {
					if class, _ := attribute(span, "class"); class == "badge" && span.FirstChild != nil && strings.TrimSpace(span.FirstChild.Data) == tt.want {
						found = true
					}
				}
				if !found {
					t.Errorf("%sのラベルに必須バッジ%qが無い", field, tt.want)
				}
				required := false
				for _, input := range elements(doc, "input") {
					if id, _ := attribute(input, "id"); id == field {
						_, required = attribute(input, "required")
					}
				}
				if !required {
					t.Errorf("%sの入力欄にrequired属性が無い", field)
				}
			}
		})
	}
}

// renderNewは属性の順序に依存せず確認できるよう、フォームをHTMLの木として読み込む。
func renderNew(t *testing.T, locale string, data spacepages.NewData) *html.Node {
	t.Helper()

	var buf bytes.Buffer
	if err := spacepages.New(data).Render(i18n.SetLocale(context.Background(), locale), &buf); err != nil {
		t.Fatalf("フォームの描画に失敗: %v", err)
	}
	doc, err := html.Parse(&buf)
	if err != nil {
		t.Fatalf("フォームのHTML解析に失敗: %v", err)
	}
	return doc
}

func elements(root *html.Node, tag string) []*html.Node {
	var result []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			result = append(result, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return result
}

func attribute(node *html.Node, name string) (string, bool) {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val, true
		}
	}
	return "", false
}
