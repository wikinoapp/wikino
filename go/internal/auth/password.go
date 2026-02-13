// Package auth はユーザー認証に関する機能を提供します。
package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// BcryptCost はbcryptのコスト値。テスト時はSetupTestMainでTestBcryptCostに変更される。
var BcryptCost = bcrypt.DefaultCost // 10

// TestBcryptCost はテスト用の低コスト値
const TestBcryptCost = bcrypt.MinCost // 4

// VerifyPassword はパスワードがハッシュと一致するかを検証します。
// Rails版の has_secure_password (bcrypt) との互換性を保ちます。
func VerifyPassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// HashPassword はパスワードをbcryptでハッシュ化します。
// BcryptCost変数で指定されたコストを使用します。
func HashPassword(plainPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
