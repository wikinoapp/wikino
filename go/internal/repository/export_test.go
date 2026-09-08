package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// exportFixture is the space and space member every export test needs.
//
// [Ja] exportFixture はどのエクスポートのテストでも必要になるスペースとスペースメンバー
type exportFixture struct {
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
}

// setupExportFixture creates a space and a member of it. The suffix keeps the unique columns
// (email, atname, space identifier) apart between the tests, which run in parallel.
//
// [Ja] setupExportFixture はスペースとそのメンバーを作成する。テストは並行に走るため、一意性の
// ある列 (メールアドレス・アットネーム・スペース識別子) を suffix で区別する。
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

// laterExportID returns the export ID that the database orders last of the two. Postgres compares
// uuid values by their bytes, and the canonical lowercase hex text puts those bytes in the same
// order, so comparing the strings answers the same as the database does.
//
// [Ja] laterExportID は 2 つのうち、データベースが後ろに並べるほうのエクスポート ID を返す。
// Postgres は uuid をバイト列の順で比較し、正規形の小文字 16 進表記はそのバイト列と同じ順に
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
		t.Fatalf("Create() error = %v", err)
	}
	if export == nil {
		t.Fatal("Create() returned nil")
	}
	if export.ID == "" {
		t.Error("export.ID is empty")
	}
	if export.SpaceID != f.spaceID {
		t.Errorf("export.SpaceID = %v, want %v", export.SpaceID, f.spaceID)
	}
	if export.QueuedByID != f.spaceMemberID {
		t.Errorf("export.QueuedByID = %v, want %v", export.QueuedByID, f.spaceMemberID)
	}
	if export.Status != model.ExportStatusQueued {
		t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusQueued)
	}
	if export.StatusChangedAt.IsZero() {
		t.Error("export.StatusChangedAt is zero")
	}
	if export.HeartbeatAt != nil {
		t.Errorf("export.HeartbeatAt = %v, want nil", export.HeartbeatAt)
	}
	if export.ObjectKey != nil {
		t.Errorf("export.ObjectKey = %v, want nil", *export.ObjectKey)
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
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export == nil {
			t.Fatal("FindByIDAndSpace() returned nil")
		}
		if export.ID != exportID {
			t.Errorf("export.ID = %v, want %v", export.ID, exportID)
		}
		if export.Status != model.ExportStatusSucceeded {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusSucceeded)
		}
		if export.ObjectKey == nil || *export.ObjectKey != "exports/found.zip" {
			t.Errorf("export.ObjectKey = %v, want exports/found.zip", export.ObjectKey)
		}
	})

	t.Run("別のスペースのエクスポートは取得できない", func(t *testing.T) {
		export, err := repo.FindByIDAndSpace(ctx, exportID, other.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export != nil {
			t.Errorf("FindByIDAndSpace() = %v, want nil", export)
		}
	})

	t.Run("UUIDでないIDはエラーにせず見つからないとして扱う", func(t *testing.T) {
		export, err := repo.FindByIDAndSpace(ctx, model.ExportID("not-a-uuid"), f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export != nil {
			t.Errorf("FindByIDAndSpace() = %v, want nil", export)
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
			t.Fatalf("FindLatestBySpace() error = %v", err)
		}
		if export == nil {
			t.Fatal("FindLatestBySpace() returned nil")
		}
		if export.ID != latestID {
			t.Errorf("export.ID = %v, want %v", export.ID, latestID)
		}
	})

	t.Run("エクスポートが無いスペースではnilを返す", func(t *testing.T) {
		export, err := repo.FindLatestBySpace(ctx, empty.spaceID)
		if err != nil {
			t.Fatalf("FindLatestBySpace() error = %v", err)
		}
		if export != nil {
			t.Errorf("FindLatestBySpace() = %v, want nil", export)
		}
	})

	// The tie on created_at is broken by id, the same key ListOlderExportsBySpace compares.
	// Without a tie-break the two exports below would be interchangeable, and which one the
	// screen shows would depend on the plan.
	//
	// [Ja] created_at が並んだときは id で決着する。これは ListOlderExportsBySpace が比較する
	// キーと同じである。タイブレークが無いと、以下の 2 件はどちらが返ってもよいことになり、
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
			t.Fatalf("FindLatestBySpace() error = %v", err)
		}
		if export == nil {
			t.Fatal("FindLatestBySpace() returned nil")
		}
		if export.ID != wantID {
			t.Errorf("export.ID = %v, want %v", export.ID, wantID)
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
			t.Fatalf("ListOlderBySpace() error = %v", err)
		}
		if len(exports) != 2 {
			t.Fatalf("len(exports) = %d, want 2", len(exports))
		}
		if exports[0].ID != oldestID {
			t.Errorf("exports[0].ID = %v, want %v", exports[0].ID, oldestID)
		}
		if exports[1].ID != middleID {
			t.Errorf("exports[1].ID = %v, want %v", exports[1].ID, middleID)
		}
		for _, export := range exports {
			if export.ID == newerID {
				t.Errorf("ListOlderBySpace() included newer export %v", newerID)
			}
		}
	})

	t.Run("別スペースのエクスポートを基準にすると何も返さない", func(t *testing.T) {
		exports, err := repo.ListOlderBySpace(ctx, f.spaceID, otherSpaceExportID)
		if err != nil {
			t.Fatalf("ListOlderBySpace() error = %v", err)
		}
		if len(exports) != 0 {
			t.Errorf("len(exports) = %d, want 0", len(exports))
		}
	})

	t.Run("別スペースを指定すると自分のスペースのエクスポートは返さない", func(t *testing.T) {
		exports, err := repo.ListOlderBySpace(ctx, other.spaceID, currentID)
		if err != nil {
			t.Fatalf("ListOlderBySpace() error = %v", err)
		}
		if len(exports) != 0 {
			t.Errorf("len(exports) = %d, want 0", len(exports))
		}
	})

	// The order of exports that share a created_at is decided by id. Comparing the same key as
	// FindLatestBySpace is what backs the promise that an older worker coming back to life does
	// not delete the export that replaced it.
	//
	// [Ja] created_at が並んだときの順序は id で決まる。FindLatestBySpace と同じキーで比較して
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
			t.Fatalf("ListOlderBySpace() error = %v", err)
		}
		if len(exports) != 1 {
			t.Fatalf("len(exports) = %d, want 1", len(exports))
		}
		if exports[0].ID != earlierID {
			t.Errorf("exports[0].ID = %v, want %v", exports[0].ID, earlierID)
		}

		exports, err = repo.ListOlderBySpace(ctx, tie.spaceID, earlierID)
		if err != nil {
			t.Fatalf("ListOlderBySpace() error = %v", err)
		}
		if len(exports) != 0 {
			t.Errorf("len(exports) = %d, want 0", len(exports))
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
			t.Fatalf("MarkStarted() error = %v", err)
		}
		if export == nil {
			t.Fatal("MarkStarted() returned nil")
		}
		if export.Status != model.ExportStatusStarted {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusStarted)
		}
		if export.HeartbeatAt == nil {
			t.Error("export.HeartbeatAt is nil")
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
			t.Fatalf("MarkStarted() error = %v", err)
		}
		if export == nil {
			t.Fatal("MarkStarted() returned nil")
		}
		if export.HeartbeatAt == nil {
			t.Fatal("export.HeartbeatAt is nil")
		}
		if !export.HeartbeatAt.After(staleHeartbeat) {
			t.Errorf("export.HeartbeatAt = %v, want after %v", *export.HeartbeatAt, staleHeartbeat)
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
			t.Fatalf("MarkStarted() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkStarted() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusSucceeded {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusSucceeded)
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
			t.Fatalf("MarkStarted() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkStarted() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusQueued {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusQueued)
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
			t.Fatalf("MarkSucceeded() error = %v", err)
		}
		if export == nil {
			t.Fatal("MarkSucceeded() returned nil")
		}
		if export.Status != model.ExportStatusSucceeded {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusSucceeded)
		}
		if export.ObjectKey == nil || *export.ObjectKey != "exports/succeeded.zip" {
			t.Errorf("export.ObjectKey = %v, want exports/succeeded.zip", export.ObjectKey)
		}
		if export.HeartbeatAt == nil {
			t.Error("export.HeartbeatAt is nil, want the heartbeat to be kept")
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
			t.Fatalf("MarkSucceeded() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkSucceeded() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusQueued {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusQueued)
		}
		if stored.ObjectKey != nil {
			t.Errorf("stored.ObjectKey = %v, want nil", *stored.ObjectKey)
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
			t.Fatalf("MarkSucceeded() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkSucceeded() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusStarted {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusStarted)
		}
		if stored.ObjectKey != nil {
			t.Errorf("stored.ObjectKey = %v, want nil", *stored.ObjectKey)
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
			t.Fatalf("MarkFailed() error = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailed() returned nil")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusFailed)
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
			t.Fatalf("MarkFailed() error = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailed() returned nil")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusFailed)
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
			t.Fatalf("MarkFailed() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailed() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusSucceeded {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusSucceeded)
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
			t.Fatalf("MarkFailed() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailed() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusStarted {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusStarted)
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
			t.Fatalf("UpdateHeartbeat() error = %v", err)
		}
		if !updated {
			t.Error("UpdateHeartbeat() = false, want true")
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.HeartbeatAt == nil {
			t.Fatal("stored.HeartbeatAt is nil")
		}
		if !stored.HeartbeatAt.After(staleHeartbeat) {
			t.Errorf("stored.HeartbeatAt = %v, want after %v", *stored.HeartbeatAt, staleHeartbeat)
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
			t.Fatalf("UpdateHeartbeat() error = %v", err)
		}
		if updated {
			t.Error("UpdateHeartbeat() = true, want false")
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.HeartbeatAt != nil {
			t.Errorf("stored.HeartbeatAt = %v, want nil", *stored.HeartbeatAt)
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
			t.Fatalf("UpdateHeartbeat() error = %v", err)
		}
		if updated {
			t.Error("UpdateHeartbeat() = true, want false")
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.HeartbeatAt != nil {
			t.Errorf("stored.HeartbeatAt = %v, want nil", *stored.HeartbeatAt)
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

	t.Run("エクスポートを状態履歴ごと削除する", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusSucceeded).
			WithObjectKey("exports/deleted.zip").
			Build()

		// The Rails version records the state as export_statuses rows that reference the export.
		// Deleting an export it created has to remove them too.
		//
		// [Ja] Rails 版は状態を、エクスポートを参照する export_statuses の行として記録している。Rails
		// が作ったエクスポートを削除するには、その行も一緒に消す必要がある。
		now := time.Now()
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO export_statuses (space_id, export_id, kind, changed_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4, $4)`,
			string(f.spaceID), string(exportID), int32(model.ExportStatusSucceeded), now,
		); err != nil {
			t.Fatalf("export_statusesの作成に失敗: %v", err)
		}

		if err := repo.Delete(ctx, exportID, f.spaceID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		export, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export != nil {
			t.Errorf("FindByIDAndSpace() = %v, want nil", export)
		}

		var statusCount int
		if err := tx.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM export_statuses WHERE export_id = $1`,
			string(exportID),
		).Scan(&statusCount); err != nil {
			t.Fatalf("export_statusesの件数取得に失敗: %v", err)
		}
		if statusCount != 0 {
			t.Errorf("export_statusesの件数 = %d, want 0", statusCount)
		}
	})

	t.Run("別スペースの指定では削除しない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			WithStatus(model.ExportStatusSucceeded).
			WithObjectKey("exports/kept.zip").
			Build()

		now := time.Now()
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO export_statuses (space_id, export_id, kind, changed_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4, $4)`,
			string(f.spaceID), string(exportID), int32(model.ExportStatusSucceeded), now,
		); err != nil {
			t.Fatalf("export_statusesの作成に失敗: %v", err)
		}

		if err := repo.Delete(ctx, exportID, other.spaceID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		export, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if export == nil {
			t.Fatal("FindByIDAndSpace() returned nil, want the export to be kept")
		}

		var statusCount int
		if err := tx.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM export_statuses WHERE export_id = $1`,
			string(exportID),
		).Scan(&statusCount); err != nil {
			t.Fatalf("export_statusesの件数取得に失敗: %v", err)
		}
		if statusCount != 1 {
			t.Errorf("export_statusesの件数 = %d, want 1", statusCount)
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
			t.Fatalf("MarkFailedIfStale() error = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailedIfStale() returned nil")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusFailed)
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
			t.Fatalf("MarkFailedIfStale() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfStale() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusStarted {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusStarted)
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
			t.Fatalf("MarkFailedIfStale() error = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailedIfStale() returned nil")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusFailed)
		}
	})

	t.Run("queuedのエクスポートは失敗へ進めない", func(t *testing.T) {
		exportID := testutil.NewExportBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithQueuedByID(f.spaceMemberID).
			Build()

		export, err := repo.MarkFailedIfStale(ctx, exportID, f.spaceID, staleBefore)
		if err != nil {
			t.Fatalf("MarkFailedIfStale() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfStale() = %v, want nil", export)
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
			t.Fatalf("MarkFailedIfUnclaimed() error = %v", err)
		}
		if export == nil {
			t.Fatal("MarkFailedIfUnclaimed() returned nil")
		}
		if export.Status != model.ExportStatusFailed {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusFailed)
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
			t.Fatalf("MarkFailedIfUnclaimed() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfUnclaimed() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusQueued {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusQueued)
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
			t.Fatalf("MarkFailedIfUnclaimed() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfUnclaimed() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusStarted {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusStarted)
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
			t.Fatalf("MarkFailedIfUnclaimed() error = %v", err)
		}
		if export != nil {
			t.Errorf("MarkFailedIfUnclaimed() = %v, want nil", export)
		}

		stored, err := repo.FindByIDAndSpace(ctx, exportID, f.spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace() error = %v", err)
		}
		if stored.Status != model.ExportStatusQueued {
			t.Errorf("stored.Status = %v, want %v", stored.Status, model.ExportStatusQueued)
		}
	})
}
