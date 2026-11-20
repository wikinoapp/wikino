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

func TestSettingsAttachments_DeleteButton(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		canDelete bool
	}{
		{name: "attachment:deleteを持つと削除のボタンが出る", canDelete: true},
		{name: "attachment:deleteを持たないと削除のボタンが出ない", canDelete: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			var buf bytes.Buffer
			err := spacepages.SettingsAttachments(spacepages.SettingsAttachmentsData{
				Space:               viewmodel.Space{Identifier: "space", Name: "開発チーム"},
				Attachments:         []viewmodel.SettingsAttachment{{ID: "attachment-id", Filename: "photo.png", ContentType: "image/png"}},
				Pagination:          viewmodel.NewPagination(1, 1, 50),
				CanDeleteAttachment: tt.canDelete,
			}).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("Render()のエラー = %v", err)
			}

			body := buf.String()
			for _, marker := range []string{`value="DELETE"`, `aria-label="photo.pngを削除"`} {
				if got := strings.Contains(body, marker); got != tt.canDelete {
					t.Errorf("%qを含むか = %v、期待値 = %v", marker, got, tt.canDelete)
				}
			}
			if !strings.Contains(body, "photo.png") {
				t.Error("ファイル名が表示されていない")
			}
		})
	}
}
