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
)

func newUpdateAPIDraftPageUC(db *sql.DB) *UpdateAPIDraftPageUsecase {
	q := query.New(db)
	return NewUpdateAPIDraftPageUsecase(
		db,
		repository.NewSpaceRepository(q),
		repository.NewPageRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
	)
}

// draftPageDigestは下書きの今の版 (ETagの引用符の中身) を返す
func draftPageDigest(t *testing.T, db *sql.DB, id model.DraftPageID, spaceID model.SpaceID) string {
	t.Helper()

	draft, err := repository.NewDraftPageRepository(query.New(db)).FindByID(t.Context(), id, spaceID)
	if err != nil || draft == nil {
		t.Fatalf("下書きの取得 = (%v, %v)、期待値 = 下書き", draft, err)
	}
	return draft.ContentDigest()
}

func TestUpdateAPIDraftPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-update", memberRole)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").WithBody("ページの本文").Build()
	draftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
		WithTitle("下書き").WithBody("下書きの本文").Build()
	uc := newUpdateAPIDraftPageUC(db)
	update := func(t *testing.T, digest string, title *string, updateTitle bool, body *string) (*UpdateAPIDraftPageOutput, error) {
		t.Helper()
		return uc.Execute(t.Context(), UpdateAPIDraftPageInput{
			Principal:       f.principal(memberRole, draftPageWriteScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{digest},
			UpdateTitle:     updateTitle,
			Title:           title,
			Body:            body,
		})
	}

	// サブテストは同じ下書きを順に更新するため、並列にしない。
	t.Run("本文だけを更新し、リビジョンを作る", func(t *testing.T) {
		before := draftPageDigest(t, db, draftID, f.space.ID)
		output, err := update(t, before, nil, false, ptr("[[まだ無いページ]]"))
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		draft := output.DraftPage
		if draft.Title == nil || *draft.Title != "下書き" || draft.Body != "[[まだ無いページ]]" {
			t.Errorf("下書き = (%v, %q)、期待値 = (\"下書き\", \"[[まだ無いページ]]\")", draft.Title, draft.Body)
		}
		if draft.Page == nil || draft.Page.ID != pageID || output.Topic.ID != f.topicIDs[1] {
			t.Errorf("下書きのページ・トピック = (%+v, %+v)、期待値 = ページ1とトピック1", draft.Page, output.Topic)
		}
		if draft.ContentDigest() == before || draft.ContentDigest() != draftPageDigest(t, db, draftID, f.space.ID) {
			t.Error("更新後の版が、更新前と異なり保存した下書きと同じになっていない")
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = '下書き' AND body = '[[まだ無いページ]]'", draftID); n != 1 {
			t.Errorf("更新した内容のリビジョンの数 = %d、期待値 = 1", n)
		}
		// Web画面の保存と同じく、Wikiリンクが指す存在しないページを未公開のページとして作る
		if n := countRows(t, db, "pages", f.space.ID, "title = 'まだ無いページ' AND published_at IS NULL"); n != 1 {
			t.Errorf("リンク先として作成されたページの数 = %d、期待値 = 1", n)
		}
		// ページ自体は書き換えない
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND title = 'ページ' AND body = 'ページの本文'", pageID); n != 1 {
			t.Errorf("元のままのページの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("タイトルをnullにするとタイトルを外す", func(t *testing.T) {
		output, err := update(t, draftPageDigest(t, db, draftID, f.space.ID), nil, true, nil)
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DraftPage.Title != nil || output.DraftPage.Body != "[[まだ無いページ]]" {
			t.Errorf("下書き = (%v, %q)、期待値 = (nil, 変えない本文)", output.DraftPage.Title, output.DraftPage.Body)
		}
		// 手動保存と同じく、タイトルの無い下書きのリビジョンは空のタイトルで記録する
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = '' AND body = '[[まだ無いページ]]'", draftID); n != 1 {
			t.Errorf("タイトルを外した内容のリビジョンの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("内容が今と同じなら下書きを書き換えず、最新のリビジョンと同じならリビジョンも作らない", func(t *testing.T) {
		before := draftPageDigest(t, db, draftID, f.space.ID)
		revisions := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID)

		output, err := update(t, before, nil, true, ptr("[[まだ無いページ]]"))
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DraftPage.ContentDigest() != before || draftPageDigest(t, db, draftID, f.space.ID) != before {
			t.Error("内容が同じ更新で下書きの版が変わった")
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != revisions {
			t.Errorf("リビジョンの数 = %d、期待値 = %d (増えない)", n, revisions)
		}
	})

	t.Run("Web画面の自動保存の後に古い版で更新すると412", func(t *testing.T) {
		before := draftPageDigest(t, db, draftID, f.space.ID)
		q := query.New(db)
		autoSave := NewAutoSaveDraftPageUsecase(db, repository.NewSpaceRepository(q), repository.NewSpaceMemberRepository(q),
			repository.NewDraftPageRepository(q), repository.NewPageRepository(q), repository.NewPageEditorRepository(q),
			repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q), repository.NewAttachmentRepository(q))
		if _, err := autoSave.Execute(t.Context(), AutoSaveDraftPageInput{
			SpaceIdentifier: f.space.Identifier, PageNumber: 1, UserID: f.userID, Title: ptr("Webのタイトル"), Body: "Webの本文",
		}); err != nil {
			t.Fatalf("自動保存のエラー = %v", err)
		}

		_, err := update(t, before, nil, false, ptr("古い版を元にした本文"))
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
			t.Fatalf("Execute()のエラー = %v、期待値 = AppErrCodePreconditionFailed (412)", err)
		}
		if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2 AND body = 'Webの本文'", draftID); n != 1 {
			t.Errorf("自動保存の内容を保った下書きの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("If-Match: *なら版を問わず更新する", func(t *testing.T) {
		output, err := uc.Execute(t.Context(), UpdateAPIDraftPageInput{
			Principal:       f.principal(memberRole, draftPageWriteScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchAny:      true,
			Body:            ptr("版を問わない本文"),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DraftPage.Body != "版を問わない本文" {
			t.Errorf("本文 = %q、期待値 = \"版を問わない本文\"", output.DraftPage.Body)
		}
	})
}

func TestUpdateAPIDraftPageUsecase_Execute_NotFoundAndForbidden(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-update-denied", memberRole)
	ownPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("自分の下書きのページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(ownPageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).WithTitle("下書きの無いページ").Build()
	otherUserID := testutil.NewUserBuilderDB(t, db).WithAtname("api_draft_update_other").WithEmail("api_draft_update_other@example.com").Build()
	otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(f.space.ID).WithUserID(otherUserID).Build()
	otherPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(3).WithTitle("他のメンバーの下書きのページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(otherPageID).WithSpaceMemberID(otherMemberID).WithTopicID(f.topicIDs[1]).Build()

	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		pageNumber  int32
		wantCode    model.AppErrorCode
	}{
		{name: "下書きの無いページは未存在", memberRole: memberRole, tokenScopes: draftPageWriteScopes, pageNumber: 2, wantCode: model.AppErrCodeResourceNotFound},
		{name: "ほかのメンバーの下書きは未存在", memberRole: memberRole, tokenScopes: draftPageWriteScopes, pageNumber: 3, wantCode: model.AppErrCodeResourceNotFound},
		{name: "draft_page:writeの無いトークンは保存できない", memberRole: memberRole, tokenScopes: []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageDelete}, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
		{name: "下書きを保存できないロールは保存できない", memberRole: model.SpaceRoleViewer, tokenScopes: draftPageWriteScopes, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := newUpdateAPIDraftPageUC(db).Execute(t.Context(), UpdateAPIDraftPageInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      tt.pageNumber,
				IfMatchAny:      true,
				Body:            ptr("更新できない本文"),
			})
			if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
				t.Fatalf("Execute()のエラー = %v、期待値 = %v", err, tt.wantCode)
			}
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
		})
	}

	if n := countRows(t, db, "draft_pages", f.space.ID, "body = '更新できない本文'"); n != 0 {
		t.Errorf("書き換えられた下書きの数 = %d、期待値 = 0", n)
	}
}

// Web画面の保存がリンク先のページを作った後、下書きを書き換える前に、APIの更新が同じリンク先を作ろうとしても、
// 循環待ちにならず、Web画面の保存の後に412を返すことを確かめる
func TestUpdateAPIDraftPageUsecase_Execute_ConcurrentWebSave(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-update-concurrent", memberRole)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()
	draftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
		WithTitle("下書き").WithBody("下書きの本文").Build()
	digest := draftPageDigest(t, db, draftID, f.space.ID)

	webTx, webPID := beginWebTx(t, db)
	q := query.New(db).WithTx(webTx)
	if _, err := resolveLinkedPageIDs(t.Context(), resolveLinkedPageIDsInput{
		Body: "[[共有リンク先]]", CurrentTopicName: "公開", SpaceID: f.space.ID, SpaceMemberID: f.spaceMemberID,
	}, repository.NewTopicRepository(q), repository.NewPageRepository(q), repository.NewPageEditorRepository(q)); err != nil {
		t.Fatalf("Web側のリンク先の解決: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	type result struct {
		output *UpdateAPIDraftPageOutput
		err    error
	}
	results := make(chan result, 1)
	go func() {
		output, err := newUpdateAPIDraftPageUC(db).Execute(ctx, UpdateAPIDraftPageInput{
			Principal: f.principal(memberRole, draftPageWriteScopes), SpaceIdentifier: f.space.Identifier,
			PageNumber: 1, IfMatchDigests: []string{digest}, Body: ptr("[[共有リンク先]]"),
		})
		results <- result{output: output, err: err}
	}()

	// APIのリンク先の作成が、Webの未コミットのリンク先のページを待つまで、Webの下書きの更新を進めない
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for blocked := false; !blocked; {
		err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity
			WHERE datname = current_database()
			  AND $1 = ANY(pg_blocking_pids(pid))
			  AND query LIKE '%CreateUnpublishedPage%'
		)`, webPID).Scan(&blocked)
		if err != nil {
			t.Fatalf("APIのロック待ちの確認: %v", err)
		}
		if blocked {
			break
		}
		select {
		case r := <-results:
			t.Fatalf("競合前にAPIが終了: output=%+v, err=%v", r.output, r.err)
		case <-ctx.Done():
			t.Fatalf("APIがWeb側のリンク先のページを待たなかった: %v", ctx.Err())
		case <-ticker.C:
		}
	}

	if _, err := repository.NewDraftPageRepository(q).Update(ctx, repository.UpdateDraftPageInput{
		ID: draftID, SpaceID: f.space.ID, TopicID: f.topicIDs[1], Title: ptr("Webの下書き"), Body: "[[共有リンク先]]", ModifiedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Web側の下書きの更新: %v", err)
	}
	if err := webTx.Commit(); err != nil {
		t.Fatalf("Web側のコミット: %v", err)
	}

	var r result
	select {
	case r = <-results:
	case <-ctx.Done():
		t.Fatalf("Web保存後もAPIが終了しなかった: %v", ctx.Err())
	}
	if ae := model.AsAppError(r.err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
		t.Fatalf("APIのエラー = %v、期待値 = AppErrCodePreconditionFailed (412)", r.err)
	}
	if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2 AND title = 'Webの下書き'", draftID); n != 1 {
		t.Errorf("Webの内容を保った下書きの数 = %d、期待値 = 1", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, "title = '共有リンク先'"); n != 1 {
		t.Errorf("共有リンク先のページの数 = %d、期待値 = 1", n)
	}
}
