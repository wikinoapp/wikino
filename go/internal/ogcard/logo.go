package ogcard

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"

	"golang.org/x/image/vector"
)

// logoPathDataはWikinoのロゴ (`static/images/icon.svg` の「W」) のSVGパスデータ
//
// ロゴを変えたときは、ここも同じパスデータに差し替え、`designRevision` を上げる。
const logoPathData = "M329.2 353.4C331.867 339.267 334.267 327 336.4 316.6C338.533 305.933 340.4 296.067 342 287C343.6 277.933 345.067 269 346.4 260.2C348 251.133 349.6 241.133 351.2 230.2C353.067 219 355.067 206.333 357.2 192.2C359.6 177.8 362.4 160.6 365.6 140.6C366.4 135.533 367.733 132.333 369.6 131C371.467 129.4 376.133 128.6 383.6 128.6C386.533 128.6 390.933 128.867 396.8 129.4C402.667 129.667 408.533 130.867 414.4 133C420.267 134.867 425.333 137.933 429.6 142.2C434.133 146.2 436.4 151.933 436.4 159.4C436.4 163.667 435.6 170.333 434 179.4C432.4 188.2 430.133 198.467 427.2 210.2C424.533 221.667 421.333 234.2 417.6 247.8C414.133 261.4 410.533 275.133 406.8 289C403.067 302.867 399.2 316.333 395.2 329.4C391.467 342.467 387.867 354.333 384.4 365C381.2 375.4 378.267 384.2 375.6 391.4C372.933 398.6 371.067 403.267 370 405.4C368.4 408.333 366.133 410.333 363.2 411.4C360.267 412.467 357.333 413 354.4 413H316.4C309.467 413 303.6 411.667 298.8 409C294.267 406.333 290.4 402.867 287.2 398.6C284.267 394.333 281.867 389.533 280 384.2C278.4 378.867 277.333 373.4 276.8 367.8L256.8 188.2C255.2 199.933 253.333 213.133 251.2 227.8C249.333 242.467 247.2 257.4 244.8 272.6C242.667 287.8 240.533 302.733 238.4 317.4C236.267 332.067 234.133 345.533 232 357.8C229.867 369.8 227.867 380.067 226 388.6C224.133 397.133 222.533 402.733 221.2 405.4C219.867 408.333 217.733 410.333 214.8 411.4C211.867 412.467 208.8 413 205.6 413H167.6C160.933 413 155.333 411.8 150.8 409.4C146.533 406.733 142.933 403.267 140 399C137.067 394.733 134.667 389.933 132.8 384.6C130.933 379 129.333 373.4 128 367.8C117.067 324.333 108.133 288.6 101.2 260.6C94.5333 232.6 89.2 210.2 85.2 193.4C81.2 176.333 78.4 163.933 76.8 156.2C75.4667 148.2 74.8 142.467 74.8 139C74.8 134.467 76.1333 131.533 78.8 130.2C81.7333 128.867 87.7333 128.2 96.8 128.2C108 128.2 117.333 129 124.8 130.6C132.533 132.2 138.8 134.733 143.6 138.2C148.4 141.4 152 145.8 154.4 151.4C157.067 156.733 158.8 163.267 159.6 171C159.6 172.067 159.733 173.667 160 175.8C160.267 177.667 160.533 180.867 160.8 185.4C161.333 189.933 162 196.2 162.8 204.2C163.867 211.933 165.067 222.333 166.4 235.4C168 248.467 169.867 264.733 172 284.2C174.4 303.4 177.2 326.467 180.4 353.4C183.333 337.133 185.6 324.067 187.2 314.2C189.067 304.067 190.667 295.267 192 287.8C193.333 280.067 194.533 272.733 195.6 265.8C196.667 258.6 197.867 249.8 199.2 239.4C200.8 228.733 202.533 215.4 204.4 199.4C206.533 183.4 209.2 162.6 212.4 137C212.667 135.133 213.733 133.667 215.6 132.6C217.733 131.533 220.267 130.733 223.2 130.2C226.133 129.667 229.333 129.4 232.8 129.4C236.533 129.133 240 129 243.2 129C253.867 129 262.933 129.667 270.4 131C277.867 132.067 284 134.067 288.8 137C293.867 139.933 297.867 143.933 300.8 149C303.733 153.8 306.133 159.8 308 167C308.8 170.467 309.733 175.267 310.8 181.4C311.867 187.267 313.067 196.733 314.4 209.8C316 222.867 317.867 240.867 320 263.8C322.4 286.467 325.467 316.333 329.2 353.4Z"

// pathOpはパスを構成する描画命令の種類
type pathOp int

const (
	pathOpMoveTo pathOp = iota
	pathOpLineTo
	pathOpCubeTo
	pathOpClose
)

// pathSegmentはパスの描画命令1つ。座標はSVGのユーザー座標系の絶対座標
type pathSegment struct {
	op  pathOp
	pts []point
}

type point struct {
	x, y float64
}

// logoViewBoxSizeはロゴのSVGのviewBoxの1辺 (`viewBox="0 0 512 512"`)
const logoViewBoxSize = 512

// logoCornerRatioは、ロゴの角丸の半径の1辺に対する比率
//
// Webでは60pxのロゴを `rounded-xl` (12px) の角丸で囲んでおり、その比率に揃える。
const logoCornerRatio = 0.2

// kappaは、4分の1の円弧を3次ベジェ曲線で近似するときの制御点の比率
const kappa = 0.5522847498

// logoはロゴのパス
type logo struct {
	segments []pathSegment
}

// parseLogoはSVGのパスデータを解析する
//
// ロゴのパスが使う絶対座標の `M` / `L` / `H` / `C` / `Z` だけに対応する。
// それ以外の命令 (相対座標や円弧など) はエラーにし、ロゴを差し替えたときに
// 描画が崩れたまま気付かないことを防ぐ。
func parseLogo(d string) (*logo, error) {
	tokens := tokenizePath(d)
	var segments []pathSegment
	var current point
	var nums []float64
	i := 0

	// 命令に続く数値をn個読む
	readNums := func(cmd string, n int) error {
		nums = nums[:0]
		for range n {
			if i >= len(tokens) {
				return fmt.Errorf("ogcard: ロゴのパスの %s に続く数値が足りない", cmd)
			}
			v, err := strconv.ParseFloat(tokens[i], 64)
			if err != nil {
				return fmt.Errorf("ogcard: ロゴのパスの %s に続く数値 %q を解析できない: %w", cmd, tokens[i], err)
			}
			nums = append(nums, v)
			i++
		}
		return nil
	}

	for i < len(tokens) {
		cmd := tokens[i]
		i++
		switch cmd {
		case "M", "L":
			if err := readNums(cmd, 2); err != nil {
				return nil, err
			}
			current = point{nums[0], nums[1]}
			op := pathOpLineTo
			if cmd == "M" {
				op = pathOpMoveTo
			}
			segments = append(segments, pathSegment{op: op, pts: []point{current}})
		case "H":
			if err := readNums(cmd, 1); err != nil {
				return nil, err
			}
			current = point{nums[0], current.y}
			segments = append(segments, pathSegment{op: pathOpLineTo, pts: []point{current}})
		case "C":
			if err := readNums(cmd, 6); err != nil {
				return nil, err
			}
			pts := []point{{nums[0], nums[1]}, {nums[2], nums[3]}, {nums[4], nums[5]}}
			current = pts[2]
			segments = append(segments, pathSegment{op: pathOpCubeTo, pts: pts})
		case "Z":
			segments = append(segments, pathSegment{op: pathOpClose})
		default:
			return nil, fmt.Errorf("ogcard: ロゴのパスに未対応の命令 %q がある", cmd)
		}
	}

	if len(segments) == 0 || segments[0].op != pathOpMoveTo {
		return nil, fmt.Errorf("ogcard: ロゴのパスが M で始まっていない")
	}

	return &logo{segments: segments}, nil
}

// tokenizePathはパスデータを命令と数値のトークンに分ける
//
// 命令の文字は数値と区切りなしで続くため (`M329.2 353.4C331.867 ...`)、
// 英字の前後で区切る。`e` は数値の指数表記に使われうるため命令として扱わない。
func tokenizePath(d string) []string {
	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			tokens = append(tokens, b.String())
			b.Reset()
		}
	}
	for _, r := range d {
		switch {
		case r == ' ' || r == ',' || r == '\n' || r == '\t':
			flush()
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z' && r != 'e'):
			flush()
			tokens = append(tokens, string(r))
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return tokens
}

// drawは、Webのロゴ (`bg-primary rounded-xl` の角丸の四角に `fill-primary-foreground` の「W」) と
// 同じ見た目のロゴを、左上を (x, y) に合わせて1辺sizeで描く
func (l *logo) draw(dst draw.Image, x, y, size int, badge, mark color.Color) {
	rect := image.Rect(x, y, x+size, y+size)
	s := float32(size)

	bg := vector.NewRasterizer(size, size)
	addRoundedSquare(bg, s, s*logoCornerRatio)
	bg.Draw(dst, rect, image.NewUniform(badge), image.Point{})

	scale := s / logoViewBoxSize
	tr := func(p point) (float32, float32) {
		return float32(p.x) * scale, float32(p.y) * scale
	}
	fg := vector.NewRasterizer(size, size)
	for _, seg := range l.segments {
		switch seg.op {
		case pathOpMoveTo:
			fg.MoveTo(tr(seg.pts[0]))
		case pathOpLineTo:
			fg.LineTo(tr(seg.pts[0]))
		case pathOpCubeTo:
			bx, by := tr(seg.pts[0])
			cx, cy := tr(seg.pts[1])
			dx, dy := tr(seg.pts[2])
			fg.CubeTo(bx, by, cx, cy, dx, dy)
		case pathOpClose:
			fg.ClosePath()
		}
	}
	fg.Draw(dst, rect, image.NewUniform(mark), image.Point{})
}

// addRoundedSquareは、1辺s・角丸の半径rの正方形のパスを加える
func addRoundedSquare(z *vector.Rasterizer, s, r float32) {
	k := r * kappa
	z.MoveTo(r, 0)
	z.LineTo(s-r, 0)
	z.CubeTo(s-r+k, 0, s, r-k, s, r)
	z.LineTo(s, s-r)
	z.CubeTo(s, s-r+k, s-r+k, s, s-r, s)
	z.LineTo(r, s)
	z.CubeTo(r-k, s, 0, s-r+k, 0, s-r)
	z.LineTo(0, r)
	z.CubeTo(0, r-k, r-k, 0, r, 0)
	z.ClosePath()
}
