// Package authはユーザー認証に関する機能を提供します。
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCostはbcryptのコスト値。テスト時はSetupTestMainでTestBcryptCostに変更される。
var BcryptCost = bcrypt.DefaultCost // 10

// TestBcryptCostはテスト用の低コスト値
const TestBcryptCost = bcrypt.MinCost // 4

// パスワード強度検証の定数
const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
)

// パスワード強度検証のsentinel error
var (
	ErrPasswordTooShort     = errors.New("パスワードが短すぎます")
	ErrPasswordTooLong      = errors.New("パスワードが長すぎます")
	ErrPasswordInvalidChars = errors.New("パスワードに使用できない文字が含まれています")
)

// ValidatePasswordStrengthはパスワードの強度をチェックする。
// NIST SP 800-63B-4準拠:
// - 最小文字数: 8文字
// - 最大文字数: 128文字
// - 印字可能ASCII文字のみ許可 (0x21〜0x7E、スペースは含まない)
// - 文字種の複雑性要件は廃止 (大文字・小文字・数字・記号の組み合わせ要求なし)
//
// エラーはsentinel errorとして返し、呼び出し側でerrors.Isにより判別する。
// authは純粋な技術ユーティリティとしてi18nに依存しないため、翻訳の解決は呼び出し側の責務となる。
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

// VerifyPasswordはパスワードがハッシュと一致するかを検証します。
// Rails版のhas_secure_password (bcrypt) との互換性を保ちます。
func VerifyPassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// HashPasswordはパスワードをbcryptでハッシュ化します。
// BcryptCost変数で指定されたコストを使用します。
func HashPassword(plainPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
