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

	// Existsはkeyのオブジェクトがバケットにあるかを返す。オブジェクトの不在はエラーにせずfalseを返し、
	// リクエストの失敗はエラーにする。ブラウザが署名付きURLへ直接アップロードしたオブジェクトが
	// 届いたかを、アップロード後の処理が確かめるのに使う。
	Exists(ctx context.Context, key string) (bool, error)

	// Uploadはinput.Bodyを読み、input.Keyの位置へ書き込む。bodyはストリームでよく、seekに
	// 対応している必要はない。
	Upload(ctx context.Context, input UploadInput) error

	// Deleteはkeyの位置のオブジェクトを削除する。すでに存在しないキーは成功として扱い、
	// リトライされた後片付けが前の試行で終わった仕事に対して失敗しないようにする。
	Delete(ctx context.Context, key string) error

	// PresignedGetURLは、それ自体は資格情報を持たずにkeyのオブジェクトをダウンロードでき、
	// expiresInの経過後は使えなくなるURLを返す。このURLを渡すことで、ブラウザはアーカイブを
	// アプリ経由でストリーミングせずストレージから直接取得できる。optsで指定した応答ヘッダーは
	// 署名に含まれ、ストレージはオブジェクトに保存した値の代わりにそれを返す。
	PresignedGetURL(ctx context.Context, key string, expiresIn time.Duration, opts PresignedGetOptions) (string, error)

	// PresignedPutURLは、それ自体は資格情報を持たずにkeyの位置へ1回のPUTでオブジェクトを書き込め、
	// expiresInの経過後は使えなくなるURLを返す。ブラウザがアプリを経由せずストレージへ直接アップロード
	// するのに使う。optsの種類・大きさ・チェックサムは署名に含まれ、ストレージはそれと違うリクエストを拒否する。
	PresignedPutURL(ctx context.Context, key string, expiresIn time.Duration, opts PresignedPutOptions) (string, error)
}

// PresignedPutOptionsは、署名付きURLでのアップロードに課す条件。どれも署名に含め、アップロードする
// リクエストのヘッダーがこれと一致しないとストレージは書き込みを拒否する。申告と違う内容を
// 書き込ませないためである。
type PresignedPutOptions struct {
	// ContentTypeはアップロードのContent-Type
	ContentType string
	// ContentLengthはアップロードの大きさ (バイト数)
	ContentLength int64
	// ContentMD5は内容のMD5をBase64にした値 (Content-MD5)
	ContentMD5 string
}

// PresignedGetOptionsは、署名付きURLでの取得に対してストレージが返す応答ヘッダーの指定。空の項目は
// 指定せず、ストレージはオブジェクトに保存した値を返す。添付ファイルの表示は、ブラウザが解釈して
// 実行しうる種類をそのまま返さないよう、ここで種類と表示のしかたを上書きする。
type PresignedGetOptions struct {
	// ResponseContentDispositionは応答のContent-Disposition
	ResponseContentDisposition string
	// ResponseContentTypeは応答のContent-Type
	ResponseContentType string
}

// UploadInputは書き込む1つのオブジェクト。ContentTypeは後でオブジェクトを取得したときに
// ストレージが返す値なので、キーから推測せずアップロード時に指定する。
type UploadInput struct {
	Key         string
	Body        io.Reader
	ContentType string
}
