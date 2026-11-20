package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func (f *deleteSpaceFixture) exportCleanupJobs(t *testing.T) []*rivertype.JobRow {
	t.Helper()
	jobs, err := f.jobClient.JobList(t.Context(), river.NewJobListParams().Kinds(dispatcher.CleanupExportObjectsArgs{}.Kind()))
	if err != nil {
		t.Fatalf("ZIPの後始末ジョブの取得に失敗: %v", err)
	}
	return jobs.Jobs
}

func exportCleanupInput(t *testing.T, job *rivertype.JobRow) CleanupExportObjectsInput {
	t.Helper()
	var args dispatcher.CleanupExportObjectsArgs
	if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
		t.Fatalf("ZIPの後始末ジョブの引数の取得に失敗: %v", err)
	}
	blobs := make([]model.LegacyExportBlob, len(args.LegacyBlobs))
	for i, blob := range args.LegacyBlobs {
		blobs[i] = model.LegacyExportBlob{ID: blob.ID, Key: blob.Key}
	}
	return CleanupExportObjectsInput{SpaceID: args.SpaceID, ExportIDs: args.ExportIDs, Keys: args.Keys, LegacyBlobs: blobs}
}

func (f *deleteSpaceFixture) exportCleanupUsecase() *CleanupExportObjectsUsecase {
	return NewCleanupExportObjectsUsecase(repository.NewExportRepository(query.New(f.db)), f.dispatcher, f.objectStorage)
}

func (f *deleteSpaceFixture) cleanupExports(t *testing.T) {
	t.Helper()
	for _, job := range f.exportCleanupJobs(t) {
		wait, err := f.exportCleanupUsecase().Execute(t.Context(), exportCleanupInput(t, job))
		if err != nil || wait != 0 {
			t.Fatalf("ZIPの後始末の結果: 待機時間 = %v、エラー = %v", wait, err)
		}
	}
}

func (f *deleteSpaceFixture) addExport(t *testing.T) model.ExportID {
	t.Helper()
	return testutil.NewExportBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithQueuedByID(f.memberID).WithStatus(model.ExportStatusQueued).Build()
}

func (f *deleteSpaceFixture) addLegacyExportBlob(t *testing.T, exportID model.ExportID, key string) string {
	t.Helper()
	var blobID string
	if err := f.db.QueryRowContext(t.Context(), `
		INSERT INTO active_storage_blobs (key, filename, content_type, service_name, byte_size, created_at)
		VALUES ($1, 'old.zip', 'application/zip', 'local', 3, NOW()) RETURNING id`, key).Scan(&blobID); err != nil {
		t.Fatalf("ZIPのblobの作成に失敗: %v", err)
	}
	f.attachExportBlob(t, exportID, blobID)
	f.objectStorage.Put(key, []byte("zip"), "application/zip")
	return blobID
}

func (f *deleteSpaceFixture) attachExportBlob(t *testing.T, exportID model.ExportID, blobID string) {
	t.Helper()
	if _, err := f.db.ExecContext(t.Context(), `
		INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		VALUES ('file', 'ExportRecord', $1, $2, NOW())`, string(exportID), blobID); err != nil {
		t.Fatalf("ZIPの関連の作成に失敗: %v", err)
	}
}

type interruptedExportStorage struct {
	*storage.FakeObjectStorage
	beforeUpload func()
	uploadErr    error
	deleteErr    error
	key          string
}

func (s *interruptedExportStorage) Upload(ctx context.Context, input storage.UploadInput) error {
	if s.beforeUpload != nil {
		s.beforeUpload()
	}
	if err := s.FakeObjectStorage.Upload(ctx, input); err != nil {
		return err
	}
	s.key = input.Key
	s.DeleteErr = s.deleteErr
	return s.uploadErr
}

func (f *deleteSpaceFixture) exportGenerator(objects storage.ObjectStorage) *GenerateExportFilesUsecase {
	q := query.New(f.db)
	return NewGenerateExportFilesUsecase(&config.Config{Env: "test", Domain: "wikino.example.com"},
		repository.NewExportRepository(q), repository.NewSpaceRepository(q), repository.NewSpaceMemberRepository(q), repository.NewUserRepository(q),
		repository.NewTopicRepository(q), repository.NewPageRepository(q), repository.NewAttachmentRepository(q), objects, nil)
}

func TestDeleteSpaceUsecase_生成途中や失敗済みのZIPも回収する(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                 string
		fixture              string
		during               bool
		uploadErr, deleteErr error
	}{
		{name: "生成中に行が消え、アップロードの応答が失われる", fixture: "upload-reply-lost", during: true, uploadErr: errors.New("アップロード後の応答が失われた")},
		{name: "生成中に行が消え、補償削除が失敗する", fixture: "compensation-failed", during: true, deleteErr: errors.New("補償削除が一時的に失敗した")},
		{name: "削除の前に生成が失敗している", fixture: "failed-before-deletion", uploadErr: errors.New("アップロード後の応答が失われた")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "export-"+tt.fixture)}
			exportID := f.addExport(t)
			objects := &interruptedExportStorage{FakeObjectStorage: f.objectStorage, uploadErr: tt.uploadErr, deleteErr: tt.deleteErr}
			remove := func() {
				f.discard(t)
				if err := f.deleteSpace(t); err != nil {
					t.Fatalf("スペースの削除に失敗: %v", err)
				}
			}
			if tt.during {
				objects.beforeUpload = remove
			}
			generator := f.exportGenerator(objects)
			input := GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID, FinalAttempt: !tt.during}
			err := generator.Execute(t.Context(), input)
			if tt.uploadErr != nil && !errors.Is(err, tt.uploadErr) {
				t.Fatalf("ZIPの生成結果 = %v、期待値 = %v", err, tt.uploadErr)
			}
			if !tt.during {
				remove()
			}
			f.objectStorage.DeleteErr = nil
			if tt.during && !f.hasObject(objects.key) {
				t.Fatal("行の削除後にZIPが書き戻されていない")
			}
			// どちらの通常ジョブを再試行しても、行が無いためZIPをたどれない。
			if err := generator.Execute(t.Context(), input); err != nil {
				t.Fatalf("生成の再試行に失敗: %v", err)
			}
			if err := f.deleteSpace(t); err != nil {
				t.Fatalf("削除の再試行に失敗: %v", err)
			}
			jobs := f.exportCleanupJobs(t)
			if len(jobs) != 1 {
				t.Fatalf("ZIPの後始末ジョブ = %d件、期待値 = 1件", len(jobs))
			}
			f.cleanupExports(t)
			f.cleanupExports(t)
			if objects.key == "" || f.hasObject(objects.key) {
				t.Error("後始末の後もZIPが残っている")
			}
			f.assertNoRows(t)
		})
	}
}

type afterExportDeletionStorage struct {
	*storage.FakeObjectStorage
	once        sync.Once
	afterDelete func() error
	err         error
}

func (s *afterExportDeletionStorage) DeleteMany(ctx context.Context, keys []string) error {
	if err := s.FakeObjectStorage.DeleteMany(ctx, keys); err != nil {
		return err
	}
	s.once.Do(func() { s.err = s.afterDelete() })
	return s.err
}

func TestDeleteSpaceUsecase_最初の実体の削除後に生成が成功してもZIPを回収する(t *testing.T) {
	t.Parallel()
	f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "export-late-success")}
	exportID := f.addExport(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	uploading, resume := make(chan struct{}), make(chan struct{})
	result := make(chan error, 1)
	objects := &interruptedExportStorage{FakeObjectStorage: f.objectStorage, beforeUpload: func() {
		close(uploading)
		select {
		case <-resume:
		case <-ctx.Done():
		}
	}}
	go func() {
		result <- f.exportGenerator(objects).Execute(ctx, GenerateExportFilesInput{ExportID: exportID, SpaceID: f.spaceID})
	}()
	select {
	case <-uploading:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	f.discard(t)
	deletionObjects := &afterExportDeletionStorage{FakeObjectStorage: f.objectStorage, afterDelete: func() error {
		close(resume)
		select {
		case err := <-result:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	if err := f.deleteSpaceUsecase(deletionObjects).Execute(ctx, DeleteSpaceInput{SpaceID: f.spaceID}); err != nil {
		t.Fatal(err)
	}
	if !f.hasObject(objects.key) {
		t.Fatal("最初の削除後にZIPが生成されていない")
	}
	f.cleanupExports(t)
	if f.hasObject(objects.key) {
		t.Error("行の削除前に完成したZIPが残っている")
	}
}

func TestCleanupExportObjectsUsecase_生成ジョブの終了を待つ(t *testing.T) {
	t.Parallel()
	f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "export-wait")}
	exportID := f.addExport(t)
	key := exportObjectKey(&model.Export{ID: exportID, SpaceID: f.spaceID})
	job, err := f.jobClient.Insert(t.Context(), dispatcher.GenerateExportFilesArgs{SpaceID: string(f.spaceID), ExportID: string(exportID)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.discard(t)
	if err := f.deleteSpace(t); err != nil {
		t.Fatal(err)
	}
	input := exportCleanupInput(t, f.exportCleanupJobs(t)[0])
	for _, args := range []dispatcher.GenerateExportFilesArgs{
		{SpaceID: "another-space", ExportID: string(exportID)},
		{SpaceID: string(f.spaceID), ExportID: "another-export"},
	} {
		if _, err := f.jobClient.Insert(t.Context(), args, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, state := range []rivertype.JobState{
		rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRetryable,
		rivertype.JobStateRunning, rivertype.JobStateScheduled,
	} {
		f.objectStorage.Put(key, []byte("zip"), "application/zip")
		if _, err := f.db.ExecContext(t.Context(), "UPDATE "+f.jobSchema+".river_job SET state = $1 WHERE id = $2", state, job.Job.ID); err != nil {
			t.Fatal(err)
		}
		wait, err := f.exportCleanupUsecase().Execute(t.Context(), input)
		if err != nil || wait <= 0 || !f.hasObject(key) {
			t.Errorf("状態 %s: 待機時間 = %v、エラー = %v、実体が残る = %v", state, wait, err, f.hasObject(key))
		}
	}
	for _, state := range []rivertype.JobState{rivertype.JobStateCompleted, rivertype.JobStateCancelled, rivertype.JobStateDiscarded} {
		f.objectStorage.Put(key, []byte("zip"), "application/zip")
		if _, err := f.db.ExecContext(t.Context(), "UPDATE "+f.jobSchema+".river_job SET state = $1, finalized_at = NOW() WHERE id = $2", state, job.Job.ID); err != nil {
			t.Fatal(err)
		}
		f.cleanupExports(t)
		if f.hasObject(key) {
			t.Errorf("生成ジョブが%sになってもZIPが残っている", state)
		}
	}
}

func TestCleanupExportObjectsUsecase_blobの確保後に失敗しても再試行できる(t *testing.T) {
	t.Parallel()
	f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "export-retry")}
	key := "export-retry-legacy"
	blobID := f.addLegacyExportBlob(t, f.addExport(t), key)
	f.discard(t)
	if err := f.deleteSpace(t); err != nil {
		t.Fatal(err)
	}
	// 初回の削除後の書き戻しと、blobを確保したあとのストレージ障害を再現する。
	f.objectStorage.Put(key, []byte("late zip"), "application/zip")
	deleteErr := errors.New("ストレージに到達できません")
	f.objectStorage.DeleteErr = deleteErr
	for _, job := range f.exportCleanupJobs(t) {
		input := exportCleanupInput(t, job)
		if len(input.LegacyBlobs) == 0 {
			continue
		}
		if _, err := f.exportCleanupUsecase().Execute(t.Context(), input); !errors.Is(err, deleteErr) {
			t.Fatalf("後始末のエラー = %v、期待値 = %v", err, deleteErr)
		}
	}
	var rows int
	if err := f.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM active_storage_blobs WHERE id = $1", blobID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 || !f.hasObject(key) {
		t.Fatal("blobだけが消え、実体が残った状態になっていない")
	}
	f.objectStorage.DeleteErr = nil
	f.cleanupExports(t)
	f.cleanupExports(t)
	if f.hasObject(key) {
		t.Error("blobが無い状態から後始末を再試行してもZIPが残っている")
	}
}

func TestCleanupExportObjectsUsecase_共有ZIPは最後の参照を消してから回収する(t *testing.T) {
	t.Parallel()
	for _, concurrent := range []bool{false, true} {
		name, fixture := "順に削除する", "sequential"
		if concurrent {
			name, fixture = "並行して削除する", "concurrent"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "zip-share-"+fixture+"-a")}
			b := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "zip-share-"+fixture+"-b")}
			b.objectStorage = a.objectStorage
			key := "zip-share-" + fixture
			blobID := a.addLegacyExportBlob(t, a.addExport(t), key)
			b.attachExportBlob(t, b.addExport(t), blobID)
			a.discard(t)
			b.discard(t)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			if concurrent {
				blocker, err := a.db.BeginTx(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = blocker.Rollback() }()
				var blockerPID int
				var locked string
				if err := blocker.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockerPID); err != nil {
					t.Fatal(err)
				}
				if err := blocker.QueryRowContext(ctx, "SELECT id FROM spaces WHERE id=$1 FOR KEY SHARE", string(a.spaceID)).Scan(&locked); err != nil {
					t.Fatal(err)
				}
				result := make(chan error, 1)
				go func() {
					result <- a.deleteSpaceUsecase(a.objectStorage).Execute(ctx, DeleteSpaceInput{SpaceID: a.spaceID})
				}()
				waitForQueryBlockedBy(t, ctx, a.db, blockerPID, "LockDiscardedSpace", result)
				if err := b.deleteSpace(t); err != nil {
					t.Fatal(err)
				}
				b.cleanupExports(t)
				if !a.hasObject(key) {
					t.Fatal("参照が残っている共有ZIPが消えた")
				}
				if err := blocker.Commit(); err != nil {
					t.Fatal(err)
				}
				if err := <-result; err != nil {
					t.Fatal(err)
				}
			} else {
				if err := a.deleteSpace(t); err != nil {
					t.Fatal(err)
				}
				a.cleanupExports(t)
				if !a.hasObject(key) {
					t.Fatal("参照が残っている共有ZIPが消えた")
				}
				if err := b.deleteSpace(t); err != nil {
					t.Fatal(err)
				}
			}
			a.cleanupExports(t)
			b.cleanupExports(t)
			if a.hasObject(key) {
				t.Error("最後の参照を消した共有ZIPが残っている")
			}
			var rows int
			if err := a.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM active_storage_blobs WHERE id=$1", blobID).Scan(&rows); err != nil || rows != 0 {
				t.Errorf("残ったblob = %d、エラー = %v", rows, err)
			}
		})
	}
}

func TestDeleteSpaceUsecase_ZIPの後始末の登録に失敗したら行を残す(t *testing.T) {
	t.Parallel()
	f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "export-enqueue-failure")}
	f.addLegacyExportBlob(t, f.addExport(t), "export-enqueue-failure")
	f.discard(t)
	insertErr := errors.New("ZIPの後始末を登録できなかった")
	f.dispatcher = dispatcher.NewDispatcher(nil, failingTxJobClient{TxJobClient: f.jobClient, err: insertErr})
	if err := f.deleteSpace(t); !errors.Is(err, insertErr) {
		t.Fatalf("削除のエラー = %v、期待値 = %v", err, insertErr)
	}
	for name, sql := range map[string]string{
		"スペース":   "SELECT COUNT(*) FROM spaces WHERE id=$1",
		"エクスポート": "SELECT COUNT(*) FROM exports WHERE space_id=$1",
		"ZIPの関連": "SELECT COUNT(*) FROM active_storage_attachments WHERE record_type='ExportRecord' AND record_id IN (SELECT id FROM exports WHERE space_id=$1)",
	} {
		var n int
		if err := f.db.QueryRowContext(t.Context(), sql, string(f.spaceID)).Scan(&n); err != nil || n != 1 {
			t.Errorf("%sの行 = %d、エラー = %v、期待値 = 1", name, n, err)
		}
	}
	if len(f.exportCleanupJobs(t)) != 0 {
		t.Error("ロールバックした後始末ジョブが残っている")
	}
	f.dispatcher = dispatcher.NewDispatcher(f.jobClient, f.jobClient)
	if err := f.deleteSpace(t); err != nil {
		t.Fatal(err)
	}
	f.cleanupExports(t)
}

func TestDeleteSpaceUsecase_ZIPのキーを一定件数ずつ読み後始末へ渡す(t *testing.T) {
	t.Parallel()
	f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "export-batches")}
	if _, err := f.db.ExecContext(t.Context(), `
		INSERT INTO exports (space_id, queued_by_id, status, status_changed_at, created_at, updated_at)
		SELECT $1, $2, 2, NOW(), NOW(), NOW() FROM generate_series(1, 1001)`, string(f.spaceID), string(f.memberID)); err != nil {
		t.Fatal(err)
	}
	f.discard(t)
	if err := f.deleteSpace(t); err != nil {
		t.Fatal(err)
	}
	jobs := f.exportCleanupJobs(t)
	if len(jobs) != 2 {
		t.Fatalf("ZIPの後始末ジョブ = %d件、期待値 = 2件", len(jobs))
	}
	ids := map[string]bool{}
	for _, job := range jobs {
		input := exportCleanupInput(t, job)
		if len(input.ExportIDs) == 0 || len(input.ExportIDs) > 1000 || len(input.Keys) != len(input.ExportIDs) {
			t.Errorf("バッチの件数: エクスポート = %d、キー = %d", len(input.ExportIDs), len(input.Keys))
		}
		for _, id := range input.ExportIDs {
			if ids[id] {
				t.Errorf("エクスポート %s が重複している", id)
			}
			ids[id] = true
		}
	}
	if len(ids) != 1001 || f.objectStorage.DeleteManyCalls() != 2 {
		t.Errorf("後始末へ渡したID = %d件、一括削除 = %d回", len(ids), f.objectStorage.DeleteManyCalls())
	}
}
