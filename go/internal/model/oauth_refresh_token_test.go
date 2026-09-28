package model

import (
	"testing"
	"time"
)

func TestOAuthRefreshToken_IsActive(t *testing.T) {
	t.Parallel()

	now := time.Now()
	past := now.Add(-time.Hour)

	tests := []struct {
		name      string
		expiresAt time.Time
		usedAt    *time.Time
		revokedAt *time.Time
		want      bool
	}{
		{name: "未使用で失効しておらず期限内なら有効", expiresAt: now.Add(time.Hour), want: true},
		{name: "期限ちょうどは期限切れとして無効", expiresAt: now, want: false},
		{name: "使用済みなら期限内でも無効", expiresAt: now.Add(time.Hour), usedAt: &past, want: false},
		{name: "失効していれば期限内でも無効", expiresAt: now.Add(time.Hour), revokedAt: &past, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token := &OAuthRefreshToken{ExpiresAt: tt.expiresAt, UsedAt: tt.usedAt, RevokedAt: tt.revokedAt}
			if got := token.IsActive(now); got != tt.want {
				t.Errorf("IsActive() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}
