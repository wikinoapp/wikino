package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestBottomNav(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		data       components.BottomNavData
		wantSearch bool
		wantSignIn bool
	}{
		{
			name: "ログイン済みの場合はメニュー・ホーム・検索が表示される",
			data: components.BottomNavData{
				CurrentPageName: templates.PageNameHome,
				SignedIn:        true,
			},
			wantSearch: true,
			wantSignIn: false,
		},
		{
			name: "未ログインの場合はメニュー・ホーム・ログインが表示される",
			data: components.BottomNavData{
				CurrentPageName: templates.PageNameWelcome,
			},
			wantSearch: false,
			wantSignIn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			var buf bytes.Buffer
			err := components.BottomNav(tt.data).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()

			if !strings.Contains(html, "md:hidden") {
				t.Error("md:hidden クラスが含まれていない")
			}

			if !strings.Contains(html, "メニュー") {
				t.Error("メニューボタンが含まれていない")
			}

			if !strings.Contains(html, "ホーム") {
				t.Error("ホームボタンが含まれていない")
			}

			if tt.wantSearch && !strings.Contains(html, "検索") {
				t.Error("検索ボタンが含まれていない")
			}

			if tt.wantSignIn {
				if !strings.Contains(html, "ログイン") {
					t.Error("ログインボタンが含まれていない")
				}
				if !strings.Contains(html, "/sign_in") {
					t.Error("ログインリンクが含まれていない")
				}
			}

			if !tt.wantSignIn && strings.Contains(html, "/sign_in") {
				t.Error("ログインリンクが表示されるべきではない")
			}

			// サイドバートグルイベントが含まれているか確認
			if !strings.Contains(html, "basecoat:sidebar") {
				t.Error("サイドバートグルイベントが含まれていない")
			}
		})
	}
}

func TestBottomNav_ログイン済みのホームパス(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	data := components.BottomNavData{
		SignedIn: true,
	}

	var buf bytes.Buffer
	err := components.BottomNav(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "/home") {
		t.Error("ログイン済みの場合、ホームリンクは /home であるべき")
	}
}

func TestBottomNav_未ログインのホームパス(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	data := components.BottomNavData{}

	var buf bytes.Buffer
	err := components.BottomNav(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if strings.Contains(html, "/home") {
		t.Error("未ログインの場合、/home リンクは表示されるべきではない")
	}
}

func TestBottomNav_スペースフィルター付き検索パス(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	data := components.BottomNavData{
		SignedIn:        true,
		SpaceIdentifier: "my-space",
	}

	var buf bytes.Buffer
	err := components.BottomNav(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "space:my-space") {
		t.Error("スペースフィルター付きの検索パスが含まれていない")
	}
}

func TestBottomNav_英語ロケール(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "en")

	data := components.BottomNavData{
		SignedIn: true,
	}

	var buf bytes.Buffer
	err := components.BottomNav(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "Menu") {
		t.Error("英語の「Menu」テキストが含まれていない")
	}
	if !strings.Contains(html, "Home") {
		t.Error("英語の「Home」テキストが含まれていない")
	}
	if !strings.Contains(html, "Search") {
		t.Error("英語の「Search」テキストが含まれていない")
	}
}
