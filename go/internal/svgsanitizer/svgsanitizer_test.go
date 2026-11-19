package svgsanitizer

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"
)

// mustSanitizeは、inをサニタイズし、結果がXMLとして読めて2回目のサニタイズで変わらないことを確かめて返す
func mustSanitize(t *testing.T, in string) string {
	t.Helper()
	out, err := Sanitize([]byte(in))
	if err != nil {
		t.Fatalf("Sanitize()のエラー = %v", err)
	}
	assertWellFormedSVG(t, out)
	again, err := Sanitize(out)
	if err != nil {
		t.Fatalf("2回目のSanitize()のエラー = %v", err)
	}
	if !bytes.Equal(again, out) {
		t.Errorf("2回目の結果が変わった\n1回目 = %s\n2回目 = %s", out, again)
	}
	return string(out)
}

// assertWellFormedSVGは、outがXMLとして読め、最上位の要素がSVGの名前空間のsvgであることを確かめる
func assertWellFormedSVG(t *testing.T, out []byte) {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(out))
	root := true
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("結果をXMLとして読めない: %v\n%s", err, out)
		}
		if se, ok := tok.(xml.StartElement); ok && root {
			root = false
			if se.Name.Space != svgNamespace || se.Name.Local != "svg" {
				t.Fatalf("最上位の要素 = %v、期待値 = SVGの名前空間のsvg", se.Name)
			}
		}
	}
}

func svgDoc(body string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">` + body + `</svg>`
}

func TestSanitize_DropsDangerousContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		// forbiddenは、結果に含まれてはいけない文字列
		forbidden []string
		// wantは、結果に含まれるべき文字列
		want []string
	}{
		{
			name:      "scriptを中身ごと落とす",
			in:        svgDoc(`<script>alert(1)</script><script><![CDATA[alert(2)]]></script><rect width="1"/>`),
			forbidden: []string{"script", "alert"},
			want:      []string{`<rect width="1"/>`},
		},
		{
			name:      "接頭辞を付けたSVGの名前空間のscriptも落とす",
			in:        svgDoc(`<x:script xmlns:x="http://www.w3.org/2000/svg">alert(1)</x:script>`),
			forbidden: []string{"script", "alert"},
		},
		{
			name:      "HTMLの名前空間の要素を中身ごと落とす",
			in:        svgDoc(`<h:script xmlns:h="http://www.w3.org/1999/xhtml">alert(1)</h:script>`),
			forbidden: []string{"script", "alert"},
		},
		{
			name:      "大文字のSCRIPTはscript要素として書き出さない",
			in:        svgDoc(`<SCRIPT>x</SCRIPT>`),
			forbidden: []string{"<SCRIPT", "<script"},
		},
		{
			name:      "styleを中身ごと落とす",
			in:        svgDoc(`<style>@import url(https://evil.example/x.css); rect{fill:red}</style><rect/>`),
			forbidden: []string{"style", "evil", "fill:red"},
			want:      []string{"<rect/>"},
		},
		{
			name:      "イベントハンドラーの属性を落とす",
			in:        `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect onclick="alert(2)" onmouseover="x" width="1"/></svg>`,
			forbidden: []string{"onload", "onclick", "onmouseover", "alert"},
			want:      []string{`<rect width="1"/>`},
		},
		{
			name:      "foreignObjectを中身ごと落とす",
			in:        svgDoc(`<foreignObject><div xmlns="http://www.w3.org/1999/xhtml"><iframe src="javascript:alert(1)"/>本文</div></foreignObject>`),
			forbidden: []string{"foreignObject", "iframe", "javascript", "本文"},
		},
		{
			name:      "javascript:のリンクはタグだけを落として図形を残す",
			in:        svgDoc(`<a xlink:href="javascript:alert(1)"><rect width="1"/></a>`),
			forbidden: []string{"<a", "javascript"},
			want:      []string{`<rect width="1"/>`},
		},
		{
			name:      "画像のjavascript:・data:・vbscript:の参照を落とす",
			in:        svgDoc(`<image href="javascript:alert(1)"/><image xlink:href="data:image/png;base64,AAAA"/><image href=" JaVaScRiPt:x"/><image href="&#106;avascript:x"/><image href="vbscript:x"/>`),
			forbidden: []string{"javascript", "JaVaScRiPt", "data:", "vbscript", "href"},
		},
		{
			name:      "画像のhttpとhttpsの参照は残す",
			in:        svgDoc(`<image href="https://example.com/a.png"/><image xlink:href="http://example.com/b.png"/><image href="a.png"/>`),
			forbidden: []string{`"a.png"`},
			want:      []string{`href="https://example.com/a.png"`, `xlink:href="http://example.com/b.png"`},
		},
		{
			name:      "useの外部への参照を落とし、文書の中への参照を残す",
			in:        svgDoc(`<use href="https://evil.example/x.svg#a"/><use xlink:href="data:image/svg+xml,&lt;svg/&gt;#a"/><use xlink:href="#shape"/>`),
			forbidden: []string{"evil", "data:"},
			want:      []string{`<use xlink:href="#shape"/>`},
		},
		{
			name:      "url()で外部を参照する属性を落とす",
			in:        svgDoc(`<rect fill="url(https://evil.example/x.svg#g)" stroke="url( 'http://evil.example/' )" mask="url(data:x)"/>`),
			forbidden: []string{"evil", "fill", "stroke", "mask", "data:"},
		},
		{
			name:      "style属性の外部への参照とエスケープとat-ruleを落とす",
			in:        svgDoc(`<rect style="fill:url(http://evil.example/)"/><circle style="fill:u\72l(http://evil.example/)"/><ellipse style="background:image-set('x.png' 1x)"/><line style="@import 'x'"/>`),
			forbidden: []string{"style", "evil", "image-set", "@import"},
		},
		{
			name:      "animateとsetでhrefを書き換えられないよう落とす",
			in:        svgDoc(`<a><animate attributeName="href" to="javascript:alert(1)"/><set attributeName="href" to="javascript:alert(2)"/><text>x</text></a>`),
			forbidden: []string{"animate", "set ", "attributeName", "javascript"},
			want:      []string{"<text>x</text>"},
		},
		{
			name:      "処理命令とコメントとDOCTYPEを落とす",
			in:        `<?xml version="1.0"?><?xml-stylesheet href="https://evil.example/x.css"?><!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd"><!-- 秘密 -->` + svgDoc(`<!-- 秘密 --><rect/>`),
			forbidden: []string{"<?", "evil", "DOCTYPE", "<!--", "秘密"},
			want:      []string{"<rect/>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := mustSanitize(t, tt.in)
			for _, s := range tt.forbidden {
				if strings.Contains(out, s) {
					t.Errorf("結果に %qが残っている\n%s", s, out)
				}
			}
			for _, s := range tt.want {
				if !strings.Contains(out, s) {
					t.Errorf("結果に %qが無い\n%s", s, out)
				}
			}
		})
	}
}

func TestSanitize_KeepsAllowedContent(t *testing.T) {
	t.Parallel()

	in := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" viewBox="0 0 100 50" width="100" height="50" preserveAspectRatio="xMidYMid meet" inkscape:version="1.4">
<title>図 &amp; 説明</title>
<defs>
<linearGradient id="g1" x1="0" y1="0" x2="1" y2="0" gradientUnits="objectBoundingBox"><stop offset="0" stop-color="#fff" stop-opacity="0.5"/></linearGradient>
<linearGradient id="g2" xlink:href="#g1" gradientTransform="rotate(45)"/>
<clipPath id="c" clipPathUnits="userSpaceOnUse"><rect width="10" height="10"/></clipPath>
<marker id="m" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0 0L10 5L0 10z"/></marker>
<symbol id="s" viewBox="0 0 10 10"><circle cx="5" cy="5" r="5"/></symbol>
</defs>
<g fill="url(#g2)" clip-path="url(#c)" fill-rule="evenodd" font-family="sans-serif" font-size="12" transform="translate(1 2) scale(2)">
<path d="M0 0H10" stroke="#000" stroke-width="2" marker-end="url(#m)" style="fill:rgb(1,2,3);opacity:0.5"/>
<use xlink:href="#s" x="1" y="1" width="5" height="5"/>
<switch><text systemLanguage="ja" x="1" y="2">日本語 &lt;1&gt;</text><text x="1" y="2">English</text></switch>
<text x="1" y="20"><tspan dx="1">文字</tspan></text>
</g>
<sodipodi:namedview xmlns:sodipodi="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd" pagecolor="#fff"/>
</svg>`
	out := mustSanitize(t, in)

	for _, s := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 50" width="100" height="50" preserveAspectRatio="xMidYMid meet">`,
		`<title>図 &amp; 説明</title>`,
		`<linearGradient id="g1" x1="0" y1="0" x2="1" y2="0" gradientUnits="objectBoundingBox"><stop offset="0" stop-color="#fff" stop-opacity="0.5"/></linearGradient>`,
		`<linearGradient id="g2" xlink:href="#g1" gradientTransform="rotate(45)"/>`,
		`<clipPath id="c" clipPathUnits="userSpaceOnUse">`,
		`<marker id="m" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="6" markerHeight="6" orient="auto">`,
		`<symbol id="s" viewBox="0 0 10 10">`,
		`<g fill="url(#g2)" clip-path="url(#c)" fill-rule="evenodd" font-family="sans-serif" font-size="12" transform="translate(1 2) scale(2)">`,
		`<path d="M0 0H10" stroke="#000" stroke-width="2" marker-end="url(#m)" style="fill:rgb(1,2,3);opacity:0.5"/>`,
		`<use xlink:href="#s" x="1" y="1" width="5" height="5"/>`,
		`<switch><text systemLanguage="ja" x="1" y="2">日本語 &lt;1&gt;</text><text x="1" y="2">English</text></switch>`,
		`<text x="1" y="20"><tspan dx="1">文字</tspan></text>`,
	} {
		if !strings.Contains(out, s) {
			t.Errorf("結果に %qが無い\n%s", s, out)
		}
	}
	for _, s := range []string{"inkscape", "sodipodi", "pagecolor", "<?xml"} {
		if strings.Contains(out, s) {
			t.Errorf("結果に %qが残っている\n%s", s, out)
		}
	}
}

func TestSanitize_Namespaces(t *testing.T) {
	t.Parallel()

	t.Run("名前空間を宣言していないSVGにSVGの名前空間を宣言する", func(t *testing.T) {
		t.Parallel()
		out := mustSanitize(t, `<svg viewBox="0 0 1 1"><rect/></svg>`)
		if want := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><rect/></svg>`; out != want {
			t.Errorf("結果 = %s、期待値 = %s", out, want)
		}
	})

	t.Run("子の要素で宣言したXLinkの接頭辞を最上位で宣言し直す", func(t *testing.T) {
		t.Parallel()
		out := mustSanitize(t, `<svg xmlns="http://www.w3.org/2000/svg"><g xmlns:l="http://www.w3.org/1999/xlink"><use l:href="#a"/></g></svg>`)
		want := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><g><use xlink:href="#a"/></g></svg>`
		if out != want {
			t.Errorf("結果 = %s、期待値 = %s", out, want)
		}
	})

	t.Run("XLinkを使わなければXLinkの名前空間を宣言しない", func(t *testing.T) {
		t.Parallel()
		out := mustSanitize(t, svgDoc(`<rect/>`))
		if strings.Contains(out, "xlink") {
			t.Errorf("結果にxlinkが残っている: %s", out)
		}
	})

	t.Run("同じXLinkのhrefを2つ書いたら最初の1つだけを残す", func(t *testing.T) {
		t.Parallel()
		out := mustSanitize(t, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:a="http://www.w3.org/1999/xlink" xmlns:b="http://www.w3.org/1999/xlink"><use a:href="#x" b:href="#y"/></svg>`)
		if !strings.Contains(out, `<use xlink:href="#x"/>`) || strings.Contains(out, "#y") {
			t.Errorf("結果 = %s", out)
		}
	})

	t.Run("XLink以外の名前空間のhrefを落とす", func(t *testing.T) {
		t.Parallel()
		out := mustSanitize(t, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="https://evil.example/"><use xlink:href="#x"/></svg>`)
		if strings.Contains(out, "href") {
			t.Errorf("結果にhrefが残っている: %s", out)
		}
	})
}

func TestSanitize_Entities(t *testing.T) {
	t.Parallel()

	t.Run("内部サブセットの実体で書いた名前空間を展開する", func(t *testing.T) {
		t.Parallel()
		in := `<?xml version="1.0"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd" [
	<!ENTITY ns_svg "http://www.w3.org/2000/svg">
	<!ENTITY ns_xlink 'http://www.w3.org/1999/xlink'>
]>
<svg xmlns="&ns_svg;" xmlns:xlink="&ns_xlink;"><use xlink:href="#a"/></svg>`
		out := mustSanitize(t, in)
		want := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><use xlink:href="#a"/></svg>`
		if out != want {
			t.Errorf("結果 = %s、期待値 = %s", out, want)
		}
	})

	t.Run("Illustratorの編集機能を保持したSVGのAdobeの名前空間を展開し、Adobeの要素と属性を落とす", func(t *testing.T) {
		t.Parallel()
		in := `<?xml version="1.0" encoding="utf-8"?>
<!-- Generator: Adobe Illustrator 16.0.0, SVG Export Plug-In . SVG Version: 6.00 Build 0)  -->
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd" [
	<!ENTITY ns_extend "http://ns.adobe.com/Extensibility/1.0/">
	<!ENTITY ns_ai "http://ns.adobe.com/AdobeIllustrator/10.0/">
	<!ENTITY ns_graphs "http://ns.adobe.com/Graphs/1.0/">
	<!ENTITY ns_custom "http://ns.adobe.com/GenericCustomNamespace/1.0/">
]>
<svg version="1.1" id="Layer_1" xmlns:x="&ns_extend;" xmlns:i="&ns_ai;" xmlns:graph="&ns_graphs;" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100" xml:space="preserve">
<switch>
	<foreignObject requiredExtensions="&ns_ai;" x="0" y="0" width="1" height="1">
		<i:pgfRef xlink:href="#adobe_illustrator_pgf"></i:pgfRef>
	</foreignObject>
	<g i:extraneous="self"><rect x="10" y="10" width="80" height="80" fill="#FF0000"/></g>
</switch>
<i:pgf id="adobe_illustrator_pgf">eJzt</i:pgf>
</svg>`
		out := mustSanitize(t, in)
		for _, s := range []string{
			`<svg xmlns="http://www.w3.org/2000/svg" id="Layer_1" viewBox="0 0 100 100">`,
			`<g><rect x="10" y="10" width="80" height="80" fill="#FF0000"/></g>`,
		} {
			if !strings.Contains(out, s) {
				t.Errorf("結果に %qが無い\n%s", s, out)
			}
		}
		for _, s := range []string{"adobe", "foreignObject", "pgf", "extraneous", "xlink", "xml:space"} {
			if strings.Contains(out, s) {
				t.Errorf("結果に %qが残っている\n%s", s, out)
			}
		}
	})

	t.Run("名前空間のURIをすべて実体で書いたIllustratorのSVGを展開する", func(t *testing.T) {
		t.Parallel()
		in := `<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd" [
	<!ENTITY ns_ai "http://ns.adobe.com/AdobeIllustrator/10.0/">
	<!ENTITY ns_svg "http://www.w3.org/2000/svg">
	<!ENTITY ns_xlink "http://www.w3.org/1999/xlink">
]>
<svg version="1.1" xmlns:i="&ns_ai;" xmlns="&ns_svg;" xmlns:xlink="&ns_xlink;" viewBox="0 0 100 100"><rect width="1" i:knockout="Off"/></svg>`
		out := mustSanitize(t, in)
		want := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="1"/></svg>`
		if out != want {
			t.Errorf("結果 = %s、期待値 = %s", out, want)
		}
	})

	t.Run("外部の実体を読まずにエラーにする", func(t *testing.T) {
		t.Parallel()
		in := `<!DOCTYPE svg [<!ENTITY x SYSTEM "file:///etc/passwd">]><svg xmlns="http://www.w3.org/2000/svg"><text>&x;</text></svg>`
		if _, err := Sanitize([]byte(in)); !errors.Is(err, ErrMalformed) {
			t.Errorf("エラー = %v、期待値 = ErrMalformed", err)
		}
	})

	t.Run("実体参照を含む一般実体は拒否する", func(t *testing.T) {
		t.Parallel()
		in := `<!DOCTYPE svg [<!ENTITY a "aaaa"><!ENTITY b "&a;&a;&a;">]><svg xmlns="http://www.w3.org/2000/svg"><text>&b;</text></svg>`
		if out, err := Sanitize([]byte(in)); !errors.Is(err, ErrMalformed) || out != nil {
			t.Errorf("結果 = %s、エラー = %v、期待値 = nil, ErrMalformed", out, err)
		}
	})

	t.Run("名前空間以外の値を持つ一般実体は拒否する", func(t *testing.T) {
		t.Parallel()

		longValue := strings.Repeat("A", 16384)
		repeated := strings.Repeat("&repeated;", 512)
		tests := []struct {
			name  string
			value string
			body  string
		}{
			{name: "短いテキストの実体", value: "text", body: `<text>&repeated;</text>`},
			{name: "長い実体を文字データで繰り返す", value: longValue, body: `<text>` + repeated + `</text>`},
			{name: "長い実体を属性で繰り返す", value: longValue, body: `<rect id="` + repeated + `"/>`},
			{name: "SVGの名前空間のURIに長い末尾が付く", value: svgNamespace + longValue, body: `<text>&repeated;</text>`},
			{name: "XLinkの名前空間のURIに末尾が付く", value: xlinkNamespace + "/extra", body: `<text>&repeated;</text>`},
			{name: "Adobeの名前空間のURIに長い末尾が付く", value: "http://ns.adobe.com/" + longValue, body: `<text>` + repeated + `</text>`},
			{name: "Adobeの名前空間のURIの形で長さの上限を超える", value: "http://ns.adobe.com/" + strings.Repeat("a", maxAdobeNamespaceLength-len("http://ns.adobe.com/")+1), body: `<text>&repeated;</text>`},
			{name: "Adobeの名前空間のURIにURIのパスに使わない文字を含む", value: "http://ns.adobe.com/a b", body: `<text>&repeated;</text>`},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				in := `<!DOCTYPE svg [<!ENTITY repeated "` + tt.value + `">]>` + svgDoc(tt.body)
				out, err := Sanitize([]byte(in))
				if !errors.Is(err, ErrMalformed) || out != nil {
					t.Errorf("結果の長さ = %d、エラー = %v、期待値 = nil, ErrMalformed", len(out), err)
				}
			})
		}
	})

	t.Run("参照しない一般実体の宣言は落とす", func(t *testing.T) {
		t.Parallel()
		in := `<!DOCTYPE svg [<!ENTITY unused "` + strings.Repeat("A", 16384) + `">]>` + svgDoc(`<rect/>`)
		out := mustSanitize(t, in)
		if want := `<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`; out != want {
			t.Errorf("結果 = %s、期待値 = %s", out, want)
		}
	})
}

func TestSanitize_ConvertsCharsetToUTF8(t *testing.T) {
	t.Parallel()

	in := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><svg xmlns=\"http://www.w3.org/2000/svg\"><text>caf\xe9</text></svg>"
	out := mustSanitize(t, in)
	if !strings.Contains(out, "<text>café</text>") {
		t.Errorf("結果 = %s", out)
	}
}

func TestSanitize_EscapesAttributeValues(t *testing.T) {
	t.Parallel()

	out := mustSanitize(t, svgDoc(`<text font-family="&quot;A&amp;B&quot; &lt;x&gt;&#9;&#10;">x</text>`))
	if want := `font-family="&quot;A&amp;B&quot; &lt;x&gt;&#x9;&#xA;"`; !strings.Contains(out, want) {
		t.Errorf("結果 = %s、期待値に %qを含む", out, want)
	}
}

func TestSanitize_StopsAtRootEnd(t *testing.T) {
	t.Parallel()

	out := mustSanitize(t, `<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg><script>alert(1)</script>`)
	if want := `<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`; out != want {
		t.Errorf("結果 = %s、期待値 = %s", out, want)
	}
}

func TestSanitize_Malformed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{name: "空", in: ""},
		{name: "XMLではない", in: "\x89PNG\r\n\x1a\n"},
		{name: "最上位がsvgではない", in: `<html xmlns="http://www.w3.org/1999/xhtml"><svg/></html>`},
		{name: "最上位がSVG以外の名前空間のsvg", in: `<svg xmlns="https://example.com/"/>`},
		{name: "閉じていない", in: `<svg xmlns="http://www.w3.org/2000/svg"><rect>`},
		{name: "タグが対応しない", in: `<svg xmlns="http://www.w3.org/2000/svg"><g></rect></svg>`},
		{name: "未定義の実体", in: `<svg xmlns="http://www.w3.org/2000/svg"><text>&nbsp;</text></svg>`},
		{name: "落とす要素の中が壊れている", in: `<svg xmlns="http://www.w3.org/2000/svg"><script><a></script></svg>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := Sanitize([]byte(tt.in))
			if !errors.Is(err, ErrMalformed) {
				t.Errorf("エラー = %v、期待値 = ErrMalformed (結果 = %s)", err, out)
			}
		})
	}
}
