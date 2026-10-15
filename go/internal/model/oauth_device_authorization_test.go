package model

import (
	"testing"
	"time"
)

func TestOAuthDeviceAuthorization_IsExpired(t *testing.T) {
	t.Parallel()

	expiresAt := time.Date(2026, time.October, 12, 12, 0, 0, 0, time.UTC)
	authorization := &OAuthDeviceAuthorization{ExpiresAt: expiresAt}
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "期限の1ナノ秒前", now: expiresAt.Add(-time.Nanosecond), want: false},
		{name: "期限と同じ時刻", now: expiresAt, want: true},
		{name: "期限の1ナノ秒後", now: expiresAt.Add(time.Nanosecond), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := authorization.IsExpired(tt.now); got != tt.want {
				t.Errorf("IsExpired(%v) = %v、期待値 = %v", tt.now, got, tt.want)
			}
		})
	}
}
