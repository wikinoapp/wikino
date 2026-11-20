package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/storage"
)

// failingReaderはremainingバイトの埋め草を返してからerrで失敗する。途中で壊れるbodyを
// テストが再現する手段である。cancelを設定すると、エラーを返す直前にそのcontextをキャンセル
// する。アップロード自身のcontextが途中でキャンセルされる状況はこれで再現する。
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
			t.Fatalf("Upload()のエラー = %v", err)
		}

		body, contentType, ok := fake.Object(key)
		if !ok {
			t.Fatalf("オブジェクトが保持されていない (key: %s)", key)
		}
		if string(body) != "archive" {
			t.Errorf("保持された本体 = %q、期待値 = %q", body, "archive")
		}
		if contentType != "application/zip" {
			t.Errorf("Content-Type = %q、期待値 = %q", contentType, "application/zip")
		}

		reader, err := fake.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get()のエラー = %v", err)
		}
		defer func() { _ = reader.Close() }()

		got, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("取得したオブジェクトの読み取りに失敗: %v", err)
		}
		if string(got) != "archive" {
			t.Errorf("Get() = %q、期待値 = %q", got, "archive")
		}
	})

	t.Run("正常系: Putで用意したオブジェクトを取得できる", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()
		fake.Put("attachments/photo.png", []byte("png"), "image/png")

		body, contentType, ok := fake.Object("attachments/photo.png")
		if !ok {
			t.Fatal("オブジェクトが保持されていない")
		}
		if !bytes.Equal(body, []byte("png")) {
			t.Errorf("保持された本体 = %q、期待値 = %q", body, "png")
		}
		if contentType != "image/png" {
			t.Errorf("Content-Type = %q、期待値 = %q", contentType, "image/png")
		}
	})

	t.Run("正常系: Existsは保持しているかを返す", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()
		fake.Put("attachments/photo.png", []byte("png"), "image/png")

		for key, want := range map[string]bool{"attachments/photo.png": true, "attachments/gone.png": false} {
			got, err := fake.Exists(ctx, key)
			if err != nil {
				t.Fatalf("Exists(%q)のエラー = %v", key, err)
			}
			if got != want {
				t.Errorf("Exists(%q) = %v、期待値 = %v", key, got, want)
			}
		}
	})

	t.Run("異常系: ExistsErrを設定するとExistsが失敗する", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("確認に失敗しました")
		fake := storage.NewFakeObjectStorage()
		fake.ExistsErr = wantErr

		if _, err := fake.Exists(ctx, "attachments/photo.png"); !errors.Is(err, wantErr) {
			t.Errorf("Exists()のエラー = %v、期待値 = %v", err, wantErr)
		}
	})

	t.Run("正常系: Keysはソートされたキーを返す", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()
		fake.Put("b.md", []byte("b"), "")
		fake.Put("a.md", []byte("a"), "")
		fake.Put("c.md", []byte("c"), "")

		got := fake.Keys()
		want := []string{"a.md", "b.md", "c.md"}
		if len(got) != len(want) {
			t.Fatalf("Keys() = %v、期待値 = %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Keys() = %v、期待値 = %v", got, want)
			}
		}
	})

	t.Run("異常系: 存在しないキーの取得はErrObjectNotFoundになる", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()

		if _, err := fake.Get(ctx, "exports/gone.zip"); !errors.Is(err, storage.ErrObjectNotFound) {
			t.Errorf("Get()のエラー = %v、期待値 = ErrObjectNotFound", err)
		}
	})

	t.Run("異常系: readerの途中エラーでは不完全なオブジェクトを保存しない", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("読み取りに失敗しました")
		fake := storage.NewFakeObjectStorage()
		const key = "exports/incomplete.zip"

		err := fake.Upload(ctx, storage.UploadInput{
			Key:  key,
			Body: &failingReader{remaining: 8, err: wantErr},
		})
		if !errors.Is(err, wantErr) {
			t.Errorf("Upload()のエラー = %v、期待値 = %v", err, wantErr)
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
			t.Fatalf("Delete()のエラー = %v", err)
		}
		if _, _, ok := fake.Object("exports/old.zip"); ok {
			t.Error("オブジェクトが残っている")
		}
		if err := fake.Delete(ctx, "exports/old.zip"); err != nil {
			t.Errorf("Delete()のエラー = %v、期待値 = nil", err)
		}
	})

	t.Run("正常系: まとめて削除し、呼び出しの回数を数える", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()
		fake.Put("attachments/a", []byte("a"), "image/png")
		fake.Put("attachments/b", []byte("b"), "image/png")
		fake.Put("attachments/other", []byte("other"), "image/png")

		if err := fake.DeleteMany(ctx, []string{"attachments/a", "attachments/b", "attachments/gone"}); err != nil {
			t.Fatalf("DeleteMany()のエラー = %v", err)
		}
		if got := fake.Keys(); len(got) != 1 || got[0] != "attachments/other" {
			t.Errorf("残ったキー = %v、期待値 = [attachments/other]", got)
		}
		if got := fake.DeleteManyCalls(); got != 1 {
			t.Errorf("DeleteManyCalls() = %d、期待値 = 1", got)
		}
	})

	t.Run("正常系: presigned URLはキーと有効期間を含む", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()

		presigned, err := fake.PresignedGetURL(ctx, "exports/wikino.zip", 24*time.Hour, storage.PresignedGetOptions{})
		if err != nil {
			t.Fatalf("PresignedGetURL()のエラー = %v", err)
		}
		want := "https://storage.example.com/exports/wikino.zip?expires_in=86400"
		if presigned != want {
			t.Errorf("PresignedGetURL() = %q、期待値 = %q", presigned, want)
		}
	})

	t.Run("正常系: 応答ヘッダーの指定をURLに含める", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()

		presigned, err := fake.PresignedGetURL(ctx, "attachments/key", 5*time.Minute, storage.PresignedGetOptions{
			ResponseContentDisposition: `attachment; filename="a.svg"`,
			ResponseContentType:        "application/octet-stream",
		})
		if err != nil {
			t.Fatalf("PresignedGetURL()のエラー = %v", err)
		}
		parsed, err := url.Parse(presigned)
		if err != nil {
			t.Fatalf("URLのパースに失敗: %v", err)
		}
		if got := parsed.Query().Get("response-content-disposition"); got != `attachment; filename="a.svg"` {
			t.Errorf("response-content-disposition = %q、期待値 = %q", got, `attachment; filename="a.svg"`)
		}
		if got := parsed.Query().Get("response-content-type"); got != "application/octet-stream" {
			t.Errorf("response-content-type = %q、期待値 = %q", got, "application/octet-stream")
		}
	})

	t.Run("正常系: アップロードのpresigned URLはキーと有効期間と条件を含む", func(t *testing.T) {
		t.Parallel()

		fake := storage.NewFakeObjectStorage()

		presigned, err := fake.PresignedPutURL(ctx, "attachments/key", 5*time.Minute, storage.PresignedPutOptions{
			ContentType:   "image/png",
			ContentLength: 1234,
			ContentMD5:    "1B2M2Y8AsgTpgAmY7PhCfg==",
		})
		if err != nil {
			t.Fatalf("PresignedPutURL()のエラー = %v", err)
		}
		parsed, err := url.Parse(presigned)
		if err != nil {
			t.Fatalf("URLのパースに失敗: %v", err)
		}
		if parsed.Path != "/attachments/key" {
			t.Errorf("パス = %q、期待値 = %q", parsed.Path, "/attachments/key")
		}
		for name, want := range map[string]string{
			"expires_in":     "300",
			"content_type":   "image/png",
			"content_length": "1234",
			"content_md5":    "1B2M2Y8AsgTpgAmY7PhCfg==",
		} {
			if got := parsed.Query().Get(name); got != want {
				t.Errorf("%s = %q、期待値 = %q", name, got, want)
			}
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
			t.Errorf("Get()のエラー = %v、期待値 = %v", err, wantErr)
		}
		if err := fake.Upload(ctx, storage.UploadInput{Key: "exports/wikino.zip"}); !errors.Is(err, wantErr) {
			t.Errorf("Upload()のエラー = %v、期待値 = %v", err, wantErr)
		}
		if err := fake.Delete(ctx, "exports/wikino.zip"); !errors.Is(err, wantErr) {
			t.Errorf("Delete()のエラー = %v、期待値 = %v", err, wantErr)
		}
		if err := fake.DeleteMany(ctx, []string{"exports/wikino.zip"}); !errors.Is(err, wantErr) {
			t.Errorf("DeleteMany()のエラー = %v、期待値 = %v", err, wantErr)
		}
		if _, err := fake.PresignedGetURL(ctx, "exports/wikino.zip", time.Hour, storage.PresignedGetOptions{}); !errors.Is(err, wantErr) {
			t.Errorf("PresignedGetURL()のエラー = %v、期待値 = %v", err, wantErr)
		}
		if _, err := fake.PresignedPutURL(ctx, "attachments/key", time.Minute, storage.PresignedPutOptions{}); !errors.Is(err, wantErr) {
			t.Errorf("PresignedPutURL()のエラー = %v、期待値 = %v", err, wantErr)
		}
	})
}
