// Package storageは、Wikinoがバイナリオブジェクトを置くS3互換オブジェクトストレージ
// (Cloudflare R2) へのInfrastructure層アダプタを提供する。オブジェクトはスペースにアップロード
// された添付ファイルと、エクスポート機能が生成するZIPアーカイブである。
//
// バケットはActiveStorage経由で読み書きするRails版と共有している。本パッケージはキーを指定して
// オブジェクトを読み書きするだけなので、ActiveStorageのレコードの規約には踏み込まず、オブジェクト
// そのもの以外には触れない。
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrObjectNotFoundは、キーに対応するオブジェクトがバケットに無いときにGetが返すエラー。
// 呼び出し側はこれでオブジェクトの不在とリクエストの失敗を区別できる。エクスポートにはこの区別が
// 要る。オブジェクトが失われた添付ファイルは報告すべき壊れたレコードだが、失敗したリクエストは
// リトライする価値があるためである。
var ErrObjectNotFound = errors.New("storage: オブジェクトが存在しません")

// ObjectStorageはアプリケーションの他の部分から見たオブジェクトストレージ。実際のバケットに
// 対する実装がS3ObjectStorageで、FakeObjectStorageはオブジェクトをメモリに保持し、テストが
// ネットワークへ出ずにエクスポート全体を動かせるようにする。
type ObjectStorage interface {
	// Getはオブジェクト本体のストリームを返す。閉じるのは呼び出し側。バケットが持っていない
	// キーにはErrObjectNotFoundを返す。
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Uploadはinput.Bodyを読み、input.Keyの位置へ書き込む。bodyはストリームでよく、seekに
	// 対応している必要はない。
	Upload(ctx context.Context, input UploadInput) error

	// Deleteはkeyの位置のオブジェクトを削除する。すでに存在しないキーは成功として扱い、
	// リトライされた後片付けが前の試行で終わった仕事に対して失敗しないようにする。
	Delete(ctx context.Context, key string) error

	// PresignedGetURLは、それ自体は資格情報を持たずにkeyのオブジェクトをダウンロードでき、
	// expiresInの経過後は使えなくなるURLを返す。このURLを渡すことで、ブラウザはアーカイブを
	// アプリ経由でストリーミングせずストレージから直接取得できる。
	PresignedGetURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}

// UploadInputは書き込む1つのオブジェクト。ContentTypeは後でオブジェクトを取得したときに
// ストレージが返す値なので、キーから推測せずアップロード時に指定する。
type UploadInput struct {
	Key         string
	Body        io.Reader
	ContentType string
}
