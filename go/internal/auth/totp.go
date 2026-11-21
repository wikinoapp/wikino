package auth

import (
	"crypto/rand"
	"encoding/base32"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	// totpIssuerは認証アプリに表示し、otpauth URIに埋め込むissuer
	totpIssuer = "Wikino"

	// totpSecretBytesは生成するTOTPの秘密鍵のバイト数。
	// 20バイト (160ビット) はRFC 4226が推奨する長さで、base32で32文字になる。
	totpSecretBytes = 20

	// totpDriftは、時計のずれを許すため照合の基準時刻を前後にずらす秒数
	totpDrift = 15 * time.Second
)

// totpOptsはコードの照合に使うパラメーター。認証アプリの既定値 (SHA1・6桁・30秒) にし、
// 基準時刻の前後1タイムステップも受け付ける。
var totpOpts = totp.ValidateOpts{
	Period:    30,
	Skew:      1,
	Digits:    otp.DigitsSix,
	Algorithm: otp.AlgorithmSHA1,
}

// totpSecretEncodingはpquerna/otpが秘密鍵に使うbase32 (標準のアルファベット・パディング無し)
var totpSecretEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateTOTPSecretは暗号論的乱数のTOTPの秘密鍵をbase32で返す。
// 返す値は認証アプリへ手で入力するキーとしてもそのまま表示できる。
func GenerateTOTPSecret() (string, error) {
	b := make([]byte, totpSecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return totpSecretEncoding.EncodeToString(b), nil
}

// BuildOTPAuthURLはbase32の秘密鍵を認証アプリへ登録するotpauth URIを返す。
// accountNameは認証アプリに表示するアカウントの名前 (メールアドレス)。
func BuildOTPAuthURL(secret, accountName string) (string, error) {
	raw, err := totpSecretEncoding.DecodeString(secret)
	if err != nil {
		return "", err
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: accountName,
		Period:      totpOpts.Period,
		Digits:      totpOpts.Digits,
		Algorithm:   totpOpts.Algorithm,
		Secret:      raw,
	})
	if err != nil {
		return "", err
	}

	return key.URL(), nil
}

// ValidateTOTPCodeは、codeがnowの時点で秘密鍵secretのコードとして通るかを返す。
//
// ログインと二要素認証の有効化で同じ許容幅にするため、どちらもこの関数で照合する。
// 基準時刻を前後15秒ずらした時点でも照合し、それぞれ前後1タイムステップまで受け付ける。
// 読めない秘密鍵や6桁の数字でないコードは、エラーにせず不一致として扱う。
func ValidateTOTPCode(code, secret string, now time.Time) bool {
	for _, at := range []time.Time{now, now.Add(-totpDrift), now.Add(totpDrift)} {
		if valid, err := totp.ValidateCustom(code, secret, at, totpOpts); err == nil && valid {
			return true
		}
	}

	return false
}
