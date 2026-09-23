// Package imageはimgproxy用のURL生成ヘルパーを提供します
//
// imgproxyはオンザフライで画像のリサイズ・フォーマット変換を行う外部サービスです。
// このパッケージはog:image配信や将来の本文中画像配信で共通利用されることを想定し、
// HMAC-SHA256で署名されたimgproxy URLを組み立てます。
//
// 署名の仕様はimgproxy v3の公式ドキュメントに準拠します。
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

// Helperはimgproxy URLを生成するヘルパー
//
// インスタンス生成時にbaseURL / key / saltを確定させるため、ハンドラーやUseCase側では
// シングルトンとしてmain.goで構築し、依存性注入で渡される想定。
type Helper struct {
	baseURL string
	key     []byte
	salt    []byte
}

// NewHelperは新しいHelperを生成する
//
// baseURLはimgproxyのベースURL (例: "https://imgproxy.example.dev")。末尾スラッシュは除去される。
// keyHex / saltHexはimgproxyが要求する16進数文字列形式の秘密鍵とsalt。
// 16進数のデコードに失敗した場合やbaseURLが空の場合はエラーを返す。
func NewHelper(baseURL, keyHex, saltHex string) (*Helper, error) {
	if baseURL == "" {
		return nil, errors.New("imgproxy: baseURLが空です")
	}

	key, err := decodeHex(keyHex)
	if err != nil {
		return nil, fmt.Errorf("imgproxy: KEYのデコードに失敗: %w", err)
	}
	salt, err := decodeHex(saltHex)
	if err != nil {
		return nil, fmt.Errorf("imgproxy: SALTのデコードに失敗: %w", err)
	}

	return &Helper{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     key,
		salt:    salt,
	}, nil
}

// ResizeOptionsはimgproxyのリサイズ・フォーマット変換オプション
//
// 現状はog:image用途で必要な項目のみを公開する。他のオプション (gravity, dpr等) は
// 必要になった時点で追加する。ExpiresAtをゼロ値で渡した場合はexpireを付与しない。
type ResizeOptions struct {
	Width     int
	Height    int
	Format    string // "auto", "webp", "avif", "jpg" など。空ならformatオプションを付けない
	ExpiresAt time.Time
}

// BuildURLはimgproxy用の署名付きURLを生成する
//
// sourceURLはimgproxyが画像を取得するソースURL。S3上の元画像を指す場合は
// "s3://{bucket}/{key}" 形式を使用する。imgproxy側のIMGPROXY_USE_S3=trueおよび
// エンドポイント設定によってS3互換ストレージから取得される。
//
// 戻り値のURLは "{baseURL}/{signature}/{processing_options}/{plain|encoded}/{source_url}" 形式。
// processing_optionsは "resize:fit:1200:630/expires:1234567890/format:auto" のように
// スラッシュ区切りで連結される。
func (h *Helper) BuildURL(sourceURL string, opts ResizeOptions) (string, error) {
	if sourceURL == "" {
		return "", errors.New("imgproxy: sourceURLが空です")
	}

	processing := buildProcessingOptions(opts)
	source := "plain/" + sourceURL

	// 署名対象は "/{processing_options}/{source_part}" の形式
	// saltはimgproxy側で署名対象の先頭に付与される
	pathToSign := "/" + processing + "/" + source

	signed := h.sign(pathToSign)

	return fmt.Sprintf("%s/%s%s", h.baseURL, signed, pathToSign), nil
}

// signはsalt + pathをHMAC-SHA256でハッシュし、base64 url-safe (パディングなし) を返す
func (h *Helper) sign(path string) string {
	mac := hmac.New(sha256.New, h.key)
	mac.Write(h.salt)
	mac.Write([]byte(path))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// buildProcessingOptionsはResizeOptionsをimgproxyのオプション文字列に変換する
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

// decodeHexは16進数文字列をバイト列にデコードする
//
// 空文字列はバイト列の長さ0として許容する (imgproxyのsigning無効モード相当)。
// ただしNewHelperは明示的に空でないことを前提としていないため、空key / saltでも
// 形式上は有効な署名が計算できる (ただしセキュリティ強度はゼロ)。
func decodeHex(s string) ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}
	// 大文字小文字を気にせずデコードできるように小文字へ正規化
	s = strings.ToLower(s)

	if len(s)%2 != 0 {
		return nil, fmt.Errorf("奇数長の16進数文字列です: 長さ=%d", len(s))
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

// hexNibbleは16進数1文字をニブル (0-15) に変換する
func hexNibble(c byte) (byte, error) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', nil
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, nil
	}
	return 0, fmt.Errorf("不正な16進数文字: %q", c)
}
