package ogcard

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"sync"
	"testing"
)

func newTestRenderer(t testing.TB) *Renderer {
	t.Helper()

	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer()のエラー = %v", err)
	}
	return r
}

func TestRendererRender(t *testing.T) {
	t.Parallel()

	r := newTestRenderer(t)

	tests := []struct {
		name string
		card Card
	}{
		{
			name: "短いタイトル",
			card: Card{SpaceName: "Wikino開発", TopicName: "設計", PageTitle: "はじめに"},
		},
		{
			name: "最大行数を超える長いタイトル",
			card: Card{SpaceName: "スペース", TopicName: "トピック", PageTitle: strings.Repeat("吾輩は猫である。名前はまだ無い。", 10)},
		},
		{
			name: "フォントに無い文字を含むタイトル",
			card: Card{SpaceName: "日記🎉", TopicName: "2026年", PageTitle: "今日は🍣を食べた👍"},
		},
		{
			name: "フォントに無い文字だけのタイトル",
			card: Card{SpaceName: "スペース", TopicName: "トピック", PageTitle: "🎉🎉🎉"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := r.Render(tt.card)
			if err != nil {
				t.Fatalf("Render()のエラー = %v", err)
			}

			cfg, err := png.DecodeConfig(bytes.NewReader(got))
			if err != nil {
				t.Fatalf("PNGとしてのデコードのエラー = %v", err)
			}
			if cfg.Width != Width || cfg.Height != Height {
				t.Errorf("画像の寸法 = %d×%d、期待値 = %d×%d", cfg.Width, cfg.Height, Width, Height)
			}
		})
	}
}

func TestRendererRender_Content(t *testing.T) {
	t.Parallel()

	r := newTestRenderer(t)
	card := Card{SpaceName: "スペース", TopicName: "トピック", PageTitle: "ページタイトル"}
	base := renderTestImage(t, r, card)

	// 文字が描かれる領域だけを見る。下端のアクセントバーは対象から外す。
	headerRegion := image.Rect(padding, padding-20, Width-padding, titleTop-20)
	titleRegion := image.Rect(padding, titleTop, Width-padding, footerBaselineY-60)
	footerRegion := image.Rect(padding, footerBaselineY-60, Width-padding, Height-accentBarHeight)

	for _, tt := range []struct {
		name   string
		region image.Rectangle
	}{
		{name: "ヘッダー", region: headerRegion},
		{name: "タイトル", region: titleRegion},
		{name: "フッター", region: footerRegion},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !regionHasInk(base, tt.region) {
				t.Errorf("%sの領域に文字が描かれていない", tt.name)
			}
		})
	}

	t.Run("フッターは右寄せ", func(t *testing.T) {
		t.Parallel()
		leftHalf := image.Rect(0, footerRegion.Min.Y, Width/2, footerRegion.Max.Y)
		if regionHasInk(base, leftHalf) {
			t.Error("フッターの左半分に描画がある")
		}
	})

	for _, tt := range []struct {
		name   string
		card   Card
		region image.Rectangle
	}{
		{name: "スペース名", card: Card{SpaceName: "別のスペース", TopicName: card.TopicName, PageTitle: card.PageTitle}, region: headerRegion},
		{name: "トピック名", card: Card{SpaceName: card.SpaceName, TopicName: "別のトピック", PageTitle: card.PageTitle}, region: headerRegion},
		{name: "ページタイトル", card: Card{SpaceName: card.SpaceName, TopicName: card.TopicName, PageTitle: "別のタイトル"}, region: titleRegion},
	} {
		t.Run(tt.name+"の変更", func(t *testing.T) {
			t.Parallel()
			changed := renderTestImage(t, r, tt.card)
			if !regionDiffers(base, changed, tt.region) {
				t.Errorf("%sを変えても該当領域の描画が変わらない", tt.name)
			}
		})
	}
}

func renderTestImage(t *testing.T, r *Renderer, card Card) image.Image {
	t.Helper()

	b, err := r.Render(card)
	if err != nil {
		t.Fatalf("Render()のエラー = %v", err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("PNGとしてのデコードのエラー = %v", err)
	}
	return img
}

func regionHasInk(img image.Image, region image.Rectangle) bool {
	region = region.Intersect(img.Bounds())
	backgroundR, backgroundG, backgroundB, backgroundA := backgroundColor.RGBA()
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if r != backgroundR || g != backgroundG || b != backgroundB || a != backgroundA {
				return true
			}
		}
	}
	return false
}

func regionDiffers(a, b image.Image, region image.Rectangle) bool {
	region = region.Intersect(a.Bounds()).Intersect(b.Bounds())
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb || aa != ba {
				return true
			}
		}
	}
	return false
}

func TestRendererRender_Deterministic(t *testing.T) {
	t.Parallel()

	r := newTestRenderer(t)
	card := Card{SpaceName: "スペース", TopicName: "トピック", PageTitle: "同じ内容なら同じ画像"}

	// 並行に描画しても互いに影響せず、同じ画像になることを確かめる
	const n = 8
	results := make([][]byte, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			b, err := r.Render(card)
			if err != nil {
				t.Errorf("Render()のエラー = %v", err)
				return
			}
			results[i] = b
		})
	}
	wg.Wait()

	for i := 1; i < n; i++ {
		if !bytes.Equal(results[0], results[i]) {
			t.Fatalf("%d回目の描画結果が1回目と異なる", i+1)
		}
	}
}

func TestHasGlyph(t *testing.T) {
	t.Parallel()

	r := newTestRenderer(t)

	tests := []struct {
		name string
		char rune
		want bool
	}{
		{name: "ASCII", char: 'A', want: true},
		{name: "ひらがな", char: 'あ', want: true},
		{name: "カタカナ", char: 'ア', want: true},
		{name: "漢字", char: '猫', want: true},
		{name: "全角記号", char: '「', want: true},
		{name: "省略記号", char: '…', want: true},
		{name: "カラー絵文字", char: '🎉', want: false},
		{name: "異体字セレクタ", char: '️', want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := r.hasBoldGlyph(tt.char); got != tt.want {
				t.Errorf("hasBoldGlyph(%q) = %v、期待値 = %v", tt.char, got, tt.want)
			}
			if got := r.hasRegularGlyph(tt.char); got != tt.want {
				t.Errorf("hasRegularGlyph(%q) = %v、期待値 = %v", tt.char, got, tt.want)
			}
		})
	}
}

func BenchmarkRendererRender(b *testing.B) {
	r := newTestRenderer(b)
	card := Card{
		SpaceName: "Wikino開発",
		TopicName: "設計",
		PageTitle: "吾輩は猫である。名前はまだ無い。どこで生れたかとんと見当がつかぬ。何でも薄暗いじめじめした所でニャーニャー泣いていた事だけは記憶している。",
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := r.Render(card); err != nil {
			b.Fatalf("Render()のエラー = %v", err)
		}
	}
}

func BenchmarkNewRenderer(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := NewRenderer(); err != nil {
			b.Fatalf("NewRenderer()のエラー = %v", err)
		}
	}
}
