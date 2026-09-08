package storage_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/storage"
)

// S3ObjectStorage and FakeObjectStorage must both satisfy the interface the rest of the
// application depends on, so that the fake can stand in for the real bucket everywhere.
//
// [Ja] S3ObjectStorage と FakeObjectStorage は、アプリケーションの他の部分が依存する interface を
// どちらも満たさなければならない。フェイクがどこでも実際のバケットの代わりになるようにするためである。
var (
	_ storage.ObjectStorage = (*storage.S3ObjectStorage)(nil)
	_ storage.ObjectStorage = (*storage.FakeObjectStorage)(nil)
)

const testBucket = "test-bucket"

type abortRequest struct {
	key             string
	uploadID        string
	contextCanceled bool
}

// fakeS3 answers the requests the adapter issues (PutObject, the multipart calls, GetObject and
// DeleteObject) in path-style form, and records enough of them for a test to tell which path an
// upload took.
//
// [Ja] fakeS3 はアダプタが発行するリクエスト (PutObject、multipart、GetObject、DeleteObject) に
// パス形式で応答し、アップロードがどの経路を通ったかをテストが判別できる程度に記録する。
type fakeS3 struct {
	mu           sync.Mutex
	objects      map[string][]byte
	contentTypes map[string]string
	putHeaders   http.Header
	getHeaders   http.Header

	// parts holds the uploaded parts per upload ID, and uploadIDs lists the IDs handed out in the
	// order they were created. Scoping the parts by upload ID keeps a second multipart attempt
	// against the same fake from completing with the parts the first one left behind.
	//
	// [Ja] parts は upload ID ごとにアップロード済みのパートを保持し、uploadIDs は払い出した ID を
	// 作成順に並べる。パートを upload ID で区切ることで、同じフェイクに対する 2 回目の multipart が
	// 1 回目の残したパートで complete してしまうことを防ぐ。
	parts     map[string]map[int][]byte
	uploadIDs []string

	deletedKeys   []string
	abortRequests []abortRequest
	failAbort     bool

	// getErrorCode and deleteErrorCode make GetObject and DeleteObject answer with a 404 carrying
	// that S3 error code. "NoSuchKey" is what a storage returns for an object that is not there;
	// any other code stands for a failure the adapter must report rather than treat as absence.
	//
	// [Ja] getErrorCode / deleteErrorCode は、その S3 エラーコードを載せた 404 を GetObject /
	// DeleteObject に返させる。"NoSuchKey" は存在しないオブジェクトに対してストレージが返すもので、
	// それ以外のコードは、不在として扱うのではなくアダプタが報告すべき失敗を表す。
	getErrorCode    string
	deleteErrorCode string
}

func newFakeS3() *fakeS3 {
	return &fakeS3{
		objects:      map[string][]byte{},
		contentTypes: map[string]string{},
		parts:        map[string]map[int][]byte{},
	}
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/"+testBucket+"/")
	query := r.URL.Query()

	switch {
	case r.Method == http.MethodPost && query.Has("uploads"):
		f.createMultipartUpload(w, key)
	case r.Method == http.MethodPut && query.Get("partNumber") != "":
		f.uploadPart(w, r, query.Get("uploadId"), query.Get("partNumber"))
	case r.Method == http.MethodPost && query.Get("uploadId") != "":
		f.completeMultipartUpload(w, key, query.Get("uploadId"))
	case r.Method == http.MethodPut:
		f.putObject(w, r, key)
	case r.Method == http.MethodGet:
		f.getObject(w, r, key)
	case r.Method == http.MethodDelete && query.Get("uploadId") != "":
		f.abortMultipartUpload(w, r, key, query.Get("uploadId"))
	case r.Method == http.MethodDelete:
		f.deleteObject(w, key)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func (f *fakeS3) createMultipartUpload(w http.ResponseWriter, key string) {
	f.mu.Lock()
	uploadID := fmt.Sprintf("test-upload-id-%d", len(f.uploadIDs)+1)
	f.uploadIDs = append(f.uploadIDs, uploadID)
	f.parts[uploadID] = map[int][]byte{}
	f.mu.Unlock()

	writeXML(w, http.StatusOK, fmt.Sprintf(
		`<InitiateMultipartUploadResult><Bucket>%s</Bucket><Key>%s</Key><UploadId>%s</UploadId></InitiateMultipartUploadResult>`,
		testBucket, key, uploadID,
	))
}

func (f *fakeS3) uploadPart(w http.ResponseWriter, r *http.Request, uploadID, partNumber string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	number, err := strconv.Atoi(partNumber)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	if f.parts[uploadID] == nil {
		f.parts[uploadID] = map[int][]byte{}
	}
	f.parts[uploadID][number] = body
	f.mu.Unlock()

	w.Header().Set("ETag", strconv.Quote("part-"+partNumber))
	w.WriteHeader(http.StatusOK)
}

func (f *fakeS3) completeMultipartUpload(w http.ResponseWriter, key, uploadID string) {
	f.mu.Lock()
	parts := f.parts[uploadID]
	numbers := make([]int, 0, len(parts))
	for number := range parts {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	var body []byte
	for _, number := range numbers {
		body = append(body, parts[number]...)
	}
	f.objects[key] = body
	f.mu.Unlock()

	writeXML(w, http.StatusOK, fmt.Sprintf(
		`<CompleteMultipartUploadResult><Bucket>%s</Bucket><Key>%s</Key><ETag>%s</ETag></CompleteMultipartUploadResult>`,
		testBucket, key, strconv.Quote("complete"),
	))
}

func (f *fakeS3) putObject(w http.ResponseWriter, r *http.Request, key string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	f.mu.Lock()
	f.objects[key] = body
	f.contentTypes[key] = r.Header.Get("Content-Type")
	f.putHeaders = r.Header.Clone()
	f.mu.Unlock()

	w.Header().Set("ETag", strconv.Quote("put"))
	w.WriteHeader(http.StatusOK)
}

func (f *fakeS3) getObject(w http.ResponseWriter, r *http.Request, key string) {
	f.mu.Lock()
	body, ok := f.objects[key]
	contentType := f.contentTypes[key]
	errorCode := f.getErrorCode
	f.getHeaders = r.Header.Clone()
	f.mu.Unlock()

	if errorCode != "" {
		writeS3Error(w, errorCode)
		return
	}
	if !ok {
		writeS3Error(w, "NoSuchKey")
		return
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (f *fakeS3) abortMultipartUpload(w http.ResponseWriter, r *http.Request, key, uploadID string) {
	f.mu.Lock()
	f.abortRequests = append(f.abortRequests, abortRequest{
		key:             key,
		uploadID:        uploadID,
		contextCanceled: r.Context().Err() != nil,
	})
	failAbort := f.failAbort
	if !failAbort {
		// A denied abort leaves the parts in the bucket, which is the state the returned error
		// reports. Only discard them when the abort is answered as successful.
		//
		// [Ja] 拒否された abort はパートをバケットに残す。返されるエラーが報告しているのはこの
		// 状態である。破棄するのは abort が成功として応答されるときだけにする。
		delete(f.parts, uploadID)
	}
	f.mu.Unlock()

	if failAbort {
		writeXML(
			w,
			http.StatusForbidden,
			`<Error><Code>AbortDenied</Code><Message>abort failed</Message></Error>`,
		)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeS3) deleteObject(w http.ResponseWriter, key string) {
	f.mu.Lock()
	errorCode := f.deleteErrorCode
	f.deletedKeys = append(f.deletedKeys, key)
	if errorCode == "" {
		delete(f.objects, key)
	}
	f.mu.Unlock()

	if errorCode != "" {
		writeS3Error(w, errorCode)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeS3) object(key string) ([]byte, string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, ok := f.objects[key]
	return body, f.contentTypes[key], ok
}

func assertMultipartAbort(t *testing.T, fake *fakeS3, key string) {
	t.Helper()

	fake.mu.Lock()
	uploadIDs := append([]string(nil), fake.uploadIDs...)
	abortRequests := append([]abortRequest(nil), fake.abortRequests...)
	failAbort := fake.failAbort
	remainingUploads := len(fake.parts)
	fake.mu.Unlock()

	if len(uploadIDs) != 1 {
		t.Fatalf("CreateMultipartUpload の呼び出し回数 = %d, want 1", len(uploadIDs))
	}
	if len(abortRequests) != 1 {
		t.Fatalf("AbortMultipartUpload の呼び出し回数 = %d, want 1", len(abortRequests))
	}
	request := abortRequests[0]
	if request.key != key {
		t.Errorf("abort の key = %q, want %q", request.key, key)
	}
	if request.uploadID != uploadIDs[0] {
		t.Errorf("abort の upload ID = %q, want %q", request.uploadID, uploadIDs[0])
	}
	if request.contextCanceled {
		t.Error("abort の request context がキャンセルされている")
	}

	// A successful abort discards the parts, while a denied one leaves them where they are.
	//
	// [Ja] 成功した abort はパートを破棄し、拒否された abort はそのまま残す。
	wantUploads := 0
	if failAbort {
		wantUploads = 1
	}
	if remainingUploads != wantUploads {
		t.Errorf("abort 後に残っている upload = %d, want %d", remainingUploads, wantUploads)
	}
}

func writeXML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + body))
}

// writeS3Error answers with a 404 carrying code. Both codes the adapter has to tell apart
// (NoSuchKey and NoSuchBucket) come back as a 404, so the status alone cannot decide the outcome.
//
// [Ja] writeS3Error は code を載せた 404 を返す。アダプタが区別しなければならない 2 つのコード
// (NoSuchKey と NoSuchBucket) はどちらも 404 で返るため、ステータスだけでは結果を決められない。
func writeS3Error(w http.ResponseWriter, code string) {
	writeXML(w, http.StatusNotFound, fmt.Sprintf(`<Error><Code>%s</Code><Message>%s</Message></Error>`, code, code))
}

// newTestStorage starts the fake storage and returns an adapter pointed at it.
//
// [Ja] newTestStorage はフェイクのストレージを起動し、それを向いたアダプタを返す。
func newTestStorage(t *testing.T) (*storage.S3ObjectStorage, *fakeS3) {
	t.Helper()
	return newTestStorageWithRegion(t, "")
}

func newTestStorageWithRegion(t *testing.T, region string) (*storage.S3ObjectStorage, *fakeS3) {
	t.Helper()

	fake := newFakeS3()
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	s, err := storage.NewS3ObjectStorage(storage.Config{
		BucketName:      testBucket,
		Endpoint:        server.URL,
		AccessKeyID:     "test-access-key-id",
		SecretAccessKey: "test-secret-access-key",
		Region:          region,
	})
	if err != nil {
		t.Fatalf("NewS3ObjectStorage() error = %v", err)
	}
	return s, fake
}

func TestNewS3ObjectStorage_IncompleteConfig(t *testing.T) {
	t.Parallel()

	complete := storage.Config{
		BucketName:      testBucket,
		Endpoint:        "https://storage.example.com",
		AccessKeyID:     "test-access-key-id",
		SecretAccessKey: "test-secret-access-key",
		Region:          "apac",
	}

	tests := []struct {
		name    string
		cfg     storage.Config
		wantErr bool
	}{
		{
			name:    "正常系: 必須項目がすべて揃っている",
			cfg:     complete,
			wantErr: false,
		},
		{
			name: "正常系: リージョンだけが空",
			cfg: func() storage.Config {
				cfg := complete
				cfg.Region = ""
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "異常系: バケット名が空",
			cfg: func() storage.Config {
				cfg := complete
				cfg.BucketName = ""
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "異常系: エンドポイントが空",
			cfg: func() storage.Config {
				cfg := complete
				cfg.Endpoint = ""
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "異常系: アクセスキー ID が空",
			cfg: func() storage.Config {
				cfg := complete
				cfg.AccessKeyID = ""
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "異常系: シークレットアクセスキーが空",
			cfg: func() storage.Config {
				cfg := complete
				cfg.SecretAccessKey = ""
				return cfg
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := storage.NewS3ObjectStorage(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NewS3ObjectStorage() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewS3ObjectStorage() error = %v", err)
			}
			if s == nil {
				t.Fatal("NewS3ObjectStorage() = nil, want non-nil")
			}
		})
	}
}

func TestS3ObjectStorage_UploadAndGet(t *testing.T) {
	t.Parallel()

	s, fake := newTestStorage(t)
	ctx := context.Background()
	const key = "exports/wikino.zip"
	body := []byte("wikino export archive")

	if err := s.Upload(ctx, storage.UploadInput{
		Key:         key,
		Body:        bytes.NewReader(body),
		ContentType: "application/zip",
	}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	stored, contentType, ok := fake.object(key)
	if !ok {
		t.Fatalf("オブジェクトが保存されていない (key: %s)", key)
	}
	if !bytes.Equal(stored, body) {
		t.Errorf("保存された本体 = %q, want %q", stored, body)
	}
	if contentType != "application/zip" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/zip")
	}

	reader, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer func() { _ = reader.Close() }()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("取得したオブジェクトの読み取りに失敗: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("Get() = %q, want %q", got, body)
	}

	fake.mu.Lock()
	putHeaders := fake.putHeaders.Clone()
	getHeaders := fake.getHeaders.Clone()
	fake.mu.Unlock()

	for name := range putHeaders {
		if strings.HasPrefix(strings.ToLower(name), "x-amz-checksum-") ||
			strings.EqualFold(name, "X-Amz-Sdk-Checksum-Algorithm") {
			t.Errorf("PutObject の任意チェックサムヘッダー %q が設定されている", name)
		}
	}
	if got := getHeaders.Get("X-Amz-Checksum-Mode"); got != "" {
		t.Errorf("GetObject の X-Amz-Checksum-Mode = %q, want empty", got)
	}
}

// TestS3ObjectStorage_UploadMultipart checks that a body larger than the uploader's single-part
// threshold is split into parts and reassembled.
//
// [Ja] TestS3ObjectStorage_UploadMultipart は、アップローダーが単一 part で扱う閾値を超える body が
// part に分割されて組み立て直されることを確認する。
func TestS3ObjectStorage_UploadMultipart(t *testing.T) {
	t.Parallel()

	s, fake := newTestStorage(t)
	ctx := context.Background()
	const key = "exports/large.zip"

	// The uploader keeps a floor of 5 MiB on the part size, so the body has to pass it to take the
	// multipart path.
	//
	// [Ja] アップローダーはパートサイズに 5 MiB の下限を持つため、multipart の経路を通すには
	// body がそれを超えている必要がある。
	body := bytes.Repeat([]byte("wikino-export-"), 6*1024*1024/14+1)

	if err := s.Upload(ctx, storage.UploadInput{
		Key:         key,
		Body:        bytes.NewReader(body),
		ContentType: "application/zip",
	}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	fake.mu.Lock()
	uploadCount := len(fake.uploadIDs)
	partCount := 0
	if uploadCount > 0 {
		partCount = len(fake.parts[fake.uploadIDs[0]])
	}
	fake.mu.Unlock()

	if uploadCount != 1 {
		t.Errorf("CreateMultipartUpload の呼び出し回数 = %d, want 1", uploadCount)
	}
	if partCount < 2 {
		t.Errorf("パート数 = %d, want >= 2", partCount)
	}

	stored, _, ok := fake.object(key)
	if !ok {
		t.Fatalf("オブジェクトが保存されていない (key: %s)", key)
	}
	if !bytes.Equal(stored, body) {
		t.Errorf("組み立て直された本体の長さ = %d, want %d", len(stored), len(body))
	}
}

func TestS3ObjectStorage_UploadMultipartFailure(t *testing.T) {
	t.Parallel()

	const (
		key           = "exports/interrupted.zip"
		firstPartSize = 5 * 1024 * 1024
	)

	t.Run("異常系: multipart に至らない失敗では abort しない", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		wantErr := errors.New("single part reader failed")
		err := s.Upload(context.Background(), storage.UploadInput{
			Key: key,
			Body: &failingReader{
				remaining: firstPartSize / 2,
				err:       wantErr,
			},
		})
		if !errors.Is(err, wantErr) {
			t.Errorf("Upload() error = %v, want %v", err, wantErr)
		}

		fake.mu.Lock()
		uploadCount := len(fake.uploadIDs)
		abortCount := len(fake.abortRequests)
		fake.mu.Unlock()

		if uploadCount != 0 {
			t.Errorf("CreateMultipartUpload の呼び出し回数 = %d, want 0", uploadCount)
		}
		if abortCount != 0 {
			t.Errorf("AbortMultipartUpload の呼び出し回数 = %d, want 0", abortCount)
		}
	})

	t.Run("異常系: reader の失敗後に同じ upload を一度 abort する", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		wantErr := errors.New("multipart reader failed")
		err := s.Upload(context.Background(), storage.UploadInput{
			Key: key,
			Body: &failingReader{
				remaining: firstPartSize,
				err:       wantErr,
			},
		})
		if !errors.Is(err, wantErr) {
			t.Errorf("Upload() error = %v, want %v", err, wantErr)
		}
		assertMultipartAbort(t, fake, key)
	})

	t.Run("異常系: upload context がキャンセルされても abort はキャンセルされない", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		err := s.Upload(ctx, storage.UploadInput{
			Key: key,
			Body: &failingReader{
				remaining: firstPartSize,
				cancel:    cancel,
				err:       context.Canceled,
			},
		})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Upload() error = %v, want context.Canceled", err)
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("upload context error = %v, want context.Canceled", ctx.Err())
		}
		assertMultipartAbort(t, fake, key)
	})

	t.Run("異常系: reader と abort の両方の失敗を返す", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		fake.mu.Lock()
		fake.failAbort = true
		fake.mu.Unlock()

		wantErr := errors.New("multipart reader failed")
		err := s.Upload(context.Background(), storage.UploadInput{
			Key: key,
			Body: &failingReader{
				remaining: firstPartSize,
				err:       wantErr,
			},
		})
		if !errors.Is(err, wantErr) {
			t.Errorf("Upload() error = %v, want %v", err, wantErr)
		}
		var abortErr interface {
			ErrorCode() string
		}
		if !errors.As(err, &abortErr) {
			t.Fatalf("Upload() error = %v, want abort API error", err)
		}
		if abortErr.ErrorCode() != "AbortDenied" {
			t.Errorf("abort error code = %q, want %q", abortErr.ErrorCode(), "AbortDenied")
		}
		assertMultipartAbort(t, fake, key)
	})
}

func TestS3ObjectStorage_Get(t *testing.T) {
	t.Parallel()

	t.Run("異常系: 存在しないキーは ErrObjectNotFound になる", func(t *testing.T) {
		t.Parallel()

		s, _ := newTestStorage(t)

		reader, err := s.Get(context.Background(), "attachments/gone.png")
		if err == nil {
			_ = reader.Close()
			t.Fatal("Get() error = nil, want ErrObjectNotFound")
		}
		if !errors.Is(err, storage.ErrObjectNotFound) {
			t.Errorf("Get() error = %v, want ErrObjectNotFound", err)
		}
	})

	// A 404 that carries NoSuchBucket says the bucket is missing or unreachable. Reporting it as
	// ErrObjectNotFound would turn a failure worth retrying into a broken attachment the export
	// records, so the adapter has to pass the original error through.
	//
	// [Ja] NoSuchBucket を伴う 404 はバケットが存在しないか到達できないことを示す。これを
	// ErrObjectNotFound として報告すると、リトライする価値のある失敗がエクスポートの記録する
	// 壊れた添付ファイルに変わってしまうため、アダプタは元のエラーをそのまま通す必要がある。
	t.Run("異常系: NoSuchKey 以外のエラーは ErrObjectNotFound にしない", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		fake.mu.Lock()
		fake.getErrorCode = "NoSuchBucket"
		fake.mu.Unlock()

		reader, err := s.Get(context.Background(), "attachments/photo.png")
		if err == nil {
			_ = reader.Close()
			t.Fatal("Get() error = nil, want error")
		}
		if errors.Is(err, storage.ErrObjectNotFound) {
			t.Errorf("Get() error = %v, want an error that is not ErrObjectNotFound", err)
		}

		var apiErr interface {
			ErrorCode() string
		}
		if !errors.As(err, &apiErr) {
			t.Fatalf("Get() error = %v, want an S3 API error", err)
		}
		if apiErr.ErrorCode() != "NoSuchBucket" {
			t.Errorf("error code = %q, want %q", apiErr.ErrorCode(), "NoSuchBucket")
		}
	})
}

func TestS3ObjectStorage_Delete(t *testing.T) {
	t.Parallel()

	t.Run("正常系: オブジェクトを削除する", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		ctx := context.Background()
		const key = "exports/old.zip"

		if err := s.Upload(ctx, storage.UploadInput{Key: key, Body: strings.NewReader("old")}); err != nil {
			t.Fatalf("Upload() error = %v", err)
		}
		if err := s.Delete(ctx, key); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		if _, _, ok := fake.object(key); ok {
			t.Errorf("オブジェクトが残っている (key: %s)", key)
		}
	})

	t.Run("正常系: 存在しないキーの削除は成功として扱う", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		fake.mu.Lock()
		fake.deleteErrorCode = "NoSuchKey"
		fake.mu.Unlock()

		if err := s.Delete(context.Background(), "exports/gone.zip"); err != nil {
			t.Errorf("Delete() error = %v, want nil", err)
		}
	})

	// Only NoSuchKey means the object is already gone. Swallowing any other 404 would report a
	// cleanup as done while the object is still there.
	//
	// [Ja] オブジェクトがすでに無いことを意味するのは NoSuchKey だけである。それ以外の 404 まで
	// 飲み込むと、オブジェクトが残っているのに後片付けが終わったことにしてしまう。
	t.Run("異常系: NoSuchKey 以外のエラーは失敗として返す", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		fake.mu.Lock()
		fake.deleteErrorCode = "NoSuchBucket"
		fake.mu.Unlock()

		err := s.Delete(context.Background(), "exports/wikino.zip")
		if err == nil {
			t.Fatal("Delete() error = nil, want error")
		}

		var apiErr interface {
			ErrorCode() string
		}
		if !errors.As(err, &apiErr) {
			t.Fatalf("Delete() error = %v, want an S3 API error", err)
		}
		if apiErr.ErrorCode() != "NoSuchBucket" {
			t.Errorf("error code = %q, want %q", apiErr.ErrorCode(), "NoSuchBucket")
		}
	})
}

func TestS3ObjectStorage_PresignedGetURL(t *testing.T) {
	t.Parallel()

	const key = "exports/wikino.zip"
	tests := []struct {
		name       string
		region     string
		wantRegion string
	}{
		{name: "正常系: リージョン未設定時は auto を使う", region: "", wantRegion: "auto"},
		{name: "正常系: 設定したリージョンを使う", region: "apac", wantRegion: "apac"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, _ := newTestStorageWithRegion(t, tt.region)
			presigned, err := s.PresignedGetURL(context.Background(), key, 24*time.Hour)
			if err != nil {
				t.Fatalf("PresignedGetURL() error = %v", err)
			}

			parsed, err := url.Parse(presigned)
			if err != nil {
				t.Fatalf("presigned URL のパースに失敗: %v", err)
			}
			if want := "/" + testBucket + "/" + key; parsed.Path != want {
				t.Errorf("presigned URL のパス = %q, want %q", parsed.Path, want)
			}
			if got := parsed.Query().Get("X-Amz-Expires"); got != strconv.Itoa(int((24 * time.Hour).Seconds())) {
				t.Errorf("X-Amz-Expires = %q, want %q", got, strconv.Itoa(int((24 * time.Hour).Seconds())))
			}
			if parsed.Query().Get("X-Amz-Signature") == "" {
				t.Error("presigned URL に署名が含まれていない")
			}

			credential := strings.Split(parsed.Query().Get("X-Amz-Credential"), "/")
			if len(credential) != 5 {
				t.Fatalf("X-Amz-Credential = %q, want 5 slash-separated fields", credential)
			}
			if got := credential[2]; got != tt.wantRegion {
				t.Errorf("署名リージョン = %q, want %q", got, tt.wantRegion)
			}
		})
	}
}
