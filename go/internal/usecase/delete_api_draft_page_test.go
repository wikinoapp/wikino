package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newDeleteAPIDraftPageUC(db *sql.DB) *DeleteAPIDraftPageUsecase {
	q := query.New(db)
	return NewDeleteAPIDraftPageUsecase(
		db,
		repository.NewPageRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)
}

// draftPageDeleteScopesは下書きを削除できるトークンのスコープ
var draftPageDeleteScopes = []model.Scope{model.ScopeDraftPageDelete}

func TestDeleteAPIDraftPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-delete", memberRole)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").WithBody("ページの本文").Build()
	draftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
		WithTitle("下書き").WithBody("下書きの本文").Build()
	testutil.NewDraftPageRevisionBuilderDB(t, db).WithDraftPageID(draftID).WithSpaceID(f.space.ID).WithSpaceMemberID(f.spaceMemberID).Build()
	testutil.NewDraftPageRevisionBuilderDB(t, db).WithDraftPageID(draftID).WithSpaceID(f.space.ID).WithSpaceMemberID(f.spaceMemberID).Build()
	uc := newDeleteAPIDraftPageUC(db)
	digest := draftPageDigest(t, db, draftID, f.space.ID)

	// サブテストは同じ下書きを順に扱うため、並列にしない。
	t.Run("古い版を前提にした削除は412で、下書きを残す", func(t *testing.T) {
		err := uc.Execute(t.Context(), DeleteAPIDraftPageInput{
			Principal:       f.principal(memberRole, draftPageDeleteScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{"古い版"},
		})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
			t.Fatalf("Execute()のエラー = %v、期待値 = AppErrCodePreconditionFailed (412)", err)
		}
		if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", draftID); n != 1 {
			t.Errorf("残った下書きの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("下書きを編集履歴ごと削除し、ページは残す", func(t *testing.T) {
		err := uc.Execute(t.Context(), DeleteAPIDraftPageInput{
			Principal:       f.principal(memberRole, draftPageDeleteScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{digest},
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", draftID); n != 0 {
			t.Errorf("残った下書きの数 = %d、期待値 = 0", n)
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != 0 {
			t.Errorf("残ったリビジョンの数 = %d、期待値 = 0", n)
		}
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND title = 'ページ' AND body = 'ページの本文'", pageID); n != 1 {
			t.Errorf("元のままのページの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("削除済みの下書きをもう一度削除すると未存在", func(t *testing.T) {
		err := uc.Execute(t.Context(), DeleteAPIDraftPageInput{
			Principal:       f.principal(memberRole, draftPageDeleteScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{digest},
		})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Fatalf("Execute()のエラー = %v、期待値 = AppErrCodeResourceNotFound (404)", err)
		}
	})
}

func TestDeleteAPIDraftPageUsecase_Execute_NotFoundAndForbidden(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-delete-denied", memberRole)
	ownPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("自分の下書きのページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(ownPageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).Build()
	otherUserID := testutil.NewUserBuilderDB(t, db).WithAtname("api_draft_delete_other").WithEmail("api_draft_delete_other@example.com").Build()
	otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(f.space.ID).WithUserID(otherUserID).Build()
	otherPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).WithTitle("他のメンバーの下書きのページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(otherPageID).WithSpaceMemberID(otherMemberID).WithTopicID(f.topicIDs[1]).Build()

	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		pageNumber  int32
		wantCode    model.AppErrorCode
	}{
		{name: "ほかのメンバーの下書きは未存在", memberRole: memberRole, tokenScopes: draftPageDeleteScopes, pageNumber: 2, wantCode: model.AppErrCodeResourceNotFound},
		{name: "draft_page:deleteの無いトークンは削除できない", memberRole: memberRole, tokenScopes: draftPageWriteScopes, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
		{name: "下書きを削除できないロールは削除できない", memberRole: model.SpaceRoleViewer, tokenScopes: draftPageDeleteScopes, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := newDeleteAPIDraftPageUC(db).Execute(t.Context(), DeleteAPIDraftPageInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      tt.pageNumber,
				IfMatchAny:      true,
			})
			if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
				t.Fatalf("Execute()のエラー = %v、期待値 = %v", err, tt.wantCode)
			}
		})
	}

	if n := countRows(t, db, "draft_pages", f.space.ID, ""); n != 2 {
		t.Errorf("残った下書きの数 = %d、期待値 = 2", n)
	}
}

// WebとAPIの双方を先行させ、後続の削除が下書きのロックを待ってから404になることを確かめる。
// 編集履歴のロックで先行する削除を止め、ロック待ちを観測してから進めるため、実行速度に依存しない
func TestDeleteAPIDraftPageUsecase_Execute_ConcurrentWebDelete(t *testing.T) {
	t.Parallel()

	for _, webFirst := range []bool{true, false} {
		name, identifier := "Webの削除が先行する", "draft-delete-web-first"
		if !webFirst {
			name, identifier = "APIの削除が先行する", "draft-delete-api-first"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := testutil.GetTestDB()
			f := setupCreateAPIPageFixture(t, db, identifier, apiTopicRegularMemberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).Build()
			draftID := testutil.NewDraftPageBuilderDB(t, db).
				WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).Build()
			testutil.NewDraftPageRevisionBuilderDB(t, db).
				WithDraftPageID(draftID).WithSpaceID(f.space.ID).WithSpaceMemberID(f.spaceMemberID).Build()

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			blocker, blockerPID := beginWebTx(t, db)
			// FKの参照先である下書きをロックせず、編集履歴の削除だけを待たせる
			if _, err := blocker.ExecContext(ctx, `SELECT id FROM draft_page_revisions WHERE draft_page_id = $1 AND space_id = $2 FOR UPDATE`, draftID, f.space.ID); err != nil {
				t.Fatalf("編集履歴のロック: %v", err)
			}

			webUC, _, _ := newDeleteDraftPageUsecaseForTest(t)
			webDelete := func() error {
				return webUC.Execute(ctx, DeleteDraftPageInput{SpaceIdentifier: f.space.Identifier, PageNumber: 1, UserID: f.userID})
			}
			apiDelete := func() error {
				return newDeleteAPIDraftPageUC(db).Execute(ctx, DeleteAPIDraftPageInput{
					Principal:       f.principal(apiTopicRegularMemberRole, draftPageDeleteScopes),
					SpaceIdentifier: f.space.Identifier, PageNumber: 1, IfMatchAny: true,
				})
			}
			first, second := webDelete, apiDelete
			if !webFirst {
				first, second = apiDelete, webDelete
			}
			firstResult, secondResult := make(chan error, 1), make(chan error, 1)
			go func() { firstResult <- first() }()

			// 先行する削除が編集履歴を待つまで、後続の削除を始めない
			firstPID := waitForDraftDeletionBlocked(t, ctx, db, blockerPID, "DeleteDraftPageRevisionsByDraftPageID", firstResult)
			go func() { secondResult <- second() }()
			waitForDraftDeletionBlocked(t, ctx, db, firstPID, "FindDraftPageByIDForUpdate", secondResult)

			if err := blocker.Rollback(); err != nil {
				t.Fatalf("編集履歴のロック解除: %v", err)
			}
			if err := <-firstResult; err != nil {
				t.Fatalf("先行する削除のエラー = %v、期待値 = nil", err)
			}
			if err := <-secondResult; model.AsAppError(err) == nil || model.AsAppError(err).Code != model.AppErrCodeResourceNotFound {
				t.Fatalf("後続の削除のエラー = %v、期待値 = AppErrCodeResourceNotFound", err)
			}
			if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", draftID); n != 0 {
				t.Errorf("残った下書きの数 = %d、期待値 = 0", n)
			}
			if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != 0 {
				t.Errorf("残った編集履歴の数 = %d、期待値 = 0", n)
			}
			if n := countRows(t, db, "pages", f.space.ID, "id = $2", pageID); n != 1 {
				t.Errorf("残ったページの数 = %d、期待値 = 1", n)
			}
		})
	}
}

// Web画面の公開とAPIの削除の双方を先行させ、どちらも下書き → 編集履歴の順にロックして循環待ちにならないことを確かめる。
// 公開が先行すれば後続の削除は404になり、削除が先行すれば後続の公開は消えた下書きの削除を飛ばして成功する
func TestDeleteAPIDraftPageUsecase_Execute_ConcurrentWebPublish(t *testing.T) {
	t.Parallel()

	for _, publishFirst := range []bool{true, false} {
		name, identifier := "Webの公開が先行する", "draft-delete-publish-first"
		if !publishFirst {
			name, identifier = "APIの削除が先行する", "draft-delete-before-publish"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			db := testutil.GetTestDB()
			f := setupCreateAPIPageFixture(t, db, identifier, apiTopicRegularMemberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").Build()
			draftID := testutil.NewDraftPageBuilderDB(t, db).
				WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).Build()
			testutil.NewDraftPageRevisionBuilderDB(t, db).
				WithDraftPageID(draftID).WithSpaceID(f.space.ID).WithSpaceMemberID(f.spaceMemberID).Build()

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			blocker, blockerPID := beginWebTx(t, db)
			// FKの参照先である下書きをロックせず、編集履歴の削除だけを待たせる
			if _, err := blocker.ExecContext(ctx, `SELECT id FROM draft_page_revisions WHERE draft_page_id = $1 AND space_id = $2 FOR UPDATE`, draftID, f.space.ID); err != nil {
				t.Fatalf("編集履歴のロック: %v", err)
			}

			q := query.New(db)
			pageRepo := repository.NewPageRepository(q)
			publishUC := NewPublishPageUsecase(db, repository.NewSpaceRepository(q), repository.NewSpaceMemberRepository(q), pageRepo,
				repository.NewPageRevisionRepository(q), repository.NewPageEditorRepository(q), repository.NewDraftPageRepository(q),
				repository.NewDraftPageRevisionRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q),
				repository.NewAttachmentRepository(q), repository.NewPageAttachmentReferenceRepository(q), validator.NewPageUpdateValidator(pageRepo))
			publish := func() error {
				_, err := publishUC.Execute(ctx, PublishPageInput{
					SpaceIdentifier: f.space.Identifier, PageNumber: 1, UserID: f.userID, Title: "公開したタイトル", Body: "公開した本文",
				})
				return err
			}
			apiDelete := func() error {
				return newDeleteAPIDraftPageUC(db).Execute(ctx, DeleteAPIDraftPageInput{
					Principal:       f.principal(apiTopicRegularMemberRole, draftPageDeleteScopes),
					SpaceIdentifier: f.space.Identifier, PageNumber: 1, IfMatchAny: true,
				})
			}
			first, second := publish, apiDelete
			if !publishFirst {
				first, second = apiDelete, publish
			}
			firstResult, secondResult := make(chan error, 1), make(chan error, 1)
			go func() { firstResult <- first() }()

			// 先行する側が下書きをロックして編集履歴を待つまで、後続を始めない
			firstPID := waitForDraftDeletionBlocked(t, ctx, db, blockerPID, "DeleteDraftPageRevisionsByDraftPageID", firstResult)
			go func() { secondResult <- second() }()
			waitForDraftDeletionBlocked(t, ctx, db, firstPID, "FindDraftPageByIDForUpdate", secondResult)

			if err := blocker.Rollback(); err != nil {
				t.Fatalf("編集履歴のロック解除: %v", err)
			}
			publishErr, deleteErr := <-firstResult, <-secondResult
			if !publishFirst {
				publishErr, deleteErr = deleteErr, publishErr
			}
			if publishErr != nil {
				t.Fatalf("公開のエラー = %v、期待値 = nil", publishErr)
			}
			if publishFirst {
				if model.AsAppError(deleteErr) == nil || model.AsAppError(deleteErr).Code != model.AppErrCodeResourceNotFound {
					t.Fatalf("後続の削除のエラー = %v、期待値 = AppErrCodeResourceNotFound", deleteErr)
				}
			} else if deleteErr != nil {
				t.Fatalf("先行する削除のエラー = %v、期待値 = nil", deleteErr)
			}
			if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", draftID); n != 0 {
				t.Errorf("残った下書きの数 = %d、期待値 = 0", n)
			}
			if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != 0 {
				t.Errorf("残った編集履歴の数 = %d、期待値 = 0", n)
			}
			if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND title = $3", pageID, "公開したタイトル"); n != 1 {
				t.Errorf("公開したタイトルのページの数 = %d、期待値 = 1", n)
			}
		})
	}
}

// waitForDraftDeletionBlockedは、queryNameのクエリを実行中の接続がblockingPIDの接続のロックを待つまで待機し、
// 待っている接続のPIDを返す。待つ前にresultの処理が終わった場合と、ctxが期限切れになった場合はテストを失敗させる
func waitForDraftDeletionBlocked(t *testing.T, ctx context.Context, db *sql.DB, blockingPID int, queryName string, result <-chan error) int {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var pid int
		err := db.QueryRowContext(ctx, `SELECT COALESCE((
			SELECT pid FROM pg_stat_activity
			WHERE datname = current_database() AND $1 = ANY(pg_blocking_pids(pid))
			AND query LIKE $2 LIMIT 1
		), 0)`, blockingPID, "%"+queryName+"%").Scan(&pid)
		if err != nil {
			t.Fatalf("ロック待ちの確認: %v", err)
		}
		if pid != 0 {
			return pid
		}
		select {
		case err := <-result:
			t.Fatalf("ロック待ちを確認する前に処理が終了: %v", err)
		case <-ctx.Done():
			t.Fatalf("期待した行のロックを待たなかった: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}
