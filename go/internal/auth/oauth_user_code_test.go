package auth

import (
	"strings"
	"testing"
)

func TestGenerateOAuthUserCode(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for range 100 {
		code, err := GenerateOAuthUserCode()
		if err != nil {
			t.Fatalf("GenerateOAuthUserCode()のエラー = %v", err)
		}
		if len(code) != 8 {
			t.Fatalf("ユーザーコード %q の文字数 = %d、期待値 = 8", code, len(code))
		}
		for _, r := range code {
			if !strings.ContainsRune("BCDFGHJKLMNPQRSTVWXZ", r) {
				t.Fatalf("ユーザーコード %q に20種の子音以外の文字 %q が含まれている", code, r)
			}
		}
		seen[code] = true
	}
	// 2.56×10^10通りから100回選んで重複する確率は無視できる
	if len(seen) != 100 {
		t.Errorf("100回の生成で異なるユーザーコードの数 = %d、期待値 = 100", len(seen))
	}
}

func TestFormatOAuthUserCode(t *testing.T) {
	t.Parallel()

	if got := FormatOAuthUserCode("BCDFGHJK"); got != "BCDF-GHJK" {
		t.Errorf("FormatOAuthUserCode() = %q、期待値 = %q", got, "BCDF-GHJK")
	}
}

func TestNormalizeOAuthUserCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{name: "区切りなしの大文字", input: "BCDFGHJK", want: "BCDFGHJK", wantOK: true},
		{name: "表示の形", input: "BCDF-GHJK", want: "BCDFGHJK", wantOK: true},
		{name: "小文字", input: "bcdf-ghjk", want: "BCDFGHJK", wantOK: true},
		{name: "空白の区切りと前後の空白", input: " BCDF GHJK ", want: "BCDFGHJK", wantOK: true},
		{name: "母音を含む", input: "BCDF-GHJA", wantOK: false},
		{name: "数字を含む", input: "BCDF-GHJ1", wantOK: false},
		{name: "短い", input: "BCDF-GHJ", wantOK: false},
		{name: "長い", input: "BCDF-GHJKL", wantOK: false},
		{name: "空", input: "", wantOK: false},
		{name: "大文字にすると子音になる非ASCIIの文字", input: "BCDF-GHJſ", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := NormalizeOAuthUserCode(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("NormalizeOAuthUserCode(%q)の成否 = %v、期待値 = %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("NormalizeOAuthUserCode(%q) = %q、期待値 = %q", tt.input, got, tt.want)
			}
		})
	}
}
