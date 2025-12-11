// Package auth はユーザー認証に関する機能を提供します。
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// VerifyPassword はパスワードがハッシュと一致するかを検証します。
// Rails版の has_secure_password (bcrypt) との互換性を保ちます。
func VerifyPassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// HashPassword はパスワードをbcryptでハッシュ化します。
// デフォルトコストを使用します（bcrypt.DefaultCost = 10）。
func HashPassword(plainPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
