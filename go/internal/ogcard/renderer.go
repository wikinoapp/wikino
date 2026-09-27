// Package ogcardはページのog:imageに使うカード画像を描画する
//
// imgproxyの署名付きURLを組み立てる `internal/image` とは別のパッケージにしている。
// 描画処理を `internal/image` に置くと、標準ライブラリの `image` / `image/draw` /
// `image/png` とパッケージ名が衝突するため。
package ogcard

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// カード画像の寸法。OGPで推奨される1200×630に揃える
const (
	Width  = 1200
	Height = 630
)

// レイアウト (寸法の単位はpx、Yは各行のベースライン)
const (
	padding         = 80
	contentMaxWidth = Width - padding*2
	accentBarHeight = 12

	// 上段: スペース名 / トピック名 (1行)
	headerFontSize  = 36
	headerBaselineY = padding + headerFontSize
	headerSeparator = " / "

	// 中段: ページタイトル (最大3行)。行数が少ないときは、3行分の領域の中で上下中央に置く
	titleFontSize   = 64
	titleLineHeight = 90
	titleMaxLines   = 3
	titleTop        = 176

	// 下段: ロゴとサービス名 (右寄せ)。タイトルより目立たないよう小さくし、アクセントバーの近くに置く
	footerFontSize  = 28
	footerBaselineY = Height - accentBarHeight - 32
	footerText      = "Wikino"
	footerLogoSize  = 36
	footerLogoGap   = 12
)

// 配色はWebの配色 (`web/style.css` の `--background` / `--app-primary` など) をsRGBに換算したもの
var (
	backgroundColor = color.RGBA{R: 250, G: 245, B: 241, A: 255}
	primaryColor    = color.RGBA{R: 18, G: 47, B: 52, A: 255}
	mutedColor      = color.RGBA{R: 125, G: 116, B: 112, A: 255}
	accentColor     = color.RGBA{R: 55, G: 143, B: 149, A: 255}
)

// Noto Sans JPのサブセット版 (https://github.com/notofonts/noto-cjk の `Sans/SubsetOTF/JP`)。
// ライセンスは同じディレクトリの `fonts/OFL.txt` を参照
var (
	//go:embed fonts/NotoSansJP-Regular.otf
	regularFontData []byte
	//go:embed fonts/NotoSansJP-Bold.otf
	boldFontData []byte
)

// Cardはカード画像に描く内容
type Card struct {
	SpaceName string
	TopicName string
	PageTitle string
}

// NewPageCardはページのカード画像に描く内容を組み立てる
//
// ページ表示画面 (og:imageのURLに含めるバージョンの計算) とカード画像のエンドポイント
// (描画と正規URLの判定) の両方から呼ぶ。両者の組み立てが食い違うと、HTMLに出したURLが
// 正規URLにならず、SNSのクローラーが毎回302を経由することになる。
func NewPageCard(space *model.Space, topic *model.Topic, page *model.Page) Card {
	var title string
	if page.Title != nil {
		title = *page.Title
	}
	return Card{
		SpaceName: space.Name,
		TopicName: topic.Name,
		PageTitle: title,
	}
}

// Rendererはカード画像を描画する
//
// フォントの解析は構築時に1回だけ行う。解析済みのフォントは読み取り専用で、
// Renderは複数のgoroutineから同時に呼び出せる。
type Renderer struct {
	regular *opentype.Font
	bold    *opentype.Font
	logo    *logo
}

// NewRendererは埋め込んだフォントを解析してRendererを生成する
func NewRenderer() (*Renderer, error) {
	regular, err := opentype.Parse(regularFontData)
	if err != nil {
		return nil, fmt.Errorf("ogcard: 標準のフォントの解析に失敗: %w", err)
	}
	bold, err := opentype.Parse(boldFontData)
	if err != nil {
		return nil, fmt.Errorf("ogcard: 太字のフォントの解析に失敗: %w", err)
	}
	logo, err := parseLogo(logoPathData)
	if err != nil {
		return nil, err
	}
	return &Renderer{regular: regular, bold: bold, logo: logo}, nil
}

// Renderはカード画像を描画し、PNGのバイト列を返す
func (r *Renderer) Render(card Card) ([]byte, error) {
	// opentype.Faceは内部にバッファを持ち並行に使えないため、描画ごとに作る
	headerFace, err := newFace(r.regular, headerFontSize)
	if err != nil {
		return nil, err
	}
	defer func() { _ = headerFace.Close() }()
	titleFace, err := newFace(r.bold, titleFontSize)
	if err != nil {
		return nil, err
	}
	defer func() { _ = titleFace.Close() }()
	footerFace, err := newFace(r.bold, footerFontSize)
	if err != nil {
		return nil, err
	}
	defer func() { _ = footerFace.Close() }()

	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	draw.Draw(img, img.Bounds(), image.NewUniform(backgroundColor), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, Height-accentBarHeight, Width, Height), image.NewUniform(accentColor), image.Point{}, draw.Src)

	maxWidth := fixed.I(contentMaxWidth)

	header := headerText(card, r.hasRegularGlyph)
	for _, line := range wrap(header, maxWidth, 1, measurer(headerFace)) {
		drawText(img, headerFace, mutedColor, line, padding, headerBaselineY)
	}

	title := sanitize(card.PageTitle, r.hasBoldGlyph)
	ascent := titleFace.Metrics().Ascent.Ceil()
	titleLines := wrap(title, maxWidth, titleMaxLines, measurer(titleFace))
	top := titleTop + (titleMaxLines-len(titleLines))*titleLineHeight/2
	for i, line := range titleLines {
		drawText(img, titleFace, primaryColor, line, padding, top+i*titleLineHeight+ascent)
	}

	r.drawFooter(img, footerFace)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("ogcard: PNGのエンコードに失敗: %w", err)
	}
	return buf.Bytes(), nil
}

// drawFooterは、ロゴとサービス名をコンテンツの右端に揃えて描く
//
// ロゴは、上下の中心を文字の大文字の高さの中心に揃える。
func (r *Renderer) drawFooter(img draw.Image, face font.Face) {
	textWidth := font.MeasureString(face, footerText).Ceil()
	textX := Width - padding - textWidth

	capHeight := face.Metrics().CapHeight.Round()
	logoX := textX - footerLogoGap - footerLogoSize
	logoY := footerBaselineY - capHeight/2 - footerLogoSize/2
	r.logo.draw(img, logoX, logoY, footerLogoSize, primaryColor, backgroundColor)

	drawText(img, face, primaryColor, footerText, textX, footerBaselineY)
}

func (r *Renderer) hasRegularGlyph(c rune) bool {
	return hasGlyph(r.regular, c)
}

func (r *Renderer) hasBoldGlyph(c rune) bool {
	return hasGlyph(r.bold, c)
}

// hasGlyphはフォントが文字のグリフを持つかを返す
//
// グリフが無い文字のインデックスは0 (.notdef) になる。
func hasGlyph(f *opentype.Font, c rune) bool {
	index, err := f.GlyphIndex(nil, c)
	return err == nil && index != 0
}

func newFace(f *opentype.Font, size float64) (font.Face, error) {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, fmt.Errorf("ogcard: フォントフェイスの生成に失敗: %w", err)
	}
	return face, nil
}

func measurer(face font.Face) measureFunc {
	return func(s string) fixed.Int26_6 {
		return font.MeasureString(face, s)
	}
}

func drawText(img draw.Image, face font.Face, c color.Color, s string, x, baselineY int) {
	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, baselineY),
	}
	d.DrawString(s)
}
