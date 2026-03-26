package components_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
	"github.com/wikinoapp/wikino/go/internal/timezone"
)

func TestRelativeTime(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name         string
		time         time.Time
		locale       string
		tz           string
		wantDatetime bool
		wantTitle    bool
		wantContains []string
		wantExcludes []string
	}{
		{
			name:         "日本語_1分未満の場合はたった今と表示される",
			time:         now.Add(-30 * time.Second),
			locale:       "ja",
			tz:           "Asia/Tokyo",
			wantDatetime: true,
			wantTitle:    true,
			wantContains: []string{"<time", "datetime=", "title=", "たった今"},
		},
		{
			name:         "日本語_分単位の相対時間が表示される",
			time:         now.Add(-5 * time.Minute),
			locale:       "ja",
			tz:           "Asia/Tokyo",
			wantDatetime: true,
			wantTitle:    true,
			wantContains: []string{"<time", "5分前"},
		},
		{
			name:         "日本語_時間単位の相対時間が表示される",
			time:         now.Add(-3 * time.Hour),
			locale:       "ja",
			tz:           "Asia/Tokyo",
			wantDatetime: true,
			wantTitle:    true,
			wantContains: []string{"<time", "3時間前"},
		},
		{
			name:         "日本語_日単位の相対時間が表示される",
			time:         now.Add(-2 * 24 * time.Hour),
			locale:       "ja",
			tz:           "Asia/Tokyo",
			wantDatetime: true,
			wantTitle:    true,
			wantContains: []string{"<time", "2日前"},
		},
		{
			name:         "日本語_3日超の場合は絶対時間にフォールバックする",
			time:         now.Add(-5 * 24 * time.Hour),
			locale:       "ja",
			tz:           "Asia/Tokyo",
			wantDatetime: true,
			wantTitle:    false,
			wantContains: []string{"<time", "/"},
			wantExcludes: []string{"title="},
		},
		{
			name:         "英語_1分未満の場合はjust nowと表示される",
			time:         now.Add(-10 * time.Second),
			locale:       "en",
			tz:           "UTC",
			wantDatetime: true,
			wantTitle:    true,
			wantContains: []string{"<time", "just now"},
		},
		{
			name:         "英語_分単位の相対時間が表示される",
			time:         now.Add(-10 * time.Minute),
			locale:       "en",
			tz:           "UTC",
			wantDatetime: true,
			wantTitle:    true,
			wantContains: []string{"<time", "10 minutes ago"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, tt.locale)
			ctx = timezone.ToContext(ctx, tt.tz)

			data := components.RelativeTimeData{Time: tt.time}

			var buf bytes.Buffer
			err := components.RelativeTime(data).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()

			if tt.wantDatetime && !strings.Contains(html, "datetime=") {
				t.Error("datetime属性が含まれていない")
			}

			if tt.wantTitle && !strings.Contains(html, "title=") {
				t.Error("title属性が含まれていない")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("出力に %q が含まれていない\n出力: %s", want, html)
				}
			}

			for _, exclude := range tt.wantExcludes {
				if strings.Contains(html, exclude) {
					t.Errorf("出力に %q が含まれるべきではない\n出力: %s", exclude, html)
				}
			}
		})
	}
}

func TestRelativeTime_datetime属性にRFC3339形式のUTC時刻が設定される(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 25, 5, 14, 0, 0, time.UTC)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")
	ctx = timezone.ToContext(ctx, "Asia/Tokyo")

	data := components.RelativeTimeData{Time: fixedTime}

	var buf bytes.Buffer
	err := components.RelativeTime(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	expectedDatetime := "2026-03-25T05:14:00Z"
	if !strings.Contains(html, expectedDatetime) {
		t.Errorf("datetime属性に %q が含まれていない\n出力: %s", expectedDatetime, html)
	}
}

func TestRelativeTime_title属性にタイムゾーン変換済みの絶対時間が設定される(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 25, 5, 14, 0, 0, time.UTC)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")
	ctx = timezone.ToContext(ctx, "Asia/Tokyo")

	data := components.RelativeTimeData{Time: fixedTime}

	var buf bytes.Buffer
	err := components.RelativeTime(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	// UTC 05:14 → Asia/Tokyo 14:14
	expectedTitle := "2026/03/25 14:14"
	if !strings.Contains(html, expectedTitle) {
		t.Errorf("title属性に %q が含まれていない\n出力: %s", expectedTitle, html)
	}
}

func TestAbsoluteTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		time         time.Time
		locale       string
		tz           string
		wantContains []string
	}{
		{
			name:   "日本語_Asia/Tokyoタイムゾーンで絶対時間が表示される",
			time:   time.Date(2026, 3, 25, 5, 14, 0, 0, time.UTC),
			locale: "ja",
			tz:     "Asia/Tokyo",
			wantContains: []string{
				"<time",
				"datetime=",
				"2026-03-25T05:14:00Z",
				"2026/03/25 14:14",
			},
		},
		{
			name:   "英語_UTCタイムゾーンで絶対時間が表示される",
			time:   time.Date(2026, 3, 25, 5, 14, 0, 0, time.UTC),
			locale: "en",
			tz:     "UTC",
			wantContains: []string{
				"<time",
				"datetime=",
				"2026-03-25T05:14:00Z",
				"2026/03/25 05:14",
			},
		},
		{
			name:   "America/New_Yorkタイムゾーンで表示される",
			time:   time.Date(2026, 3, 25, 5, 14, 0, 0, time.UTC),
			locale: "ja",
			tz:     "America/New_York",
			wantContains: []string{
				"<time",
				"2026/03/25 01:14",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, tt.locale)
			ctx = timezone.ToContext(ctx, tt.tz)

			data := components.AbsoluteTimeData{Time: tt.time}

			var buf bytes.Buffer
			err := components.AbsoluteTime(data).Render(ctx, &buf)
			if err != nil {
				t.Fatalf("レンダリングに失敗: %v", err)
			}

			html := buf.String()

			for _, want := range tt.wantContains {
				if !strings.Contains(html, want) {
					t.Errorf("出力に %q が含まれていない\n出力: %s", want, html)
				}
			}
		})
	}
}

func TestAbsoluteTime_title属性がない(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")
	ctx = timezone.ToContext(ctx, "Asia/Tokyo")

	data := components.AbsoluteTimeData{
		Time: time.Date(2026, 3, 25, 5, 14, 0, 0, time.UTC),
	}

	var buf bytes.Buffer
	err := components.AbsoluteTime(data).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("レンダリングに失敗: %v", err)
	}

	html := buf.String()

	if strings.Contains(html, "title=") {
		t.Errorf("AbsoluteTimeにtitle属性が含まれるべきではない\n出力: %s", html)
	}
}
