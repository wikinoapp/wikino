// Package storage provides Infrastructure-layer adapters for the S3-compatible object storage
// (Cloudflare R2) that Wikino keeps its binary objects in: the attachments uploaded to a space,
// and the ZIP archives the export feature produces.
//
// The bucket is shared with the Rails version, which reaches it through ActiveStorage. This
// package only reads and writes objects by key, so it stays clear of the ActiveStorage record
// conventions and touches nothing but the object itself.
//
// [Ja] storage パッケージは、Wikino がバイナリオブジェクトを置く S3 互換オブジェクトストレージ
// (Cloudflare R2) への Infrastructure 層アダプタを提供する。オブジェクトはスペースにアップロード
// された添付ファイルと、エクスポート機能が生成する ZIP アーカイブである。
//
// バケットは ActiveStorage 経由で読み書きする Rails 版と共有している。本パッケージはキーを指定して
// オブジェクトを読み書きするだけなので、ActiveStorage のレコードの規約には踏み込まず、オブジェクト
// そのもの以外には触れない。
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrObjectNotFound is what Get returns when the bucket holds nothing under the key. Callers can
// tell a missing object from a failed request with it, which the export needs: an attachment whose
// object is gone is a broken record to report, while a request that failed is worth retrying.
//
// [Ja] ErrObjectNotFound は、キーに対応するオブジェクトがバケットに無いときに Get が返すエラー。
// 呼び出し側はこれでオブジェクトの不在とリクエストの失敗を区別できる。エクスポートにはこの区別が
// 要る。オブジェクトが失われた添付ファイルは報告すべき壊れたレコードだが、失敗したリクエストは
// リトライする価値があるためである。
var ErrObjectNotFound = errors.New("storage: オブジェクトが存在しません")

// ObjectStorage is the object storage seen from the rest of the application. S3ObjectStorage
// implements it against the real bucket, and FakeObjectStorage keeps the objects in memory so that
// tests can exercise a whole export without reaching the network.
//
// [Ja] ObjectStorage はアプリケーションの他の部分から見たオブジェクトストレージ。実際のバケットに
// 対する実装が S3ObjectStorage で、FakeObjectStorage はオブジェクトをメモリに保持し、テストが
// ネットワークへ出ずにエクスポート全体を動かせるようにする。
type ObjectStorage interface {
	// Get returns the object body as a stream. The caller closes it. A key the bucket does not
	// hold gives ErrObjectNotFound.
	//
	// [Ja] Get はオブジェクト本体のストリームを返す。閉じるのは呼び出し側。バケットが持っていない
	// キーには ErrObjectNotFound を返す。
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Upload reads input.Body and writes it to input.Key. The body may be a stream and need not
	// support seeking.
	//
	// [Ja] Upload は input.Body を読み、input.Key の位置へ書き込む。body はストリームでよく、seek に
	// 対応している必要はない。
	Upload(ctx context.Context, input UploadInput) error

	// Delete removes the object under key. A key that is already gone counts as success, which
	// keeps a retried cleanup from failing on the work its earlier attempt finished.
	//
	// [Ja] Delete は key の位置のオブジェクトを削除する。すでに存在しないキーは成功として扱い、
	// リトライされた後片付けが前の試行で終わった仕事に対して失敗しないようにする。
	Delete(ctx context.Context, key string) error

	// PresignedGetURL returns a URL that downloads the object under key without any credentials of
	// its own, and stops working after expiresIn. Handing the URL out lets the browser fetch the
	// archive straight from the storage instead of streaming it through the app.
	//
	// [Ja] PresignedGetURL は、それ自体は資格情報を持たずに key のオブジェクトをダウンロードでき、
	// expiresIn の経過後は使えなくなる URL を返す。この URL を渡すことで、ブラウザはアーカイブを
	// アプリ経由でストリーミングせずストレージから直接取得できる。
	PresignedGetURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}

// UploadInput is one object to write. ContentType is what the storage answers with when the object
// is fetched later, so it is set at upload time rather than guessed from the key.
//
// [Ja] UploadInput は書き込む 1 つのオブジェクト。ContentType は後でオブジェクトを取得したときに
// ストレージが返す値なので、キーから推測せずアップロード時に指定する。
type UploadInput struct {
	Key         string
	Body        io.Reader
	ContentType string
}
