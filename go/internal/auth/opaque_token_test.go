package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateOpaqueToken(t *testing.T) {
	t.Parallel()

	token, err := GenerateOpaqueToken(PersonalAccessTokenPrefix)
	if err != nil {
		t.Fatalf("GenerateOpaqueToken()のエラー = %v", err)
	}

	if !strings.HasPrefix(token, string(PersonalAccessTokenPrefix)) {
		t.Fatalf("トークン %q が接頭辞 %q で始まっていない", token, PersonalAccessTokenPrefix)
	}

	body := strings.TrimPrefix(token, string(PersonalAccessTokenPrefix))
	decoded, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		t.Fatalf("本体 %q がbase64url (パディングなし) として読めない: %v", body, err)
	}
	if len(decoded) != 32 {
		t.Errorf("本体のバイト数 = %d、期待値 = 32", len(decoded))
	}

	other, err := GenerateOpaqueToken(PersonalAccessTokenPrefix)
	if err != nil {
		t.Fatalf("GenerateOpaqueToken()のエラー = %v", err)
	}
	if token == other {
		t.Error("2回の生成で同じトークンが返った")
	}
}

func TestDigestOpaqueToken(t *testing.T) {
	t.Parallel()

	digest := DigestOpaqueToken("wkp_example")

	// "wkp_example" のSHA-256を16進で表したもの
	want := "2981f7ea007088d1f83121ae595bb9e1757f30f1273671b520a1bd5495458dc4"
	if digest != want {
		t.Errorf("DigestOpaqueToken() = %q、期待値 = %q", digest, want)
	}
	if DigestOpaqueToken("wkp_other") == digest {
		t.Error("異なるトークンから同じダイジェストが返った")
	}
}

func TestOpaqueTokenLastChars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
		want  string
	}{
		{name: "末尾の4文字を返す", token: "wkp_abcdefgh", want: "efgh"},
		{name: "4文字以下ならそのまま返す", token: "abc", want: "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := OpaqueTokenLastChars(tt.token); got != tt.want {
				t.Errorf("OpaqueTokenLastChars(%q) = %q、期待値 = %q", tt.token, got, tt.want)
			}
		})
	}
}

func TestGenerateOAuthClientID(t *testing.T) {
	t.Parallel()

	clientID, err := GenerateOAuthClientID()
	if err != nil {
		t.Fatalf("GenerateOAuthClientID()のエラー = %v", err)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(clientID)
	if err != nil {
		t.Fatalf("クライアントID %q がbase64url (パディングなし) として読めない: %v", clientID, err)
	}
	if len(decoded) != 16 {
		t.Errorf("クライアントIDのバイト数 = %d、期待値 = 16", len(decoded))
	}

	other, err := GenerateOAuthClientID()
	if err != nil {
		t.Fatalf("GenerateOAuthClientID()のエラー = %v", err)
	}
	if clientID == other {
		t.Error("2回の生成で同じクライアントIDが返った")
	}
}
