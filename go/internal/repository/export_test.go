package repository

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// exportFixtureはどのエクスポートのテストでも必要になるスペースとスペースメンバー
type exportFixture struct {
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
}

// setupExportFixtureはスペースとそのメンバーを作成する。テストは並行に走るため、一意性の
// ある列 (メールアドレス・アットネーム・スペース識別子) をsuffixで区別する。
func setupExportFixture(t *testing.T, tx *sql.Tx, suffix string) exportFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("export-" + suffix + "@example.com").
		WithAtname("export_" + suffix).
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("export-" + suffix + "-space").
		WithName("Export " + suffix).
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	return exportFixture{spaceID: spaceID, spaceMemberID: spaceMemberID}
}

// laterExportIDは2つのうち、データベースが後ろに並べるほうのエクスポートIDを返す。
// Postgresはuuidをバイト列の順で比較し、正規形の小文字16進表記はそのバイト列と同じ順に
// 並ぶため、文字列の比較でデータベースと同じ答えが得られる。
func laterExportID(a, b model.ExportID) model.ExportID {
	if a > b {
		return a
	}
	return b
}

func TestExportRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "create")

	export, err := repo.Create(ctx, CreateExportInput{
		SpaceID:    f.spaceID,
		QueuedByID: f.spaceMemberID,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if export == nil {
		t.Fatal("Create()がnilを返した")
	}
	if export.ID == "" {
		t.Error("export.IDが空")
	}
	if export.SpaceID != f.spaceID {
		t.Errorf("export.SpaceID = %v、期待値 = %v", export.SpaceID, f.spaceID)
	}
	if export.QueuedByID != f.spaceMemberID {
		t.Errorf("export.QueuedByID = %v、期待値 = %v", export.QueuedByID, f.spaceMemberID)
	}
	if export.Status != model.ExportStatusQueued {
		t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusQueued)
	}
	if export.StatusChangedAt.IsZero() {
		t.Error("export.StatusChangedAtがゼロ値")
	}
	if export.HeartbeatAt != nil {
		t.Errorf("export.HeartbeatAt = %v、期待値 = nil", export.HeartbeatAt)
	}
	if export.ObjectKey != nil {
		t.Errorf("export.ObjectKey = %v、期待値 = nil", *export.ObjectKey)
	}
}

func TestExportRepository_FindByIDAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "find")
	other := setupExportFixture(t, tx, "find-other")

	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithObjectKey("exports/found.zip").
		Build()

	t.Run("スペース内のエクスポートを取得できる", func(t *testing.T) {
		export, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("FindByIDAndSpace()がnilを返した")
		}
		if export.ID != exportID {
			t.Errorf("export.ID = %v、期待値 = %v", export.ID, exportID)
		}
		if export.Status != model.ExportStatusSucceeded {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusSucceeded)
		}
		if export.ObjectKey == nil || *export.ObjectKey != "exports/found.zip" {
			t.Errorf("export.ObjectKey = %v、期待値 = exports/found.zip", export.ObjectKey)
		}
	})

	t.Run("別のスペースのエクスポートは取得できない", func(t *testing.T) {
		export, err := repo.FindByIDAndSpace(ctx, exportID, other.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("FindByIDAndSpace() = %v、期待値 = nil", export)
		}
	})

	t.Run("UUIDでないIDはエラーにせず見つからないとして扱う", func(t *testing.T) {
		export, err := repo.FindByIDAndSpace(ctx, model.ExportID("not-a-uuid"), f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("FindByIDAndSpace() = %v、期待値 = nil", export)
		}
	})
}

func TestExportRepository_FindLatestBySpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "latest")
	empty := setupExportFixture(t, tx, "latest-empty")

	now := time.Now()
	testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithCreatedAt(now.Add(-2 * time.Hour)).
		Build()
	latestID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithCreatedAt(now).
		Build()
	testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithCreatedAt(now.Add(-1 * time.Hour)).
		Build()

	t.Run("最新のエクスポートを返す", func(t *testing.T) {
		export, err := repo.FindLatestBySpace(ctx, f.spaceID)
		if err != nil {
			t.Fatalf("FindLatestBySpace()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("FindLatestBySpace()がnilを返した")
		}
		if export.ID != latestID {
			t.Errorf("export.ID = %v、期待値 = %v", export.ID, latestID)
		}
	})

	t.Run("エクスポートが無いスペースではnilを返す", func(t *testing.T) {
		export, err := repo.FindLatestBySpace(ctx, empty.spaceID)
		if err != nil {
			t.Fatalf("FindLatestBySpace()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("FindLatestBySpace() = %v、期待値 = nil", export)
		}
	})

	// created_atが並んだときはidで決着する。これはListOlderExportsBySpaceが比較する
	// キーと同じである。タイブレークが無いと、以下の2件はどちらが返ってもよいことになり、
	// 画面に出るエクスポートが実行計画次第で変わる。
	t.Run("created_atが同じときはidの大きいエクスポートを返す", func(t *testing.T) {
		tie := setupExportFixture(t, tx, "latest-tie")
		sameTime := now.Add(-3 * time.Hour)

		firstID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(tie.spaceID).
			WithQueuedByID(tie.spaceMemberID).
			WithCreatedAt(sameTime).
			Build()
		secondID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(tie.spaceID).
			WithQueuedByID(tie.spaceMemberID).
			WithCreatedAt(sameTime).
			Build()
		wantID := laterExportID(firstID, secondID)

		export, err := repo.FindLatestBySpace(ctx, tie.spaceID)
		if err != nil {
			t.Fatalf("FindLatestBySpace()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("FindLatestBySpace()がnilを返した")
		}
		if export.ID != wantID {
			t.Errorf("export.ID = %v、期待値 = %v", export.ID, wantID)
		}
	})
}

func TestExportRepository_ListOlderBySpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "list")
	other := setupExportFixture(t, tx, "list-other")

	now := time.Now()
	oldestID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithCreatedAt(now.Add(-2 * time.Hour)).
		Build()
	middleID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithCreatedAt(now.Add(-1 * time.Hour)).
		Build()
	currentID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithCreatedAt(now).
		Build()
	newerID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithCreatedAt(now.Add(1 * time.Hour)).
		Build()
	otherSpaceExportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(other.spaceID).
		WithQueuedByID(other.spaceMemberID).
		WithCreatedAt(now.Add(-3 * time.Hour)).
		Build()

	t.Run("基準より古い同じスペースのエクスポートを古い順に返す", func(t *testing.T) {
		exports, err := repo.ListOlderBySpace(ctx, f.spaceID, currentID)
		if err != nil {
			t.Fatalf("ListOlderBySpace()のエラー = %v", err)
		}
		if len(exports) != 2 {
			t.Fatalf("len(exports) = %d、期待値 = 2", len(exports))
		}
		if exports[0].ID != oldestID {
			t.Errorf("exports[0].ID = %v、期待値 = %v", exports[0].ID, oldestID)
		}
		if exports[1].ID != middleID {
			t.Errorf("exports[1].ID = %v、期待値 = %v", exports[1].ID, middleID)
		}
		for _, export := range exports {
			if export.ID == newerID {
				t.Errorf("ListOlderBySpace()の結果に新しいエクスポート%vが含まれている", newerID)
			}
		}
	})

	t.Run("別スペースのエクスポートを基準にすると何も返さない", func(t *testing.T) {
		exports, err := repo.ListOlderBySpace(ctx, f.spaceID, otherSpaceExportID)
		if err != nil {
			t.Fatalf("ListOlderBySpace()のエラー = %v", err)
		}
		if len(exports) != 0 {
			t.Errorf("len(exports) = %d、期待値 = 0", len(exports))
		}
	})

	t.Run("別スペースを指定すると自分のスペースのエクスポートは返さない", func(t *testing.T) {
		exports, err := repo.ListOlderBySpace(ctx, other.spaceID, currentID)
		if err != nil {
			t.Fatalf("ListOlderBySpace()のエラー = %v", err)
		}
		if len(exports) != 0 {
			t.Errorf("len(exports) = %d、期待値 = 0", len(exports))
		}
	})

	// created_atが並んだときの順序はidで決まる。FindLatestBySpaceと同じキーで比較して
	// いることが、復帰した古いワーカーに後継を削除させないという保証の根拠になっている。
	t.Run("created_atが同じときはidで古い順が決まる", func(t *testing.T) {
		tie := setupExportFixture(t, tx, "list-tie")
		sameTime := now.Add(-4 * time.Hour)

		firstID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(tie.spaceID).
			WithQueuedByID(tie.spaceMemberID).
			WithCreatedAt(sameTime).
			Build()
		secondID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(tie.spaceID).
			WithQueuedByID(tie.spaceMemberID).
			WithCreatedAt(sameTime).
			Build()
		laterID := laterExportID(firstID, secondID)
		earlierID := firstID
		if laterID == firstID {
			earlierID = secondID
		}

		exports, err := repo.ListOlderBySpace(ctx, tie.spaceID, laterID)
		if err != nil {
			t.Fatalf("ListOlderBySpace()のエラー = %v", err)
		}
		if len(exports) != 1 {
			t.Fatalf("len(exports) = %d、期待値 = 1", len(exports))
		}
		if exports[0].ID != earlierID {
			t.Errorf("exports[0].ID = %v、期待値 = %v", exports[0].ID, earlierID)
		}

		exports, err = repo.ListOlderBySpace(ctx, tie.spaceID, earlierID)
		if err != nil {
			t.Fatalf("ListOlderBySpace()のエラー = %v", err)
		}
		if len(exports) != 0 {
			t.Errorf("len(exports) = %d、期待値 = 0", len(exports))
		}
	})
}

func TestExportRepository_MarkStarted(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "start")
	other := setupExportFixture(t, tx, "start-other")

	t.Run("queuedから開始できheartbeatが打たれる", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusQueued).
			Build()

		export, err := repo.MarkStarted(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("MarkStarted()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("MarkStarted()がnilを返した")
		}
		if export.Status != model.ExportStatusStarted {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusStarted)
		}
		if export.HeartbeatAt == nil {
			t.Error("export.HeartbeatAtがnil")
		}
	})

	t.Run("リトライでstartedからも開始できheartbeatが更新される", func(t *testing.T) {
		staleHeartbeat := time.Now().Add(-1 * time.Hour)
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			WithHeartbeatAt(staleHeartbeat).
			Build()

		export, err := repo.MarkStarted(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("MarkStarted()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("MarkStarted()がnilを返した")
		}
		if export.HeartbeatAt == nil {
			t.Fatal("export.HeartbeatAtがnil")
		}
		if !export.HeartbeatAt.After(staleHeartbeat) {
			t.Errorf("export.HeartbeatAt = %v、期待値 = %vより後", *export.HeartbeatAt, staleHeartbeat)
		}
	})

	t.Run("完了したエクスポートは巻き戻さない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusSucceeded).
			Build()

		export, err := repo.MarkStarted(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("MarkStarted()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkStarted() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusSucceeded {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusSucceeded)
		}
	})

	t.Run("別スペースの指定では開始しない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusQueued).
			Build()

		export, err := repo.MarkStarted(ctx, exportID, other.spaceID)
		if err != nil {
			t.Fatalf("MarkStarted()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkStarted() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusQueued {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusQueued)
		}
	})
}

func TestExportRepository_MarkSucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "succeed")
	other := setupExportFixture(t, tx, "succeed-other")

	t.Run("startedから成功へ進みZIPの位置を記録する", func(t *testing.T) {
		heartbeat := time.Now().Add(-1 * time.Minute)
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			WithHeartbeatAt(heartbeat).
			Build()

		export, err := repo.MarkSucceeded(ctx, exportID, f.spaceID, "exports/succeeded.zip")
		if err != nil {
			t.Fatalf("MarkSucceeded()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("MarkSucceeded()がnilを返した")
		}
		if export.Status != model.ExportStatusSucceeded {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusSucceeded)
		}
		if export.ObjectKey == nil || *export.ObjectKey != "exports/succeeded.zip" {
			t.Errorf("export.ObjectKey = %v、期待値 = exports/succeeded.zip", export.ObjectKey)
		}
		if export.HeartbeatAt == nil {
			t.Error("export.HeartbeatAtがnil、期待値 = ハートビートが残っている")
		}
	})

	t.Run("queuedからは成功へ進めない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusQueued).
			Build()

		export, err := repo.MarkSucceeded(ctx, exportID, f.spaceID, "exports/ignored.zip")
		if err != nil {
			t.Fatalf("MarkSucceeded()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkSucceeded() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusQueued {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusQueued)
		}
		if stored.ObjectKey != nil {
			t.Errorf("stored.ObjectKey = %v、期待値 = nil", *stored.ObjectKey)
		}
	})

	t.Run("別スペースの指定では成功にしない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			Build()

		export, err := repo.MarkSucceeded(ctx, exportID, other.spaceID, "exports/other.zip")
		if err != nil {
			t.Fatalf("MarkSucceeded()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkSucceeded() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusStarted {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusStarted)
		}
		if stored.ObjectKey != nil {
			t.Errorf("stored.ObjectKey = %v、期待値 = nil", *stored.ObjectKey)
		}
	})
}

func TestExportRepository_MarkFailed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "fail")
	other := setupExportFixture(t, tx, "fail-other")

	t.Run("startedから失敗へ進む", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			Build()

		export, err := repo.MarkFailed(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("MarkFailed()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailed()がnilを返した")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusFailed)
		}
	})

	t.Run("開始を記録できなかった試行のためqueuedからも失敗へ進む", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusQueued).
			Build()

		export, err := repo.MarkFailed(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("MarkFailed()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailed()がnilを返した")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusFailed)
		}
	})

	t.Run("成功したエクスポートは失敗へ進めない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusSucceeded).
			WithObjectKey("exports/kept.zip").
			Build()

		export, err := repo.MarkFailed(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("MarkFailed()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailed() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusSucceeded {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusSucceeded)
		}
	})

	t.Run("別スペースの指定では失敗にしない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			Build()

		export, err := repo.MarkFailed(ctx, exportID, other.spaceID)
		if err != nil {
			t.Fatalf("MarkFailed()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailed() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusStarted {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusStarted)
		}
	})
}

func TestExportRepository_UpdateHeartbeat(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "heartbeat")
	other := setupExportFixture(t, tx, "heartbeat-other")

	t.Run("処理中のエクスポートのheartbeatを更新する", func(t *testing.T) {
		staleHeartbeat := time.Now().Add(-1 * time.Hour)
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			WithHeartbeatAt(staleHeartbeat).
			Build()

		updated, err := repo.UpdateHeartbeat(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("UpdateHeartbeat()のエラー = %v", err)
		}
		if !updated {
			t.Error("UpdateHeartbeat() = false、期待値 = true")
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.HeartbeatAt == nil {
			t.Fatal("stored.HeartbeatAtがnil")
		}
		if !stored.HeartbeatAt.After(staleHeartbeat) {
			t.Errorf("stored.HeartbeatAt = %v、期待値 = %vより後", *stored.HeartbeatAt, staleHeartbeat)
		}
	})

	t.Run("完了したエクスポートのheartbeatは更新しない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusFailed).
			Build()

		updated, err := repo.UpdateHeartbeat(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("UpdateHeartbeat()のエラー = %v", err)
		}
		if updated {
			t.Error("UpdateHeartbeat() = true、期待値 = false")
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.HeartbeatAt != nil {
			t.Errorf("stored.HeartbeatAt = %v、期待値 = nil", *stored.HeartbeatAt)
		}
	})

	t.Run("別スペースの指定ではheartbeatを更新しない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			Build()

		updated, err := repo.UpdateHeartbeat(ctx, exportID, other.spaceID)
		if err != nil {
			t.Fatalf("UpdateHeartbeat()のエラー = %v", err)
		}
		if updated {
			t.Error("UpdateHeartbeat() = true、期待値 = false")
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.HeartbeatAt != nil {
			t.Errorf("stored.HeartbeatAt = %v、期待値 = nil", *stored.HeartbeatAt)
		}
	})
}

func TestExportRepository_Delete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "delete")
	other := setupExportFixture(t, tx, "delete-other")

	t.Run("エクスポートを削除する", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusSucceeded).
			WithObjectKey("exports/deleted.zip").
			Build()

		if err := repo.Delete(ctx, exportID, f.spaceID); err != nil {
			t.Fatalf("Delete()のエラー = %v", err)
		}

		export, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("FindByIDAndSpace() = %v、期待値 = nil", export)
		}
	})

	t.Run("別スペースの指定では削除しない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusSucceeded).
			WithObjectKey("exports/kept.zip").
			Build()

		if err := repo.Delete(ctx, exportID, other.spaceID); err != nil {
			t.Fatalf("Delete()のエラー = %v", err)
		}

		export, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("FindByIDAndSpace()がnilを返した、期待値 = エクスポートが残っている")
		}
	})
}

func TestExportRepository_MarkFailedIfStale(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "stale")
	staleBefore := time.Now().Add(-5 * time.Minute)

	t.Run("heartbeatが古いstartedを失敗へ進める", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			WithHeartbeatAt(staleBefore.Add(-time.Minute)).
			Build()

		export, err := repo.MarkFailedIfStale(ctx, exportID, f.spaceID, staleBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfStale()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailedIfStale()がnilを返した")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusFailed)
		}
	})

	t.Run("heartbeatが新しいstartedには手を触れない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			WithHeartbeatAt(time.Now()).
			Build()

		export, err := repo.MarkFailedIfStale(ctx, exportID, f.spaceID, staleBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfStale()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfStale() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusStarted {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusStarted)
		}
	})

	t.Run("heartbeatを持たないstartedは止まったものとして扱う", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			Build()

		export, err := repo.MarkFailedIfStale(ctx, exportID, f.spaceID, staleBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfStale()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailedIfStale()がnilを返した")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusFailed)
		}
	})

	t.Run("queuedのエクスポートは失敗へ進めない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			Build()

		export, err := repo.MarkFailedIfStale(ctx, exportID, f.spaceID, staleBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfStale()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfStale() = %v、期待値 = nil", export)
		}
	})
}

func TestExportRepository_MarkFailedIfUnclaimed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "unclaimed")
	other := setupExportFixture(t, tx, "unclaimed-other")
	unclaimedBefore := time.Now().Add(-model.ExportQueuedStaleAfter)

	t.Run("拾われないまま閾値を過ぎたqueuedを失敗へ進める", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatusChangedAt(unclaimedBefore.Add(-time.Minute)).
			Build()

		export, err := repo.MarkFailedIfUnclaimed(ctx, exportID, f.spaceID, unclaimedBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfUnclaimed()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailedIfUnclaimed()がnilを返した")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusFailed)
		}
	})

	t.Run("まだワーカーを待っているqueuedには手を触れない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatusChangedAt(time.Now()).
			Build()

		export, err := repo.MarkFailedIfUnclaimed(ctx, exportID, f.spaceID, unclaimedBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfUnclaimed()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfUnclaimed() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusQueued {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusQueued)
		}
	})

	t.Run("ジョブがワーカーへ届いたstartedには手を触れない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusStarted).
			WithStatusChangedAt(unclaimedBefore.Add(-time.Minute)).
			WithHeartbeatAt(time.Now()).
			Build()

		export, err := repo.MarkFailedIfUnclaimed(ctx, exportID, f.spaceID, unclaimedBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfUnclaimed()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfUnclaimed() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusStarted {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusStarted)
		}
	})

	t.Run("別スペースの指定では失敗にしない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatusChangedAt(unclaimedBefore.Add(-time.Minute)).
			Build()

		export, err := repo.MarkFailedIfUnclaimed(ctx, exportID, other.spaceID, unclaimedBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfUnclaimed()のエラー = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfUnclaimed() = %v、期待値 = nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if stored.Status != model.ExportStatusQueued {
			t.Errorf("stored.Status = %v、期待値 = %v", stored.Status, model.ExportStatusQueued)
		}
	})
}

// 親の削除に使うエクスポートの全FKがON DELETE CASCADEを保つことを検証する。
// 下の振る舞いテストはRailsの削除順に従い、スペースを削除する前にqueued_by_id経由で
// exportsを削除するため、space_idのFKを実行できない。
func TestExportRepository_ForeignKeysUseCascadeDelete(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	ctx := context.Background()

	constraintNames := []string{
		"fk_rails_703ee3dae6",
		"fk_rails_7fa4a1a0c0",
	}
	rows, err := db.QueryContext(ctx, `
		SELECT constraint_name, delete_rule
		FROM information_schema.referential_constraints
		WHERE constraint_schema = current_schema()
		  AND constraint_name = ANY($1)
	`, pq.Array(constraintNames))
	if err != nil {
		t.Fatalf("外部キーの削除規則の取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	found := make(map[string]bool, len(constraintNames))
	for rows.Next() {
		var constraintName, deleteRule string
		if err := rows.Scan(&constraintName, &deleteRule); err != nil {
			t.Fatalf("外部キーの削除規則の読み取りに失敗: %v", err)
		}
		found[constraintName] = true
		if deleteRule != "CASCADE" {
			t.Errorf("%sのdelete_rule = %q、期待値 = CASCADE", constraintName, deleteRule)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("外部キーの削除規則の走査に失敗: %v", err)
	}

	for _, constraintName := range constraintNames {
		if !found[constraintName] {
			t.Errorf("外部キー%sが見つかりません", constraintName)
		}
	}
}

// Rails側のスペース削除が頼るON DELETE CASCADEの契約を検証する。exportsは
// Rails版が知らないGo側の行であるため、スペースとそのメンバーを削除したとき、
// exportsへの明示的なDELETEなしで一緒に消える必要がある。
func TestExportRepository_CascadeOnSpaceDelete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	f := setupExportFixture(t, tx, "cascade")

	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.spaceMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithObjectKey("exports/cascade.zip").
		Build()

	// Rails版と同じ順序で削除する。先にメンバー、次にスペース。
	if _, err := tx.ExecContext(
		ctx, `DELETE FROM space_members WHERE space_id = $1`, string(f.spaceID),
	); err != nil {
		t.Fatalf("space_membersの削除に失敗: %v", err)
	}
	if _, err := tx.ExecContext(
		ctx, `DELETE FROM spaces WHERE id = $1`, string(f.spaceID),
	); err != nil {
		t.Fatalf("spacesの削除に失敗: %v", err)
	}

	var exportCount int
	if err := tx.QueryRowContext(
		ctx, `SELECT COUNT(*) FROM exports WHERE id = $1`, string(exportID),
	).Scan(&exportCount); err != nil {
		t.Fatalf("exportsの件数取得に失敗: %v", err)
	}
	if exportCount != 0 {
		t.Errorf("exportsの件数 = %d、期待値 = 0", exportCount)
	}
}

// insertLegacyExportBlobは、Rails版のエクスポートのZIPのblobを作り、IDを返す
func insertLegacyExportBlob(t *testing.T, tx *sql.Tx, key string) string {
	t.Helper()
	var blobID string
	if err := tx.QueryRowContext(context.Background(), `
		INSERT INTO active_storage_blobs (key, filename, content_type, service_name, byte_size, created_at)
		VALUES ($1, 'old.zip', 'application/zip', 'local', 3, NOW()) RETURNING id`, key).Scan(&blobID); err != nil {
		t.Fatalf("blobの作成に失敗: %v", err)
	}
	return blobID
}

// attachLegacyExportBlobは、Railsのhas_one_attached :fileと同じ形で、blobをエクスポートに関連付ける
func attachLegacyExportBlob(t *testing.T, tx *sql.Tx, exportID model.ExportID, blobID string) {
	t.Helper()
	if _, err := tx.ExecContext(context.Background(), `
		INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		VALUES ('file', 'ExportRecord', $1, $2, NOW())`, string(exportID), blobID); err != nil {
		t.Fatalf("blobの関連付けに失敗: %v", err)
	}
}

func TestExportRepository_ObjectsBySpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewExportRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	f := setupExportFixture(t, tx, "by-space")
	other := setupExportFixture(t, tx, "by-space-other")
	newExport := func(fx exportFixture, objectKey string) model.ExportID {
		b := testutil.NewExportBuilder(t, tx).WithSpaceID(fx.spaceID).WithQueuedByID(fx.spaceMemberID).WithStatus(model.ExportStatusSucceeded)
		if objectKey != "" {
			b = b.WithObjectKey(objectKey)
		}
		return b.Build()
	}

	newExport(f, "exports/by-space.zip")
	legacyExportID := newExport(f, "")
	attachLegacyExportBlob(t, tx, legacyExportID, insertLegacyExportBlob(t, tx, "legacy-by-space"))
	// ほかのスペースのエクスポートも参照しているZIPは、実体を残す
	sharedBlobID := insertLegacyExportBlob(t, tx, "legacy-by-space-shared")
	attachLegacyExportBlob(t, tx, legacyExportID, sharedBlobID)
	otherExportID := newExport(other, "exports/by-space-other.zip")
	attachLegacyExportBlob(t, tx, otherExportID, sharedBlobID)

	var exports []*model.Export
	var after model.ExportID
	for {
		batch, err := repo.ListForDeletionBySpaceAfter(ctx, f.spaceID, after, 1)
		if err != nil {
			t.Fatalf("エクスポートの取得に失敗: %v", err)
		}
		if len(batch) == 0 {
			break
		}
		if len(batch) != 1 || batch[0].SpaceID != f.spaceID || batch[0].ID <= after {
			t.Fatalf("取得したバッチ = %+v、1件ずつ対象スペースのIDの昇順を期待", batch)
		}
		exports = append(exports, batch...)
		after = batch[0].ID
	}
	if len(exports) != 2 {
		t.Fatalf("エクスポートの数 = %d、期待値 = 2 (キーが未記録の行も含む)", len(exports))
	}
	var recordedKeys []string
	for _, export := range exports {
		if export.ObjectKey != nil {
			recordedKeys = append(recordedKeys, *export.ObjectKey)
		}
	}
	if !slices.Equal(recordedKeys, []string{"exports/by-space.zip"}) {
		t.Errorf("記録されたキー = %v", recordedKeys)
	}

	var candidates []model.LegacyExportBlob
	var afterBlob string
	for {
		batch, err := repo.ListLegacyBlobsBySpaceAfter(ctx, f.spaceID, afterBlob, 1)
		if err != nil {
			t.Fatalf("ZIPのblobの取得に失敗: %v", err)
		}
		if len(batch) == 0 {
			break
		}
		if len(batch) != 1 || batch[0].ID <= afterBlob {
			t.Fatalf("ZIPのバッチ = %+v、1件ずつIDの昇順を期待", batch)
		}
		candidates = append(candidates, batch...)
		afterBlob = batch[0].ID
	}
	if len(candidates) != 2 {
		t.Fatalf("ZIPのblobの数 = %d、期待値 = 2 (共有中も含む)", len(candidates))
	}
	for _, blob := range candidates {
		if blob.Shared != (blob.Key == "legacy-by-space-shared") {
			t.Errorf("ZIPの共有状態 = %+v", blob)
		}
	}

	if err := repo.DeleteLegacyFilesBySpace(ctx, f.spaceID); err != nil {
		t.Fatalf("DeleteLegacyFilesBySpace()のエラー = %v", err)
	}
	// 関連の削除後に参照を再確認する。blobを確保したあとの再試行でもキーを失わない。
	for range 2 {
		keys, err := repo.ClaimUnreferencedLegacyBlobs(ctx, candidates)
		if err != nil || !slices.Equal(keys, []string{"legacy-by-space"}) {
			t.Fatalf("回収してよいキー = %v、エラー = %v", keys, err)
		}
	}

	var blobs []string
	rows, err := tx.QueryContext(ctx, `SELECT key FROM active_storage_blobs WHERE key LIKE 'legacy-by-space%' ORDER BY key`)
	if err != nil {
		t.Fatalf("blobの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			t.Fatalf("blobの読み取りに失敗: %v", err)
		}
		blobs = append(blobs, key)
	}
	if !slices.Equal(blobs, []string{"legacy-by-space-shared"}) {
		t.Errorf("残ったblob = %v、期待値 = [legacy-by-space-shared]", blobs)
	}

	var attachments int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM active_storage_attachments
		WHERE record_type = 'ExportRecord' AND record_id = ANY($1::uuid[])`,
		"{"+string(legacyExportID)+","+string(otherExportID)+"}").Scan(&attachments); err != nil {
		t.Fatalf("関連の数の取得に失敗: %v", err)
	}
	if attachments != 1 {
		t.Errorf("残った関連の数 = %d、期待値 = 1 (ほかのスペースのエクスポートの関連だけ)", attachments)
	}
}
