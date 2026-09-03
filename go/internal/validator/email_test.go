package validator_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestIsValidEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{name: "通常のアドレス", email: "user@example.com", want: true},
		{name: "サブアドレス付き", email: "user+tag@example.com", want: true},
		{name: "空文字列", email: "", want: false},
		{name: "アットマークが無い", email: "invalid-email", want: false},
		{name: "ローカル部が無い", email: "@example.com", want: false},
		{name: "ドメインが無い", email: "user@", want: false},
		{name: "アドレスが2つ並んでいる", email: "a@example.com, b@example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := validator.IsValidEmail(tt.email); got != tt.want {
				t.Errorf("IsValidEmail(%q) = %v であることを期待したが %v だった", tt.email, tt.want, got)
			}
		})
	}
}

func TestCanonicalEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		email         string
		wantAddress   string
		wantCanonical bool
	}{
		{
			name:          "アドレスだけが書かれている",
			email:         "user@example.com",
			wantAddress:   "user@example.com",
			wantCanonical: true,
		},
		// The three cases below are what separates this function from
		// IsValidEmail: parsing accepts them all and hands back the same
		// address, while what was written differs from it.
		//
		// [Ja] 以下の 3 件が本関数と IsValidEmail を分けるもの。解釈はいずれも受理
		// して同じアドレスを返すが、書かれた文字列はそれと食い違う。
		{
			name:          "前後に空白がある",
			email:         " user@example.com ",
			wantAddress:   "user@example.com",
			wantCanonical: false,
		},
		{
			name:          "末尾に空白がある",
			email:         "user@example.com ",
			wantAddress:   "user@example.com",
			wantCanonical: false,
		},
		{
			name:          "表示名が付いている",
			email:         "テストユーザー <user@example.com>",
			wantAddress:   "user@example.com",
			wantCanonical: false,
		},
		{
			name:          "解釈できない",
			email:         "invalid-email",
			wantAddress:   "",
			wantCanonical: false,
		},
		{
			name:          "空文字列",
			email:         "",
			wantAddress:   "",
			wantCanonical: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			address, canonical := validator.CanonicalEmail(tt.email)
			if address != tt.wantAddress {
				t.Errorf("CanonicalEmail(%q) のアドレスが %q であることを期待したが %q だった", tt.email, tt.wantAddress, address)
			}
			if canonical != tt.wantCanonical {
				t.Errorf("CanonicalEmail(%q) の判定が %v であることを期待したが %v だった", tt.email, tt.wantCanonical, canonical)
			}
		})
	}
}
