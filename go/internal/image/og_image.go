package image

import (
	"errors"
	"time"
)

// og:image 用のリサイズ・フォーマット・署名 TTL は Builder 内部の定数として固定する。
// og:image は OGP 推奨の 1200x630 でサーブし、フォーマットは jpg に固定する。SNS クローラ
// (Slack / X / Discord 等) の中には WebP / AVIF を扱えないものがあり、`auto` ではクローラ
// 側のサムネイル生成が壊れるケースがあるため。署名 TTL は HTML キャッシュ寿命 (s-maxage=300)
// を十分に上回る 1 時間とし、CDN+ブラウザのキャッシュ追従にも余裕を持たせる。
const (
	ogImageWidth        = 1200
	ogImageHeight       = 630
	ogImageFormat       = "jpg"
	ogImageSignatureTTL = time.Hour
)

// OgImageBuilder は og:image 配信用の imgproxy URL を組み立てるヘルパー
//
// og:image 専用のリサイズ・フォーマット・TTL ポリシーをここに集約する。本文中画像など
// 他用途で別ポリシーが必要になった場合は、別の Builder を用意する想定。
//
// helper / bucket は main.go で構築時に確定させる。bucket が空の場合は構築自体を行わず、
// 呼び出し側 (Handler) が nil チェックして 500 を返すことで「imgproxy 設定が不完全な
// 状態でフラグ ON ユーザーが到達したケース」を可視化する。
type OgImageBuilder struct {
	helper *Helper
	bucket string
}

// NewOgImageBuilder は新しい OgImageBuilder を生成する
//
// helper と bucket はいずれも空でないことを前提とする (空の場合はエラー)。空の値で
// Builder を構築するのは設定ミスを覆い隠すだけなので、main.go 側で WARN ログを出して
// Builder 自体を構築しないほうが運用上の事故を見つけやすい。
func NewOgImageBuilder(helper *Helper, bucket string) (*OgImageBuilder, error) {
	if helper == nil {
		return nil, errors.New("og_image: helper が nil です")
	}
	if bucket == "" {
		return nil, errors.New("og_image: bucket が空です")
	}
	return &OgImageBuilder{helper: helper, bucket: bucket}, nil
}

// BuildOgImageURL は元画像の S3 blob key から og:image 用の署名付き imgproxy URL を組み立てる
//
// now は署名 TTL の起点。テスト容易性のため引数で受け取る (本番は time.Now() を渡す)。
// 戻り値は "{baseURL}/{signature}/resize:fit:1200:630/expires:.../format:jpg/plain/s3://{bucket}/{key}" 形式。
func (b *OgImageBuilder) BuildOgImageURL(blobKey string, now time.Time) (string, error) {
	if blobKey == "" {
		return "", errors.New("og_image: blobKey が空です")
	}
	sourceURL := "s3://" + b.bucket + "/" + blobKey
	return b.helper.BuildURL(sourceURL, ResizeOptions{
		Width:     ogImageWidth,
		Height:    ogImageHeight,
		Format:    ogImageFormat,
		ExpiresAt: now.Add(ogImageSignatureTTL),
	})
}
