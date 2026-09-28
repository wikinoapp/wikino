package model

import (
	"testing"
	"time"
)

func TestPersonalAccessToken_IsActive(t *testing.T) {
	t.Parallel()

	now := time.Now()
	revokedAt := now.Add(-time.Hour)

	tests := []struct {
		name      string
		expiresAt time.Time
		revokedAt *time.Time
		want      bool
	}{
		{name: "失効しておらず期限内なら有効", expiresAt: now.Add(time.Hour), want: true},
		{name: "期限ちょうどは期限切れとして無効", expiresAt: now, want: false},
		{name: "期限を過ぎていれば無効", expiresAt: now.Add(-time.Second), want: false},
		{name: "失効していれば期限内でも無効", expiresAt: now.Add(time.Hour), revokedAt: &revokedAt, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token := &PersonalAccessToken{ExpiresAt: tt.expiresAt, RevokedAt: tt.revokedAt}
			if got := token.IsActive(now); got != tt.want {
				t.Errorf("IsActive() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}
