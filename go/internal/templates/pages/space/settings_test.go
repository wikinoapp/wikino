package space_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	spacepages "github.com/wikinoapp/wikino/go/internal/templates/pages/space"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// 今のロールではspace:writeとspace:deleteが常に一緒に付くため、2つの判定で項目が別々に出ることは
// テンプレートに直接値を渡して確かめる。
func TestSettings_項目の出し分け(t *testing.T) {
	t.Parallel()

	const (
		generalLink  = `href="/s/space/settings/general"`
		deletionLink = `href="/s/space/settings/deletion/new"`
	)

	for _, tt := range []struct {
		name           string
		canUpdateSpace bool
		canDeleteSpace bool
	}{
		{name: "両方を持つと一般と削除の項目が出る", canUpdateSpace: true, canDeleteSpace: true},
		{name: "space:writeだけなら削除の項目が出ない", canUpdateSpace: true},
		{name: "space:deleteだけなら削除の項目だけが出る", canDeleteSpace: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			var buf bytes.Buffer
			err := spacepages.Settings(spacepages.SettingsData{
				Space:          viewmodel.Space{Identifier: "space", Name: "開発チーム"},
				CanUpdateSpace: tt.canUpdateSpace,
				CanDeleteSpace: tt.canDeleteSpace,
			}).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("Render()のエラー = %v", err)
			}

			body := buf.String()
			if got := strings.Contains(body, generalLink); got != tt.canUpdateSpace {
				t.Errorf("一般の項目を含むか = %v、期待値 = %v", got, tt.canUpdateSpace)
			}
			if got := strings.Contains(body, deletionLink); got != tt.canDeleteSpace {
				t.Errorf("削除の項目を含むか = %v、期待値 = %v", got, tt.canDeleteSpace)
			}
		})
	}
}
