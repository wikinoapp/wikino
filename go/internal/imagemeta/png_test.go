package imagemeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image/png"
	"testing"
)

// encodeTestPNGは、メタデータの無いPNGを作る
func encodeTestPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, testImage(8, 6)); err != nil {
		t.Fatalf("PNGの作成に失敗: %v", err)
	}
	return buf.Bytes()
}

// pngIHDREndは、PNGのシグネチャとIHDRのチャンクの終わりの位置
const pngIHDREnd = 8 + 12 + 13

// withPNGChunksは、PNGのIHDRの直後にチャンクchunksを差し込む
func withPNGChunks(base []byte, chunks ...[]byte) []byte {
	b := append([]byte{}, base[:pngIHDREnd]...)
	for _, c := range chunks {
		b = append(b, c...)
	}
	return append(b, base[pngIHDREnd:]...)
}

// pngMetadataChunksは、取り除くべきメタデータのチャンク (向きorientationを持つeXIf・テキスト・更新日時・
// 知らない補助チャンク) を返す
func pngMetadataChunks(orientation uint16) [][]byte {
	return [][]byte{
		pngChunkBytes("eXIf", exifTIFF(binary.BigEndian, orientation)),
		pngChunkBytes("tEXt", append([]byte("Comment\x00"), secretComment...)),
		pngChunkBytes("zTXt", append([]byte("Raw profile type iptc\x00\x00"), secretIPTC...)),
		pngChunkBytes("iTXt", append([]byte("XML:com.adobe.xmp\x00\x00\x00\x00\x00"), secretXMP...)),
		pngChunkBytes("tIME", []byte{0x07, 0xEA, 11, 15, 10, 0, 0}),
		pngChunkBytes("caBX", secretGPS),
	}
}

// pngKeptChunksは、残すべき表示のためのチャンク (カラープロファイル・ガンマ・画素の縦横比) を返す
func pngKeptChunks() [][]byte {
	return [][]byte{
		pngChunkBytes("iCCP", append([]byte("icc\x00\x00"), fakeICC...)),
		pngChunkBytes("gAMA", []byte{0, 0, 0xB1, 0x8F}),
		pngChunkBytes("pHYs", []byte{0, 0, 0x16, 0x25, 0, 0, 0x16, 0x25, 1}),
	}
}

func TestStrip_PNG(t *testing.T) {
	t.Parallel()

	base := encodeTestPNG(t)
	wantImage, err := png.Decode(bytes.NewReader(base))
	if err != nil {
		t.Fatalf("元のPNGの復号に失敗: %v", err)
	}

	t.Run("メタデータとIENDより後ろのデータを落とし、表示のためのチャンクと画素を保つ", func(t *testing.T) {
		t.Parallel()

		chunks := append(pngMetadataChunks(1), pngKeptChunks()...)
		input := append(withPNGChunks(base, chunks...), secretTrailer...)

		out := mustStrip(t, input)

		assertNotContains(t, out, secretGPS, secretXMP, secretIPTC, secretComment, secretTrailer, []byte("tIME"), []byte("eXIf"))
		if want := withPNGChunks(base, pngKeptChunks()...); !bytes.Equal(out, want) {
			t.Error("結果が、残すチャンクだけを持つPNGと一致しません")
		}
		got, err := png.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("結果の復号に失敗: %v", err)
		}
		assertSamePixels(t, wantImage, got)
		assertIdempotent(t, out)
	})

	t.Run("既定でない向きは、向きだけを持つeXIfで保つ", func(t *testing.T) {
		t.Parallel()

		out := mustStrip(t, withPNGChunks(base, pngMetadataChunks(6)...))

		assertNotContains(t, out, secretGPS, secretXMP, secretComment)
		if want := withPNGChunks(base, pngChunkBytes("eXIf", orientationTIFF(6))); !bytes.Equal(out, want) {
			t.Error("結果が、向きだけを持つeXIfのPNGと一致しません")
		}
		assertIdempotent(t, out)
	})

	t.Run("アニメーションのチャンクを保つ", func(t *testing.T) {
		t.Parallel()

		idatAt := bytes.Index(base, []byte("IDAT")) - 4
		idatLen := int(binary.BigEndian.Uint32(base[idatAt:]))
		idat := base[idatAt+8 : idatAt+8+idatLen]
		frame := func(seq uint32) []byte {
			b := binary.BigEndian.AppendUint32(nil, seq)
			b = binary.BigEndian.AppendUint32(b, 8)
			b = binary.BigEndian.AppendUint32(b, 6)
			b = append(b, make([]byte, 8)...)
			return pngChunkBytes("fcTL", append(b, 0, 1, 0, 0))
		}
		fdAT := pngChunkBytes("fdAT", append(binary.BigEndian.AppendUint32(nil, 2), idat...))
		animated := append([]byte{}, base[:pngIHDREnd]...)
		animated = append(animated, pngChunkBytes("acTL", []byte{0, 0, 0, 2, 0, 0, 0, 0})...)
		animated = append(animated, frame(0)...)
		animated = append(animated, base[idatAt:idatAt+12+idatLen]...)
		animated = append(animated, frame(1)...)
		animated = append(animated, fdAT...)
		animated = append(animated, base[idatAt+12+idatLen:]...)

		input := withPNGChunks(animated, pngMetadataChunks(1)...)

		out := mustStrip(t, input)

		if !bytes.Equal(out, animated) {
			t.Error("結果が、メタデータの無いアニメーションのPNGと一致しません")
		}
	})

	t.Run("メタデータの無いPNGは同じバイト列を返す", func(t *testing.T) {
		t.Parallel()

		input := withPNGChunks(base, pngKeptChunks()...)

		out := mustStrip(t, input)

		if &out[0] != &input[0] || len(out) != len(input) {
			t.Error("入力と同じバイト列が返りませんでした")
		}
	})
}

func TestStrip_PNG_Malformed(t *testing.T) {
	t.Parallel()

	base := encodeTestPNG(t)
	badCRC := bytes.Clone(base)
	badCRC[pngIHDREnd-1] ^= 0xFF
	tooLong := bytes.Clone(base)
	binary.BigEndian.PutUint32(tooLong[pngIHDREnd:], 0xFFFFFFF0)

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "IENDが無い", input: base[:len(base)-12]},
		{name: "IHDRが空", input: append(append(bytes.Clone(pngSignature), pngChunkBytes("IHDR", nil)...), base[pngIHDREnd:]...)},
		{name: "IHDRが重複している", input: withPNGChunks(base, base[8:pngIHDREnd])},
		{name: "IDATが無い", input: append(bytes.Clone(base[:pngIHDREnd]), pngChunkBytes("IEND", nil)...)},
		{name: "IDATが空のチャンクだけ", input: withPNGChunks(append(bytes.Clone(base[:pngIHDREnd]), pngChunkBytes("IEND", nil)...), pngChunkBytes("IDAT", nil))},
		{name: "IENDの長さが0でない", input: append(bytes.Clone(base[:len(base)-12]), pngChunkBytes("IEND", []byte{0})...)},
		{name: "チャンクが途中で切れている", input: base[:pngIHDREnd+10]},
		{name: "CRCが一致しない", input: badCRC},
		{name: "チャンクの長さが範囲を超える", input: tooLong},
		{name: "最初のチャンクがIHDRではない", input: append(append(bytes.Clone(pngSignature), pngChunkBytes("tEXt", []byte("a\x00b"))...), base[8:]...)},
		{name: "チャンクの種類が英字でない", input: withPNGChunks(base, pngChunkBytes("ab1d", nil))},
		{name: "シグネチャだけ", input: bytes.Clone(pngSignature)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Strip(tt.input); !errors.Is(err, ErrMalformed) {
				t.Errorf("Strip()のエラー = %v、期待値 = ErrMalformed", err)
			}
		})
	}
}

func TestStrip_PNG_EmptyIDAT(t *testing.T) {
	t.Parallel()

	base := encodeTestPNG(t)
	input := withPNGChunks(base, pngChunkBytes("IDAT", nil))

	out := mustStrip(t, input)

	if !bytes.Equal(out, input) || &out[0] != &input[0] {
		t.Error("画像データの前の空のIDATチャンクが変わりました")
	}
	got, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("結果の復号に失敗: %v", err)
	}
	assertSamePixels(t, testImage(8, 6), got)
	assertIdempotent(t, out)
}
