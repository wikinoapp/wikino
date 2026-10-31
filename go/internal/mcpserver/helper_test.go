package mcpserver

import (
	"os"
	"path/filepath"
	"testing"
)

// testOpenAPIDescriptionは、公開APIのOpenAPI記述 (正本のYAML) を読み込んで返す。
// 記述を埋め込むパッケージ (`api`) への依存を増やさないよう、ファイルから読む
func testOpenAPIDescription(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("OpenAPI記述の読み込みに失敗: %v", err)
	}
	return data
}
