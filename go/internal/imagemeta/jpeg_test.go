package imagemeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"testing"
)

// jpegSegmentBytesは、マーカーmarkerと中身payloadのJPEGのセグメントを作る
func jpegSegmentBytes(marker byte, payload []byte) []byte {
	b := []byte{0xFF, marker}
	b = binary.BigEndian.AppendUint16(b, uint16(2+len(payload)))
	return append(b, payload...)
}

// encodeTestJPEGは、メタデータの無いJPEGを作る
func encodeTestJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, testImage(32, 24), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("JPEGの作成に失敗: %v", err)
	}
	return buf.Bytes()
}

// withJPEGSegmentsは、JPEGのSOIの直後にセグメントsegmentsを差し込む
func withJPEGSegments(base []byte, segments ...[]byte) []byte {
	b := append([]byte{}, base[:2]...)
	for _, s := range segments {
		b = append(b, s...)
	}
	return append(b, base[2:]...)
}

var (
	jfifSegment  = jpegSegmentBytes(jpegMarkerAPP0, append(bytes.Clone(jfifIdentifier), 1, 2, 0, 0, 1, 0, 1, 0, 0))
	iccSegment   = jpegSegmentBytes(jpegMarkerAPP2, append(append(bytes.Clone(iccIdentifier), 1, 1), fakeICC...))
	adobeSegment = jpegSegmentBytes(jpegMarkerAPP14, append(bytes.Clone(adobeIdentifier), 0, 100, 0, 0, 0, 0, 1))
)

// jpegMetadataSegmentsは、取り除くべきメタデータのセグメント (向きorientationを持つEXIF・XMP・IPTC・コメント・
// MPF・そのほかのアプリケーションのセグメント) を返す
func jpegMetadataSegments(orientation uint16) [][]byte {
	return [][]byte{
		jpegSegmentBytes(jpegMarkerAPP1, append(bytes.Clone(exifIdentifier), exifTIFF(binary.LittleEndian, orientation)...)),
		jpegSegmentBytes(jpegMarkerAPP1, append([]byte("http://ns.adobe.com/xap/1.0/\x00"), secretXMP...)),
		jpegSegmentBytes(0xED, append([]byte("Photoshop 3.0\x00"), secretIPTC...)),
		jpegSegmentBytes(jpegMarkerCOM, secretComment),
		jpegSegmentBytes(jpegMarkerAPP2, append([]byte("MPF\x00"), secretGPS...)),
		jpegSegmentBytes(0xEB, append([]byte("JP"), secretXMP...)),
	}
}

func TestStrip_JPEG(t *testing.T) {
	t.Parallel()

	base := encodeTestJPEG(t)
	wantImage, err := jpeg.Decode(bytes.NewReader(base))
	if err != nil {
		t.Fatalf("元のJPEGの復号に失敗: %v", err)
	}

	t.Run("メタデータと画像の終わりより後ろのデータを落とし、JFIF・カラープロファイル・Adobeと画素を保つ", func(t *testing.T) {
		t.Parallel()

		segments := append([][]byte{jfifSegment}, jpegMetadataSegments(1)...)
		segments = append(segments, iccSegment, adobeSegment)
		input := append(withJPEGSegments(base, segments...), secretTrailer...)

		out := mustStrip(t, input)

		assertNotContains(t, out, secretGPS, secretXMP, secretIPTC, secretComment, secretTrailer, exifIdentifier)
		want := withJPEGSegments(base, jfifSegment, iccSegment, adobeSegment)
		if !bytes.Equal(out, want) {
			t.Errorf("結果が、残すセグメントだけを持つJPEGと一致しません (長さ %d、期待値 %d)", len(out), len(want))
		}
		got, err := jpeg.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("結果の復号に失敗: %v", err)
		}
		assertSamePixels(t, wantImage, got)
		assertIdempotent(t, out)
	})

	t.Run("既定でない向きは、向きだけを持つEXIFで保つ", func(t *testing.T) {
		t.Parallel()

		input := withJPEGSegments(base, append([][]byte{jfifSegment}, jpegMetadataSegments(6)...)...)

		out := mustStrip(t, input)

		assertNotContains(t, out, secretGPS, secretXMP, secretComment)
		want := withJPEGSegments(base, jfifSegment, jpegSegmentBytes(jpegMarkerAPP1, append(bytes.Clone(exifIdentifier), orientationTIFF(6)...)))
		if !bytes.Equal(out, want) {
			t.Errorf("結果が、JFIFと向きだけを持つEXIFのJPEGと一致しません")
		}
		assertIdempotent(t, out)
	})

	t.Run("2つ目のEXIFの向きは使わない", func(t *testing.T) {
		t.Parallel()

		first := jpegSegmentBytes(jpegMarkerAPP1, append(bytes.Clone(exifIdentifier), exifTIFF(binary.BigEndian, 1)...))
		second := jpegSegmentBytes(jpegMarkerAPP1, append(bytes.Clone(exifIdentifier), exifTIFF(binary.BigEndian, 6)...))

		out := mustStrip(t, withJPEGSegments(base, first, second))

		if !bytes.Equal(out, base) {
			t.Error("最初のEXIFが既定の向きなのに、EXIFが残っています")
		}
	})

	for _, name := range []string{"progressive.jpg", "restart.jpg"} {
		t.Run(name+"のスキャンの区切りを読み、画素を保つ", func(t *testing.T) {
			t.Parallel()

			original := readTestdata(t, name)
			wantOriginal, err := jpeg.Decode(bytes.NewReader(original))
			if err != nil {
				t.Fatalf("元のJPEGの復号に失敗: %v", err)
			}
			input := append(withJPEGSegments(original, jpegMetadataSegments(1)...), secretTrailer...)

			out := mustStrip(t, input)

			if !bytes.Equal(out, original) {
				t.Error("メタデータを落とした結果が、元のJPEGと一致しません")
			}
			got, err := jpeg.Decode(bytes.NewReader(out))
			if err != nil {
				t.Fatalf("結果の復号に失敗: %v", err)
			}
			assertSamePixels(t, wantOriginal, got)
		})
	}

	t.Run("メタデータの無いJPEGは同じバイト列を返す", func(t *testing.T) {
		t.Parallel()

		input := withJPEGSegments(base, jfifSegment, iccSegment)

		out := mustStrip(t, input)

		if &out[0] != &input[0] || len(out) != len(input) {
			t.Error("入力と同じバイト列が返りませんでした")
		}
	})
}

func TestStrip_JPEG_Malformed(t *testing.T) {
	t.Parallel()

	base := encodeTestJPEG(t)
	sosAt := bytes.Index(base, []byte{0xFF, jpegMarkerSOS})
	sofAt := bytes.Index(base, []byte{0xFF, jpegMarkerSOF0})
	sofEnd := sofAt + 2 + int(binary.BigEndian.Uint16(base[sofAt+2:]))
	sosEnd := sosAt + 2 + int(binary.BigEndian.Uint16(base[sosAt+2:]))

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "画像の終わりが無い", input: base[:len(base)-2]},
		{name: "スキャンの途中で切れている", input: base[:sosAt+40]},
		{name: "SOIだけ", input: []byte{0xFF, jpegMarkerSOI}},
		{name: "SOIとEOIだけ", input: []byte{0xFF, jpegMarkerSOI, 0xFF, jpegMarkerEOI}},
		{name: "フレームが無い", input: append(bytes.Clone(base[:sofAt]), base[sofEnd:]...)},
		{name: "スキャンが無い", input: append(bytes.Clone(base[:sosAt]), 0xFF, jpegMarkerEOI)},
		{name: "フレームヘッダーが短い", input: append(append(bytes.Clone(base[:sofAt]), jpegSegmentBytes(jpegMarkerSOF0, nil)...), base[sofEnd:]...)},
		{name: "スキャンヘッダーが短い", input: append(append(bytes.Clone(base[:sosAt]), jpegSegmentBytes(jpegMarkerSOS, nil)...), base[sosEnd:]...)},
		{name: "スキャンの画像データが無い", input: append(bytes.Clone(base[:sosEnd]), 0xFF, jpegMarkerEOI)},
		{name: "セグメントの長さが範囲を超える", input: append([]byte{0xFF, jpegMarkerSOI, 0xFF, jpegMarkerAPP1, 0xFF, 0xFF}, base[2:]...)},
		{name: "セグメントの長さが2未満", input: append([]byte{0xFF, jpegMarkerSOI, 0xFF, jpegMarkerAPP1, 0x00, 0x01}, base[2:]...)},
		{name: "マーカーの無いデータが続く", input: []byte{0xFF, jpegMarkerSOI, 0x12, 0x34}},
		{name: "SOIが2回ある", input: append([]byte{0xFF, jpegMarkerSOI}, base...)},
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

// TestStrip_JPEG_DecodesBoundsは、向きを保ったJPEGが、向きを持たない画像として標準のデコーダーでも読めることを確かめる
func TestStrip_JPEG_DecodesBounds(t *testing.T) {
	t.Parallel()

	out := mustStrip(t, withJPEGSegments(encodeTestJPEG(t), jpegMetadataSegments(8)...))

	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("結果の読み込みに失敗: %v", err)
	}
	if got := (image.Point{X: cfg.Width, Y: cfg.Height}); got != (image.Point{X: 32, Y: 24}) {
		t.Errorf("大きさ = %v、期待値 = (32,24)", got)
	}
	if got := parseOrientation(out[2+4+len(exifIdentifier):]); got != 8 {
		t.Errorf("向き = %d、期待値 = 8", got)
	}
}

func TestStrip_JPEG_MarkerFill(t *testing.T) {
	t.Parallel()

	base := encodeTestJPEG(t)
	for _, marker := range []byte{base[3], jpegMarkerDHT, jpegMarkerSOF0, jpegMarkerSOS, jpegMarkerEOI} {
		t.Run(fmt.Sprintf("マーカー0x%02Xの前の詰め物を保つ", marker), func(t *testing.T) {
			t.Parallel()

			at := bytes.Index(base, []byte{0xFF, marker})
			if at < 0 {
				t.Fatal("テスト画像に対象のマーカーがありません")
			}
			input := append(bytes.Clone(base[:at]), 0xFF, 0xFF)
			input = append(input, base[at:]...)
			wantImage, err := jpeg.Decode(bytes.NewReader(input))
			if err != nil {
				t.Fatalf("詰め物を持つJPEGの復号に失敗: %v", err)
			}

			out := mustStrip(t, input)

			if !bytes.Equal(out, input) || &out[0] != &input[0] {
				t.Error("メタデータの無い入力の詰め物が変わりました")
			}
			withMetadata := append(withJPEGSegments(input, jpegMetadataSegments(1)...), secretTrailer...)
			stripped := mustStrip(t, withMetadata)
			if !bytes.Equal(stripped, input) {
				t.Error("メタデータの除去で詰め物が変わりました")
			}
			got, err := jpeg.Decode(bytes.NewReader(stripped))
			if err != nil {
				t.Fatalf("結果の復号に失敗: %v", err)
			}
			assertSamePixels(t, wantImage, got)
			assertIdempotent(t, stripped)
		})
	}
}
