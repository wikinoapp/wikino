package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// OpaqueTokenPrefixはopaqueトークンの種類を表す接頭辞。ログや漏洩時に種類を判別できる
// ように、値の先頭に付ける
type OpaqueTokenPrefix string

// PersonalAccessTokenPrefixは個人アクセストークンの接頭辞
const PersonalAccessTokenPrefix OpaqueTokenPrefix = "wkp_"

// OAuthClientSecretPrefixはOAuthアプリのクライアントシークレットの接頭辞
const OAuthClientSecretPrefix OpaqueTokenPrefix = "wks_"

// OAuthAccessTokenPrefixはOAuthのアクセストークンの接頭辞。公開APIのトークン照合は、
// この接頭辞で個人アクセストークンと見分ける
const OAuthAccessTokenPrefix OpaqueTokenPrefix = "wka_"

// OAuthRefreshTokenPrefixはOAuthのリフレッシュトークンの接頭辞
const OAuthRefreshTokenPrefix OpaqueTokenPrefix = "wkr_"

// OAuthAuthorizationCodePrefixはOAuthの認可コードの接頭辞。コードはリダイレクトの直後に
// 数分で使い切られ、ログや漏洩時に種類を見分ける場面が無いため、接頭辞を付けない
const OAuthAuthorizationCodePrefix OpaqueTokenPrefix = ""

// opaqueTokenRandomBytesはopaqueトークンの本体にする乱数のバイト数
const opaqueTokenRandomBytes = 32

// oauthClientIDRandomBytesはOAuthアプリのクライアントIDにする乱数のバイト数
const oauthClientIDRandomBytes = 16

// opaqueTokenLastCharsLengthは一覧でトークンを見分けるために残す末尾の文字数
const opaqueTokenLastCharsLength = 4

// GenerateOpaqueTokenは、接頭辞に32バイトの暗号学的乱数をbase64url (パディングなし) で
// 表したものを続けたトークンを生成する
func GenerateOpaqueToken(prefix OpaqueTokenPrefix) (string, error) {
	b := make([]byte, opaqueTokenRandomBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return string(prefix) + base64.RawURLEncoding.EncodeToString(b), nil
}

// DigestOpaqueTokenはトークンのSHA-256ダイジェストを16進文字列で返す。データベースには
// トークンの値ではなくこのダイジェストを保存し、照合もダイジェスト同士で行う
func DigestOpaqueToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// OpaqueTokenLastCharsは、一覧でトークンを見分けるために保存する末尾の数文字を返す
func OpaqueTokenLastChars(token string) string {
	if len(token) <= opaqueTokenLastCharsLength {
		return token
	}
	return token[len(token)-opaqueTokenLastCharsLength:]
}

// GenerateOAuthClientIDは、OAuthアプリのクライアントIDとして、16バイトの暗号学的乱数を
// base64url (パディングなし) で表したものを生成する。クライアントIDは秘密ではなく、
// 認可要求のURLにそのまま載るため、接頭辞を付けずURLで扱いやすい文字だけにする
func GenerateOAuthClientID() (string, error) {
	b := make([]byte, oauthClientIDRandomBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
