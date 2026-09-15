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

// S3ObjectStorageとFakeObjectStorageは、アプリケーションの他の部分が依存するinterfaceを
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

// fakeS3はアダプタが発行するリクエスト (PutObject、multipart、GetObject、DeleteObject) に
// パス形式で応答し、アップロードがどの経路を通ったかをテストが判別できる程度に記録する。
type fakeS3 struct {
	mu           sync.Mutex
	objects      map[string][]byte
	contentTypes map[string]string
	putHeaders   http.Header
	getHeaders   http.Header

	// partsはupload IDごとにアップロード済みのパートを保持し、uploadIDsは払い出したIDを
	// 作成順に並べる。パートをupload IDで区切ることで、同じフェイクに対する2回目のmultipartが
	// 1回目の残したパートでcompleteしてしまうことを防ぐ。
	parts     map[string]map[int][]byte
	uploadIDs []string

	deletedKeys   []string
	abortRequests []abortRequest
	failAbort     bool

	// getErrorCode / deleteErrorCodeは、そのS3エラーコードを載せた404をGetObject /
	// DeleteObjectに返させる。"NoSuchKey" は存在しないオブジェクトに対してストレージが返すもので、
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
		// 拒否されたabortはパートをバケットに残す。返されるエラーが報告しているのはこの
		// 状態である。破棄するのはabortが成功として応答されるときだけにする。
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
		t.Fatalf("CreateMultipartUploadの呼び出し回数 = %d、期待値 = 1", len(uploadIDs))
	}
	if len(abortRequests) != 1 {
		t.Fatalf("AbortMultipartUploadの呼び出し回数 = %d、期待値 = 1", len(abortRequests))
	}
	request := abortRequests[0]
	if request.key != key {
		t.Errorf("abortのkey = %q、期待値 = %q", request.key, key)
	}
	if request.uploadID != uploadIDs[0] {
		t.Errorf("abortのupload ID = %q、期待値 = %q", request.uploadID, uploadIDs[0])
	}
	if request.contextCanceled {
		t.Error("abortのrequest contextがキャンセルされている")
	}

	// 成功したabortはパートを破棄し、拒否されたabortはそのまま残す。
	wantUploads := 0
	if failAbort {
		wantUploads = 1
	}
	if remainingUploads != wantUploads {
		t.Errorf("abort後に残っているupload = %d、期待値 = %d", remainingUploads, wantUploads)
	}
}

func writeXML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + body))
}

// writeS3Errorはcodeを載せた404を返す。アダプタが区別しなければならない2つのコード
// (NoSuchKeyとNoSuchBucket) はどちらも404で返るため、ステータスだけでは結果を決められない。
func writeS3Error(w http.ResponseWriter, code string) {
	writeXML(w, http.StatusNotFound, fmt.Sprintf(`<Error><Code>%s</Code><Message>%s</Message></Error>`, code, code))
}

// newTestStorageはフェイクのストレージを起動し、それを向いたアダプタを返す。
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
		t.Fatalf("NewS3ObjectStorage()のエラー = %v", err)
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
			name: "異常系: アクセスキーIDが空",
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
					t.Fatal("NewS3ObjectStorage()のエラー = nil、期待値 = エラー")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewS3ObjectStorage()のエラー = %v", err)
			}
			if s == nil {
				t.Fatal("NewS3ObjectStorage() = nil、期待値 = nilではない")
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
		t.Fatalf("Upload()のエラー = %v", err)
	}

	stored, contentType, ok := fake.object(key)
	if !ok {
		t.Fatalf("オブジェクトが保存されていない (key: %s)", key)
	}
	if !bytes.Equal(stored, body) {
		t.Errorf("保存された本体 = %q、期待値 = %q", stored, body)
	}
	if contentType != "application/zip" {
		t.Errorf("Content-Type = %q、期待値 = %q", contentType, "application/zip")
	}

	reader, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get()のエラー = %v", err)
	}
	defer func() { _ = reader.Close() }()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("取得したオブジェクトの読み取りに失敗: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("Get() = %q、期待値 = %q", got, body)
	}

	fake.mu.Lock()
	putHeaders := fake.putHeaders.Clone()
	getHeaders := fake.getHeaders.Clone()
	fake.mu.Unlock()

	for name := range putHeaders {
		if strings.HasPrefix(strings.ToLower(name), "x-amz-checksum-") ||
			strings.EqualFold(name, "X-Amz-Sdk-Checksum-Algorithm") {
			t.Errorf("PutObjectの任意チェックサムヘッダー%qが設定されている", name)
		}
	}
	if got := getHeaders.Get("X-Amz-Checksum-Mode"); got != "" {
		t.Errorf("GetObjectのX-Amz-Checksum-Mode = %q、期待値 = 空", got)
	}
}

// TestS3ObjectStorage_UploadMultipartは、アップローダーが単一partで扱う閾値を超えるbodyが
// partに分割されて組み立て直されることを確認する。
func TestS3ObjectStorage_UploadMultipart(t *testing.T) {
	t.Parallel()

	s, fake := newTestStorage(t)
	ctx := context.Background()
	const key = "exports/large.zip"

	// アップローダーはパートサイズに5 MiBの下限を持つため、multipartの経路を通すには
	// bodyがそれを超えている必要がある。
	body := bytes.Repeat([]byte("wikino-export-"), 6*1024*1024/14+1)

	if err := s.Upload(ctx, storage.UploadInput{
		Key:         key,
		Body:        bytes.NewReader(body),
		ContentType: "application/zip",
	}); err != nil {
		t.Fatalf("Upload()のエラー = %v", err)
	}

	fake.mu.Lock()
	uploadCount := len(fake.uploadIDs)
	partCount := 0
	if uploadCount > 0 {
		partCount = len(fake.parts[fake.uploadIDs[0]])
	}
	fake.mu.Unlock()

	if uploadCount != 1 {
		t.Errorf("CreateMultipartUploadの呼び出し回数 = %d、期待値 = 1", uploadCount)
	}
	if partCount < 2 {
		t.Errorf("パート数 = %d、期待値 = 2以上", partCount)
	}

	stored, _, ok := fake.object(key)
	if !ok {
		t.Fatalf("オブジェクトが保存されていない (key: %s)", key)
	}
	if !bytes.Equal(stored, body) {
		t.Errorf("組み立て直された本体の長さ = %d、期待値 = %d", len(stored), len(body))
	}
}

func TestS3ObjectStorage_UploadMultipartFailure(t *testing.T) {
	t.Parallel()

	const (
		key           = "exports/interrupted.zip"
		firstPartSize = 5 * 1024 * 1024
	)

	t.Run("異常系: multipartに至らない失敗ではabortしない", func(t *testing.T) {
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
			t.Errorf("Upload()のエラー = %v、期待値 = %v", err, wantErr)
		}

		fake.mu.Lock()
		uploadCount := len(fake.uploadIDs)
		abortCount := len(fake.abortRequests)
		fake.mu.Unlock()

		if uploadCount != 0 {
			t.Errorf("CreateMultipartUploadの呼び出し回数 = %d、期待値 = 0", uploadCount)
		}
		if abortCount != 0 {
			t.Errorf("AbortMultipartUploadの呼び出し回数 = %d、期待値 = 0", abortCount)
		}
	})

	t.Run("異常系: readerの失敗後に同じuploadを一度abortする", func(t *testing.T) {
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
			t.Errorf("Upload()のエラー = %v、期待値 = %v", err, wantErr)
		}
		assertMultipartAbort(t, fake, key)
	})

	t.Run("異常系: upload contextがキャンセルされてもabortはキャンセルされない", func(t *testing.T) {
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
			t.Errorf("Upload()のエラー = %v、期待値 = context.Canceled", err)
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("uploadのcontextのエラー = %v、期待値 = context.Canceled", ctx.Err())
		}
		assertMultipartAbort(t, fake, key)
	})

	t.Run("異常系: readerとabortの両方の失敗を返す", func(t *testing.T) {
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
			t.Errorf("Upload()のエラー = %v、期待値 = %v", err, wantErr)
		}
		var abortErr interface {
			ErrorCode() string
		}
		if !errors.As(err, &abortErr) {
			t.Fatalf("Upload()のエラー = %v、期待値 = abort APIのエラー", err)
		}
		if abortErr.ErrorCode() != "AbortDenied" {
			t.Errorf("abortのエラーコード = %q、期待値 = %q", abortErr.ErrorCode(), "AbortDenied")
		}
		assertMultipartAbort(t, fake, key)
	})
}

func TestS3ObjectStorage_Get(t *testing.T) {
	t.Parallel()

	t.Run("異常系: 存在しないキーはErrObjectNotFoundになる", func(t *testing.T) {
		t.Parallel()

		s, _ := newTestStorage(t)

		reader, err := s.Get(context.Background(), "attachments/gone.png")
		if err == nil {
			_ = reader.Close()
			t.Fatal("Get()のエラー = nil、期待値 = ErrObjectNotFound")
		}
		if !errors.Is(err, storage.ErrObjectNotFound) {
			t.Errorf("Get()のエラー = %v、期待値 = ErrObjectNotFound", err)
		}
	})

	// NoSuchBucketを伴う404はバケットが存在しないか到達できないことを示す。これを
	// ErrObjectNotFoundとして報告すると、リトライする価値のある失敗がエクスポートの記録する
	// 壊れた添付ファイルに変わってしまうため、アダプタは元のエラーをそのまま通す必要がある。
	t.Run("異常系: NoSuchKey以外のエラーはErrObjectNotFoundにしない", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		fake.mu.Lock()
		fake.getErrorCode = "NoSuchBucket"
		fake.mu.Unlock()

		reader, err := s.Get(context.Background(), "attachments/photo.png")
		if err == nil {
			_ = reader.Close()
			t.Fatal("Get()のエラー = nil、期待値 = エラー")
		}
		if errors.Is(err, storage.ErrObjectNotFound) {
			t.Errorf("Get()のエラー = %v、期待値 = ErrObjectNotFound以外のエラー", err)
		}

		var apiErr interface {
			ErrorCode() string
		}
		if !errors.As(err, &apiErr) {
			t.Fatalf("Get()のエラー = %v、期待値 = S3 APIのエラー", err)
		}
		if apiErr.ErrorCode() != "NoSuchBucket" {
			t.Errorf("エラーコード = %q、期待値 = %q", apiErr.ErrorCode(), "NoSuchBucket")
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
			t.Fatalf("Upload()のエラー = %v", err)
		}
		if err := s.Delete(ctx, key); err != nil {
			t.Fatalf("Delete()のエラー = %v", err)
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
			t.Errorf("Delete()のエラー = %v、期待値 = nil", err)
		}
	})

	// オブジェクトがすでに無いことを意味するのはNoSuchKeyだけである。それ以外の404まで
	// 飲み込むと、オブジェクトが残っているのに後片付けが終わったことにしてしまう。
	t.Run("異常系: NoSuchKey以外のエラーは失敗として返す", func(t *testing.T) {
		t.Parallel()

		s, fake := newTestStorage(t)
		fake.mu.Lock()
		fake.deleteErrorCode = "NoSuchBucket"
		fake.mu.Unlock()

		err := s.Delete(context.Background(), "exports/wikino.zip")
		if err == nil {
			t.Fatal("Delete()のエラー = nil、期待値 = エラー")
		}

		var apiErr interface {
			ErrorCode() string
		}
		if !errors.As(err, &apiErr) {
			t.Fatalf("Delete()のエラー = %v、期待値 = S3 APIのエラー", err)
		}
		if apiErr.ErrorCode() != "NoSuchBucket" {
			t.Errorf("エラーコード = %q、期待値 = %q", apiErr.ErrorCode(), "NoSuchBucket")
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
		{name: "正常系: リージョン未設定時はautoを使う", region: "", wantRegion: "auto"},
		{name: "正常系: 設定したリージョンを使う", region: "apac", wantRegion: "apac"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, _ := newTestStorageWithRegion(t, tt.region)
			presigned, err := s.PresignedGetURL(context.Background(), key, 24*time.Hour)
			if err != nil {
				t.Fatalf("PresignedGetURL()のエラー = %v", err)
			}

			parsed, err := url.Parse(presigned)
			if err != nil {
				t.Fatalf("presigned URLのパースに失敗: %v", err)
			}
			if want := "/" + testBucket + "/" + key; parsed.Path != want {
				t.Errorf("presigned URLのパス = %q、期待値 = %q", parsed.Path, want)
			}
			if got := parsed.Query().Get("X-Amz-Expires"); got != strconv.Itoa(int((24 * time.Hour).Seconds())) {
				t.Errorf("X-Amz-Expires = %q、期待値 = %q", got, strconv.Itoa(int((24 * time.Hour).Seconds())))
			}
			if parsed.Query().Get("X-Amz-Signature") == "" {
				t.Error("presigned URLに署名が含まれていない")
			}

			credential := strings.Split(parsed.Query().Get("X-Amz-Credential"), "/")
			if len(credential) != 5 {
				t.Fatalf("X-Amz-Credential = %q、期待値 = スラッシュ区切りの5つのフィールド", credential)
			}
			if got := credential[2]; got != tt.wantRegion {
				t.Errorf("署名リージョン = %q、期待値 = %q", got, tt.wantRegion)
			}
		})
	}
}
