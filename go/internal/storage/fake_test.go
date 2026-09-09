package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/storage"
)

// failingReader hands out remaining bytes of filler and then fails with err, which is how a test
// drives a body that breaks partway through. Setting cancel makes it cancel that context just
// before the error, reproducing an upload whose own context is canceled mid-flight.
//
// [Ja] failingReader は remaining バイトの埋め草を返してから err で失敗する。途中で壊れる body を
// テストが再現する手段である。cancel を設定すると、エラーを返す直前にその context をキャンセル
// する。アップロード自身の context が途中でキャンセルされる状況はこれで再現する。
type failingReader struct {
	remaining int
	cancel    context.CancelFunc
	err       error
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.remaining > 0 {
		n := min(len(p), r.remaining)
		for i := range p[:n] {
			p[i] = 'x'
		}
		r.remaining -= n
		return n, nil
	}
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	return 0, r.err
}

func TestFakeObjectStorage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("正常系: アップロードしたオブジェクトを取得できる", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()
		const key = "exports/wikino.zip"

		if err := fake.Upload(ctx, storage.UploadInput{
			Key:         key,
			Body:        strings.NewReader("archive"),
			ContentType: "application/zip",
		}); err != nil {
			t.Fatalf("Upload() error = %v", err)
		}

		body, contentType, ok := fake.Object(key)
		if !ok {
			t.Fatalf("オブジェクトが保持されていない (key: %s)", key)
		}
		if string(body) != "archive" {
			t.Errorf("保持された本体 = %q, want %q", body, "archive")
		}
		if contentType != "application/zip" {
			t.Errorf("Content-Type = %q, want %q", contentType, "application/zip")
		}

		reader, err := fake.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		defer func() { _ = reader.Close() }()

		got, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("取得したオブジェクトの読み取りに失敗: %v", err)
		}
		if string(got) != "archive" {
			t.Errorf("Get() = %q, want %q", got, "archive")
		}
	})

	t.Run("正常系: Put で用意したオブジェクトを取得できる", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()
		fake.Put("attachments/photo.png", []byte("png"), "image/png")

		body, contentType, ok := fake.Object("attachments/photo.png")
		if !ok {
			t.Fatal("オブジェクトが保持されていない")
		}
		if !bytes.Equal(body, []byte("png")) {
			t.Errorf("保持された本体 = %q, want %q", body, "png")
		}
		if contentType != "image/png" {
			t.Errorf("Content-Type = %q, want %q", contentType, "image/png")
		}
	})

	t.Run("正常系: Keys はソートされたキーを返す", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()
		fake.Put("b.md", []byte("b"), "")
		fake.Put("a.md", []byte("a"), "")
		fake.Put("c.md", []byte("c"), "")

		got := fake.Keys()
		want := []string{"a.md", "b.md", "c.md"}
		if len(got) != len(want) {
			t.Fatalf("Keys() = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Keys() = %v, want %v", got, want)
			}
		}
	})

	t.Run("異常系: 存在しないキーの取得は ErrObjectNotFound になる", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()

		if _, err := fake.Get(ctx, "exports/gone.zip"); !errors.Is(err, storage.ErrObjectNotFound) {
			t.Errorf("Get() error = %v, want ErrObjectNotFound", err)
		}
	})

	t.Run("異常系: reader の途中エラーでは不完全なオブジェクトを保存しない", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("読み取りに失敗しました")
		fake := storage.NewFakeObjectStorage()
		const key = "exports/incomplete.zip"

		err := fake.Upload(ctx, storage.UploadInput{
			Key:  key,
			Body: &failingReader{remaining: 8, err: wantErr},
		})
		if !errors.Is(err, wantErr) {
			t.Errorf("Upload() error = %v, want %v", err, wantErr)
		}
		if _, _, ok := fake.Object(key); ok {
			t.Errorf("不完全なオブジェクトが保存されている (key: %s)", key)
		}
	})

	t.Run("正常系: 削除は存在しないキーでも成功する", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()
		fake.Put("exports/old.zip", []byte("old"), "application/zip")

		if err := fake.Delete(ctx, "exports/old.zip"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if _, _, ok := fake.Object("exports/old.zip"); ok {
			t.Error("オブジェクトが残っている")
		}
		if err := fake.Delete(ctx, "exports/old.zip"); err != nil {
			t.Errorf("Delete() error = %v, want nil", err)
		}
	})

	t.Run("正常系: presigned URL はキーと有効期間を含む", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()

		presigned, err := fake.PresignedGetURL(ctx, "exports/wikino.zip", 24*time.Hour)
		if err != nil {
			t.Fatalf("PresignedGetURL() error = %v", err)
		}
		want := "https://storage.example.com/exports/wikino.zip?expires_in=86400"
		if presigned != want {
			t.Errorf("PresignedGetURL() = %q, want %q", presigned, want)
		}
	})

	t.Run("異常系: エラーを設定した操作は失敗する", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("ストレージに到達できません")
		fake := storage.NewFakeObjectStorage()
		fake.GetErr = wantErr
		fake.UploadErr = wantErr
		fake.DeleteErr = wantErr
		fake.PresignErr = wantErr

		if _, err := fake.Get(ctx, "exports/wikino.zip"); !errors.Is(err, wantErr) {
			t.Errorf("Get() error = %v, want %v", err, wantErr)
		}
		if err := fake.Upload(ctx, storage.UploadInput{Key: "exports/wikino.zip"}); !errors.Is(err, wantErr) {
			t.Errorf("Upload() error = %v, want %v", err, wantErr)
		}
		if err := fake.Delete(ctx, "exports/wikino.zip"); !errors.Is(err, wantErr) {
			t.Errorf("Delete() error = %v, want %v", err, wantErr)
		}
		if _, err := fake.PresignedGetURL(ctx, "exports/wikino.zip", time.Hour); !errors.Is(err, wantErr) {
			t.Errorf("PresignedGetURL() error = %v, want %v", err, wantErr)
		}
	})
}
