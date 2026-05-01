// Package image は imgproxy 用の URL 生成ヘルパーを提供します
//
// imgproxy はオンザフライで画像のリサイズ・フォーマット変換を行う外部サービスです。
// このパッケージは og:image 配信や将来の本文中画像配信で共通利用されることを想定し、
// HMAC-SHA256 で署名された imgproxy URL を組み立てます。
//
// 署名の仕様は imgproxy v3 の公式ドキュメントに準拠します。
// https://docs.imgproxy.net/usage/signing_url
package image

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Helper は imgproxy URL を生成するヘルパー
//
// インスタンス生成時に baseURL / key / salt を確定させるため、ハンドラーや UseCase 側では
// シングルトンとして main.go で構築し、依存性注入で渡される想定。
type Helper struct {
	baseURL string
	key     []byte
	salt    []byte
}

// NewHelper は新しい Helper を生成する
//
// baseURL は imgproxy のベース URL (例: "https://imgproxy.example.dev")。末尾スラッシュは除去される。
// keyHex / saltHex は imgproxy が要求する 16 進数文字列形式の秘密鍵と salt。
// 16 進数のデコードに失敗した場合や baseURL が空の場合はエラーを返す。
func NewHelper(baseURL, keyHex, saltHex string) (*Helper, error) {
	if baseURL == "" {
		return nil, errors.New("imgproxy: baseURL が空です")
	}

	key, err := decodeHex(keyHex)
	if err != nil {
		return nil, fmt.Errorf("imgproxy: KEY のデコードに失敗: %w", err)
	}
	salt, err := decodeHex(saltHex)
	if err != nil {
		return nil, fmt.Errorf("imgproxy: SALT のデコードに失敗: %w", err)
	}

	return &Helper{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     key,
		salt:    salt,
	}, nil
}

// ResizeOptions は imgproxy のリサイズ・フォーマット変換オプション
//
// 現状は og:image 用途で必要な項目のみを公開する。他のオプション (gravity, dpr 等) は
// 必要になった時点で追加する。ExpiresAt をゼロ値で渡した場合は expire を付与しない。
type ResizeOptions struct {
	Width     int
	Height    int
	Format    string // "auto", "webp", "avif", "jpg" など。空なら format オプションを付けない
	ExpiresAt time.Time
}

// BuildURL は imgproxy 用の署名付き URL を生成する
//
// sourceURL は imgproxy が画像を取得するソース URL。S3 上の元画像を指す場合は
// "s3://{bucket}/{key}" 形式を使用する。imgproxy 側の IMGPROXY_USE_S3=true および
// エンドポイント設定によって S3 互換ストレージから取得される。
//
// 戻り値の URL は "{baseURL}/{signature}/{processing_options}/{plain|encoded}/{source_url}" 形式。
// processing_options は "resize:fit:1200:630/expires:1234567890/format:auto" のように
// スラッシュ区切りで連結される。
func (h *Helper) BuildURL(sourceURL string, opts ResizeOptions) (string, error) {
	if sourceURL == "" {
		return "", errors.New("imgproxy: sourceURL が空です")
	}

	processing := buildProcessingOptions(opts)
	source := "plain/" + sourceURL

	// 署名対象は "/{processing_options}/{source_part}" の形式
	// salt は imgproxy 側で署名対象の先頭に付与される
	pathToSign := "/" + processing + "/" + source

	signed := h.sign(pathToSign)

	return fmt.Sprintf("%s/%s%s", h.baseURL, signed, pathToSign), nil
}

// sign は salt + path を HMAC-SHA256 でハッシュし、base64 url-safe (パディングなし) を返す
func (h *Helper) sign(path string) string {
	mac := hmac.New(sha256.New, h.key)
	mac.Write(h.salt)
	mac.Write([]byte(path))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// buildProcessingOptions は ResizeOptions を imgproxy のオプション文字列に変換する
//
// 出力例: "resize:fit:1200:630/expires:1234567890/format:auto"
func buildProcessingOptions(opts ResizeOptions) string {
	parts := make([]string, 0, 3)

	if opts.Width > 0 || opts.Height > 0 {
		parts = append(parts, fmt.Sprintf("resize:fit:%d:%d", opts.Width, opts.Height))
	}
	if !opts.ExpiresAt.IsZero() {
		parts = append(parts, fmt.Sprintf("expires:%d", opts.ExpiresAt.Unix()))
	}
	if opts.Format != "" {
		parts = append(parts, "format:"+opts.Format)
	}

	return strings.Join(parts, "/")
}

// decodeHex は 16 進数文字列をバイト列にデコードする
//
// 空文字列はバイト列の長さ 0 として許容する (imgproxy の signing 無効モード相当)。
// ただし NewHelper は明示的に空でないことを前提としていないため、空 key / salt でも
// 形式上は有効な署名が計算できる (ただしセキュリティ強度はゼロ)。
func decodeHex(s string) ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}
	// 大文字小文字を気にせずデコードできるように小文字へ正規化
	s = strings.ToLower(s)

	if len(s)%2 != 0 {
		return nil, fmt.Errorf("奇数長の 16 進数文字列です: 長さ=%d", len(s))
	}

	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		hi, err := hexNibble(s[i*2])
		if err != nil {
			return nil, err
		}
		lo, err := hexNibble(s[i*2+1])
		if err != nil {
			return nil, err
		}
		out[i] = hi<<4 | lo
	}
	return out, nil
}

// hexNibble は 16 進数 1 文字をニブル (0-15) に変換する
func hexNibble(c byte) (byte, error) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', nil
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, nil
	}
	return 0, fmt.Errorf("不正な 16 進数文字: %q", c)
}
