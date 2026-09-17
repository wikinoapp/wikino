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

// FakeObjectStorageはエクスポートのテストが使うメモリ上のObjectStorage。テストは
// エクスポートが読む添付ファイルを用意し、書き出されたアーカイブを読み返し、任意の1操作を
// 失敗させられる。
//
// エラーのフィールドはテスト対象を動かす前に設定する。各呼び出しで読むだけで、呼び出しの最中に
// 変更することは想定していない。
type FakeObjectStorage struct {
	// GetErr / UploadErr / DeleteErr / PresignErrは、対応する呼び出しを保持しているオブジェクト
	// に触れずにそのエラーで失敗させる。
	GetErr     error
	UploadErr  error
	DeleteErr  error
	PresignErr error

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

// PresignedGetURLはキーと有効期間から組み立てたURLを返す。リダイレクトがどのオブジェクトを
// 指し、どれだけの間有効なのかをテストが判別できるようにするためである。
func (f *FakeObjectStorage) PresignedGetURL(_ context.Context, key string, expiresIn time.Duration) (string, error) {
	if f.PresignErr != nil {
		return "", f.PresignErr
	}

	presigned := url.URL{
		Scheme:   "https",
		Host:     fakePresignedURLHost,
		Path:     "/" + key,
		RawQuery: url.Values{"expires_in": []string{strconv.FormatInt(int64(expiresIn.Seconds()), 10)}}.Encode(),
	}
	return presigned.String(), nil
}
