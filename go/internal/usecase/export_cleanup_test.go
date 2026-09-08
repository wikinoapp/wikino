package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// exportCleanupStorage injects storage failures and records contexts at the I/O boundary.
//
// [Ja] exportCleanupStorage はストレージの失敗を注入し、I/O 境界の context を記録する。
type exportCleanupStorage struct {
	*storage.FakeObjectStorage
	deleteErr     error
	uploadContext context.Context
	deleteContext context.Context
	afterUpload   func(context.Context) error
}

func (s *exportCleanupStorage) Upload(ctx context.Context, input storage.UploadInput) error {
	s.uploadContext = ctx
	if err := s.FakeObjectStorage.Upload(ctx, input); err != nil {
		return err
	}
	if s.afterUpload != nil {
		return s.afterUpload(ctx)
	}
	return nil
}
func (s *exportCleanupStorage) Delete(ctx context.Context, key string) error {
	s.deleteContext = ctx
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.FakeObjectStorage.Delete(ctx, key)
}

// exportOutcomeSender checks that heartbeat has stopped and notification has a live deadline.
//
// [Ja] exportOutcomeSender は heartbeat の停止と、通知用の有効な期限を確認する。
type exportOutcomeSender struct {
	fakeExportSender
	t             *testing.T
	objectStorage *exportCleanupStorage
}

func (s *exportOutcomeSender) SendSucceeded(ctx context.Context, to, url, appURL, locale string) error {
	s.t.Helper()
	if s.objectStorage.uploadContext.Err() == nil {
		s.t.Error("generation context remains active after success")
	}
	if ctx.Err() != nil {
		s.t.Errorf("notification context canceled: %v", ctx.Err())
	}
	if _, ok := ctx.Deadline(); !ok {
		s.t.Error("notification has no deadline")
	}
	return s.fakeExportSender.SendSucceeded(ctx, to, url, appURL, locale)
}

func TestGenerateExportFilesUsecase_OutcomeContexts(t *testing.T) {
	// Sequential execution avoids disturbing the existing temporary-file cleanup assertions.
	//
	// [Ja] 既存の一時ファイル数の検証に干渉しないよう逐次実行する。
	for _, replaced := range []bool{false, true} {
		name := "success"
		if replaced {
			name = "replaced"
		}
		t.Run(name, func(t *testing.T) {
			f := setupGenerateExportFixture(t, "outcome-"+name)
			id := f.queuedExport(t)
			objectStorage := &exportCleanupStorage{FakeObjectStorage: f.objectStorage}
			f.usecase.objectStorage = objectStorage
			sender := &exportOutcomeSender{t: t, objectStorage: objectStorage}
			f.usecase.sender = sender
			if replaced {
				objectStorage.afterUpload = func(ctx context.Context) error { _, err := f.exportRepo.MarkFailed(ctx, id, f.spaceID); return err }
			} else {
				f.objectStorage.Put("outcome-old.zip", []byte("old"), exportContentType)
				testutil.NewExportBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithQueuedByID(f.memberID).WithStatus(model.ExportStatusSucceeded).WithObjectKey("outcome-old.zip").WithCreatedAt(time.Now().Add(-time.Hour)).Build()
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if err := f.usecase.Execute(ctx, GenerateExportFilesInput{ExportID: id, SpaceID: f.spaceID, FinalAttempt: true}); err != nil {
				t.Fatal(err)
			}
			wantMail := 1
			if replaced {
				wantMail = 0
			}
			if len(sender.succeededURLs) != wantMail {
				t.Errorf("success mails=%d, want %d", len(sender.succeededURLs), wantMail)
			}
			if objectStorage.deleteContext == nil {
				t.Fatal("cleanup not called")
			}
			if _, ok := objectStorage.deleteContext.Deadline(); !ok {
				t.Error("cleanup has no deadline")
			}
			if keys := f.objectStorage.Keys(); len(keys) != wantMail {
				t.Errorf("remaining objects=%v", keys)
			}
		})
	}
}

// addLegacyExportFile creates the associations used by Rails has_one_attached :file.
//
// [Ja] addLegacyExportFile は Rails の has_one_attached :file が使う関連を作成する。
func addLegacyExportFile(t *testing.T, f generateExportFixture, id model.ExportID, key string) string {
	t.Helper()
	var blobID string
	err := f.db.QueryRowContext(context.Background(), `INSERT INTO active_storage_blobs (key,filename,content_type,service_name,byte_size,created_at)
 VALUES ($1,'old.zip','application/zip','local',3,now()) RETURNING id`, key).Scan(&blobID)
	if err != nil {
		t.Fatal(err)
	}
	attachLegacyExportFile(t, f, id, blobID)
	f.objectStorage.Put(key, []byte("old"), exportContentType)
	return blobID
}

// attachLegacyExportFile links an existing blob to an export for shared-blob tests.
//
// [Ja] attachLegacyExportFile は共有 blob のテスト用に既存 blob をエクスポートへ関連付ける。
func attachLegacyExportFile(t *testing.T, f generateExportFixture, id model.ExportID, blobID string) {
	t.Helper()
	_, err := f.db.ExecContext(context.Background(), `INSERT INTO active_storage_attachments (name,record_type,record_id,blob_id,created_at)
 VALUES ('file','ExportRecord',$1,$2,now())`, id.String(), blobID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateExportFilesUsecase_CleanupRetry(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		name := "go"
		if legacy {
			name = "rails"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := setupGenerateExportFixture(t, "retry-cleanup-"+name)
			key := "cleanup-retry-" + name + ".zip"
			builder := testutil.NewExportBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithQueuedByID(f.memberID).WithStatus(model.ExportStatusSucceeded).WithCreatedAt(time.Now().Add(-time.Hour))
			if !legacy {
				builder.WithObjectKey(key)
			}
			oldID := builder.Build()
			blobID := ""
			if legacy {
				blobID = addLegacyExportFile(t, f, oldID, key)
			} else {
				f.objectStorage.Put(key, []byte("old"), exportContentType)
			}
			currentID := f.queuedExport(t)
			ctx := context.Background()
			current, err := f.exportRepo.FindByIDAndSpace(ctx, currentID, f.spaceID)
			if err != nil {
				t.Fatal(err)
			}
			objectStorage := &exportCleanupStorage{FakeObjectStorage: f.objectStorage, deleteErr: errors.New("storage unavailable")}
			f.usecase.objectStorage = objectStorage
			f.usecase.deleteReplacedExports(ctx, current)
			old, err := f.exportRepo.FindByIDAndSpace(ctx, oldID, f.spaceID)
			if err != nil {
				t.Fatal(err)
			}
			if old == nil {
				t.Fatal("record lost after failed storage deletion")
			}
			if legacy {
				files, err := f.exportRepo.ListLegacyFiles(ctx, oldID, f.spaceID)
				if err != nil || len(files) != 1 {
					t.Fatalf("legacy association lost: %v, %v", files, err)
				}
			}
			objectStorage.deleteErr = nil
			f.usecase.deleteReplacedExports(ctx, current)
			old, err = f.exportRepo.FindByIDAndSpace(ctx, oldID, f.spaceID)
			if err != nil {
				t.Fatal(err)
			}
			if old != nil {
				t.Fatal("old record remains after successful retry")
			}
			if keys := f.objectStorage.Keys(); len(keys) != 0 {
				t.Errorf("old ZIP remains: %v", keys)
			}
			if legacy {
				var exists bool
				if err := f.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM active_storage_blobs WHERE id=$1)", blobID).Scan(&exists); err != nil {
					t.Fatal(err)
				}
				if exists {
					t.Error("unreferenced legacy blob remains")
				}
			}
		})
	}
}

func TestGenerateExportFilesUsecase_LegacySpaceIsolation(t *testing.T) {
	t.Parallel()
	f := setupGenerateExportFixture(t, "legacy-isolation")
	other := setupGenerateExportFixture(t, "legacy-isolation-other")
	oldID := testutil.NewExportBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithQueuedByID(f.memberID).WithStatus(model.ExportStatusSucceeded).WithCreatedAt(time.Now().Add(-time.Hour)).Build()
	blobID := addLegacyExportFile(t, f, oldID, "shared-legacy.zip")
	otherID := other.queuedExport(t)
	attachLegacyExportFile(t, other, otherID, blobID)
	ctx := context.Background()
	if files, err := f.exportRepo.ListLegacyFiles(ctx, oldID, other.spaceID); err != nil || len(files) != 0 {
		t.Fatalf("cross-space lookup: %v %v", files, err)
	}
	if err := f.exportRepo.Delete(ctx, oldID, other.spaceID); err != nil {
		t.Fatal(err)
	}
	files, err := f.exportRepo.ListLegacyFiles(ctx, oldID, f.spaceID)
	if err != nil || len(files) != 1 || !files[0].Shared {
		t.Fatalf("shared archive lost or misclassified: %v %v", files, err)
	}
	currentID := f.queuedExport(t)
	current, err := f.exportRepo.FindByIDAndSpace(ctx, currentID, f.spaceID)
	if err != nil {
		t.Fatal(err)
	}
	f.usecase.deleteReplacedExports(ctx, current)
	old, err := f.exportRepo.FindByIDAndSpace(ctx, oldID, f.spaceID)
	if err != nil || old != nil {
		t.Fatalf("old export remains: %v %v", old, err)
	}
	files, err = other.exportRepo.ListLegacyFiles(ctx, otherID, other.spaceID)
	if err != nil || len(files) != 1 {
		t.Fatalf("other export lost its archive: %v %v", files, err)
	}
	if _, _, ok := f.objectStorage.Object("shared-legacy.zip"); !ok {
		t.Error("shared object deleted")
	}
}

func TestGenerateExportFilesUsecase_UnrecordedArchiveCleanup(t *testing.T) {
	// Sequential execution avoids disturbing the existing temporary-file cleanup assertions.
	//
	// [Ja] 既存の一時ファイル数の検証に干渉しないよう逐次実行する。
	for _, failure := range []string{"success-write", "replacement-delete"} {
		t.Run(failure, func(t *testing.T) {
			f := setupGenerateExportFixture(t, "unrecorded-"+failure)
			ctx := context.Background()
			id := f.queuedExport(t)
			key := exportObjectKey(&model.Export{ID: id, SpaceID: f.spaceID})
			objectStorage := &exportCleanupStorage{FakeObjectStorage: f.objectStorage, deleteErr: errors.New("storage unavailable")}
			f.usecase.objectStorage = objectStorage

			if failure == "success-write" {
				export, err := f.exportRepo.MarkStarted(ctx, id, f.spaceID)
				if err != nil || export == nil {
					t.Fatalf("start export: %v, %v", export, err)
				}
				tx, err := f.db.BeginTx(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = tx.Rollback() }()
				f.usecase.exportRepo = f.exportRepo.WithTx(tx)
				// End the real transaction after upload so recording success fails at the DB boundary.
				// Then use a live connection for the final-attempt failure handling performed by Execute.
				//
				// [Ja] アップロード直後に実トランザクションを終了し、DB 境界で成功記録を失敗させる。
				// その後、有効な接続で Execute と同じ最終試行の失敗処理を行う。
				objectStorage.afterUpload = func(context.Context) error { return tx.Rollback() }
				err = f.usecase.generate(ctx, export)
				if !errors.Is(err, sql.ErrTxDone) {
					t.Fatalf("success write error = %v, want transaction done", err)
				}
				f.usecase.exportRepo = f.exportRepo
				f.usecase.recordFailure(ctx, id, f.spaceID, true, err)
			} else {
				objectStorage.afterUpload = func(ctx context.Context) error {
					_, err := f.exportRepo.MarkFailed(ctx, id, f.spaceID)
					return err
				}
				if err := f.usecase.Execute(ctx, GenerateExportFilesInput{ExportID: id, SpaceID: f.spaceID, FinalAttempt: true}); err != nil {
					t.Fatal(err)
				}
			}

			old, err := f.exportRepo.FindByIDAndSpace(ctx, id, f.spaceID)
			if err != nil || old == nil || old.Status != model.ExportStatusFailed || old.ObjectKey != nil {
				t.Fatalf("failed export without recorded key = %v, %v", old, err)
			}
			if _, _, ok := f.objectStorage.Object(key); !ok {
				t.Fatal("uploaded archive missing before cleanup")
			}
			objectStorage.afterUpload = nil
			for _, unavailable := range []bool{true, false} {
				if !unavailable {
					objectStorage.deleteErr = nil
				}
				currentID := f.queuedExport(t)
				if err := f.usecase.Execute(ctx, GenerateExportFilesInput{ExportID: currentID, SpaceID: f.spaceID, FinalAttempt: true}); err != nil {
					t.Fatal(err)
				}
				current, err := f.exportRepo.FindByIDAndSpace(ctx, currentID, f.spaceID)
				if err != nil || current == nil || current.Status != model.ExportStatusSucceeded {
					t.Fatalf("next export did not succeed: %v, %v", current, err)
				}
				old, err = f.exportRepo.FindByIDAndSpace(ctx, id, f.spaceID)
				if err != nil {
					t.Fatal(err)
				}
				if (old != nil) != unavailable {
					t.Fatalf("storage unavailable=%v: old record = %v", unavailable, old)
				}
				if _, _, ok := f.objectStorage.Object(key); ok != unavailable {
					t.Fatalf("storage unavailable=%v: old archive exists=%v", unavailable, ok)
				}
				if _, _, ok := f.objectStorage.Object(*current.ObjectKey); !ok {
					t.Fatal("current archive was deleted")
				}
			}
		})
	}
}
