package cimd

import "testing"

func TestIsClientIDMetadataDocumentURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		clientID string
		want     bool
	}{
		{clientID: "https://claude.ai/oauth/claude-code-client-metadata", want: true},
		{clientID: "https://example.com/client.json", want: true},
		{clientID: "http://claude.ai/oauth/client", want: false},
		{clientID: "wikino-cli", want: false},
		{clientID: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.clientID, func(t *testing.T) {
			t.Parallel()

			if got := IsClientIDMetadataDocumentURL(tt.clientID); got != tt.want {
				t.Errorf("IsClientIDMetadataDocumentURL(%q) = %v、期待値 = %v", tt.clientID, got, tt.want)
			}
		})
	}
}

func TestIsAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		clientID string
		want     bool
	}{
		{name: "許可リストのホストのHTTPSのURL", clientID: "https://claude.ai/oauth/claude-code-client-metadata", want: true},
		{name: "ChatGPTのホスト", clientID: "https://chatgpt.com/oauth/client-metadata.json", want: true},
		{name: "VS Codeのホスト", clientID: "https://vscode.dev/oauth/client-metadata.json", want: true},
		{name: "VS Code Insidersのホスト", clientID: "https://insiders.vscode.dev/oauth/client-metadata.json", want: true},
		{name: "クエリを含むURL", clientID: "https://claude.ai/oauth/metadata?v=1", want: true},
		{name: "許可リストの外のホスト", clientID: "https://example.com/client.json", want: false},
		{name: "許可リストのホストのサブドメイン", clientID: "https://evil.claude.ai/client.json", want: false},
		{name: "許可リストのホストを接頭辞に持つホスト", clientID: "https://claude.ai.example.com/client.json", want: false},
		{name: "既定のポートを明示したURL", clientID: "https://claude.ai:443/client.json", want: false},
		{name: "HTTPのURL", clientID: "http://claude.ai/client.json", want: false},
		{name: "パスの無いURL", clientID: "https://claude.ai", want: false},
		{name: "ルートだけのパス", clientID: "https://claude.ai/", want: false},
		{name: "ユーザー情報を含むURL", clientID: "https://user@claude.ai/client.json", want: false},
		{name: "フラグメントを含むURL", clientID: "https://claude.ai/client.json#x", want: false},
		{name: "空のフラグメントを含むURL", clientID: "https://claude.ai/client.json#", want: false},
		{name: "ドットのセグメントを含むURL", clientID: "https://claude.ai/a/../client.json", want: false},
		{name: "1つのドットのセグメントを含むURL", clientID: "https://claude.ai/./client.json", want: false},
		{name: "ホストの大文字小文字が違うURL", clientID: "https://Claude.ai/client.json", want: false},
		{name: "URLでない文字列", clientID: "wikino-cli", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsAllowed(tt.clientID); got != tt.want {
				t.Errorf("IsAllowed(%q) = %v、期待値 = %v", tt.clientID, got, tt.want)
			}
		})
	}
}
