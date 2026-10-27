package validator

import (
	"strings"
	"testing"
)

func TestParseOAuthDeviceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		want   *string
		wantOK bool
	}{
		{name: "送られなければnil", raw: "", want: nil, wantOK: true},
		{name: "空白だけならnil", raw: " 　 ", want: nil, wantOK: true},
		{name: "前後の空白を除く", raw: "  my-laptop ", want: oauthDeviceNamePtr("my-laptop"), wantOK: true},
		{name: "100文字は受け付ける", raw: strings.Repeat("あ", 100), want: oauthDeviceNamePtr(strings.Repeat("あ", 100)), wantOK: true},
		{name: "100文字を超えると受け付けない", raw: strings.Repeat("あ", 101), wantOK: false},
		{name: "制御文字を含むと受け付けない", raw: "my\nlaptop", wantOK: false},
		{name: "不正UTF-8を受け付けない", raw: "my\xfflaptop", wantOK: false},
		{name: "途中で切れたUTF-8を受け付けない", raw: "my\xe3\x81", wantOK: false},
		{name: "正しいUTF-8の置換文字を受け付ける", raw: "my\ufffdlaptop", want: oauthDeviceNamePtr("my\ufffdlaptop"), wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseOAuthDeviceName(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v、期待値 = %v", ok, tt.wantOK)
			}
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Errorf("端末名 = %v、期待値 = %v", oauthDeviceNameValue(got), oauthDeviceNameValue(tt.want))
			}
		})
	}
}

func oauthDeviceNamePtr(s string) *string { return &s }

func oauthDeviceNameValue(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
