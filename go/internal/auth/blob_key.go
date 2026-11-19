package auth

import (
	"crypto/rand"
	"math/big"
)

// blobKeyCharsetは、添付ファイルのオブジェクトのキーに使う文字。ActiveStorageのキー
// (SecureRandom.base36) と同じく英小文字と数字にする。大文字と小文字を区別しないファイル
// システムやURLでも、別のキーが同じものとして扱われないようにするためである
const blobKeyCharset = "0123456789abcdefghijklmnopqrstuvwxyz"

// blobKeyLengthはキーの文字数。ActiveStorageのキーと同じ28文字にする (36種の28文字で約140ビット)
const blobKeyLength = 28

// GenerateBlobKeyは、添付ファイルのオブジェクトを置くキーを暗号学的乱数で生成する。
// キーはサーバーが決め、利用者が送ったファイル名を含めない
func GenerateBlobKey() (string, error) {
	charsetSize := big.NewInt(int64(len(blobKeyCharset)))
	b := make([]byte, blobKeyLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, charsetSize)
		if err != nil {
			return "", err
		}
		b[i] = blobKeyCharset[n.Int64()]
	}
	return string(b), nil
}
