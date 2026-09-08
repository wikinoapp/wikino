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

// fakePresignedURLHost is the host FakeObjectStorage builds its presigned URLs on. It resolves
// nowhere, so a test that follows the URL by mistake fails instead of reaching a real address.
//
// [Ja] fakePresignedURLHost は FakeObjectStorage が presigned URL を組み立てるホスト。名前解決
// できないため、誤って URL を辿ったテストは実在するアドレスに到達せず失敗する。
const fakePresignedURLHost = "storage.example.com"

// FakeObjectStorage is the in-memory ObjectStorage the tests of the export use. It lets a test
// seed the attachments an export reads, read back the archive it wrote, and make any single
// operation fail.
//
// Set the error fields before the code under test runs; they are read on every call and are not
// meant to change while one is in flight.
//
// [Ja] FakeObjectStorage はエクスポートのテストが使うメモリ上の ObjectStorage。テストは
// エクスポートが読む添付ファイルを用意し、書き出されたアーカイブを読み返し、任意の 1 操作を
// 失敗させられる。
//
// エラーのフィールドはテスト対象を動かす前に設定する。各呼び出しで読むだけで、呼び出しの最中に
// 変更することは想定していない。
type FakeObjectStorage struct {
	// GetErr, UploadErr, DeleteErr and PresignErr make the matching call fail with that error
	// without touching the stored objects.
	//
	// [Ja] GetErr / UploadErr / DeleteErr / PresignErr は、対応する呼び出しを保持しているオブジェクト
	// に触れずにそのエラーで失敗させる。
	GetErr     error
	UploadErr  error
	DeleteErr  error
	PresignErr error

	mu      sync.Mutex
	objects map[string]fakeObject
}

// fakeObject is one stored object.
//
// [Ja] fakeObject は保持している 1 つのオブジェクト。
type fakeObject struct {
	body        []byte
	contentType string
}

// NewFakeObjectStorage returns an empty FakeObjectStorage.
//
// [Ja] NewFakeObjectStorage は空の FakeObjectStorage を返す。
func NewFakeObjectStorage() *FakeObjectStorage {
	return &FakeObjectStorage{objects: map[string]fakeObject{}}
}

// Put stores an object without going through Upload, so that a test can set up what the code
// under test is expected to read.
//
// [Ja] Put は Upload を経由せずオブジェクトを保持する。テスト対象が読むはずのものをテストが
// 用意できるようにするためである。
func (f *FakeObjectStorage) Put(key string, body []byte, contentType string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = fakeObject{body: bytes.Clone(body), contentType: contentType}
}

// Object returns the stored body and content type of key.
//
// [Ja] Object は key に対して保持している body と content type を返す。
func (f *FakeObjectStorage) Object(key string) (body []byte, contentType string, ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	obj, ok := f.objects[key]
	if !ok {
		return nil, "", false
	}
	return bytes.Clone(obj.body), obj.contentType, true
}

// Keys returns every stored key in sorted order, so that an assertion over the whole bucket does
// not depend on the order a map hands its entries out in.
//
// [Ja] Keys は保持しているすべてのキーをソートして返す。バケット全体に対する検証が、map が
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

// Get returns the stored object body.
//
// [Ja] Get は保持しているオブジェクト本体を返す。
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

// Upload reads the body to the end and stores it. Reading it fully is what the real upload does,
// so a body that fails partway through is a failure here as well.
//
// [Ja] Upload は body を最後まで読んで保持する。実際のアップロードも最後まで読むため、途中で
// 失敗する body はここでも失敗になる。
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

// Delete removes the stored object, treating a key that is not there as success.
//
// [Ja] Delete は保持しているオブジェクトを削除する。存在しないキーは成功として扱う。
func (f *FakeObjectStorage) Delete(_ context.Context, key string) error {
	if f.DeleteErr != nil {
		return f.DeleteErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	return nil
}

// PresignedGetURL returns a URL built from the key and the lifetime, so that a test can tell which
// object a redirect points at and how long it stays valid.
//
// [Ja] PresignedGetURL はキーと有効期間から組み立てた URL を返す。リダイレクトがどのオブジェクトを
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
