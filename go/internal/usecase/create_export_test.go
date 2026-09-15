package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// failingJobInserterはどのジョブも受け付けない。エクスポートは記録されたのに、それを生成する
// ものが存在しない経路へテストが到達するための実装である。
type failingJobInserter struct{}

func (f *failingJobInserter) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return nil, errors.New("insert failed")
}

// createExportFixtureはエクスポート開始のテストに必要なスペース・メンバー・リポジトリ
type createExportFixture struct {
	db         *sql.DB
	identifier model.SpaceIdentifier
	spaceID    model.SpaceID
	memberID   model.SpaceMemberID
	userID     model.UserID
	exportRepo *repository.ExportRepository
}

// setupCreateExportFixtureは、すべてのスコープを持つメンバーが1人いるスペースを作成する。
// テストは同じデータベースに対して並行に走るため、一意性のある列をsuffixで区別する。
func setupCreateExportFixture(t *testing.T, suffix string) createExportFixture {
	t.Helper()

	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-export-" + suffix + "@example.com").
		WithAtname("create_export_" + suffix).
		Build()

	identifier := "create-export-" + suffix
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(identifier).
		Build()

	memberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	return createExportFixture{
		db:         db,
		identifier: model.SpaceIdentifier(identifier),
		spaceID:    spaceID,
		memberID:   memberID,
		userID:     userID,
		exportRepo: repository.NewExportRepository(query.New(db)),
	}
}

// newCreateExportUsecaseは、渡したジョブ投入器を使うUseCaseを組み立てる。
func newCreateExportUsecase(f createExportFixture, inserter dispatcher.JobInserter) *CreateExportUsecase {
	queries := query.New(f.db)
	return NewCreateExportUsecase(
		f.db,
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewExportRepository(queries),
		dispatcher.NewDispatcher(inserter),
	)
}

func TestCreateExportUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	t.Run("エクスポートを記録して生成ジョブを投入する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateExportFixture(t, "ok")
		inserter := &mockJobInserter{}
		uc := newCreateExportUsecase(f, inserter)

		output, err := uc.Execute(ctx, CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Export.Status != model.ExportStatusQueued {
			t.Errorf("Export.Status = %v、期待値 = %v", output.Export.Status, model.ExportStatusQueued)
		}
		if !inserter.called {
			t.Error("生成ジョブが投入されていません")
		}
		args, ok := inserter.args.(dispatcher.GenerateExportFilesArgs)
		if !ok {
			t.Fatalf("投入されたジョブの型 = %T、期待値 = GenerateExportFilesArgs", inserter.args)
		}
		if args.ExportID != output.Export.ID.String() {
			t.Errorf("args.ExportID = %q、期待値 = %q", args.ExportID, output.Export.ID.String())
		}
	})

	t.Run("実行中のエクスポートがあると開始を拒否する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateExportFixture(t, "running")
		testutil.NewExportBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.memberID).
			WithStatus(model.ExportStatusStarted).
			WithHeartbeatAt(time.Now()).
			Build()

		inserter := &mockJobInserter{}
		uc := newCreateExportUsecase(f, inserter)

		_, err := uc.Execute(ctx, CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID})
		appErr := model.AsAppError(err)
		if appErr == nil {
			t.Fatalf("Execute()のエラー = %v、期待値 = AppError", err)
		}
		if appErr.Code != model.AppErrCodeConflict {
			t.Errorf("appErr.Code = %v、期待値 = %v", appErr.Code, model.AppErrCodeConflict)
		}
		if inserter.called {
			t.Error("拒否したのに生成ジョブが投入されています")
		}
	})

	t.Run("キュー投入待ちのエクスポートがあると開始を拒否する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateExportFixture(t, "queued")
		testutil.NewExportBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.memberID).
			WithStatusChangedAt(time.Now()).
			Build()

		uc := newCreateExportUsecase(f, &mockJobInserter{})

		if _, err := uc.Execute(ctx, CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID}); model.AsAppError(err) == nil {
			t.Fatalf("Execute()のエラー = %v、期待値 = AppError", err)
		}
	})

	// ジョブがワーカーへ届かなかったqueuedは、それを解放するものが無いままスペースを
	// 保ち続ける。そのスペースは二度とエクスポートできなくなるため、ワーカーを待つ時間にも
	// heartbeatを待つ時間と同じように上限を置く。
	t.Run("拾われなかったqueuedを失敗にして新しいエクスポートを開始する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateExportFixture(t, "unclaimed")
		unclaimedID := testutil.NewExportBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.memberID).
			WithStatusChangedAt(time.Now().Add(-model.ExportQueuedStaleAfter - time.Minute)).
			Build()

		uc := newCreateExportUsecase(f, &mockJobInserter{})

		output, err := uc.Execute(ctx, CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Export.ID == unclaimedID {
			t.Error("拾われなかったエクスポートが再利用されています")
		}

		unclaimed, err := f.exportRepo.FindByIDAndSpace(ctx, unclaimedID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if unclaimed.Status != model.ExportStatusFailed {
			t.Errorf("unclaimed.Status = %v、期待値 = %v", unclaimed.Status, model.ExportStatusFailed)
		}
	})

	t.Run("止まったエクスポートを失敗にして新しいエクスポートを開始する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateExportFixture(t, "stale")
		staleID := testutil.NewExportBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.memberID).
			WithStatus(model.ExportStatusStarted).
			WithHeartbeatAt(time.Now().Add(-model.ExportHeartbeatStaleAfter - time.Minute)).
			Build()

		uc := newCreateExportUsecase(f, &mockJobInserter{})

		output, err := uc.Execute(ctx, CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Export.ID == staleID {
			t.Error("止まったエクスポートが再利用されています")
		}

		stale, err := f.exportRepo.FindByIDAndSpace(ctx, staleID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stale.Status != model.ExportStatusFailed {
			t.Errorf("stale.Status = %v、期待値 = %v", stale.Status, model.ExportStatusFailed)
		}
	})

	t.Run("エクスポート権限が無いメンバーは開始できない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateExportFixture(t, "reader")
		readerID := testutil.NewUserBuilderDB(t, f.db).
			WithEmail("create-export-reader-only@example.com").
			WithAtname("create_export_reader_only").
			Build()
		testutil.NewSpaceMemberBuilderDB(t, f.db).
			WithSpaceID(f.spaceID).
			WithUserID(readerID).
			WithScopes([]model.Scope{model.ScopePageRead}).
			Build()

		uc := newCreateExportUsecase(f, &mockJobInserter{})

		_, err := uc.Execute(ctx, CreateExportInput{SpaceIdentifier: f.identifier, UserID: readerID})
		appErr := model.AsAppError(err)
		if appErr == nil {
			t.Fatalf("Execute()のエラー = %v、期待値 = AppError", err)
		}
		if appErr.Code != model.AppErrCodeForbidden {
			t.Errorf("appErr.Code = %v、期待値 = %v", appErr.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("ジョブの投入に失敗したらエクスポートを残さない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateExportFixture(t, "enqueue-failure")
		uc := newCreateExportUsecase(f, &failingJobInserter{})

		if _, err := uc.Execute(ctx, CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID}); err == nil {
			t.Fatal("Execute()のエラー = nil、期待値 = エラー")
		}

		latest, err := f.exportRepo.FindLatestBySpace(ctx, f.spaceID)
		if err != nil {
			t.Fatalf("FindLatestBySpace()のエラー = %v", err)
		}
		if latest != nil {
			t.Errorf("FindLatestBySpace() = %v、期待値 = nil", latest)
		}
	})
}

// cancelingExportInserterはジョブ投入中のリクエストキャンセルを再現する。
type cancelingExportInserter struct{ cancel context.CancelFunc }

func (f cancelingExportInserter) Insert(context.Context, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	f.cancel()
	return nil, context.Canceled
}

func TestCreateExportUsecase_CanceledEnqueue(t *testing.T) {
	t.Parallel()
	f := setupCreateExportFixture(t, "cancel-enqueue")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uc := newCreateExportUsecase(f, cancelingExportInserter{cancel: cancel})
	_, err := uc.Execute(ctx, CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute()のエラー = %v、期待値 = context.Canceled", err)
	}
	latest, err := f.exportRepo.FindLatestBySpace(context.Background(), f.spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if latest != nil {
		t.Fatalf("取り残されたキュー投入済みのエクスポート: %v", latest)
	}
	if _, err := newCreateExportUsecase(f, &mockJobInserter{}).Execute(context.Background(), CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID}); err != nil {
		t.Fatalf("キャンセル後に再試行できない: %v", err)
	}
}

func TestCreateExportUsecase_ConcurrentStarts(t *testing.T) {
	t.Parallel()
	for _, previous := range []string{"none", "succeeded", "stale"} {
		caseName := "既存のエクスポートなし"
		switch previous {
		case "succeeded":
			caseName = "成功したエクスポートあり"
		case "stale":
			caseName = "停止したエクスポートあり"
		}
		t.Run(caseName, func(t *testing.T) {
			t.Parallel()
			f := setupCreateExportFixture(t, "concurrent-"+previous)
			if previous != "none" {
				builder := testutil.NewExportBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithQueuedByID(f.memberID).WithStatus(model.ExportStatusSucceeded)
				if previous == "stale" {
					builder.WithStatus(model.ExportStatusStarted).WithHeartbeatAt(time.Now().Add(-2 * model.ExportHeartbeatStaleAfter))
				}
				builder.Build()
			}
			const callers = 8
			start := make(chan struct{})
			results := make(chan error, callers)
			for range callers {
				go func() {
					<-start
					uc := newCreateExportUsecase(f, &mockJobInserter{})
					_, err := uc.Execute(context.Background(), CreateExportInput{SpaceIdentifier: f.identifier, UserID: f.userID})
					results <- err
				}()
			}
			close(start)
			succeeded := 0
			for range callers {
				err := <-results
				if err == nil {
					succeeded++
					continue
				}
				appErr := model.AsAppError(err)
				if appErr == nil || appErr.Code != model.AppErrCodeConflict {
					t.Errorf("予期しないエラー: %v", err)
				}
			}
			if succeeded != 1 {
				t.Fatalf("開始に成功した回数 = %d、期待値 = 1", succeeded)
			}
		})
	}
}
