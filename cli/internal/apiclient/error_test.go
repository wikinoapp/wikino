package apiclient

import (
	"maps"
	"testing"
	"time"
)

func TestParseBearerChallenge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   map[string]string
	}{
		{
			name:   "引用したパラメーター",
			header: `Bearer error="insufficient_scope", scope="topic:read"`,
			want:   map[string]string{"error": "insufficient_scope", "scope": "topic:read"},
		},
		{
			name:   "引用しないパラメーターとエスケープ",
			header: `Bearer error=invalid_token, error_description="a \"quoted\" value"`,
			want:   map[string]string{"error": "invalid_token", "error_description": `a "quoted" value`},
		},
		{
			name:   "パラメーターの無いBearer",
			header: `Bearer`,
			want:   map[string]string{},
		},
		{
			name:   "前に別のスキームのチャレンジがある",
			header: `Basic realm="wikino", Bearer scope="page:write"`,
			want:   map[string]string{"scope": "page:write"},
		},
		{
			name:   "Bearerのチャレンジが無い",
			header: `Basic realm="wikino"`,
			want:   map[string]string{},
		},
		{
			name:   "空",
			header: "",
			want:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseBearerChallenge(tt.header); !maps.Equal(got, tt.want) {
				t.Errorf("パラメーターが違います: 実際 %v, 期待 %v", got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		value  string
		want   time.Duration
		wantOK bool
	}{
		{name: "秒数", value: "30", want: 30 * time.Second, wantOK: true},
		{name: "0秒", value: "0", want: 0, wantOK: true},
		{name: "HTTPの日時", value: "Thu, 01 Oct 2026 09:01:00 GMT", want: time.Minute, wantOK: true},
		{name: "過ぎた日時は0", value: "Thu, 01 Oct 2026 08:00:00 GMT", want: 0, wantOK: true},
		{name: "負の秒数", value: "-1"},
		{name: "解釈できない値", value: "soon"},
		{name: "空", value: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseRetryAfter(tt.value, now)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("再試行までの時間が違います: 実際 (%v, %t), 期待 (%v, %t)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
