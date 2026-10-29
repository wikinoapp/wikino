package oauth_authorization_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	oauthauthorizationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_authorization"
)

// renderSelectSpaceは、スペースの選択肢spacesでスペースを選ぶ段を日本語で描画する
func renderSelectSpace(t *testing.T, spaces []oauthauthorizationpages.SpaceOption) string {
	t.Helper()

	var buf bytes.Buffer
	data := oauthauthorizationpages.SelectSpaceData{
		ApplicationName: "Claude",
		ClientHost:      "claude.ai",
		UserAtname:      "alice",
		Spaces:          spaces,
		RedirectHost:    "claude.ai",
	}
	if err := oauthauthorizationpages.SelectSpace(data).Render(i18n.SetLocale(context.Background(), i18n.LangJa), &buf); err != nil {
		t.Fatalf("スペースを選ぶ段の描画に失敗: %v", err)
	}
	return buf.String()
}

func TestSelectSpace_連携を許可する権限の無いスペースは理由を添えて選べないようにする(t *testing.T) {
	t.Parallel()

	body := renderSelectSpace(t, []oauthauthorizationpages.SpaceOption{
		{Identifier: "writable", Name: "書けるスペース", Selectable: true},
		{Identifier: "readonly", Name: "許可できないスペース", Selectable: false},
	})

	for _, want := range []string{
		"claude.aiが提供するアプリです",
		`value="writable"`,
		`value="readonly" disabled aria-describedby="space-option-1-hint"`,
		`id="space-option-1-hint"`,
		"このスペースでアプリとの連携を許可する権限がありません",
		`form="select-space-form"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("本文に%qが含まれていない", want)
		}
	}
	if strings.Contains(body, `value="writable" disabled`) {
		t.Error("選べるスペースが選べないようになっている")
	}
}

func TestSelectSpace_選べるスペースが無ければ次へのボタンを出さない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		spaces []oauthauthorizationpages.SpaceOption
		want   string
	}{
		{name: "メンバーであるスペースが無い", want: "連携できるスペースがありません"},
		{
			name:   "どのスペースでも許可する権限が無い",
			spaces: []oauthauthorizationpages.SpaceOption{{Identifier: "readonly", Name: "許可できないスペース"}},
			want:   "次のスペースのどれでも、アプリとの連携を許可する権限がありません",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := renderSelectSpace(t, tt.spaces)
			if !strings.Contains(body, tt.want) {
				t.Errorf("本文に%qが含まれていない", tt.want)
			}
			if strings.Contains(body, "次へ") || strings.Contains(body, `type="radio"`) {
				t.Error("選べるスペースが無いのに、スペースの選択肢か次へのボタンが含まれている")
			}
			if !strings.Contains(body, `value="deny"`) {
				t.Error("アプリに戻るボタンが含まれていない")
			}
		})
	}
}
