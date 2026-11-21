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
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const (
	// defaultRegionはリージョンを指定しないデプロイに対してR2が期待する値。R2の
	// エンドポイントは全世界で1つで、リージョンはリクエストの署名を組み立てるためだけに読まれる。
	defaultRegion = "auto"

	// abortMultipartUploadTimeoutは、失敗したmultipart uploadの後片付けを行う呼び出しの
	// 上限時間。後片付けはアップロードから切り離したcontextで走るため、これ以外にハングを
	// 止めるものが無い。
	abortMultipartUploadTimeout = 30 * time.Second
)

// Configはバケットへ到達するために必要な設定を保持する。呼び出し側が環境変数から値を詰める
// ことで、本パッケージをinternal/configから独立させる。
type Config struct {
	BucketName      string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string

	// Regionは空でもよく、その場合はdefaultRegionを使う。
	Region string
}

// S3ObjectStorageはS3互換のバケットを実体とするObjectStorage。
type S3ObjectStorage struct {
	client        *s3.Client
	presignClient *s3.PresignClient

	// feature/s3/managerはfeature/s3/transfermanagerへの移行が案内されdeprecatedだが、
	// 後継はまだv0のプレビューモジュールでAPIについて何も保証していない。安定版v1のmanager
	// を使い続け、transfermanagerがv1になった時点で見直す。
	uploader *manager.Uploader //nolint:staticcheck // SA1019: see the comment above
	bucket   string
}

// NewS3ObjectStorageは設定されたバケット用のクライアントを構築する。Configが欠けている
// 場合は、最初のリクエストで失敗するクライアントを返さずエラーにする。設定が足りないデプロイを
// 組み立ての場所で報告するためである。
func NewS3ObjectStorage(cfg Config) (*S3ObjectStorage, error) {
	switch {
	case cfg.BucketName == "":
		return nil, errors.New("storage: BucketNameが空です")
	case cfg.Endpoint == "":
		return nil, errors.New("storage: Endpointが空です")
	case cfg.AccessKeyID == "":
		return nil, errors.New("storage: AccessKeyIDが空です")
	case cfg.SecretAccessKey == "":
		return nil, errors.New("storage: SecretAccessKeyが空です")
	}

	region := cfg.Region
	if region == "" {
		region = defaultRegion
	}

	awsCfg := aws.Config{
		Region:       region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		BaseEndpoint: aws.String(cfg.Endpoint),

		// 追加のチェックサムは、操作が要求する場合にだけ送受信する。そうしないとSDKは
		// すべてのリクエストにCRC32を付ける。
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// バケット名をホスト名ではなくURLのパスに置く。R2はどちらの形式も受け付け、
		// パス形式はバケットごとのDNSを持たないエンドポイントに対しても動作する。テストが
		// 相手にするのはこの形のエンドポイントである。
		o.UsePathStyle = true
	})

	return &S3ObjectStorage{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		// LeavePartsOnErrorはアップローダー組み込みのabortを無効化する。組み込みのabort
		// はuploadと同じcontextを使い回すため、contextのキャンセルで失敗したアップロードは
		// abortまでキャンセルされ、アップロード済みのパートがバケットに残って課金される。
		// 代わりにUploadが切り離したcontextで自らabortする。
		uploader: manager.NewUploader(client, func(u *manager.Uploader) { //nolint:staticcheck // SA1019: see the uploader field comment
			u.LeavePartsOnError = true
		}),
		bucket: cfg.BucketName,
	}, nil
}

// Getはオブジェクト本体のストリームを返す。
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

// ExistsはHEADでkeyのオブジェクトがあるかを確かめる。HEADの応答は本文を持たず、オブジェクトの
// 不在とバケットの不在をエラーコードで見分けられないため、どちらの404もオブジェクトの不在として扱う。
// バケットの設定の誤りは、取得や書き込みのエラーとして別に現れる。
func (s *S3ObjectStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) || isNoSuchKey(err) {
		return false, nil
	}
	return false, fmt.Errorf("storage: オブジェクトの存在の確認に失敗しました (key: %s): %w", key, err)
}

// Uploadはbody全体をメモリに保持せず、一定サイズのパートをバッファしながらバケットへ
// ストリーミングする。アップローダーはbodyがパートサイズを超えた時点でmultipartの形式へ
// 切り替えるため、単一のPUTでは送れないサイズのZIPも呼び出し側は何もせずに送れる。
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
			// どちらの失敗も残す。abortのエラーはパートが残っている理由を示し、uploadの
			// エラーはそもそもabortが必要になった理由を示すためである。
			err = errors.Join(err, abortErr)
		}
		return fmt.Errorf("storage: オブジェクトのアップロードに失敗しました (key: %s): %w", input.Key, err)
	}
	return nil
}

// abortMultipartUploadは、失敗したUploadがバケットに残したパートを破棄する。アップロード
// がmultipartの形式に至っていない場合は何もしない。アップロードから切り離したcontextで走る
// ため、そのcontextのキャンセルが原因で失敗した場合でも実行される。
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
		return fmt.Errorf("storage: multipart uploadの中断に失敗しました (key: %s): %w", key, err)
	}
	return nil
}

// Deleteはkeyの位置のオブジェクトを削除する。
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

// deleteObjectsBatchSizeは、1回のDeleteObjectsで消すキーの上限。S3のDeleteObjectsが1回に受け付ける
// キーの数の上限 (1000) に合わせる
const deleteObjectsBatchSize = 1000

// DeleteManyはkeysをdeleteObjectsBatchSizeずつのDeleteObjectsで削除する。DeleteObjectsは存在しない
// キーを成功として扱う。応答に含まれたキーごとの失敗は、最初の1件をエラーにして返す
func (s *S3ObjectStorage) DeleteMany(ctx context.Context, keys []string) error {
	for start := 0; start < len(keys); start += deleteObjectsBatchSize {
		batch := keys[start:min(start+deleteObjectsBatchSize, len(keys))]
		objects := make([]types.ObjectIdentifier, len(batch))
		for i, key := range batch {
			objects[i] = types.ObjectIdentifier{Key: aws.String(key)}
		}

		out, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("storage: オブジェクトの一括削除に失敗しました (件数: %d): %w", len(batch), err)
		}
		for _, e := range out.Errors {
			if aws.ToString(e.Code) == "NoSuchKey" {
				continue
			}
			return fmt.Errorf("storage: オブジェクトの一括削除に失敗しました (key: %s, code: %s): %s",
				aws.ToString(e.Key), aws.ToString(e.Code), aws.ToString(e.Message))
		}
	}
	return nil
}

// PresignedGetURLはkeyのオブジェクトに対するGETに署名する。
func (s *S3ObjectStorage) PresignedGetURL(ctx context.Context, key string, expiresIn time.Duration, opts PresignedGetOptions) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}
	if opts.ResponseContentDisposition != "" {
		input.ResponseContentDisposition = aws.String(opts.ResponseContentDisposition)
	}
	if opts.ResponseContentType != "" {
		input.ResponseContentType = aws.String(opts.ResponseContentType)
	}

	req, err := s.presignClient.PresignGetObject(ctx, input, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("storage: presigned URLの生成に失敗しました (key: %s): %w", key, err)
	}
	return req.URL, nil
}

// PresignedPutURLはkeyへのPUTに、種類・大きさ・チェックサムを含めて署名する。
func (s *S3ObjectStorage) PresignedPutURL(ctx context.Context, key string, expiresIn time.Duration, opts PresignedPutOptions) (string, error) {
	req, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(opts.ContentType),
		ContentLength: aws.Int64(opts.ContentLength),
		ContentMD5:    aws.String(opts.ContentMD5),
	}, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("storage: アップロードのpresigned URLの生成に失敗しました (key: %s): %w", key, err)
	}
	return req.URL, nil
}

// isNoSuchKeyはerrがオブジェクトの不在を示しているかどうかを返す。それを意味するコードは
// NoSuchKeyだけである。NoSuchBucketを伴う404はバケット自体が存在しないか到達できないことを
// 示しており、オブジェクトの不在として扱うのではなく報告すべき本物の失敗である。
func isNoSuchKey(err error) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey"
}
