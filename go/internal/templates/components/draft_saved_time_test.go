package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

func TestDraftSavedTime_タイムゾーン変換(t *testing.T) {
	t.Parallel()

	// UTC 05:30 を基準とする
	modifiedAt := time.Date(2025, 1, 15, 5, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		timeZone     string
		expectedTime string
	}{
		{
			name:         "Asia/Tokyoの場合はJST(UTC+9)で表示される",
			timeZone:     "Asia/Tokyo",
			expectedTime: "14:30",
		},
		{
			name:         "空文字の場合はUTCで表示される",
			timeZone:     "",
			expectedTime: "05:30",
		},
		{
			name:         "不正なタイムゾーンの場合はUTCで表示される",
			timeZone:     "Invalid/TimeZone",
			expectedTime: "05:30",
		},
		{
			name:         "America/New_Yorkの場合はEST(UTC-5)で表示される",
			timeZone:     "America/New_York",
			expectedTime: "00:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")

			var buf bytes.Buffer
			err := components.DraftSavedTime(modifiedAt, tt.timeZone).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()
			if !strings.Contains(html, tt.expectedTime) {
				t.Errorf("出力に期待する時刻 %q が含まれていない\n出力: %s", tt.expectedTime, html)
			}
		})
	}
}
