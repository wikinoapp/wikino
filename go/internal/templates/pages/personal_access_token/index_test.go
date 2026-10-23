package personal_access_token_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	patpages "github.com/wikinoapp/wikino/go/internal/templates/pages/personal_access_token"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// mainTitleActionsMarkupはMainTitleの操作領域を開くタグ
const mainTitleActionsMarkup = `<div class="flex w-full flex-none items-center justify-end gap-2 md:w-auto md:min-h-9">`

func TestIndex_発行ボタンはタイトルの操作領域に出す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		canCreate bool
	}{
		{name: "発行できるメンバーには出す", canCreate: true},
		{name: "発行できないメンバーには出さない", canCreate: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			data := patpages.IndexData{
				Space:     viewmodel.Space{Identifier: "pat-template"},
				CanCreate: tt.canCreate,
			}
			if err := patpages.Index(data).Render(i18n.SetLocale(context.Background(), i18n.LangJa), &buf); err != nil {
				t.Fatalf("一覧の描画に失敗: %v", err)
			}
			body := buf.String()
			newLink := `href="/s/pat-template/settings/personal_access_tokens/new"`

			start := strings.Index(body, mainTitleActionsMarkup)
			if !tt.canCreate {
				if start != -1 || strings.Contains(body, newLink) {
					t.Error("発行できないメンバーに発行ボタンが出ている")
				}
				return
			}
			if start == -1 {
				t.Fatal("タイトルの操作領域が無い")
			}
			actions := body[start:]
			actions = actions[:strings.Index(actions, "</div>")]
			for _, want := range []string{newLink, `data-size="sm"`, "トークンを発行する"} {
				if !strings.Contains(actions, want) {
					t.Errorf("操作領域に%qが含まれていない", want)
				}
			}
		})
	}
}
