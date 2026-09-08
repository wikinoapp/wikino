package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const (
	// defaultRegion is the region R2 expects when the deployment does not name one. R2 serves a
	// single global endpoint and reads the region only to build the request signature.
	//
	// [Ja] defaultRegion はリージョンを指定しないデプロイに対して R2 が期待する値。R2 の
	// エンドポイントは全世界で 1 つで、リージョンはリクエストの署名を組み立てるためだけに読まれる。
	defaultRegion = "auto"

	// abortMultipartUploadTimeout bounds the call that cleans up after a failed multipart upload.
	// The cleanup runs on a context detached from the upload, so nothing else stops it from
	// hanging.
	//
	// [Ja] abortMultipartUploadTimeout は、失敗した multipart upload の後片付けを行う呼び出しの
	// 上限時間。後片付けはアップロードから切り離した context で走るため、これ以外にハングを
	// 止めるものが無い。
	abortMultipartUploadTimeout = 30 * time.Second
)

// Config holds what it takes to reach the bucket. The caller fills it from the environment so that
// this package does not depend on internal/config.
//
// [Ja] Config はバケットへ到達するために必要な設定を保持する。呼び出し側が環境変数から値を詰める
// ことで、本パッケージを internal/config から独立させる。
type Config struct {
	BucketName      string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string

	// Region may stay empty, and then defaultRegion is used.
	//
	// [Ja] Region は空でもよく、その場合は defaultRegion を使う。
	Region string
}

// S3ObjectStorage is the ObjectStorage backed by an S3-compatible bucket.
//
// [Ja] S3ObjectStorage は S3 互換のバケットを実体とする ObjectStorage。
type S3ObjectStorage struct {
	client        *s3.Client
	presignClient *s3.PresignClient

	// feature/s3/manager is deprecated in favor of feature/s3/transfermanager, but the successor
	// is still a v0 preview module with no promise about its API. Stay on the stable v1 manager
	// and revisit once transfermanager reaches v1.
	//
	// [Ja] feature/s3/manager は feature/s3/transfermanager への移行が案内され deprecated だが、
	// 後継はまだ v0 のプレビューモジュールで API について何も保証していない。安定版 v1 の manager
	// を使い続け、transfermanager が v1 になった時点で見直す。
	uploader *manager.Uploader //nolint:staticcheck // SA1019: see the comment above
	bucket   string
}

// NewS3ObjectStorage builds the client for the configured bucket. An incomplete Config is an
// error rather than a client that fails on its first request, so that a deployment missing a
// setting is reported where it is set up.
//
// [Ja] NewS3ObjectStorage は設定されたバケット用のクライアントを構築する。Config が欠けている
// 場合は、最初のリクエストで失敗するクライアントを返さずエラーにする。設定が足りないデプロイを
// 組み立ての場所で報告するためである。
func NewS3ObjectStorage(cfg Config) (*S3ObjectStorage, error) {
	switch {
	case cfg.BucketName == "":
		return nil, errors.New("storage: BucketName が空です")
	case cfg.Endpoint == "":
		return nil, errors.New("storage: Endpoint が空です")
	case cfg.AccessKeyID == "":
		return nil, errors.New("storage: AccessKeyID が空です")
	case cfg.SecretAccessKey == "":
		return nil, errors.New("storage: SecretAccessKey が空です")
	}

	region := cfg.Region
	if region == "" {
		region = defaultRegion
	}

	awsCfg := aws.Config{
		Region:       region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		BaseEndpoint: aws.String(cfg.Endpoint),

		// Send and verify the additional checksums only where the operation requires them. The
		// SDK otherwise adds a CRC32 to every request, which the Rails version had to turn off
		// against this same bucket as well (see rails/config/storage.yml).
		//
		// [Ja] 追加のチェックサムは、操作が要求する場合にだけ送受信する。そうしないと SDK は
		// すべてのリクエストに CRC32 を付ける。同じバケットに対して Rails 版でも無効化が必要に
		// なったものである (rails/config/storage.yml を参照)。
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// Keep the bucket name in the URL path instead of the host name. R2 accepts both forms,
		// and the path form also works against an endpoint that has no per-bucket DNS, which is
		// what the tests run against.
		//
		// [Ja] バケット名をホスト名ではなく URL のパスに置く。R2 はどちらの形式も受け付け、
		// パス形式はバケットごとの DNS を持たないエンドポイントに対しても動作する。テストが
		// 相手にするのはこの形のエンドポイントである。
		o.UsePathStyle = true
	})

	return &S3ObjectStorage{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		// LeavePartsOnError turns off the uploader's own abort, which reuses the upload context:
		// an upload that failed because the context was canceled would cancel its abort too, and
		// the parts already uploaded would stay in the bucket and be billed. Upload aborts them
		// itself on a detached context instead.
		//
		// [Ja] LeavePartsOnError はアップローダー組み込みの abort を無効化する。組み込みの abort
		// は upload と同じ context を使い回すため、context のキャンセルで失敗したアップロードは
		// abort までキャンセルされ、アップロード済みのパートがバケットに残って課金される。
		// 代わりに Upload が切り離した context で自ら abort する。
		uploader: manager.NewUploader(client, func(u *manager.Uploader) { //nolint:staticcheck // SA1019: see the uploader field comment
			u.LeavePartsOnError = true
		}),
		bucket: cfg.BucketName,
	}, nil
}

// Get returns the object body as a stream.
//
// [Ja] Get はオブジェクト本体のストリームを返す。
func (s *S3ObjectStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNoSuchKey(err) {
			return nil, fmt.Errorf("%w (key: %s)", ErrObjectNotFound, key)
		}
		return nil, fmt.Errorf("storage: オブジェクトの取得に失敗しました (key: %s): %w", key, err)
	}
	return out.Body, nil
}

// Upload streams the body to the bucket, buffering bounded parts instead of holding the whole
// body in memory. The uploader switches to the multipart form once the body passes its part size,
// so a ZIP too large for a single PUT is sent in parts without the caller doing anything.
//
// [Ja] Upload は body 全体をメモリに保持せず、一定サイズのパートをバッファしながらバケットへ
// ストリーミングする。アップローダーは body がパートサイズを超えた時点で multipart の形式へ
// 切り替えるため、単一の PUT では送れないサイズの ZIP も呼び出し側は何もせずに送れる。
func (s *S3ObjectStorage) Upload(ctx context.Context, input UploadInput) error {
	putInput := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(input.Key),
		Body:   input.Body,
	}
	if input.ContentType != "" {
		putInput.ContentType = aws.String(input.ContentType)
	}

	if _, err := s.uploader.Upload(ctx, putInput); err != nil { //nolint:staticcheck // SA1019: see the uploader field comment
		if abortErr := s.abortMultipartUpload(ctx, input.Key, err); abortErr != nil {
			// Keep both failures: the abort tells why the parts are still there, and the upload
			// error tells why the abort was needed at all.
			//
			// [Ja] どちらの失敗も残す。abort のエラーはパートが残っている理由を示し、upload の
			// エラーはそもそも abort が必要になった理由を示すためである。
			err = errors.Join(err, abortErr)
		}
		return fmt.Errorf("storage: オブジェクトのアップロードに失敗しました (key: %s): %w", input.Key, err)
	}
	return nil
}

// abortMultipartUpload discards the parts a failed Upload left in the bucket, and does nothing
// when the upload never reached the multipart form. It runs on a context detached from the upload
// so that it still happens when the upload failed because that context was canceled.
//
// [Ja] abortMultipartUpload は、失敗した Upload がバケットに残したパートを破棄する。アップロード
// が multipart の形式に至っていない場合は何もしない。アップロードから切り離した context で走る
// ため、その context のキャンセルが原因で失敗した場合でも実行される。
func (s *S3ObjectStorage) abortMultipartUpload(ctx context.Context, key string, uploadErr error) error {
	var multiErr manager.MultiUploadFailure
	if !errors.As(uploadErr, &multiErr) {
		return nil
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), abortMultipartUploadTimeout)
	defer cancel()

	if _, err := s.client.AbortMultipartUpload(cleanupCtx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(multiErr.UploadID()),
	}); err != nil {
		return fmt.Errorf("storage: multipart upload の中断に失敗しました (key: %s): %w", key, err)
	}
	return nil
}

// Delete removes the object under key.
//
// [Ja] Delete は key の位置のオブジェクトを削除する。
func (s *S3ObjectStorage) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		if isNoSuchKey(err) {
			return nil
		}
		return fmt.Errorf("storage: オブジェクトの削除に失敗しました (key: %s): %w", key, err)
	}
	return nil
}

// PresignedGetURL signs a GET for the object under key.
//
// [Ja] PresignedGetURL は key のオブジェクトに対する GET に署名する。
func (s *S3ObjectStorage) PresignedGetURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("storage: presigned URL の生成に失敗しました (key: %s): %w", key, err)
	}
	return req.URL, nil
}

// isNoSuchKey reports whether err says the object is not there. NoSuchKey is the only code that
// means it: a 404 carrying NoSuchBucket says the bucket itself is missing or unreachable, which is
// a real failure to report rather than an object to treat as absent.
//
// [Ja] isNoSuchKey は err がオブジェクトの不在を示しているかどうかを返す。それを意味するコードは
// NoSuchKey だけである。NoSuchBucket を伴う 404 はバケット自体が存在しないか到達できないことを
// 示しており、オブジェクトの不在として扱うのではなく報告すべき本物の失敗である。
func isNoSuchKey(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey"
}
