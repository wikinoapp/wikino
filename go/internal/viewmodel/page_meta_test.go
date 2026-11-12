package viewmodel_test

import (
	"context"
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/darkmode"
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

// ダークモードのフラグはミドルウェアがコンテキストに載せた判定から読み、判定が無いときはライトにする
func TestDefaultPageMeta_DarkModeEnabled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Domain: "wikino.example.com"}

	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{
			name: "判定が有効を返すときはtrue",
			ctx:  darkmode.ToContext(context.Background(), func() bool { return true }),
			want: true,
		},
		{
			name: "判定が無効を返すときはfalse",
			ctx:  darkmode.ToContext(context.Background(), func() bool { return false }),
			want: false,
		},
		{
			name: "判定がコンテキストに無いときはfalse",
			ctx:  context.Background(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			meta := viewmodel.DefaultPageMeta(tt.ctx, cfg)

			if meta.DarkModeEnabled != tt.want {
				t.Errorf("DarkModeEnabled = %v、期待値 = %v", meta.DarkModeEnabled, tt.want)
			}
		})
	}
}
