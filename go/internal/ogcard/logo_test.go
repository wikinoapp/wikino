package ogcard

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestParseLogo(t *testing.T) {
	t.Parallel()

	t.Run("ロゴのパスを解析できる", func(t *testing.T) {
		t.Parallel()

		l, err := parseLogo(logoPathData)
		if err != nil {
			t.Fatalf("parseLogo()のエラー = %v", err)
		}
		if got := l.segments[0].op; got != pathOpMoveTo {
			t.Errorf("最初の命令 = %v、期待値 = MoveTo", got)
		}
		if got := l.segments[len(l.segments)-1].op; got != pathOpClose {
			t.Errorf("最後の命令 = %v、期待値 = ClosePath", got)
		}
	})

	t.Run("Hは直前のY座標を引き継ぐ", func(t *testing.T) {
		t.Parallel()

		l, err := parseLogo("M10 20H30Z")
		if err != nil {
			t.Fatalf("parseLogo()のエラー = %v", err)
		}
		if got, want := l.segments[1].pts[0], (point{30, 20}); got != want {
			t.Errorf("Hの座標 = %v、期待値 = %v", got, want)
		}
	})

	tests := []struct {
		name    string
		d       string
		wantErr string
	}{
		{name: "相対座標の命令", d: "M10 20l5 5Z", wantErr: "未対応の命令"},
		{name: "円弧の命令", d: "M10 20A5 5 0 0 1 30 30Z", wantErr: "未対応の命令"},
		{name: "数値が足りない", d: "M10 20C1 2 3 4", wantErr: "数値が足りない"},
		{name: "数値として読めない", d: "M10 x20Z", wantErr: "解析できない"},
		{name: "Mで始まらない", d: "L10 20Z", wantErr: "M で始まっていない"},
		{name: "空", d: "", wantErr: "M で始まっていない"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseLogo(tt.d)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("parseLogo(%q)のエラー = %v、期待値 = %qを含むエラー", tt.d, err, tt.wantErr)
			}
		})
	}
}

func TestLogoDraw(t *testing.T) {
	t.Parallel()

	l, err := parseLogo(logoPathData)
	if err != nil {
		t.Fatalf("parseLogo()のエラー = %v", err)
	}

	badge := color.RGBA{R: 255, A: 255}
	mark := color.RGBA{B: 255, A: 255}
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	l.draw(img, 10, 10, 64, badge, mark)

	tests := []struct {
		name string
		x, y int
		want color.RGBA
	}{
		{name: "描画範囲の外は塗らない", x: 5, y: 5, want: color.RGBA{}},
		{name: "角丸の外側は塗らない", x: 10, y: 10, want: color.RGBA{}},
		{name: "Wの外側は四角の色で塗る", x: 42, y: 14, want: badge},
		// viewBoxの (128, 250) はWの左の縦画の内側
		{name: "Wの内側はWの色で塗る", x: 10 + 128*64/logoViewBoxSize, y: 10 + 250*64/logoViewBoxSize, want: mark},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := img.RGBAAt(tt.x, tt.y); got != tt.want {
				t.Errorf("(%d, %d)の色 = %v、期待値 = %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}
