package export_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestShow_Processing(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-show-running", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusStarted).
		WithHeartbeatAt(time.Now()).
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, http.MethodGet,
		"/s/exp-show-running/settings/exports/"+exportID.String(),
		map[string]string{"space_identifier": "exp-show-running", "export_id": exportID.String()},
		userID,
	))

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "完了したらメールでお知らせします") {
		t.Error("処理中の説明が表示されていない")
	}
	if strings.Contains(body, "ダウンロードする") {
		t.Error("処理中なのにダウンロードの導線が表示されている")
	}
}

func TestShow_Succeeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-show-ok", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithStatusChangedAt(time.Now()).
		WithObjectKey("exports/space/archive.zip").
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, http.MethodGet,
		"/s/exp-show-ok/settings/exports/"+exportID.String(),
		map[string]string{"space_identifier": "exp-show-ok", "export_id": exportID.String()},
		userID,
	))

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "エクスポートが完了しました") {
		t.Error("完了の説明が表示されていない")
	}
	if !strings.Contains(body, "/s/exp-show-ok/settings/exports/"+exportID.String()+"/download") {
		t.Error("ダウンロードの導線が表示されていない")
	}
}

// TestShow_SucceededAfterDownloadExpired covers the export that finished long enough ago that its
// archive can no longer be handed out. The screen has to offer another export instead of a download
// that would answer 404.
//
// [Ja] TestShow_SucceededAfterDownloadExpired は、アーカイブをもう渡せなくなるほど前に完了した
// エクスポートを対象とする。画面は 404 になるダウンロードではなく、もう一度のエクスポートを
// 提示する必要がある。
func TestShow_SucceededAfterDownloadExpired(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-show-expired", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithStatusChangedAt(time.Now().Add(-model.ExportDownloadExpiration - time.Minute)).
		WithObjectKey("exports/space/archive.zip").
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, http.MethodGet,
		"/s/exp-show-expired/settings/exports/"+exportID.String(),
		map[string]string{"space_identifier": "exp-show-expired", "export_id": exportID.String()},
		userID,
	))

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if strings.Contains(body, "/download") {
		t.Error("期限切れなのにダウンロードの導線が表示されている")
	}
	if !strings.Contains(body, "/s/exp-show-expired/settings/exports/new") {
		t.Error("再実行の導線が表示されていない")
	}
}

func TestShow_Failed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-show-failed", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusFailed).
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, http.MethodGet,
		"/s/exp-show-failed/settings/exports/"+exportID.String(),
		map[string]string{"space_identifier": "exp-show-failed", "export_id": exportID.String()},
		userID,
	))

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "エクスポートを完了できませんでした") {
		t.Error("失敗の説明が表示されていない")
	}
	if !strings.Contains(body, "/s/exp-show-failed/settings/exports/new") {
		t.Error("再実行の導線が表示されていない")
	}
}

// TestShow_StaleStartedShownAsFailed covers the export a stopped worker left behind. Its status
// stays started forever, so the screen goes by the heartbeat as the usecase does and shows it as a
// failure with a way to try again.
//
// [Ja] TestShow_StaleStartedShownAsFailed は、停止したワーカーが取り残したエクスポートを対象と
// する。その状態は started のまま残り続けるため、画面は UseCase と同じく heartbeat で判断し、
// やり直しの導線を添えた失敗として表示する。
func TestShow_StaleStartedShownAsFailed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-show-stale", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusStarted).
		WithHeartbeatAt(time.Now().Add(-model.ExportHeartbeatStaleAfter - time.Minute)).
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, http.MethodGet,
		"/s/exp-show-stale/settings/exports/"+exportID.String(),
		map[string]string{"space_identifier": "exp-show-stale", "export_id": exportID.String()},
		userID,
	))

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "エクスポートを完了できませんでした") {
		t.Error("停止したエクスポートが失敗として表示されていない")
	}
	if !strings.Contains(body, "/s/exp-show-stale/settings/exports/new") {
		t.Error("再実行の導線が表示されていない")
	}
}

// TestShow_NotFoundForExportOfAnotherSpace guards the space scoping of the lookup: knowing the ID
// of an export must not be enough to read it from a space the viewer happens to be able to export.
//
// [Ja] TestShow_NotFoundForExportOfAnotherSpace は取得のスペーススコープを守る。エクスポートの ID
// を知っていることが、たまたまエクスポートできる別のスペース経由でそれを読む理由になってはならない。
func TestShow_NotFoundForExportOfAnotherSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, _, _ := exportSpace(t, tx, "exp-show-mine", nil)
	_, otherSpaceID, otherMemberID := exportSpace(t, tx, "exp-show-theirs", nil)
	otherExportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithQueuedByID(otherMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithObjectKey("exports/other/archive.zip").
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, http.MethodGet,
		"/s/exp-show-mine/settings/exports/"+otherExportID.String(),
		map[string]string{"space_identifier": "exp-show-mine", "export_id": otherExportID.String()},
		userID,
	))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}
