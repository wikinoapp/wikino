package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"
)

// fakePresignedURLHostはFakeObjectStorageがpresigned URLを組み立てるホスト。名前解決
// できないため、誤ってURLを辿ったテストは実在するアドレスに到達せず失敗する。
const fakePresignedURLHost = "storage.example.com"

// FakeObjectStorageは、テストとe2eのサーバーが使うメモリ上のObjectStorage。テストは、テスト対象が
// 読むオブジェクトを用意し、書き出されたものを読み返し、任意の1操作を失敗させられる。
// e2eのサーバー (APP_ENV=test_e2e) は、NewServedFakeObjectStorageで作ったものをオブジェクトストレージの
// 代わりに使い、ブラウザからの署名付きURLへの送受信をServeHTTPで受ける。
//
// エラーのフィールドはテスト対象を動かす前に設定する。各呼び出しで読むだけで、呼び出しの最中に
// 変更することは想定していない。
type FakeObjectStorage struct {
	// GetErr / ExistsErr / UploadErr / DeleteErr / PresignErrは、対応する呼び出しを保持している
	// オブジェクトに触れずにそのエラーで失敗させる。PresignErrは取得とアップロードの両方の署名を失敗させる。
	GetErr     error
	ExistsErr  error
	UploadErr  error
	DeleteErr  error
	PresignErr error

	// servedはtrueのとき、署名付きURLをFakeObjectStoragePathPrefixの下の相対パスにする。
	// e2eのサーバーがServeHTTPで受けるためである
	served bool

	mu      sync.Mutex
	objects map[string]fakeObject
}

// fakeObjectは保持している1つのオブジェクト。
type fakeObject struct {
	body        []byte
	contentType string
}

// NewFakeObjectStorageは空のFakeObjectStorageを返す。
func NewFakeObjectStorage() *FakeObjectStorage {
	return &FakeObjectStorage{objects: map[string]fakeObject{}}
}

// NewServedFakeObjectStorageは、署名付きURLがFakeObjectStoragePathPrefixの下を指す空のFakeObjectStorageを返す。
// e2eのサーバーがオブジェクトストレージの代わりに使い、ブラウザの直接アップロードと取得をServeHTTPで受ける
func NewServedFakeObjectStorage() *FakeObjectStorage {
	f := NewFakeObjectStorage()
	f.served = true
	return f
}

// presignedURLは、keyのオブジェクトを指し、queryを付けた署名付きURLを組み立てる
func (f *FakeObjectStorage) presignedURL(key string, query url.Values) string {
	if f.served {
		return (&url.URL{Path: FakeObjectStoragePathPrefix + key, RawQuery: query.Encode()}).String()
	}
	return (&url.URL{Scheme: "https", Host: fakePresignedURLHost, Path: "/" + key, RawQuery: query.Encode()}).String()
}

// PutはUploadを経由せずオブジェクトを保持する。テスト対象が読むはずのものをテストが
// 用意できるようにするためである。
func (f *FakeObjectStorage) Put(key string, body []byte, contentType string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = fakeObject{body: bytes.Clone(body), contentType: contentType}
}

// Objectはkeyに対して保持しているbodyとcontent typeを返す。
func (f *FakeObjectStorage) Object(key string) (body []byte, contentType string, ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	obj, ok := f.objects[key]
	if !ok {
		return nil, "", false
	}
	return bytes.Clone(obj.body), obj.contentType, true
}

// Keysは保持しているすべてのキーをソートして返す。バケット全体に対する検証が、mapが
// エントリを返す順序に依存しないようにするためである。
func (f *FakeObjectStorage) Keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	keys := make([]string, 0, len(f.objects))
	for key := range f.objects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Getは保持しているオブジェクト本体を返す。
func (f *FakeObjectStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	if f.GetErr != nil {
		return nil, f.GetErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	obj, ok := f.objects[key]
	if !ok {
		return nil, fmt.Errorf("%w (key: %s)", ErrObjectNotFound, key)
	}
	return io.NopCloser(bytes.NewReader(obj.body)), nil
}

// Existsはkeyのオブジェクトを保持しているかを返す。
func (f *FakeObjectStorage) Exists(_ context.Context, key string) (bool, error) {
	if f.ExistsErr != nil {
		return false, f.ExistsErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[key]
	return ok, nil
}

// Uploadはbodyを最後まで読んで保持する。実際のアップロードも最後まで読むため、途中で
// 失敗するbodyはここでも失敗になる。
func (f *FakeObjectStorage) Upload(_ context.Context, input UploadInput) error {
	if f.UploadErr != nil {
		return f.UploadErr
	}

	body, err := io.ReadAll(input.Body)
	if err != nil {
		return fmt.Errorf("storage: オブジェクトのアップロードに失敗しました (key: %s): %w", input.Key, err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[input.Key] = fakeObject{body: body, contentType: input.ContentType}
	return nil
}

// Deleteは保持しているオブジェクトを削除する。存在しないキーは成功として扱う。
func (f *FakeObjectStorage) Delete(_ context.Context, key string) error {
	if f.DeleteErr != nil {
		return f.DeleteErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	return nil
}

// PresignedGetURLはキーと有効期間と応答ヘッダーの指定から組み立てたURLを返す。リダイレクトが
// どのオブジェクトを指し、どれだけの間有効で、どの応答ヘッダーを返させるのかをテストが判別できる
// ようにするためである。応答ヘッダーのクエリ名はS3の署名付きURLに揃える。
func (f *FakeObjectStorage) PresignedGetURL(_ context.Context, key string, expiresIn time.Duration, opts PresignedGetOptions) (string, error) {
	if f.PresignErr != nil {
		return "", f.PresignErr
	}

	query := url.Values{"expires_in": []string{strconv.FormatInt(int64(expiresIn.Seconds()), 10)}}
	if opts.ResponseContentDisposition != "" {
		query.Set("response-content-disposition", opts.ResponseContentDisposition)
	}
	if opts.ResponseContentType != "" {
		query.Set("response-content-type", opts.ResponseContentType)
	}
	return f.presignedURL(key, query), nil
}

// PresignedPutURLはキーと有効期間とアップロードの条件から組み立てたURLを返す。アップロードの署名が
// どのキーを指し、どれだけの間有効で、どの種類・大きさ・チェックサムを課すのかをテストが判別できる
// ようにするためである。
func (f *FakeObjectStorage) PresignedPutURL(_ context.Context, key string, expiresIn time.Duration, opts PresignedPutOptions) (string, error) {
	if f.PresignErr != nil {
		return "", f.PresignErr
	}

	query := url.Values{
		"expires_in":     []string{strconv.FormatInt(int64(expiresIn.Seconds()), 10)},
		"content_type":   []string{opts.ContentType},
		"content_length": []string{strconv.FormatInt(opts.ContentLength, 10)},
		"content_md5":    []string{opts.ContentMD5},
	}
	return f.presignedURL(key, query), nil
}
