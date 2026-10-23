package viewmodel_test

import (
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// preconnectはウィジェットの描画に追従させる。components.Turnstileはsite keyがあるときだけ
// 描画するため、空のキーではオリジンを宣言しない。
func TestPageMeta_AddTurnstilePreconnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		siteKey string
		want    []string
	}{
		{
			name:    "site keyがあるときはオリジンを宣言する",
			siteKey: "test-site-key",
			want:    []string{"https://challenges.cloudflare.com"},
		},
		{
			name:    "site keyが空のときは宣言しない",
			siteKey: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var meta viewmodel.PageMeta
			meta.AddTurnstilePreconnect(tt.siteKey)

			if !slices.Equal(meta.PreconnectOrigins, tt.want) {
				t.Errorf("PreconnectOrigins = %v、期待値 = %v", meta.PreconnectOrigins, tt.want)
			}
		})
	}
}
