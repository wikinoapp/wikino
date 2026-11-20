package imagemeta

import (
	"encoding/binary"
	"testing"
)

func TestParseOrientation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tiff []byte
		want uint16
	}{
		{name: "リトルエンディアンの向きを読める", tiff: exifTIFF(binary.LittleEndian, 6), want: 6},
		{name: "ビッグエンディアンの向きを読める", tiff: exifTIFF(binary.BigEndian, 8), want: 8},
		{name: "向きを書いた最小のEXIFを読める", tiff: orientationTIFF(3), want: 3},
		{name: "向きが無ければ0", tiff: exifTIFF(binary.BigEndian, 0), want: 0},
		{name: "範囲外の向きは0", tiff: exifTIFF(binary.BigEndian, 9), want: 0},
		{name: "バイト順が不正なら0", tiff: append([]byte("XX"), exifTIFF(binary.BigEndian, 6)[2:]...), want: 0},
		{name: "IFDの位置が範囲外なら0", tiff: []byte{'M', 'M', 0, 42, 0, 0, 0xFF, 0xFF}, want: 0},
		{name: "IFDが途中で切れていれば0", tiff: exifTIFF(binary.BigEndian, 6)[:20], want: 0},
		{name: "短すぎれば0", tiff: []byte("MM"), want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseOrientation(tt.tiff); got != tt.want {
				t.Errorf("parseOrientation() = %d、期待値 = %d", got, tt.want)
			}
		})
	}
}
