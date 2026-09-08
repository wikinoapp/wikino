package export_download_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/export_download"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupHandler builds the download handler over the given transaction, with an in-memory object
// storage so that the redirect can be checked without reaching the network.
//
// [Ja] setupHandler は与えられたトランザクションの上にダウンロードのハンドラーを組み立てる。
// オブジェクトストレージはメモリ上のものを使い、ネットワークへ出ずにリダイレクトを検証できるように
// する。
func setupHandler(t *testing.T, queries *query.Queries) *export_download.Handler {
	t.Helper()

	return export_download.NewHandler(usecase.NewGetExportDownloadUsecase(
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewExportRepository(queries),
		storage.NewFakeObjectStorage(),
	))
}

// newRequest builds a request carrying the chi URL parameters and the signed-in user.
//
// [Ja] newRequest は chi の URL パラメータとログイン中のユーザーを載せたリクエストを組み立てる。
func newRequest(t *testing.T, spaceIdentifier string, exportID model.ExportID, userID model.UserID) *http.Request {
	t.Helper()

	path := "/s/" + spaceIdentifier + "/settings/exports/" + exportID.String() + "/download"
	req := httptest.NewRequest(http.MethodGet, path, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", spaceIdentifier)
	rctx.URLParams.Add("export_id", exportID.String())

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "export-user"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// exportSpace seeds a space with one member and returns what the tests address them by.
//
// [Ja] exportSpace はメンバーが 1 人いるスペースを用意し、テストがそれらを指すための値を返す。
func exportSpace(t *testing.T, tx *sql.Tx, identifier string, scopes []model.Scope) (model.UserID, model.SpaceID, model.SpaceMemberID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(identifier).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		Build()
	memberBuilder := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if scopes != nil {
		memberBuilder = memberBuilder.WithScopes(scopes)
	}

	return userID, spaceID, memberBuilder.Build()
}

func TestShow_RedirectsToPresignedURL(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-dl-ok", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithStatusChangedAt(time.Now()).
		WithObjectKey("exports/exp-dl-ok/archive.zip").
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, "exp-dl-ok", exportID, userID))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusSeeOther)
	}

	location := rr.Header().Get("Location")
	if !strings.Contains(location, "exports/exp-dl-ok/archive.zip") {
		t.Errorf("Location = %q, ZIP のオブジェクトを指していない", location)
	}
}

// TestShow_NotFoundAfterExpiration keeps a link that outlived its archive from being answered. The
// screen stops offering the download at the same point, so this is reached by a link kept elsewhere
// (the completion mail, a bookmark).
//
// [Ja] TestShow_NotFoundAfterExpiration は、アーカイブより長生きしたリンクに応答しないことを守る。
// 画面も同じ時点でダウンロードの提示をやめるため、ここに到達するのは別の場所に残ったリンク
// (完了メール・ブックマーク) からである。
func TestShow_NotFoundAfterExpiration(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-dl-expired", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithStatusChangedAt(time.Now().Add(-model.ExportDownloadExpiration - time.Minute)).
		WithObjectKey("exports/exp-dl-expired/archive.zip").
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, "exp-dl-expired", exportID, userID))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestShow_NotFoundWhileStillRunning(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-dl-running", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusStarted).
		WithHeartbeatAt(time.Now()).
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, "exp-dl-running", exportID, userID))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestShow_NotFoundWithoutExportPermission(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-dl-forbidden", []model.Scope{model.ScopeSpaceRead})
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithStatusChangedAt(time.Now()).
		WithObjectKey("exports/exp-dl-forbidden/archive.zip").
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, "exp-dl-forbidden", exportID, userID))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// TestShow_NotFoundForExportOfAnotherSpace guards the space scoping of the lookup: knowing the ID
// of an export must not be enough to have its archive signed through a space the viewer happens to
// be able to export.
//
// [Ja] TestShow_NotFoundForExportOfAnotherSpace は取得のスペーススコープを守る。エクスポートの ID
// を知っていることが、たまたまエクスポートできる別のスペース経由でそのアーカイブに署名させる理由に
// なってはならない。
func TestShow_NotFoundForExportOfAnotherSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, _, _ := exportSpace(t, tx, "exp-dl-mine", nil)
	_, otherSpaceID, otherMemberID := exportSpace(t, tx, "exp-dl-theirs", nil)
	otherExportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithQueuedByID(otherMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithStatusChangedAt(time.Now()).
		WithObjectKey("exports/exp-dl-theirs/archive.zip").
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, queries).Show(rr, newRequest(t, "exp-dl-mine", otherExportID, userID))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}
