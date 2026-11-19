package imagemeta

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// テストで、取り除かれたことを確かめるメタデータの中身
var (
	secretGPS     = []byte("GPS-35.6812N-139.7671E")
	secretXMP     = []byte("<x:xmpmeta>XMP-SECRET</x:xmpmeta>")
	secretComment = []byte("COMMENT-SECRET")
	secretIPTC    = []byte("IPTC-SECRET")
	secretTrailer = []byte("ftypmp42TRAILER-SECRET")
	fakeICC       = []byte("FAKE-ICC-PROFILE-DATA")
)

// exifTIFFは、向きorientation (0なら持たない) と、撮影位置に見立てた文字列を持つEXIFのTIFFの構造を、
// バイト順orderで作る
func exifTIFF(order binary.AppendByteOrder, orientation uint16) []byte {
	entries := uint16(1)
	if orientation != 0 {
		entries++
	}
	b := make([]byte, 0, 64)
	if order == binary.AppendByteOrder(binary.LittleEndian) {
		b = append(b, 'I', 'I')
	} else {
		b = append(b, 'M', 'M')
	}
	b = order.AppendUint16(b, 42)
	b = order.AppendUint32(b, 8)
	b = order.AppendUint16(b, entries)
	dataOffset := uint32(8 + 2 + 12*int(entries) + 4)
	// 機種名のタグに撮影位置に見立てた文字列を置く
	b = order.AppendUint16(b, 0x010F)
	b = order.AppendUint16(b, 2)
	b = order.AppendUint32(b, uint32(len(secretGPS)))
	b = order.AppendUint32(b, dataOffset)
	if orientation != 0 {
		b = order.AppendUint16(b, orientationTag)
		b = order.AppendUint16(b, tiffTypeShort)
		b = order.AppendUint32(b, 1)
		b = order.AppendUint16(b, orientation)
		b = append(b, 0, 0)
	}
	b = order.AppendUint32(b, 0)
	return append(b, secretGPS...)
}

// testImageは、画素ごとに色の違う小さな画像を返す
func testImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: uint8((x + y) * 7), A: 255})
		}
	}
	return img
}

// readTestdataは、testdataのファイルを読む
func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("%sの読み込みに失敗: %v", name, err)
	}
	return data
}

// mustStripは、Stripを呼び、エラーなら失敗にする
func mustStrip(t *testing.T, data []byte) []byte {
	t.Helper()
	out, err := Strip(data)
	if err != nil {
		t.Fatalf("Strip()のエラー = %v", err)
	}
	return out
}

// assertNotContainsは、outにsecretsのどれも含まれないことを確かめる
func assertNotContains(t *testing.T, out []byte, secrets ...[]byte) {
	t.Helper()
	for _, s := range secrets {
		if bytes.Contains(out, s) {
			t.Errorf("結果に %q が残っています", s)
		}
	}
}

// assertIdempotentは、結果をもう一度処理しても同じバイト列が返ることを確かめる
func assertIdempotent(t *testing.T, out []byte) {
	t.Helper()
	again := mustStrip(t, out)
	if !bytes.Equal(again, out) {
		t.Error("2回目の処理で結果が変わりました")
	}
}

// assertSamePixelsは、2つの画像の画素がすべて同じことを確かめる
func assertSamePixels(t *testing.T, want, got image.Image) {
	t.Helper()
	if want.Bounds() != got.Bounds() {
		t.Fatalf("画像の範囲 = %v、期待値 = %v", got.Bounds(), want.Bounds())
	}
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if want.At(x, y) != got.At(x, y) {
				t.Fatalf("(%d, %d) の画素 = %v、期待値 = %v", x, y, got.At(x, y), want.At(x, y))
			}
		}
	}
}

// pngChunkBytesは、種類typと中身bodyのPNGのチャンクを作る
func pngChunkBytes(typ string, body []byte) []byte {
	b := binary.BigEndian.AppendUint32(nil, uint32(len(body)))
	b = append(b, typ...)
	b = append(b, body...)
	return binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(b[4:]))
}

// webpChunkBytesは、種類fourCCと中身bodyのWebPのチャンクを、詰め物を付けて作る
func webpChunkBytes(fourCC string, body []byte) []byte {
	b := append([]byte(fourCC), binary.LittleEndian.AppendUint32(nil, uint32(len(body)))...)
	b = append(b, body...)
	if len(body)%2 == 1 {
		b = append(b, 0)
	}
	return b
}

// webpRIFFは、チャンクchunksを並べたWebPのファイルを作る
func webpRIFF(chunks ...[]byte) []byte {
	body := bytes.Join(chunks, nil)
	b := append([]byte("RIFF"), binary.LittleEndian.AppendUint32(nil, uint32(4+len(body)))...)
	b = append(b, "WEBP"...)
	return append(b, body...)
}

// vp8xBodyは、フラグflagsと大きさw×hを持つVP8Xのチャンクの中身を作る
func vp8xBody(flags byte, w, h int) []byte {
	b := []byte{flags, 0, 0, 0}
	b = append(b, byte(w-1), byte((w-1)>>8), byte((w-1)>>16))
	return append(b, byte(h-1), byte((h-1)>>8), byte((h-1)>>16))
}
