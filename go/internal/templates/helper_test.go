package templates_test

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/timezone"
)

func TestFormatDateTime(t *testing.T) {
	t.Parallel()

	utcTime := time.Date(2026, 3, 25, 5, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		timeZone string
		expected string
	}{
		{
			name:     "UTCの場合",
			timeZone: "UTC",
			expected: "2026/03/25 05:30",
		},
		{
			name:     "Asia/Tokyoの場合はJST(UTC+9)で表示される",
			timeZone: "Asia/Tokyo",
			expected: "2026/03/25 14:30",
		},
		{
			name:     "America/New_Yorkの場合はEDT(UTC-4)で表示される",
			timeZone: "America/New_York",
			expected: "2026/03/25 01:30",
		},
		{
			name:     "タイムゾーン未設定の場合はUTCで表示される",
			timeZone: "",
			expected: "2026/03/25 05:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tt.timeZone != "" {
				ctx = timezone.ToContext(ctx, tt.timeZone)
			}

			got := templates.FormatDateTime(ctx, utcTime)
			if got != tt.expected {
				t.Errorf("FormatDateTime() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	t.Parallel()

	utcTime := time.Date(2026, 3, 25, 5, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		timeZone string
		expected string
	}{
		{
			name:     "UTCの場合",
			timeZone: "UTC",
			expected: "05:30",
		},
		{
			name:     "Asia/Tokyoの場合はJST(UTC+9)で表示される",
			timeZone: "Asia/Tokyo",
			expected: "14:30",
		},
		{
			name:     "America/New_Yorkの場合はEDT(UTC-4)で表示される",
			timeZone: "America/New_York",
			expected: "01:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = timezone.ToContext(ctx, tt.timeZone)

			got := templates.FormatTime(ctx, utcTime)
			if got != tt.expected {
				t.Errorf("FormatTime() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRelativeTime(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name     string
		locale   string
		input    time.Time
		expected string
	}{
		{
			name:     "1分未満の場合はたった今",
			locale:   "ja",
			input:    now.Add(-30 * time.Second),
			expected: "たった今",
		},
		{
			name:     "1分前",
			locale:   "ja",
			input:    now.Add(-1 * time.Minute),
			expected: "1分前",
		},
		{
			name:     "30分前",
			locale:   "ja",
			input:    now.Add(-30 * time.Minute),
			expected: "30分前",
		},
		{
			name:     "59分前",
			locale:   "ja",
			input:    now.Add(-59 * time.Minute),
			expected: "59分前",
		},
		{
			name:     "1時間前",
			locale:   "ja",
			input:    now.Add(-1 * time.Hour),
			expected: "1時間前",
		},
		{
			name:     "12時間前",
			locale:   "ja",
			input:    now.Add(-12 * time.Hour),
			expected: "12時間前",
		},
		{
			name:     "23時間前",
			locale:   "ja",
			input:    now.Add(-23 * time.Hour),
			expected: "23時間前",
		},
		{
			name:     "1日前",
			locale:   "ja",
			input:    now.Add(-24 * time.Hour),
			expected: "1日前",
		},
		{
			name:     "2日前",
			locale:   "ja",
			input:    now.Add(-48 * time.Hour),
			expected: "2日前",
		},
		{
			name:     "3日超の場合は絶対時間にフォールバック",
			locale:   "ja",
			input:    time.Date(2026, 1, 15, 5, 30, 0, 0, time.UTC),
			expected: "2026/01/15 05:30",
		},
		{
			name:     "英語: 1分未満の場合はjust now",
			locale:   "en",
			input:    now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "英語: 5分前",
			locale:   "en",
			input:    now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "英語: 3時間前",
			locale:   "en",
			input:    now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "英語: 2日前",
			locale:   "en",
			input:    now.Add(-48 * time.Hour),
			expected: "2 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, tt.locale)
			ctx = timezone.ToContext(ctx, "UTC")

			got := templates.RelativeTime(ctx, tt.input)
			if got != tt.expected {
				t.Errorf("RelativeTime() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRelativeTime_タイムゾーンを考慮したフォールバック(t *testing.T) {
	t.Parallel()

	utcTime := time.Date(2026, 1, 15, 5, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		timeZone string
		expected string
	}{
		{
			name:     "UTCの場合",
			timeZone: "UTC",
			expected: "2026/01/15 05:30",
		},
		{
			name:     "Asia/Tokyoの場合はJST(UTC+9)で表示される",
			timeZone: "Asia/Tokyo",
			expected: "2026/01/15 14:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, "ja")
			ctx = timezone.ToContext(ctx, tt.timeZone)

			got := templates.RelativeTime(ctx, utcTime)
			if got != tt.expected {
				t.Errorf("RelativeTime() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsRelativeTime(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected bool
	}{
		{
			name:     "1分未満の場合はtrue",
			input:    now.Add(-30 * time.Second),
			expected: true,
		},
		{
			name:     "30分前の場合はtrue",
			input:    now.Add(-30 * time.Minute),
			expected: true,
		},
		{
			name:     "12時間前の場合はtrue",
			input:    now.Add(-12 * time.Hour),
			expected: true,
		},
		{
			name:     "2日前の場合はtrue",
			input:    now.Add(-48 * time.Hour),
			expected: true,
		},
		{
			name:     "71時間59分前の場合はtrue",
			input:    now.Add(-71*time.Hour - 59*time.Minute),
			expected: true,
		},
		{
			name:     "5日前の場合はfalse",
			input:    now.Add(-5 * 24 * time.Hour),
			expected: false,
		},
		{
			name:     "30日前の場合はfalse",
			input:    now.Add(-30 * 24 * time.Hour),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := templates.IsRelativeTime(tt.input)
			if got != tt.expected {
				t.Errorf("IsRelativeTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatDateTime_不正なタイムゾーンでUTCにフォールバック(t *testing.T) {
	t.Parallel()

	utcTime := time.Date(2026, 3, 25, 5, 30, 0, 0, time.UTC)

	ctx := context.Background()
	ctx = timezone.ToContext(ctx, "Invalid/TimeZone")

	got := templates.FormatDateTime(ctx, utcTime)
	expected := "2026/03/25 05:30"
	if got != expected {
		t.Errorf("FormatDateTime() with invalid timezone = %q, want %q", got, expected)
	}
}
