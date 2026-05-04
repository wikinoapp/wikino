// Package auth はユーザー認証に関する機能を提供します。
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost はbcryptのコスト値。テスト時はSetupTestMainでTestBcryptCostに変更される。
var BcryptCost = bcrypt.DefaultCost // 10

// TestBcryptCost はテスト用の低コスト値
const TestBcryptCost = bcrypt.MinCost // 4

// パスワード強度検証の定数
const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

// パスワード強度検証のsentinel error
var (
	ErrPasswordTooShort     = errors.New("password is too short")
	ErrPasswordTooLong      = errors.New("password is too long")
	ErrPasswordInvalidChars = errors.New("password contains invalid characters")
)

// ValidatePasswordStrength はパスワードの強度をチェックする。
// NIST SP 800-63B-4 準拠:
// - 最小文字数: 8文字
// - 最大文字数: 128文字
// - 印字可能ASCII文字のみ許可（0x21〜0x7E、スペースは含まない）
// - 文字種の複雑性要件は廃止（大文字・小文字・数字・記号の組み合わせ要求なし）
//
// エラーは sentinel error として返し、呼び出し側で errors.Is により判別する。
// auth は純粋な技術ユーティリティとして i18n に依存しないため、翻訳の解決は呼び出し側の責務となる。
func ValidatePasswordStrength(password string) error {
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}
	for _, char := range password {
		if char < 0x21 || char > 0x7E {
			return ErrPasswordInvalidChars
		}
	}
	return nil
}

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
