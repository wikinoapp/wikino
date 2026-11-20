package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func (f *deleteAttachmentFixture) cleanupJobs(t *testing.T) []*rivertype.JobRow {
	t.Helper()
	jobs, err := f.jobClient.JobList(t.Context(), river.NewJobListParams().Kinds(dispatcher.CleanupAttachmentObjectsArgs{}.Kind()))
	if err != nil {
		t.Fatalf("後始末ジョブの取得に失敗: %v", err)
	}
	return jobs.Jobs
}

func cleanupInput(t *testing.T, job *rivertype.JobRow) CleanupAttachmentObjectsInput {
	t.Helper()
	var args dispatcher.CleanupAttachmentObjectsArgs
	if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
		t.Fatalf("後始末ジョブの引数の取得に失敗: %v", err)
	}
	return CleanupAttachmentObjectsInput{SpaceID: args.SpaceID, AttachmentIDs: args.AttachmentIDs, Keys: args.Keys}
}

func (f *deleteAttachmentFixture) cleanup(t *testing.T) {
	t.Helper()
	uc := NewCleanupAttachmentObjectsUsecase(f.dispatcher, f.objectStorage)
	for _, job := range f.cleanupJobs(t) {
		wait, err := uc.Execute(t.Context(), cleanupInput(t, job))
		if err != nil || wait != 0 {
			t.Fatalf("後始末の結果: 待機時間 = %v、エラー = %v", wait, err)
		}
	}
}

func TestDeleteAttachmentUsecase_行の削除後もキーを保持して再試行できる(t *testing.T) {
	t.Parallel()
	f := setupDeleteAttachmentFixture(t, "durable-cleanup")
	id, key := f.add(t, "target")
	variantKey := f.addVariant(t, id, "target")
	if err := f.execute(t, id); err != nil {
		t.Fatalf("削除に失敗: %v", err)
	}
	f.assertNoRows(t, id, key)
	jobs := f.cleanupJobs(t)
	if len(jobs) != 1 {
		t.Fatalf("後始末ジョブの数 = %d、期待値 = 1", len(jobs))
	}
	input := cleanupInput(t, jobs[0])
	if input.SpaceID != string(f.spaceID) || !slices.Equal(input.AttachmentIDs, []string{string(id)}) {
		t.Errorf("後始末の対象が異なる: %+v", input)
	}
	keys := slices.Clone(input.Keys)
	slices.Sort(keys)
	want := []string{key, variantKey}
	slices.Sort(want)
	if !slices.Equal(keys, want) {
		t.Errorf("保存したキー = %v、期待値 = %v", keys, want)
	}

	// 行を消したあとの書き戻しと、2回目の実体の削除の失敗を再現する。
	f.objectStorage.Put(key, []byte("written back"), "image/png")
	deleteErr := errors.New("ストレージが一時的に停止した")
	f.objectStorage.DeleteErr = deleteErr
	uc := NewCleanupAttachmentObjectsUsecase(f.dispatcher, f.objectStorage)
	if _, err := uc.Execute(t.Context(), input); !errors.Is(err, deleteErr) {
		t.Errorf("後始末のエラー = %v、期待値 = %v", err, deleteErr)
	}
	if ae := model.AsAppError(f.execute(t, id)); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("削除済みの添付ファイルのエラー = %v", ae)
	}
	// リクエストからは見つからなくても、DBに残ったジョブだけで再試行できる。
	f.objectStorage.DeleteErr = nil
	f.cleanup(t)
	f.cleanup(t)
	if f.hasObject(key) || f.hasObject(variantKey) {
		t.Error("後始末を再試行しても実体が残っている")
	}
}

func TestDeleteAttachmentUsecase_最初の削除の後に作られたVariantも回収する(t *testing.T) {
	t.Parallel()
	f := setupDeleteAttachmentFixture(t, "late-variant")
	id, key := f.add(t, "target")
	var variantKey string
	objectStorage := &writeBackObjectStorage{FakeObjectStorage: f.objectStorage}
	objectStorage.onDelete = func(deleted string) {
		if deleted == key && variantKey == "" {
			variantKey = f.addVariant(t, id, "late")
		}
	}
	if err := f.usecaseWith(objectStorage).Execute(t.Context(), DeleteAttachmentInput{
		SpaceIdentifier: f.identifier, UserID: f.userID, AttachmentID: id,
	}); err != nil {
		t.Fatalf("削除に失敗: %v", err)
	}
	if variantKey == "" || !f.hasObject(variantKey) {
		t.Fatal("遅れて作られたvariantを再現できていない")
	}
	f.cleanup(t)
	if f.hasObject(variantKey) {
		t.Error("行を消したvariantの実体が残っている")
	}
	f.assertNoRows(t, id, key)
}

type failingTxJobClient struct {
	dispatcher.TxJobClient
	err error
}

func (c failingTxJobClient) InsertTx(ctx context.Context, tx *sql.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	if _, err := c.TxJobClient.InsertTx(ctx, tx, args, opts); err != nil {
		return nil, err
	}
	return nil, c.err
}

func TestDeleteAttachmentUsecase_後始末の登録に失敗したら行を残す(t *testing.T) {
	t.Parallel()
	f := setupDeleteAttachmentFixture(t, "enqueue-failure")
	id, key := f.add(t, "target")
	f.addVariant(t, id, "target")
	insertErr := errors.New("後始末を登録できなかった")
	f.dispatcher = dispatcher.NewDispatcher(nil, failingTxJobClient{TxJobClient: f.jobClient, err: insertErr})
	if err := f.execute(t, id); !errors.Is(err, insertErr) {
		t.Errorf("削除のエラー = %v、期待値 = %v", err, insertErr)
	}
	f.assertRowsRemain(t, id, key, map[string]int{"attachments": 1, "active_storage_blobs": 2, "active_storage_variant_records": 1})
	if len(f.cleanupJobs(t)) != 0 {
		t.Error("ロールバックした後始末ジョブが残っている")
	}
	// 実体が先に消えていても、残した行を使って削除をやり直せる。
	f.dispatcher = dispatcher.NewDispatcher(f.jobClient, f.jobClient)
	if err := f.execute(t, id); err != nil {
		t.Fatalf("削除の再試行に失敗: %v", err)
	}
	f.cleanup(t)
	f.assertNoRows(t, id, key)
}

func TestCleanupAttachmentObjectsUsecase_処理ジョブの終了を待つ(t *testing.T) {
	t.Parallel()
	f := setupDeleteAttachmentFixture(t, "wait-processing")
	id, key := f.add(t, "target")
	job, err := f.jobClient.Insert(t.Context(), dispatcher.ProcessAttachmentArgs{SpaceID: string(f.spaceID), AttachmentID: string(id)}, nil)
	if err != nil {
		t.Fatalf("処理ジョブの登録に失敗: %v", err)
	}
	if err := f.execute(t, id); err != nil {
		t.Fatalf("削除に失敗: %v", err)
	}
	input := cleanupInput(t, f.cleanupJobs(t)[0])
	uc := NewCleanupAttachmentObjectsUsecase(f.dispatcher, f.objectStorage)
	// スペースか添付ファイルが違う処理ジョブは、回収を妨げない。
	for _, args := range []dispatcher.ProcessAttachmentArgs{
		{SpaceID: "another-space", AttachmentID: string(id)},
		{SpaceID: string(f.spaceID), AttachmentID: "another-attachment"},
	} {
		if _, err := f.jobClient.Insert(t.Context(), args, nil); err != nil {
			t.Fatalf("別の処理ジョブの登録に失敗: %v", err)
		}
	}
	for _, state := range []rivertype.JobState{
		rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRetryable,
		rivertype.JobStateRunning, rivertype.JobStateScheduled,
	} {
		// 終了前の書き戻しを再現する。終了前に回収しても、その後また書き戻されうる。
		f.objectStorage.Put(key, []byte("written back"), "image/png")
		if _, err := f.db.ExecContext(t.Context(), "UPDATE "+f.jobSchema+".river_job SET state = $1 WHERE id = $2", state, job.Job.ID); err != nil {
			t.Fatalf("処理ジョブの状態の変更に失敗: %v", err)
		}
		wait, err := uc.Execute(t.Context(), input)
		if err != nil || wait <= 0 || !f.hasObject(key) {
			t.Errorf("状態 %s: 待機時間 = %v、エラー = %v、実体が残る = %v", state, wait, err, f.hasObject(key))
		}
	}
	for _, state := range []rivertype.JobState{rivertype.JobStateCompleted, rivertype.JobStateCancelled, rivertype.JobStateDiscarded} {
		f.objectStorage.Put(key, []byte("written back"), "image/png")
		if _, err := f.db.ExecContext(t.Context(), "UPDATE "+f.jobSchema+".river_job SET state = $1, finalized_at = NOW() WHERE id = $2", state, job.Job.ID); err != nil {
			t.Fatalf("処理ジョブの終了に失敗: %v", err)
		}
		f.cleanup(t)
		if f.hasObject(key) {
			t.Errorf("処理ジョブが%sになっても実体が残っている", state)
		}
	}
}
