package auth

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateSecureToken は安全なセッショントークンを生成する
// Rails の has_secure_token と互換性のある形式で生成する
// 24バイトのランダムデータをBase64 URL-safeエンコードして32文字のトークンを生成
func GenerateSecureToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
