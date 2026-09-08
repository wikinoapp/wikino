package usecase

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// fakeExportSender records the mails an export would have sent.
//
// [Ja] fakeExportSender は、エクスポートが送ろうとしたメールを記録する。
type fakeExportSender struct {
	succeededURLs []string
	failedURLs    []string
}

func (f *fakeExportSender) SendSucceeded(_ context.Context, _, downloadURL, _, _ string) error {
	f.succeededURLs = append(f.succeededURLs, downloadURL)
	return nil
}

func (f *fakeExportSender) SendFailed(_ context.Context, _, exportURL, _, _ string) error {
	f.failedURLs = append(f.failedURLs, exportURL)
	return nil
}

// generateExportFixture is the space an export is generated from, together with what the test
// needs to look at the result.
//
// [Ja] generateExportFixture はエクスポートの生成元になるスペースと、テストが結果を見るために
// 必要になるもの
type generateExportFixture struct {
	db            *sql.DB
	spaceID       model.SpaceID
	memberID      model.SpaceMemberID
	objectStorage *storage.FakeObjectStorage
	sender        *fakeExportSender
	exportRepo    *repository.ExportRepository
	usecase       *GenerateExportFilesUsecase
}

// setupGenerateExportFixture creates a space with one member and wires the UseCase against a fake
// storage and a fake mail sender.
//
// [Ja] setupGenerateExportFixture は、メンバーが 1 人いるスペースを作り、フェイクのストレージと
// フェイクのメール送信を使う UseCase を組み立てる。
func setupGenerateExportFixture(t *testing.T, suffix string) generateExportFixture {
	t.Helper()

	db := testutil.GetTestDB()
	queries := query.New(db)

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("generate-export-" + suffix + "@example.com").
		WithAtname("generate_export_" + suffix).
		Build()

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("generate-export-" + suffix).
		Build()

	memberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	objectStorage := storage.NewFakeObjectStorage()
	sender := &fakeExportSender{}
	exportRepo := repository.NewExportRepository(queries)

	uc := NewGenerateExportFilesUsecase(
		&config.Config{Env: "test", Domain: "wikino.example.com"},
		exportRepo,
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewUserRepository(queries),
		repository.NewTopicRepository(queries),
		repository.NewPageRepository(queries),
		repository.NewAttachmentRepository(queries),
		objectStorage,
		sender,
	)

	return generateExportFixture{
		db:            db,
		spaceID:       spaceID,
		memberID:      memberID,
		objectStorage: objectStorage,
		sender:        sender,
		exportRepo:    exportRepo,
		usecase:       uc,
	}
}

// queuedExport records an export waiting for its job, which is the state Execute starts from.
//
// [Ja] queuedExport はジョブの実行を待っているエクスポートを記録する。Execute はこの状態から
// 処理を始める。
func (f generateExportFixture) queuedExport(t *testing.T) model.ExportID {
	t.Helper()
	return testutil.NewExportBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.memberID).
		Build()
}

// archiveEntries reads the archive an export uploaded and returns each entry by name.
//
// [Ja] archiveEntries はエクスポートがアップロードしたアーカイブを読み、各エントリを名前ごとに返す。
func archiveEntries(t *testing.T, objectStorage *storage.FakeObjectStorage, objectKey string) map[string]string {
	t.Helper()

	body, _, ok := objectStorage.Object(objectKey)
	if !ok {
		t.Fatalf("オブジェクト %q がアップロードされていません (keys: %v)", objectKey, objectStorage.Keys())
	}

	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("ZIPの読み取りに失敗: %v", err)
	}

	entries := map[string]string{}
	for _, file := range reader.File {
		opened, err := file.Open()
		if err != nil {
			t.Fatalf("エントリ %q を開けませんでした: %v", file.Name, err)
		}
		content, err := io.ReadAll(opened)
		_ = opened.Close()
		if err != nil {
			t.Fatalf("エントリ %q を読めませんでした: %v", file.Name, err)
		}
		entries[file.Name] = string(content)
	}

	return entries
}

// tempFileCount counts the working files an export leaves in the temporary directory. Every one of
// them is removed before Execute returns, so the count is the same before and after.
//
// [Ja] tempFileCount は、エクスポートが一時ディレクトリへ残した作業用のファイルを数える。それらは
// すべて Execute が戻る前に削除されるため、実行の前後で数は変わらない。
func tempFileCount(t *testing.T) int {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "wikino-export-*"))
	if err != nil {
		t.Fatalf("一時ファイルの検索に失敗: %v", err)
	}
	return len(matches)
}

func TestGenerateExportFilesUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// The subtests run one after another so that the count of leftover working files is not
	// disturbed by another export writing its own at the same time.
	//
	// [Ja] サブテストは順に実行する。残っている作業用ファイルの数を、別のエクスポートが同時に
	// 自分のファイルを書くことで乱されないようにするためである。

	t.Run("トピックごとにページと添付ファイルを書き出す", func(t *testing.T) {
		f := setupGenerateExportFixture(t, "archive")

		topicID := testutil.NewTopicBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithNumber(1).
			WithName("設計: メモ").
			Build()

		attachmentID := testutil.NewAttachmentBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithSpaceMemberID(f.memberID).
			WithFilename("図 1.png").
			WithBlobKey("blob-archive-1").
			Build()
		f.objectStorage.Put("blob-archive-1", []byte("PNG"), "image/png")

		pageID := testutil.NewPageBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("API | 設計").
			WithBody("[[用語]] を見る\n\n![図](/attachments/" + attachmentID.String() + ")\n").
			Build()
		testutil.NewPageBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("用語").
			WithBody("用語の説明\n").
			Build()

		refRepo := repository.NewPageAttachmentReferenceRepository(query.New(f.db))
		if _, err := refRepo.CreateBatch(ctx, pageID, f.spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("添付ファイル参照の作成に失敗: %v", err)
		}

		exportID := f.queuedExport(t)
		before := tempFileCount(t)

		if err := f.usecase.Execute(ctx, GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID, FinalAttempt: true}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if after := tempFileCount(t); after != before {
			t.Errorf("一時ファイルが残っています: before = %d, after = %d", before, after)
		}

		export, err := f.exportRepo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export.Status != model.ExportStatusSucceeded {
			t.Fatalf("export.Status = %v, want %v", export.Status, model.ExportStatusSucceeded)
		}
		if export.ObjectKey == nil {
			t.Fatal("export.ObjectKey = nil, want a key")
		}

		entries := archiveEntries(t, f.objectStorage, *export.ObjectKey)

		pagePath := "設計： メモ/API ｜ 設計.md"
		body, ok := entries[pagePath]
		if !ok {
			t.Fatalf("エントリ %q がありません (entries: %v)", pagePath, entryNames(entries))
		}
		if !strings.HasPrefix(body, "---\nwikino_title: \"API | 設計\"\n---\n\n") {
			t.Errorf("frontmatter が期待と異なります: %q", body)
		}
		if !strings.Contains(body, "[[設計： メモ/用語.md]]") {
			t.Errorf("Wiki リンクが書き換えられていません: %q", body)
		}
		if !strings.Contains(body, "![図](attachments/%E5%9B%B3%201.png)") {
			t.Errorf("添付ファイルのリンクが書き換えられていません: %q", body)
		}

		if _, ok := entries["設計： メモ/attachments/図 1.png"]; !ok {
			t.Errorf("添付ファイルの複製がありません (entries: %v)", entryNames(entries))
		}

		if len(f.sender.succeededURLs) != 1 {
			t.Fatalf("完了メールの送信数 = %d, want 1", len(f.sender.succeededURLs))
		}
		wantURL := "https://wikino.example.com/s/generate-export-archive/settings/exports/" + exportID.String() + "/download"
		if f.sender.succeededURLs[0] != wantURL {
			t.Errorf("ダウンロードURL = %q, want %q", f.sender.succeededURLs[0], wantURL)
		}
	})

	t.Run("成功時に古いエクスポートとアーカイブを削除する", func(t *testing.T) {
		f := setupGenerateExportFixture(t, "cleanup")

		topicID := testutil.NewTopicBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithNumber(1).
			WithName("メモ").
			Build()
		testutil.NewPageBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("ページ").
			Build()

		f.objectStorage.Put("exports/old.zip", []byte("old"), exportContentType)
		oldID := testutil.NewExportBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.memberID).
			WithStatus(model.ExportStatusSucceeded).
			WithObjectKey("exports/old.zip").
			WithCreatedAt(time.Now().Add(-time.Hour)).
			Build()

		exportID := f.queuedExport(t)

		if err := f.usecase.Execute(ctx, GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID, FinalAttempt: true}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		old, err := f.exportRepo.FindByIDAndSpace(ctx, oldID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if old != nil {
			t.Errorf("古いエクスポートが残っています: %v", old)
		}
		if _, _, ok := f.objectStorage.Object("exports/old.zip"); ok {
			t.Error("古いアーカイブが残っています")
		}
	})

	t.Run("添付ファイルの取得に失敗したら最終試行で失敗として記録する", func(t *testing.T) {
		f := setupGenerateExportFixture(t, "missing-object")

		topicID := testutil.NewTopicBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithNumber(1).
			WithName("メモ").
			Build()
		attachmentID := testutil.NewAttachmentBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithSpaceMemberID(f.memberID).
			WithFilename("失われた.png").
			WithBlobKey("blob-missing").
			Build()
		pageID := testutil.NewPageBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("ページ").
			Build()

		refRepo := repository.NewPageAttachmentReferenceRepository(query.New(f.db))
		if _, err := refRepo.CreateBatch(ctx, pageID, f.spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("添付ファイル参照の作成に失敗: %v", err)
		}

		exportID := f.queuedExport(t)
		before := tempFileCount(t)

		err := f.usecase.Execute(ctx, GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID, FinalAttempt: true})
		if err == nil {
			t.Fatal("Execute() error = nil, want error")
		}

		if after := tempFileCount(t); after != before {
			t.Errorf("一時ファイルが残っています: before = %d, after = %d", before, after)
		}

		export, err := f.exportRepo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusFailed)
		}
		if len(f.sender.failedURLs) != 1 {
			t.Errorf("失敗メールの送信数 = %d, want 1", len(f.sender.failedURLs))
		}
	})

	t.Run("最終試行でない失敗では失敗として記録しない", func(t *testing.T) {
		f := setupGenerateExportFixture(t, "retryable")

		topicID := testutil.NewTopicBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithNumber(1).
			WithName("メモ").
			Build()
		attachmentID := testutil.NewAttachmentBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithSpaceMemberID(f.memberID).
			WithFilename("失われた.png").
			WithBlobKey("blob-retryable").
			Build()
		pageID := testutil.NewPageBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("ページ").
			Build()

		refRepo := repository.NewPageAttachmentReferenceRepository(query.New(f.db))
		if _, err := refRepo.CreateBatch(ctx, pageID, f.spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("添付ファイル参照の作成に失敗: %v", err)
		}

		exportID := f.queuedExport(t)

		if err := f.usecase.Execute(ctx, GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID}); err == nil {
			t.Fatal("Execute() error = nil, want error")
		}

		export, err := f.exportRepo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export.Status != model.ExportStatusStarted {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusStarted)
		}
		if len(f.sender.failedURLs) != 0 {
			t.Errorf("失敗メールの送信数 = %d, want 0", len(f.sender.failedURLs))
		}
	})

	t.Run("完了済みのエクスポートには何もしない", func(t *testing.T) {
		f := setupGenerateExportFixture(t, "done")

		exportID := testutil.NewExportBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.memberID).
			WithStatus(model.ExportStatusSucceeded).
			WithObjectKey("exports/kept.zip").
			Build()

		if err := f.usecase.Execute(ctx, GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID, FinalAttempt: true}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if keys := f.objectStorage.Keys(); len(keys) != 0 {
			t.Errorf("アップロードされたオブジェクト = %v, want none", keys)
		}
		if len(f.sender.succeededURLs) != 0 {
			t.Errorf("完了メールの送信数 = %d, want 0", len(f.sender.succeededURLs))
		}
	})

	// An attempt that cannot even record that it started leaves the export queued, and a queued
	// export holds its space. The last attempt therefore has to leave a result behind, which the
	// detached context is what makes possible.
	//
	// [Ja] 開始を記録することさえできなかった試行は、エクスポートを queued のまま残す。queued の
	// エクスポートはスペースを保つため、最後の試行は結果を残す必要がある。それを可能にしているのが
	// 切り離した context である。
	t.Run("最終試行が開始記録に失敗しても失敗として記録する", func(t *testing.T) {
		f := setupGenerateExportFixture(t, "start-failure")
		exportID := f.queuedExport(t)

		canceled, cancel := context.WithCancel(ctx)
		cancel()

		if err := f.usecase.Execute(canceled, GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID, FinalAttempt: true}); err == nil {
			t.Fatal("Execute() error = nil, want error")
		}

		export, err := f.exportRepo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusFailed)
		}
		if len(f.sender.failedURLs) != 1 {
			t.Errorf("失敗メールの送信数 = %d, want 1", len(f.sender.failedURLs))
		}
	})

	t.Run("最終試行でなければ開始記録の失敗をqueuedのまま残す", func(t *testing.T) {
		f := setupGenerateExportFixture(t, "start-failure-retryable")
		exportID := f.queuedExport(t)

		canceled, cancel := context.WithCancel(ctx)
		cancel()

		if err := f.usecase.Execute(canceled, GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID}); err == nil {
			t.Fatal("Execute() error = nil, want error")
		}

		export, err := f.exportRepo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export.Status != model.ExportStatusQueued {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusQueued)
		}
		if len(f.sender.failedURLs) != 0 {
			t.Errorf("失敗メールの送信数 = %d, want 0", len(f.sender.failedURLs))
		}
	})
}

// entryNames returns the names of the archive entries, for a failure message that says what the
// archive actually held.
//
// [Ja] entryNames はアーカイブのエントリ名を返す。アーカイブが実際に何を持っていたのかを伝える
// 失敗メッセージのために使う。
func entryNames(entries map[string]string) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	return names
}
