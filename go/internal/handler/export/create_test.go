package export_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	exporthandler "github.com/wikinoapp/wikino/go/internal/handler/export"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// exportJobInserter records jobs without starting a worker, and can simulate an enqueue failure.
//
// [Ja] exportJobInserter はワーカーを起動せずにジョブを記録し、投入失敗も再現する。
type exportJobInserter struct {
	args []river.JobArgs
	err  error
}

func (i *exportJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	i.args = append(i.args, args)
	return &rivertype.JobInsertResult{}, i.err
}

func TestCreate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		membership   string
		anonymous    bool
		previous     *model.ExportStatus
		enqueueError bool
		status       int
		count        int
		jobs         int
		flashType    session.FlashType
		flashMessage string
	}{
		{name: "success", status: http.StatusSeeOther, count: 1, jobs: 1, flashType: session.FlashSuccess, flashMessage: "エクスポートを開始しました"},
		{name: "queued", previous: new(model.ExportStatusQueued), status: http.StatusSeeOther, count: 1, flashType: session.FlashError, flashMessage: "エクスポートを実行中です。完了してからお試しください"},
		{name: "started", previous: new(model.ExportStatusStarted), status: http.StatusSeeOther, count: 1, flashType: session.FlashError, flashMessage: "エクスポートを実行中です。完了してからお試しください"},
		{name: "reader", membership: "reader", status: http.StatusNotFound},
		{name: "outsider", membership: "none", status: http.StatusNotFound},
		{name: "anonymous", anonymous: true, status: http.StatusFound},
		{name: "enqueue-failure", enqueueError: true, status: http.StatusInternalServerError, jobs: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			db := testutil.GetTestDB()
			identifier := "handler-export-" + tc.name
			userID := testutil.NewUserBuilderDB(t, db).WithEmail(identifier + "@example.com").WithAtname(identifier).Build()
			spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
			// The usecase opens its own transaction, so fixtures must be committed and cleaned up explicitly.
			//
			// [Ja] UseCaseが自身でトランザクションを開くため、フィクスチャをコミットし、明示的に後片付けする。
			t.Cleanup(func() {
				for _, statement := range []string{
					"DELETE FROM exports WHERE space_id = $1",
					"DELETE FROM space_members WHERE space_id = $1",
					"DELETE FROM spaces WHERE id = $1",
				} {
					if _, err := db.ExecContext(ctx, statement, spaceID); err != nil {
						t.Error(err)
					}
				}
				if _, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", userID); err != nil {
					t.Error(err)
				}
			})
			var memberID model.SpaceMemberID
			if tc.membership != "none" {
				builder := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID)
				if tc.membership == "reader" {
					builder.WithScopes([]model.Scope{model.ScopeSpaceRead})
				}
				memberID = builder.Build()
			}
			var previousID model.ExportID
			if tc.previous != nil {
				previousID = testutil.NewExportBuilderDB(t, db).WithSpaceID(spaceID).WithQueuedByID(memberID).WithStatus(*tc.previous).WithHeartbeatAt(time.Now()).Build()
			}
			inserter := &exportJobInserter{}
			if tc.enqueueError {
				inserter.err = errors.New("enqueue failed")
			}
			queries := query.New(db)
			exportRepo := repository.NewExportRepository(queries)
			flashMgr := session.NewFlashManager("", false, true)
			h := exporthandler.NewHandler(&config.Config{Env: "test"}, flashMgr, nil, nil,
				usecase.NewCreateExportUsecase(db, repository.NewSpaceRepository(queries), repository.NewSpaceMemberRepository(queries), exportRepo, dispatcher.NewDispatcher(inserter)))
			req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/exports", map[string]string{"space_identifier": identifier}, userID)
			if tc.anonymous {
				req = req.WithContext(middleware.SetUserToContext(req.Context(), nil))
			}
			rr := httptest.NewRecorder()
			h.Create(rr, req)
			if rr.Code != tc.status {
				t.Fatalf("status = %d, want %d", rr.Code, tc.status)
			}
			if len(inserter.args) != tc.jobs {
				t.Errorf("jobs = %d, want %d", len(inserter.args), tc.jobs)
			}
			var count int
			if err := db.QueryRowContext(ctx, "SELECT count(*) FROM exports WHERE space_id = $1", spaceID).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != tc.count {
				t.Errorf("exports = %d, want %d", count, tc.count)
			}
			if tc.status == http.StatusSeeOther {
				latest, err := exportRepo.FindLatestBySpace(ctx, spaceID)
				if err != nil || latest == nil {
					t.Fatalf("latest = %v, error = %v", latest, err)
				}
				wantLocation := "/s/" + identifier + "/settings/exports/" + latest.ID.String()
				if rr.Header().Get("Location") != wantLocation {
					t.Errorf("Location = %q, want %q", rr.Header().Get("Location"), wantLocation)
				}
				if tc.previous != nil && latest.ID != previousID {
					t.Error("競合時に既存エクスポートが置き換わっています")
				}
				if tc.previous == nil {
					args, ok := inserter.args[0].(dispatcher.GenerateExportFilesArgs)
					if !ok || args.ExportID != latest.ID.String() || args.SpaceID != string(spaceID) {
						t.Errorf("job args = %#v", inserter.args[0])
					}
					if latest.Status != model.ExportStatusQueued {
						t.Errorf("status = %v, want queued", latest.Status)
					}
				}
			} else if tc.anonymous && rr.Header().Get("Location") != "/sign_in" {
				t.Errorf("Location = %q", rr.Header().Get("Location"))
			}
			flashReq := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, cookie := range rr.Result().Cookies() {
				flashReq.AddCookie(cookie)
			}
			flash := flashMgr.GetFlash(httptest.NewRecorder(), flashReq)
			if tc.flashType == "" {
				if flash != nil {
					t.Errorf("unexpected flash = %#v", flash)
				}
			} else if flash == nil || flash.Type != tc.flashType || flash.Message != tc.flashMessage {
				t.Errorf("flash = %#v, want %s %q", flash, tc.flashType, tc.flashMessage)
			}
		})
	}
}
