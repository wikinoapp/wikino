package auth

import (
	"regexp"
	"testing"
)

func TestGenerateBlobKey(t *testing.T) {
	t.Parallel()

	pattern := regexp.MustCompile(`^[0-9a-z]{28}$`)
	seen := map[string]bool{}
	for range 100 {
		key, err := GenerateBlobKey()
		if err != nil {
			t.Fatalf("GenerateBlobKey()のエラー = %v", err)
		}
		if !pattern.MatchString(key) {
			t.Errorf("GenerateBlobKey() = %q、期待値 = 英小文字と数字の28文字", key)
		}
		if seen[key] {
			t.Errorf("GenerateBlobKey() = %qが重複した", key)
		}
		seen[key] = true
	}
}
