package auth

import (
	"crypto/rand"
	"math/big"
)

const (
	// RecoveryCodeCountは二要素認証を有効にしたときに発行するリカバリーコードの個数
	RecoveryCodeCount = 10

	// recoveryCodeLengthはリカバリーコード1個の文字数
	recoveryCodeLength = 8

	// recoveryCodeAlphabetはリカバリーコードに使う文字 (小文字の英字と数字)
	recoveryCodeAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// GenerateRecoveryCodesは、8文字の小文字英数字のリカバリーコードを暗号論的乱数で10個作る。
//
// Rails版と同じ形式にし、どちらの実装で発行したコードもログインのリカバリーコードの検証で通るようにする。
func GenerateRecoveryCodes() ([]string, error) {
	codes := make([]string, RecoveryCodeCount)
	alphabetSize := big.NewInt(int64(len(recoveryCodeAlphabet)))
	for i := range codes {
		b := make([]byte, recoveryCodeLength)
		for j := range b {
			n, err := rand.Int(rand.Reader, alphabetSize)
			if err != nil {
				return nil, err
			}
			b[j] = recoveryCodeAlphabet[n.Int64()]
		}
		codes[i] = string(b)
	}

	return codes, nil
}
