package imagemeta

import (
	"errors"
	"testing"
)

func TestStrip_UnsupportedFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "GIF", input: []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")},
		{name: "SVG", input: []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)},
		{name: "WebPでないRIFF", input: []byte("RIFF\x04\x00\x00\x00WAVE")},
		{name: "空", input: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Strip(tt.input); !errors.Is(err, ErrUnsupportedFormat) {
				t.Errorf("Strip()のエラー = %v、期待値 = ErrUnsupportedFormat", err)
			}
		})
	}
}
