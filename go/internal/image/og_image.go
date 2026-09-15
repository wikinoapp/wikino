package image

import (
	"errors"
	"time"
)

// og:image用のリサイズ・フォーマット・署名TTLはBuilder内部の定数として固定する。
// og:imageはOGP推奨の1200x630でサーブし、フォーマットはjpgに固定する。SNSクローラ
// (Slack / X / Discord等) の中にはWebP / AVIFを扱えないものがあり、`auto` ではクローラ
// 側のサムネイル生成が壊れるケースがあるため。署名TTLはHTMLキャッシュ寿命 (s-maxage=300)
// を十分に上回る1時間とし、CDN+ブラウザのキャッシュ追従にも余裕を持たせる。
const (
	ogImageWidth        = 1200
	ogImageHeight       = 630
	ogImageFormat       = "jpg"
	ogImageSignatureTTL = time.Hour
)

// OgImageBuilderはog:image配信用のimgproxy URLを組み立てるヘルパー
//
// og:image専用のリサイズ・フォーマット・TTLポリシーをここに集約する。本文中画像など
// 他用途で別ポリシーが必要になった場合は、別のBuilderを用意する想定。
//
// helper / bucketはmain.goで構築時に確定させる。bucketが空の場合は構築自体を行わず、
// 呼び出し側 (Handler) がnilチェックして500を返すことで「imgproxy設定が不完全な
// 状態でフラグONユーザーが到達したケース」を可視化する。
type OgImageBuilder struct {
	helper *Helper
	bucket string
}

// NewOgImageBuilderは新しいOgImageBuilderを生成する
//
// helperとbucketはいずれも空でないことを前提とする (空の場合はエラー)。空の値で
// Builderを構築するのは設定ミスを覆い隠すだけなので、main.go側でWARNログを出して
// Builder自体を構築しないほうが運用上の事故を見つけやすい。
func NewOgImageBuilder(helper *Helper, bucket string) (*OgImageBuilder, error) {
	if helper == nil {
		return nil, errors.New("og_image: helperがnilです")
	}
	if bucket == "" {
		return nil, errors.New("og_image: bucketが空です")
	}
	return &OgImageBuilder{helper: helper, bucket: bucket}, nil
}

// BuildOgImageURLは元画像のS3 blob keyからog:image用の署名付きimgproxy URLを組み立てる
//
// nowは署名TTLの起点。テスト容易性のため引数で受け取る (本番はtime.Now() を渡す)。
// 戻り値は "{baseURL}/{signature}/resize:fit:1200:630/expires:.../format:jpg/plain/s3://{bucket}/{key}" 形式。
func (b *OgImageBuilder) BuildOgImageURL(blobKey string, now time.Time) (string, error) {
	if blobKey == "" {
		return "", errors.New("og_image: blobKeyが空です")
	}
	sourceURL := "s3://" + b.bucket + "/" + blobKey
	return b.helper.BuildURL(sourceURL, ResizeOptions{
		Width:     ogImageWidth,
		Height:    ogImageHeight,
		Format:    ogImageFormat,
		ExpiresAt: now.Add(ogImageSignatureTTL),
	})
}
