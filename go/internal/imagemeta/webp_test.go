package imagemeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"testing"

	"golang.org/x/image/webp"
)

// webpImageChunkは、testdataの単純な形式のWebPから、画像のチャンクを取り出す
func webpImageChunk(t *testing.T, name string) []byte {
	t.Helper()
	data := readTestdata(t, name)
	if string(data[12:15]) != "VP8" {
		t.Fatalf("%sは単純な形式のWebPではありません", name)
	}
	return data[webpHeaderSize:]
}

// webpMetadataChunksは、取り除くべきメタデータのチャンク (向きorientationを持つEXIF・XMP・知らないチャンク) を返す
func webpMetadataChunks(orientation uint16) [][]byte {
	return [][]byte{
		webpChunkBytes("EXIF", exifTIFF(binary.LittleEndian, orientation)),
		webpChunkBytes("XMP ", secretXMP),
		webpChunkBytes("ABCD", secretComment),
	}
}

func decodeWebP(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("WebPの復号に失敗: %v", err)
	}
	return img
}

func TestStrip_WebP(t *testing.T) {
	t.Parallel()

	t.Run("EXIF・XMPと知らないチャンクを落としてフラグを直し、カラープロファイル・アルファと画素を保つ", func(t *testing.T) {
		t.Parallel()

		simple := readTestdata(t, "lossless_alpha.webp")
		imageChunk := webpImageChunk(t, "lossless_alpha.webp")
		iccp := webpChunkBytes("ICCP", fakeICC)
		flags := byte(0x20 | 0x10 | webpFlagEXIF | webpFlagXMP)
		chunks := [][]byte{webpChunkBytes("VP8X", vp8xBody(flags, 16, 16)), iccp, imageChunk}
		input := append(webpRIFF(append(chunks, webpMetadataChunks(1)...)...), secretTrailer...)

		out := mustStrip(t, input)

		assertNotContains(t, out, secretGPS, secretXMP, secretComment, secretTrailer)
		want := webpRIFF(webpChunkBytes("VP8X", vp8xBody(0x20|0x10, 16, 16)), iccp, imageChunk)
		if !bytes.Equal(out, want) {
			t.Error("結果が、カラープロファイルと画像だけを持ち、メタデータのフラグを落としたWebPと一致しません")
		}
		assertSamePixels(t, decodeWebP(t, simple), decodeWebP(t, out))
		assertIdempotent(t, out)
	})

	t.Run("既定でない向きは、向きだけを持つEXIFで保ち、EXIFのフラグを残す", func(t *testing.T) {
		t.Parallel()

		imageChunk := webpImageChunk(t, "lossy.webp")
		chunks := append([][]byte{webpChunkBytes("VP8X", vp8xBody(webpFlagEXIF|webpFlagXMP, 16, 16)), imageChunk}, webpMetadataChunks(6)...)

		out := mustStrip(t, webpRIFF(chunks...))

		assertNotContains(t, out, secretGPS, secretXMP)
		want := webpRIFF(webpChunkBytes("VP8X", vp8xBody(webpFlagEXIF, 16, 16)), imageChunk, webpChunkBytes("EXIF", orientationTIFF(6)))
		if !bytes.Equal(out, want) {
			t.Error("結果が、向きだけを持つEXIFのWebPと一致しません")
		}
		assertSamePixels(t, decodeWebP(t, readTestdata(t, "lossy.webp")), decodeWebP(t, out))
		assertIdempotent(t, out)
	})

	t.Run("JPEGと同じ識別子から始まるEXIFの向きも読む", func(t *testing.T) {
		t.Parallel()

		imageChunk := webpImageChunk(t, "lossy.webp")
		exif := webpChunkBytes("EXIF", append(bytes.Clone(exifIdentifier), exifTIFF(binary.BigEndian, 3)...))

		out := mustStrip(t, webpRIFF(webpChunkBytes("VP8X", vp8xBody(webpFlagEXIF, 16, 16)), imageChunk, exif))

		want := webpRIFF(webpChunkBytes("VP8X", vp8xBody(webpFlagEXIF, 16, 16)), imageChunk, webpChunkBytes("EXIF", orientationTIFF(3)))
		if !bytes.Equal(out, want) {
			t.Error("結果が、向きだけを持つEXIFのWebPと一致しません")
		}
	})

	t.Run("アニメーションのチャンクを保つ", func(t *testing.T) {
		t.Parallel()

		animated := readTestdata(t, "animated.webp")
		chunks := animated[webpHeaderSize:]
		vp8xEnd := 8 + 10
		withMeta := bytes.Clone(chunks[:vp8xEnd])
		withMeta[8] |= webpFlagEXIF | webpFlagXMP
		withMeta = append(withMeta, chunks[vp8xEnd:]...)
		input := webpRIFF(append([][]byte{withMeta}, webpMetadataChunks(1)...)...)

		out := mustStrip(t, input)

		if !bytes.Equal(out, animated) {
			t.Error("結果が、元のアニメーションのWebPと一致しません")
		}
	})

	t.Run("単純な形式は、RIFFより後ろのデータだけを落とす", func(t *testing.T) {
		t.Parallel()

		simple := readTestdata(t, "lossy.webp")

		out := mustStrip(t, append(bytes.Clone(simple), secretTrailer...))

		if !bytes.Equal(out, simple) {
			t.Error("結果が、元のWebPと一致しません")
		}
	})

	t.Run("メタデータの無いWebPは同じバイト列を返す", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"lossy.webp", "lossless_alpha.webp", "lossy_alpha.webp", "animated.webp", "animated_alpha.webp"} {
			input := readTestdata(t, name)
			out := mustStrip(t, input)
			if &out[0] != &input[0] || len(out) != len(input) {
				t.Errorf("%s: 入力と同じバイト列が返りませんでした", name)
			}
		}
	})
}

func TestStrip_WebP_Malformed(t *testing.T) {
	t.Parallel()

	simple := readTestdata(t, "lossy.webp")
	tooLarge := bytes.Clone(simple)
	binary.LittleEndian.PutUint32(tooLarge[4:8], uint32(len(simple)))
	chunkTooLarge := bytes.Clone(simple)
	binary.LittleEndian.PutUint32(chunkTooLarge[16:20], uint32(len(simple)))
	imageChunk := simple[webpHeaderSize:]

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "RIFFの大きさがファイルを超える", input: tooLarge},
		{name: "チャンクの大きさがRIFFを超える", input: chunkTooLarge},
		{name: "チャンクが無い", input: webpRIFF()},
		{name: "チャンクのヘッダーが途中で切れている", input: webpRIFF([]byte("VP8"))},
		{name: "VP8Xが短すぎる", input: webpRIFF(webpChunkBytes("VP8X", []byte{0, 0}), imageChunk)},
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

func TestStrip_WebP_SimpleWithExtraChunks(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"lossy.webp", "lossless_alpha.webp"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			imageChunk := webpImageChunk(t, name)
			for _, extra := range webpMetadataChunks(1) {
				t.Run(string(extra[:4]), func(t *testing.T) {
					t.Parallel()

					input := webpRIFF(imageChunk, extra)
					// 単純な形式の後ろにあるチャンクを無視するデコーダーでも読める入力
					decodeWebP(t, input)

					if _, err := Strip(input); !errors.Is(err, ErrMalformed) {
						t.Errorf("Strip()のエラー = %v、期待値 = ErrMalformed", err)
					}
				})
			}
		})
	}
}

func TestStrip_WebP_FrameMetadata(t *testing.T) {
	t.Parallel()

	// animated_alpha.webpは、各フレームにALPHとVP8を持つ
	for _, name := range []string{"animated.webp", "animated_alpha.webp"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			original := readTestdata(t, name)
			chunks, err := parseWebPChunks(original[webpHeaderSize:])
			if err != nil {
				t.Fatal(err)
			}
			var withMetadata [][]byte
			for _, c := range chunks {
				if c.fourCC != "ANMF" {
					withMetadata = append(withMetadata, c.raw)
					continue
				}
				frame := bytes.Clone(c.body)
				frame = append(frame, webpChunkBytes("ABCD", secretGPS)...)
				frame = append(frame, webpChunkBytes("XMP ", secretXMP)...)
				withMetadata = append(withMetadata, webpChunkBytes("ANMF", frame))
			}

			out := mustStrip(t, webpRIFF(withMetadata...))

			assertNotContains(t, out, secretGPS, secretXMP)
			if !bytes.Equal(out, original) {
				t.Error("メタデータの除去後のフレームとRIFFの長さが元のアニメーションと一致しません")
			}
			assertIdempotent(t, out)
		})
	}
}

// lossyAlphaChunksは、testdataの透過のある非可逆のWebPから、VP8X・ALPH・VP8のチャンクを取り出す
func lossyAlphaChunks(t *testing.T) (vp8x, alph, vp8 webpChunk) {
	t.Helper()
	chunks, err := parseWebPChunks(readTestdata(t, "lossy_alpha.webp")[webpHeaderSize:])
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 3 || chunks[0].fourCC != "VP8X" || chunks[1].fourCC != "ALPH" || chunks[2].fourCC != "VP8 " {
		t.Fatal("lossy_alpha.webpがVP8X・ALPH・VP8の構成ではありません")
	}
	return chunks[0], chunks[1], chunks[2]
}

// hasTransparentPixelは、imgに完全には不透明でない画素があるかどうかを返す
func hasTransparentPixel(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0xFFFF {
				return true
			}
		}
	}
	return false
}

func TestStrip_WebP_LossyAlpha(t *testing.T) {
	t.Parallel()

	original := readTestdata(t, "lossy_alpha.webp")
	wantImage := decodeWebP(t, original)
	if !hasTransparentPixel(wantImage) {
		t.Fatal("lossy_alpha.webpに透過する画素がありません")
	}
	vp8x, alph, vp8 := lossyAlphaChunks(t)
	header := bytes.Clone(vp8x.body)
	header[0] |= webpFlagEXIF | webpFlagXMP
	chunks := append([][]byte{webpChunkBytes("VP8X", header), alph.raw, vp8.raw}, webpMetadataChunks(1)...)

	out := mustStrip(t, webpRIFF(chunks...))

	assertNotContains(t, out, secretGPS, secretXMP, secretComment)
	if !bytes.Equal(out, original) {
		t.Error("結果が、ALPHを保ってメタデータのフラグを落とした元のWebPと一致しません")
	}
	assertSamePixels(t, wantImage, decodeWebP(t, out))
	assertIdempotent(t, out)
}

func TestStrip_WebP_MalformedAlpha(t *testing.T) {
	t.Parallel()

	vp8x, alph, vp8 := lossyAlphaChunks(t)
	vp8l := webpImageChunk(t, "lossless_alpha.webp")
	animated, err := parseWebPChunks(readTestdata(t, "animated_alpha.webp")[webpHeaderSize:])
	if err != nil {
		t.Fatal(err)
	}
	if len(animated) < 2 || animated[0].fourCC != "VP8X" || animated[1].fourCC != "ANIM" {
		t.Fatal("animated_alpha.webpがVP8X・ANIMから始まっていません")
	}
	frameHeader := make([]byte, 16)

	images := []struct {
		name   string
		chunks [][]byte
	}{
		{name: "ALPHが画像の後ろにある", chunks: [][]byte{vp8.raw, alph.raw}},
		{name: "ALPHが2つある", chunks: [][]byte{alph.raw, alph.raw, vp8.raw}},
		{name: "ALPHが空", chunks: [][]byte{webpChunkBytes("ALPH", nil), vp8.raw}},
		{name: "ALPHがVP8Lと組になっている", chunks: [][]byte{alph.raw, vp8l}},
	}
	for _, img := range images {
		frame := append(bytes.Clone(frameHeader), bytes.Join(img.chunks, nil)...)
		inputs := []struct {
			name  string
			input []byte
		}{
			{name: "静止画", input: webpRIFF(append([][]byte{vp8x.raw}, img.chunks...)...)},
			{name: "アニメーションのフレーム", input: webpRIFF(animated[0].raw, animated[1].raw, webpChunkBytes("ANMF", frame))},
		}
		for _, in := range inputs {
			t.Run(img.name+"/"+in.name, func(t *testing.T) {
				t.Parallel()
				if _, err := Strip(in.input); !errors.Is(err, ErrMalformed) {
					t.Errorf("Strip()のエラー = %v、期待値 = ErrMalformed", err)
				}
			})
		}
	}
}

func TestStrip_WebP_MissingImageStructure(t *testing.T) {
	t.Parallel()

	imageChunk := webpImageChunk(t, "lossy.webp")
	vp8 := imageChunk[8 : 8+binary.LittleEndian.Uint32(imageChunk[4:8])]
	lossless := webpImageChunk(t, "lossless_alpha.webp")
	vp8l := lossless[8 : 8+binary.LittleEndian.Uint32(lossless[4:8])]
	staticHeader := webpChunkBytes("VP8X", vp8xBody(0, 16, 16))
	animationHeader := webpChunkBytes("VP8X", vp8xBody(webpFlagAnimation, 16, 16))
	animation := webpChunkBytes("ANIM", make([]byte, 6))
	frameHeader := make([]byte, 16)
	frame := webpChunkBytes("ANMF", append(bytes.Clone(frameHeader), imageChunk...))
	badFrameChunk := bytes.Clone(imageChunk)
	binary.LittleEndian.PutUint32(badFrameChunk[4:8], uint32(len(imageChunk)))

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "VP8Xだけ", input: webpRIFF(staticHeader)},
		{name: "画像が無くメタデータだけ", input: webpRIFF(staticHeader, webpChunkBytes("EXIF", secretGPS))},
		{name: "VP8が空", input: webpRIFF(webpChunkBytes("VP8 ", nil))},
		{name: "VP8Lが空", input: webpRIFF(webpChunkBytes("VP8L", nil))},
		{name: "VP8がヘッダーだけ", input: webpRIFF(webpChunkBytes("VP8 ", vp8[:10]))},
		{name: "VP8Lがヘッダーだけ", input: webpRIFF(webpChunkBytes("VP8L", vp8l[:5]))},
		{name: "単純な形式に画像が2つ", input: webpRIFF(imageChunk, imageChunk)},
		{name: "VP8Xが長すぎる", input: webpRIFF(webpChunkBytes("VP8X", append(vp8xBody(0, 16, 16), 0)), imageChunk)},
		{name: "アニメーションにANIMが無い", input: webpRIFF(animationHeader, frame)},
		{name: "アニメーションにフレームが無い", input: webpRIFF(animationHeader, animation)},
		{name: "ANIMの長さが不正", input: webpRIFF(animationHeader, webpChunkBytes("ANIM", make([]byte, 5)), frame)},
		{name: "静止画にアニメーションがある", input: webpRIFF(staticHeader, animation, frame)},
		{name: "アニメーションの画像がANMFの外にある", input: webpRIFF(animationHeader, animation, imageChunk, frame)},
		{name: "ANMFのヘッダーが短い", input: webpRIFF(animationHeader, animation, webpChunkBytes("ANMF", make([]byte, 15)))},
		{name: "ANMFのヘッダーだけ", input: webpRIFF(animationHeader, animation, webpChunkBytes("ANMF", frameHeader))},
		{name: "ANMF内のチャンクヘッダーが短い", input: webpRIFF(animationHeader, animation, webpChunkBytes("ANMF", append(bytes.Clone(frameHeader), []byte("VP8")...)))},
		{name: "ANMF内のチャンクが範囲を超える", input: webpRIFF(animationHeader, animation, webpChunkBytes("ANMF", append(bytes.Clone(frameHeader), badFrameChunk...)))},
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
