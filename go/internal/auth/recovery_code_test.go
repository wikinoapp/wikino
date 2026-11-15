package auth

import (
	"regexp"
	"testing"
)

func TestGenerateRecoveryCodes(t *testing.T) {
	t.Parallel()

	codes, err := GenerateRecoveryCodes()
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes()のエラー = %v", err)
	}
	if len(codes) != RecoveryCodeCount {
		t.Fatalf("個数 = %d、期待値 = %d", len(codes), RecoveryCodeCount)
	}

	// ログインのリカバリーコードの検証と同じ形式
	format := regexp.MustCompile(`^[a-z0-9]{8}$`)
	seen := map[string]bool{}
	for _, code := range codes {
		if !format.MatchString(code) {
			t.Errorf("コード = %q、8文字の小文字英数字を期待", code)
		}
		seen[code] = true
	}
	if len(seen) != len(codes) {
		t.Errorf("重複の無いコードの個数 = %d、期待値 = %d", len(seen), len(codes))
	}
}
