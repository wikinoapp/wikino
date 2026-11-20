package usecase

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // ActiveStorageのチェックサムの形式を確かめるため
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// processAttachmentSecretは、テストの画像とSVGのメタデータに埋め、処理の後に残っていないことを確かめる文字列
const processAttachmentSecret = "secret-location-35.6812N-139.7671E"

// processAttachmentFixtureは、アップロード後の処理のテストが使うスペースとUseCase
type processAttachmentFixture struct {
	db            *sql.DB
	spaceID       model.SpaceID
	memberID      model.SpaceMemberID
	objectStorage *storage.FakeObjectStorage
	usecase       *ProcessAttachmentUsecase
	suffix        string
}

func setupProcessAttachmentFixture(t *testing.T, suffix string) *processAttachmentFixture {
	t.Helper()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("process-attachment-" + suffix + "@example.com").
		WithAtname("process_attachment_" + strings.ReplaceAll(suffix, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("process-attachment-" + suffix).Build()
	memberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()

	objectStorage := storage.NewFakeObjectStorage()
	return &processAttachmentFixture{
		db:            db,
		spaceID:       spaceID,
		memberID:      memberID,
		objectStorage: objectStorage,
		usecase:       NewProcessAttachmentUsecase(db, repository.NewAttachmentRepository(query.New(db)), objectStorage),
		suffix:        suffix,
	}
}

// writeThenFailObjectStorageは、実体の書き戻しは完了したが、応答がエラーになった状態を再現する。
type writeThenFailObjectStorage struct {
	*storage.FakeObjectStorage
	beforeUpload  func()
	afterUpload   func()
	uploadErr     error
	cleanupCtxErr error
}

func (s *writeThenFailObjectStorage) Upload(ctx context.Context, input storage.UploadInput) error {
	s.beforeUpload()
	if err := s.FakeObjectStorage.Upload(ctx, input); err != nil {
		return err
	}
	s.afterUpload()
	return s.uploadErr
}

func (s *writeThenFailObjectStorage) Delete(ctx context.Context, key string) error {
	s.cleanupCtxErr = ctx.Err()
	if s.cleanupCtxErr != nil {
		return s.cleanupCtxErr
	}
	return s.FakeObjectStorage.Delete(ctx, key)
}

func TestProcessAttachmentUsecase_書き戻しの失敗時も削除済みの実体を片付ける(t *testing.T) {
	t.Parallel()

	uploadErr := errors.New("書き戻し後に応答が失われた")
	cleanupErr := errors.New("実体の後始末に失敗した")
	for _, tt := range []struct {
		name      string
		suffix    string
		cancel    bool
		deleteErr error
	}{
		{name: "書き戻しのエラー", suffix: "write-reply-error"},
		{name: "contextの打ち切り", suffix: "write-reply-cancel", cancel: true},
		{name: "後始末のエラーも保持する", suffix: "write-cleanup-error", deleteErr: cleanupErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupProcessAttachmentFixture(t, tt.suffix)
			id, key := f.add(t, "target", "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), time.Now(), model.AttachmentProcessingStatusPending)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			objectStorage := &writeThenFailObjectStorage{
				FakeObjectStorage: f.objectStorage,
				uploadErr:         uploadErr,
				beforeUpload: func() {
					if err := f.objectStorage.Delete(t.Context(), key); err != nil {
						t.Fatalf("実体の削除に失敗: %v", err)
					}
					tx, err := f.db.BeginTx(t.Context(), nil)
					if err != nil {
						t.Fatalf("トランザクションの開始に失敗: %v", err)
					}
					defer func() { _ = tx.Rollback() }()
					repo := repository.NewAttachmentRepository(query.New(tx))
					if _, err := repo.DeleteByIDsAndSpace(t.Context(), []model.AttachmentID{id}, f.spaceID); err != nil {
						t.Fatalf("行の削除に失敗: %v", err)
					}
					if err := tx.Commit(); err != nil {
						t.Fatalf("コミットに失敗: %v", err)
					}
					f.objectStorage.DeleteErr = tt.deleteErr
				},
				afterUpload: func() {
					if tt.cancel {
						cancel()
					}
				},
			}
			uc := NewProcessAttachmentUsecase(f.db, repository.NewAttachmentRepository(query.New(f.db)), objectStorage)
			_, err := uc.Execute(ctx, ProcessAttachmentInput{AttachmentID: id, SpaceID: f.spaceID})
			if !errors.Is(err, uploadErr) {
				t.Errorf("エラー = %v、書き戻しのエラーを保持することを期待", err)
			}
			if tt.deleteErr != nil && !errors.Is(err, tt.deleteErr) {
				t.Errorf("エラー = %v、後始末のエラーを保持することを期待", err)
			}
			if objectStorage.cleanupCtxErr != nil {
				t.Errorf("後始末のcontextのエラー = %v、打ち切りから切り離すことを期待", objectStorage.cleanupCtxErr)
			}
			if _, _, exists := f.objectStorage.Object(key); tt.deleteErr == nil && exists {
				t.Error("削除済みの行に書き戻した実体が残っている")
			}
		})
	}
}

// addは、種類contentTypeの添付ファイルの行を作り、contentがnilでなければオブジェクトを置く
func (f *processAttachmentFixture) add(t *testing.T, name, contentType string, content []byte, attachedAt time.Time, status model.AttachmentProcessingStatus) (model.AttachmentID, string) {
	t.Helper()

	key := "process-attachment-" + f.suffix + "-" + name
	id := testutil.NewAttachmentBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.memberID).
		WithContentType(contentType).
		WithBlobKey(key).
		WithAttachedAt(attachedAt).
		WithProcessingStatus(status).
		Build()
	if content != nil {
		f.objectStorage.Put(key, content, contentType)
	}
	return id, key
}

func (f *processAttachmentFixture) execute(t *testing.T, id model.AttachmentID) (*ProcessAttachmentOutput, error) {
	t.Helper()
	return f.usecase.Execute(context.Background(), ProcessAttachmentInput{AttachmentID: id, SpaceID: f.spaceID})
}

// mustCompleteは、1回の処理で添付ファイルが完了になることを確かめる
func (f *processAttachmentFixture) mustComplete(t *testing.T, id model.AttachmentID) {
	t.Helper()
	out, err := f.execute(t, id)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if out.RetryAfter != 0 {
		t.Errorf("RetryAfter = %v、期待値 = 0", out.RetryAfter)
	}
	f.assertStatus(t, id, model.AttachmentProcessingStatusCompleted)
}

func (f *processAttachmentFixture) assertStatus(t *testing.T, id model.AttachmentID, want model.AttachmentProcessingStatus) {
	t.Helper()
	var got model.AttachmentProcessingStatus
	if err := f.db.QueryRow(`SELECT processing_status FROM attachments WHERE id = $1`, string(id)).Scan(&got); err != nil {
		t.Fatalf("状態の取得に失敗: %v", err)
	}
	if got != want {
		t.Errorf("状態 = %d、期待値 = %d", got, want)
	}
}

// blobは、添付ファイルのblobのバイト数とチェックサムを返す
func (f *processAttachmentFixture) blob(t *testing.T, id model.AttachmentID) (int64, sql.NullString) {
	t.Helper()
	var byteSize int64
	var checksum sql.NullString
	if err := f.db.QueryRow(`SELECT asb.byte_size, asb.checksum FROM attachments a
		INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
		INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
		WHERE a.id = $1`, string(id)).Scan(&byteSize, &checksum); err != nil {
		t.Fatalf("blobの取得に失敗: %v", err)
	}
	return byteSize, checksum
}

func (f *processAttachmentFixture) object(t *testing.T, key string) ([]byte, string) {
	t.Helper()
	body, contentType, ok := f.objectStorage.Object(key)
	if !ok {
		t.Fatalf("オブジェクトが無い (key: %s)", key)
	}
	return body, contentType
}

func testJPEGWithEXIF(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, testProcessImage(), nil); err != nil {
		t.Fatalf("JPEGのエンコードに失敗: %v", err)
	}
	encoded := buf.Bytes()
	payload := append([]byte("Exif\x00\x00MM\x00\x2a\x00\x00\x00\x08\x00\x00"), processAttachmentSecret...)
	segment := binary.BigEndian.AppendUint16([]byte{0xFF, 0xE1}, uint16(len(payload)+2))
	segment = append(segment, payload...)
	return append(append(append([]byte{}, encoded[:2]...), segment...), encoded[2:]...)
}

func testPNGWithText(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, testProcessImage()); err != nil {
		t.Fatalf("PNGのエンコードに失敗: %v", err)
	}
	encoded := buf.Bytes()
	// シグネチャ (8バイト) とIHDR (25バイト) の後ろにtEXtを入れる
	const afterIHDR = 8 + 25
	chunk := testPNGChunk("tEXt", []byte("Comment\x00"+processAttachmentSecret))
	return append(append(append([]byte{}, encoded[:afterIHDR]...), chunk...), encoded[afterIHDR:]...)
}

func testPNGChunk(chunkType string, data []byte) []byte {
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	chunk = append(chunk, chunkType...)
	chunk = append(chunk, data...)
	return binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
}

// testWebPWithEXIFは、1×1の可逆圧縮の画像にEXIFを付けた拡張形式のWebPを返す
func testWebPWithEXIF() []byte {
	vp8l := []byte{0x2f, 0x00, 0x00, 0x00, 0x10, 0x07, 0x10, 0x11, 0x11, 0x88, 0x88, 0xfe, 0x07}
	exif := []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x00" + processAttachmentSecret)
	var body []byte
	body = append(body, "WEBP"...)
	body = appendTestWebPChunk(body, "VP8X", []byte{0x08, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	body = appendTestWebPChunk(body, "VP8L", vp8l)
	body = appendTestWebPChunk(body, "EXIF", exif)
	out := append([]byte("RIFF"), binary.LittleEndian.AppendUint32(nil, uint32(len(body)))...)
	return append(out, body...)
}

func appendTestWebPChunk(dst []byte, fourCC string, data []byte) []byte {
	dst = append(dst, fourCC...)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(data)))
	dst = append(dst, data...)
	if len(data)%2 == 1 {
		dst = append(dst, 0)
	}
	return dst
}

func testProcessImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	return img
}

func testChecksum(content []byte) string {
	sum := md5.Sum(content) //nolint:gosec // ActiveStorageのチェックサムの形式
	return base64.StdEncoding.EncodeToString(sum[:])
}

func TestProcessAttachmentUsecase_RemovesMetadata(t *testing.T) {
	t.Parallel()

	f := setupProcessAttachmentFixture(t, "metadata")
	now := time.Now()

	tests := []struct {
		name        string
		contentType string
		content     []byte
	}{
		{name: "jpeg", contentType: "image/jpeg", content: testJPEGWithEXIF(t)},
		{name: "png", contentType: "image/png", content: testPNGWithText(t)},
		{name: "webp", contentType: "image/webp", content: testWebPWithEXIF()},
		{name: "svg", contentType: "image/svg+xml", content: []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><desc>` + processAttachmentSecret + `</desc><rect onclick="x" width="1"/></svg>`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, key := f.add(t, tt.name, tt.contentType, tt.content, now, model.AttachmentProcessingStatusPending)
			f.mustComplete(t, id)

			body, contentType := f.object(t, key)
			if bytes.Equal(body, tt.content) {
				t.Fatal("オブジェクトが書き換えられていない")
			}
			for _, s := range []string{"script", "onclick"} {
				if bytes.Contains(body, []byte(s)) {
					t.Errorf("オブジェクトに %qが残っている", s)
				}
			}
			if tt.name != "svg" && bytes.Contains(body, []byte(processAttachmentSecret)) {
				t.Error("オブジェクトにメタデータが残っている")
			}
			if contentType != tt.contentType {
				t.Errorf("Content-Type = %q、期待値 = %q", contentType, tt.contentType)
			}

			byteSize, checksum := f.blob(t, id)
			if byteSize != int64(len(body)) {
				t.Errorf("blobのバイト数 = %d、期待値 = %d", byteSize, len(body))
			}
			if checksum.String != testChecksum(body) {
				t.Errorf("blobのチェックサム = %q、期待値 = %q", checksum.String, testChecksum(body))
			}

			// 2回目は処理待ちでないため、何もしない
			f.mustComplete(t, id)
			again, _ := f.object(t, key)
			if !bytes.Equal(again, body) {
				t.Error("2回目の処理でオブジェクトが変わった")
			}
		})
	}
}

func TestProcessAttachmentUsecase_CompletesWithoutChanges(t *testing.T) {
	t.Parallel()

	f := setupProcessAttachmentFixture(t, "unchanged")
	now := time.Now()

	for _, contentType := range []string{"image/gif", "video/mp4", "application/pdf"} {
		t.Run(contentType, func(t *testing.T) {
			t.Parallel()

			content := []byte("content of " + contentType + " " + processAttachmentSecret)
			id, key := f.add(t, strings.ReplaceAll(contentType, "/", "-"), contentType, content, now, model.AttachmentProcessingStatusPending)
			f.mustComplete(t, id)

			if body, _ := f.object(t, key); !bytes.Equal(body, content) {
				t.Error("オブジェクトが変わった")
			}
			if byteSize, checksum := f.blob(t, id); byteSize != 1024 || checksum.Valid {
				t.Errorf("blob = (%d, %v)、期待値 = 変わらない (1024, NULL)", byteSize, checksum)
			}
		})
	}
}

func TestProcessAttachmentUsecase_WaitsForUpload(t *testing.T) {
	t.Parallel()

	f := setupProcessAttachmentFixture(t, "wait")
	// 並行するテストで処理が遅れても先送りする時間が変わらないよう、UseCaseの時刻を固定する
	// attached_atはマイクロ秒までしか持たないため、秒に揃える
	now := time.Now().Truncate(time.Second)
	f.usecase.now = func() time.Time { return now }

	tests := []struct {
		name    string
		elapsed time.Duration
		want    time.Duration
	}{
		{name: "事前署名の直後は5秒先送りする", elapsed: 0, want: 5 * time.Second},
		{name: "経過時間だけ先送りする", elapsed: 2 * time.Minute, want: 2 * time.Minute},
		{name: "先送りは5分までにする", elapsed: 30 * time.Minute, want: 5 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, _ := f.add(t, strings.ReplaceAll(tt.elapsed.String(), ".", "_"), "image/png", nil, now.Add(-tt.elapsed), model.AttachmentProcessingStatusPending)
			out, err := f.execute(t, id)
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if out.RetryAfter != tt.want {
				t.Errorf("RetryAfter = %v、期待値 = %v", out.RetryAfter, tt.want)
			}
			f.assertStatus(t, id, model.AttachmentProcessingStatusPending)
		})
	}

	t.Run("上限の時間を過ぎたら失敗にする", func(t *testing.T) {
		t.Parallel()

		id, _ := f.add(t, "expired", "image/png", nil, now.Add(-model.AttachmentUploadDeadline-time.Second), model.AttachmentProcessingStatusPending)
		out, err := f.execute(t, id)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if out.RetryAfter != 0 {
			t.Errorf("RetryAfter = %v、期待値 = 0", out.RetryAfter)
		}
		f.assertStatus(t, id, model.AttachmentProcessingStatusFailed)
	})
}

func TestAttachmentUploadWait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		elapsed time.Duration
		want    time.Duration
	}{
		{elapsed: -time.Minute, want: 5 * time.Second},
		{elapsed: 3 * time.Second, want: 5 * time.Second},
		{elapsed: 40 * time.Second, want: 40 * time.Second},
		{elapsed: 10 * time.Minute, want: 5 * time.Minute},
		{elapsed: model.AttachmentUploadDeadline - 2*time.Second, want: 3 * time.Second},
	}
	for _, tt := range tests {
		if got := attachmentUploadWait(tt.elapsed); got != tt.want {
			t.Errorf("attachmentUploadWait(%v) = %v、期待値 = %v", tt.elapsed, got, tt.want)
		}
	}
}

func TestProcessAttachmentUsecase_FailsUnprocessableContent(t *testing.T) {
	t.Parallel()

	f := setupProcessAttachmentFixture(t, "unprocessable")
	now := time.Now()

	tests := []struct {
		name        string
		contentType string
		content     []byte
	}{
		{name: "壊れたJPEG", contentType: "image/jpeg", content: []byte("\xff\xd8\xff\xe1\x00")},
		{name: "画像でない内容", contentType: "image/png", content: []byte("not an image")},
		{name: "XMLでないSVG", contentType: "image/svg+xml", content: []byte("<svg><g></svg>")},
		{name: "上限を超える画像", contentType: "image/png", content: make([]byte, model.AttachmentMaxByteSize("image/png")+1)},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, key := f.add(t, string(rune('a'+i)), tt.contentType, tt.content, now, model.AttachmentProcessingStatusPending)
			out, err := f.execute(t, id)
			if err != nil {
				t.Fatalf("Execute()のエラー = %v、期待値 = nil (再試行しない)", err)
			}
			if out.RetryAfter != 0 {
				t.Errorf("RetryAfter = %v、期待値 = 0", out.RetryAfter)
			}
			f.assertStatus(t, id, model.AttachmentProcessingStatusFailed)
			if body, _ := f.object(t, key); !bytes.Equal(body, tt.content) {
				t.Error("処理できない内容のオブジェクトが変わった")
			}
		})
	}
}

// interruptingObjectStorageは、存在の確認や取得が呼ばれたときに、指定した関数を呼んでから
// フェイクのストレージに処理を渡す。試行のcontextを途中で打ち切り、その後の失敗の記録だけを失敗させるのに使う
type interruptingObjectStorage struct {
	*storage.FakeObjectStorage
	onExists func()
	onGet    func()
}

func (s *interruptingObjectStorage) Exists(ctx context.Context, key string) (bool, error) {
	if s.onExists != nil {
		s.onExists()
	}
	return s.FakeObjectStorage.Exists(ctx, key)
}

func (s *interruptingObjectStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if s.onGet != nil {
		s.onGet()
	}
	return s.FakeObjectStorage.Get(ctx, key)
}

func TestProcessAttachmentUsecase_RetriesFailureRecording(t *testing.T) {
	t.Parallel()

	t.Run("処理できない内容の失敗を記録できなければ、処理待ちに戻して次の試行で記録する", func(t *testing.T) {
		t.Parallel()

		f := setupProcessAttachmentFixture(t, "record-unprocessable")
		content := []byte("not an image")
		id, key := f.add(t, "broken", "image/png", content, time.Now(), model.AttachmentProcessingStatusPending)

		// 内容を取得したあとに試行のcontextを打ち切り、失敗の記録だけを失敗させる
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		f.usecase.objectStorage = &interruptingObjectStorage{FakeObjectStorage: f.objectStorage, onGet: cancel}

		if out, err := f.usecase.Execute(ctx, ProcessAttachmentInput{AttachmentID: id, SpaceID: f.spaceID}); err == nil || out != nil {
			t.Fatalf("Execute() = (%+v, %v)、期待値 = nil, エラー", out, err)
		}
		f.assertStatus(t, id, model.AttachmentProcessingStatusPending)
		if body, _ := f.object(t, key); !bytes.Equal(body, content) {
			t.Error("失敗を記録できなかった試行でオブジェクトが変わった")
		}

		f.usecase.objectStorage = f.objectStorage
		out, err := f.execute(t, id)
		if err != nil {
			t.Fatalf("次の試行のExecute()のエラー = %v、期待値 = nil", err)
		}
		if out.RetryAfter != 0 {
			t.Errorf("RetryAfter = %v、期待値 = 0", out.RetryAfter)
		}
		f.assertStatus(t, id, model.AttachmentProcessingStatusFailed)
	})

	t.Run("上限の時間を過ぎた失敗を記録できなければ、エラーを返して処理待ちのまま残す", func(t *testing.T) {
		t.Parallel()

		f := setupProcessAttachmentFixture(t, "record-expired")
		id, _ := f.add(t, "missing", "image/png", nil, time.Now().Add(-model.AttachmentUploadDeadline-time.Second), model.AttachmentProcessingStatusPending)

		// 存在を確かめたあとに試行のcontextを打ち切り、失敗の記録だけを失敗させる
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		f.usecase.objectStorage = &interruptingObjectStorage{FakeObjectStorage: f.objectStorage, onExists: cancel}

		if _, err := f.usecase.Execute(ctx, ProcessAttachmentInput{AttachmentID: id, SpaceID: f.spaceID}); err == nil {
			t.Fatal("Execute()のエラー = nil、期待値 = エラー (riverの再試行で記録し直す)")
		}
		f.assertStatus(t, id, model.AttachmentProcessingStatusPending)

		f.usecase.objectStorage = f.objectStorage
		out, err := f.execute(t, id)
		if err != nil {
			t.Fatalf("次の試行のExecute()のエラー = %v、期待値 = nil", err)
		}
		if out.RetryAfter != 0 {
			t.Errorf("RetryAfter = %v、期待値 = 0", out.RetryAfter)
		}
		f.assertStatus(t, id, model.AttachmentProcessingStatusFailed)
	})
}

func TestProcessAttachmentUsecase_RetriesStorageFailures(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("一時的な失敗")
	tests := []struct {
		name   string
		inject func(*storage.FakeObjectStorage)
	}{
		{name: "存在の確認", inject: func(s *storage.FakeObjectStorage) { s.ExistsErr = storageErr }},
		{name: "取得", inject: func(s *storage.FakeObjectStorage) { s.GetErr = storageErr }},
		{name: "書き戻し", inject: func(s *storage.FakeObjectStorage) { s.UploadErr = storageErr }},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupProcessAttachmentFixture(t, "retry-"+string(rune('a'+i)))
			content := testJPEGWithEXIF(t)
			id, key := f.add(t, "photo", "image/jpeg", content, time.Now(), model.AttachmentProcessingStatusPending)
			tt.inject(f.objectStorage)

			if _, err := f.execute(t, id); !errors.Is(err, storageErr) {
				t.Fatalf("Execute()のエラー = %v、期待値 = %v", err, storageErr)
			}
			// 次の試行で処理し直せるよう、処理待ちのまま (処理待ちに戻して) 残す
			f.assertStatus(t, id, model.AttachmentProcessingStatusPending)
			if body, _ := f.object(t, key); !bytes.Equal(body, content) {
				t.Error("失敗した試行でオブジェクトが変わった")
			}
			if byteSize, checksum := f.blob(t, id); byteSize != 1024 || checksum.Valid {
				t.Errorf("失敗した試行でblobが変わった: (%d, %v)", byteSize, checksum)
			}

			f.objectStorage.ExistsErr, f.objectStorage.GetErr, f.objectStorage.UploadErr = nil, nil, nil
			f.mustComplete(t, id)
		})
	}
}

func TestProcessAttachmentUsecase_DoesNothing(t *testing.T) {
	t.Parallel()

	f := setupProcessAttachmentFixture(t, "nothing")
	now := time.Now()
	content := testJPEGWithEXIF(t)

	t.Run("行が無い", func(t *testing.T) {
		t.Parallel()
		out, err := f.execute(t, model.AttachmentID("00000000-0000-0000-0000-000000000000"))
		if err != nil || out.RetryAfter != 0 {
			t.Errorf("Execute() = (%+v, %v)、期待値 = 何もしない", out, err)
		}
	})

	for _, status := range []model.AttachmentProcessingStatus{
		model.AttachmentProcessingStatusProcessing,
		model.AttachmentProcessingStatusCompleted,
		model.AttachmentProcessingStatusFailed,
	} {
		t.Run("処理待ちでない", func(t *testing.T) {
			t.Parallel()

			id, key := f.add(t, "status-"+string(rune('0'+status)), "image/jpeg", content, now, status)
			out, err := f.execute(t, id)
			if err != nil || out.RetryAfter != 0 {
				t.Errorf("Execute() = (%+v, %v)、期待値 = 何もしない", out, err)
			}
			f.assertStatus(t, id, status)
			if body, _ := f.object(t, key); !bytes.Equal(body, content) {
				t.Error("オブジェクトが変わった")
			}
		})
	}

	t.Run("ほかのスペースを指定した", func(t *testing.T) {
		t.Parallel()

		other := setupProcessAttachmentFixture(t, "nothing-other")
		id, key := f.add(t, "other-space", "image/jpeg", content, now, model.AttachmentProcessingStatusPending)
		out, err := other.usecase.Execute(context.Background(), ProcessAttachmentInput{AttachmentID: id, SpaceID: other.spaceID})
		if err != nil || out.RetryAfter != 0 {
			t.Errorf("Execute() = (%+v, %v)、期待値 = 何もしない", out, err)
		}
		f.assertStatus(t, id, model.AttachmentProcessingStatusPending)
		if body, _ := f.object(t, key); !bytes.Equal(body, content) {
			t.Error("オブジェクトが変わった")
		}
	})
}

func TestProcessAttachmentUsecase_LeavesOtherAttachments(t *testing.T) {
	t.Parallel()

	f := setupProcessAttachmentFixture(t, "others")
	now := time.Now()
	content := testJPEGWithEXIF(t)
	id, _ := f.add(t, "target", "image/jpeg", content, now, model.AttachmentProcessingStatusPending)
	otherID, otherKey := f.add(t, "other", "image/jpeg", content, now, model.AttachmentProcessingStatusPending)

	f.mustComplete(t, id)

	f.assertStatus(t, otherID, model.AttachmentProcessingStatusPending)
	if body, _ := f.object(t, otherKey); !bytes.Equal(body, content) {
		t.Error("ほかの添付ファイルのオブジェクトが変わった")
	}
	if byteSize, checksum := f.blob(t, otherID); byteSize != 1024 || checksum.Valid {
		t.Errorf("ほかの添付ファイルのblobが変わった: (%d, %v)", byteSize, checksum)
	}
}

func TestProcessAttachmentUsecase_UpdatesUnchangedImage(t *testing.T) {
	t.Parallel()

	f := setupProcessAttachmentFixture(t, "clean-image")
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, testProcessImage(), nil); err != nil {
		t.Fatalf("JPEGのエンコードに失敗: %v", err)
	}
	content := encoded.Bytes()
	id, key := f.add(t, "photo", "image/jpeg", content, time.Now(), model.AttachmentProcessingStatusPending)
	f.objectStorage.UploadErr = errors.New("内容が変わらない画像はUploadを呼ばない")

	f.mustComplete(t, id)

	if body, contentType := f.object(t, key); !bytes.Equal(body, content) || contentType != "image/jpeg" {
		t.Error("内容が変わらない画像のオブジェクトが変わった")
	}
	if byteSize, checksum := f.blob(t, id); byteSize != int64(len(content)) || checksum.String != testChecksum(content) {
		t.Errorf("blob = (%d, %v)、期待値 = (%d, %s)", byteSize, checksum, len(content), testChecksum(content))
	}
}

func TestProcessAttachmentUsecase_RetriesDatabaseFailureAfterUpload(t *testing.T) {
	t.Parallel()

	f := setupProcessAttachmentFixture(t, "retry-db")
	content := testJPEGWithEXIF(t)
	id, key := f.add(t, "photo", "image/jpeg", content, time.Now(), model.AttachmentProcessingStatusPending)

	// リポジトリの接続は保ち、完了を記録するトランザクションの開始だけを失敗させる
	closedDB, err := sql.Open("postgres", "")
	if err != nil {
		t.Fatalf("DBの生成に失敗: %v", err)
	}
	if err := closedDB.Close(); err != nil {
		t.Fatalf("DBを閉じるのに失敗: %v", err)
	}
	f.usecase.db = closedDB

	if out, err := f.execute(t, id); err == nil || out != nil {
		t.Fatalf("Execute() = (%+v, %v)、期待値 = nil, エラー", out, err)
	}
	f.assertStatus(t, id, model.AttachmentProcessingStatusPending)
	processed, _ := f.object(t, key)
	if bytes.Equal(processed, content) || bytes.Contains(processed, []byte(processAttachmentSecret)) {
		t.Fatal("DBの失敗前に、メタデータを除去した内容が書き戻されていない")
	}
	if byteSize, checksum := f.blob(t, id); byteSize != 1024 || checksum.Valid {
		t.Errorf("失敗した試行でblobが変わった: (%d, %v)", byteSize, checksum)
	}

	// 既に処理されたオブジェクトを再アップロードせず、blobと完了状態を記録し直す
	f.usecase.db = f.db
	f.objectStorage.UploadErr = errors.New("再試行で再アップロードしない")
	f.mustComplete(t, id)

	if body, contentType := f.object(t, key); !bytes.Equal(body, processed) || contentType != "image/jpeg" {
		t.Error("再試行で処理済みのオブジェクトが変わった")
	}
	if byteSize, checksum := f.blob(t, id); byteSize != int64(len(processed)) || checksum.String != testChecksum(processed) {
		t.Errorf("再試行後のblob = (%d, %v)、期待値 = (%d, %s)", byteSize, checksum, len(processed), testChecksum(processed))
	}
}

func TestProcessAttachmentUsecase_Release(t *testing.T) {
	t.Parallel()

	t.Run("親のcontextがキャンセルされても処理待ちに戻す", func(t *testing.T) {
		t.Parallel()

		f := setupProcessAttachmentFixture(t, "release-cancel")
		id, _ := f.add(t, "photo", "image/jpeg", nil, time.Now(), model.AttachmentProcessingStatusProcessing)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		if err := f.usecase.release(ctx, &model.Attachment{ID: id, SpaceID: f.spaceID}); err != nil {
			t.Fatalf("release()のエラー = %v、期待値 = nil", err)
		}
		f.assertStatus(t, id, model.AttachmentProcessingStatusPending)
	})

	t.Run("ロックを待つ復旧更新を独立した期限で打ち切る", func(t *testing.T) {
		t.Parallel()

		f := setupProcessAttachmentFixture(t, "release-timeout")
		id, _ := f.add(t, "photo", "image/jpeg", nil, time.Now(), model.AttachmentProcessingStatusProcessing)
		f.usecase.cleanupTimeout = 50 * time.Millisecond
		tx, err := f.db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatalf("ロック用のトランザクションの開始に失敗: %v", err)
		}
		defer func() { _ = tx.Rollback() }()
		if _, err := tx.ExecContext(t.Context(), `UPDATE attachments SET updated_at=updated_at WHERE id=$1`, string(id)); err != nil {
			t.Fatalf("対象の行のロックに失敗: %v", err)
		}

		done := make(chan error, 1)
		go func() {
			done <- f.usecase.release(t.Context(), &model.Attachment{ID: id, SpaceID: f.spaceID})
		}()
		select {
		case err := <-done:
			if err == nil {
				t.Error("ロックを保持したまま復旧更新が成功した、期待値 = 期限切れのエラー")
			}
		case <-time.After(2 * time.Second):
			// 退行して期限が無くなっても、テストの行のロックとgoroutineを残さない
			_ = tx.Rollback()
			select {
			case err := <-done:
				if err != nil {
					t.Logf("ロック解除後の復旧更新のエラー: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Error("ロックを解除しても復旧更新が終了しない")
			}
			t.Fatal("復旧更新が独立した期限で打ち切られなかった")
		}

		if err := tx.Rollback(); err != nil {
			t.Fatalf("ロックの解除に失敗: %v", err)
		}
		f.assertStatus(t, id, model.AttachmentProcessingStatusProcessing)
	})
}
