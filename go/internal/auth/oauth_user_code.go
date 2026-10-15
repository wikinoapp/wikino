package auth

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// oauthUserCodeCharsetは、デバイス認可のユーザーコードに使う文字。読み違えやすい母音と、
// 数字・記号と見分けにくい文字を除いた20種の子音である (RFC 8628 §6.1)。母音を除くと、
// 偶然に単語の形になることも避けられる
const oauthUserCodeCharset = "BCDFGHJKLMNPQRSTVWXZ"

// oauthUserCodeLengthはユーザーコードの文字数。20種の8文字で約2.56×10^10通りになる
const oauthUserCodeLength = 8

// oauthUserCodeSeparatorは、ユーザーコードを読みやすく表示するために4文字ごとに挟む区切り
const oauthUserCodeSeparator = "-"

// GenerateOAuthUserCodeは、デバイス認可のユーザーコードを暗号学的乱数で生成する。
// 区切りを含まない8文字の大文字で返す。この形のダイジェストを保存し、表示には
// FormatOAuthUserCodeで区切りを挟む
func GenerateOAuthUserCode() (string, error) {
	charsetSize := big.NewInt(int64(len(oauthUserCodeCharset)))
	b := make([]byte, oauthUserCodeLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, charsetSize)
		if err != nil {
			return "", err
		}
		b[i] = oauthUserCodeCharset[n.Int64()]
	}
	return string(b), nil
}

// FormatOAuthUserCodeは、GenerateOAuthUserCodeが返したユーザーコードを `BCDF-GHJK` のように
// 4文字ごとに区切って表示する形にする
func FormatOAuthUserCode(code string) string {
	half := len(code) / 2
	return code[:half] + oauthUserCodeSeparator + code[half:]
}

// NormalizeOAuthUserCodeは、利用者が入力したユーザーコードから区切りと空白を除いて大文字に
// そろえ、GenerateOAuthUserCodeと同じ形にする。大文字小文字と区切りの有無は問わない。
// ユーザーコードとして成り立たない入力ではfalseを返す。
//
// 大文字にそろえるのはASCIIの小文字だけにし、ASCII以外の文字は拒否する。Unicodeの大文字への
// 変換で子音になる文字 (`ſ` など) を含む入力を、同じコードとして通さないため
func NormalizeOAuthUserCode(input string) (string, bool) {
	var b strings.Builder
	for _, r := range input {
		if 'a' <= r && r <= 'z' {
			r -= 'a' - 'A'
		}
		switch {
		case string(r) == oauthUserCodeSeparator || r == ' ' || r == '\t':
			continue
		case strings.ContainsRune(oauthUserCodeCharset, r):
			b.WriteRune(r)
		default:
			return "", false
		}
	}
	if b.Len() != oauthUserCodeLength {
		return "", false
	}
	return b.String(), true
}
