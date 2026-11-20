package model_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestAttachment_IsImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		contentType string
		want        bool
	}{
		{contentType: "image/png", want: true},
		{contentType: "image/svg+xml", want: true},
		{contentType: "application/pdf", want: false},
		{contentType: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			t.Parallel()

			if got := (&model.Attachment{ContentType: tt.contentType}).IsImage(); got != tt.want {
				t.Errorf("IsImage() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}
