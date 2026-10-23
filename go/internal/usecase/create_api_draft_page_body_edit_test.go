package usecase

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newCreateAPIDraftPageBodyEditUC(db *sql.DB) *CreateAPIDraftPageBodyEditUsecase {
	q := query.New(db)
	return NewCreateAPIDraftPageBodyEditUsecase(
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

func TestCreateAPIDraftPageBodyEditUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole

	t.Run("本文を順に置換してリビジョンを作り、タイトルとページに触れない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-draft-body-edit", memberRole)
		pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithTitle("ページ").WithBody("ページの本文").Build()
		draftID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
			WithTitle("下書き").WithBody("- [ ] **6b-1**\r\n- [ ] **6b-2**\r\n関連: なし").Build()
		before := draftPageDigest(t, db, draftID, f.space.ID)

		output, err := newCreateAPIDraftPageBodyEditUC(db).Execute(t.Context(), CreateAPIDraftPageBodyEditInput{
			Principal:       f.principal(memberRole, draftPageWriteScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits: []model.BodyEdit{
				{OldStr: "- [ ] **6b-1**\n- [ ]", NewStr: "- [x] **6b-1**\n- [ ]"},
				{OldStr: "- [ ]", NewStr: "- [x]"},
				{OldStr: "なし", NewStr: "[[まだ無いページ]]"},
			},
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		draft := output.DraftPage
		want := "- [x] **6b-1**\n- [x] **6b-2**\n関連: [[まだ無いページ]]"
		if draft.Title == nil || *draft.Title != "下書き" || draft.Body != want {
			t.Errorf("下書き = (%v, %q)、期待値 = (\"下書き\", %q)", draft.Title, draft.Body, want)
		}
		if draft.Page == nil || draft.Page.ID != pageID || output.Topic.ID != f.topicIDs[1] {
			t.Errorf("下書きのページ・トピック = (%+v, %+v)、期待値 = ページ1とトピック1", draft.Page, output.Topic)
		}
		if draft.ContentDigest() == before || draft.ContentDigest() != draftPageDigest(t, db, draftID, f.space.ID) {
			t.Error("置換後の版が、置換前と異なり保存した下書きと同じになっていない")
		}
		// 編集の数ではなく、1回の手動保存としてリビジョンを1件作る
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != 1 {
			t.Errorf("リビジョンの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = '下書き' AND body = $3", draftID, want); n != 1 {
			t.Errorf("置換した内容のリビジョンの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "pages", f.space.ID, "title = 'まだ無いページ' AND published_at IS NULL"); n != 1 {
			t.Errorf("リンク先として作成されたページの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND title = 'ページ' AND body = 'ページの本文'", pageID); n != 1 {
			t.Errorf("元のままのページの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "page_revisions", f.space.ID, ""); n != 0 {
			t.Errorf("ページのリビジョンの数 = %d、期待値 = 0", n)
		}
	})

	t.Run("タイトルの無い下書きはタイトルの無いまま置換する", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-draft-body-edit-untitled", memberRole)
		pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
			WithNilTitle().WithBody("").WithUnpublished().Build()
		draftID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
			WithNilTitle().WithBody("foo bar foo").Build()

		output, err := newCreateAPIDraftPageBodyEditUC(db).Execute(t.Context(), CreateAPIDraftPageBodyEditInput{
			Principal:       f.principal(memberRole, draftPageWriteScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "foo", NewStr: "baz", ReplaceAll: true}},
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DraftPage.Title != nil || output.DraftPage.Body != "baz bar baz" {
			t.Errorf("下書き = (%v, %q)、期待値 = (nil, \"baz bar baz\")", output.DraftPage.Title, output.DraftPage.Body)
		}
		// 置換は下書きの保存で、未公開のページを公開しない
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND published_at IS NULL", pageID); n != 1 {
			t.Error("未公開のページが公開された")
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = '' AND body = 'baz bar baz'", draftID); n != 1 {
			t.Errorf("空のタイトルで記録したリビジョンの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("置換しても本文が変わらなければ下書きを書き換えず、最新のリビジョンと同じならリビジョンも作らない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAPIPageFixture(t, db, "api-draft-body-edit-noop", memberRole)
		pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").Build()
		draftID := testutil.NewDraftPageBuilderDB(t, db).
			WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
			WithTitle("Draft Revision Title").WithBody("Draft revision body").Build()
		// ビルダーの既定のタイトルと本文で、下書きと同じ内容のリビジョンを作る
		testutil.NewDraftPageRevisionBuilderDB(t, db).WithSpaceID(f.space.ID).WithDraftPageID(draftID).WithSpaceMemberID(f.spaceMemberID).Build()
		before := draftPageDigest(t, db, draftID, f.space.ID)

		output, err := newCreateAPIDraftPageBodyEditUC(db).Execute(t.Context(), CreateAPIDraftPageBodyEditInput{
			Principal:       f.principal(memberRole, draftPageWriteScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Edits:           []model.BodyEdit{{OldStr: "revision", NewStr: "revision"}},
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DraftPage.ContentDigest() != before || draftPageDigest(t, db, draftID, f.space.ID) != before {
			t.Error("本文を変えない置換で下書きの版が変わった")
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != 1 {
			t.Errorf("リビジョンの数 = %d、期待値 = 1 (増えない)", n)
		}
	})
}

func TestCreateAPIDraftPageBodyEditUsecase_Execute_Mismatch(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole

	tests := []struct {
		name        string
		edits       []model.BodyEdit
		wantField   string
		wantMessage string
	}{
		{
			name: "一致が0件の編集があれば、前の編集も適用せずに何番目かを返す",
			edits: []model.BodyEdit{
				{OldStr: "foo", NewStr: "[[作らないページ]]"},
				{OldStr: "無い", NewStr: "y"},
			},
			wantField:   "edits/1/old_str",
			wantMessage: "一致する箇所が本文にありません",
		},
		{
			name:        "replace_allでなく一致が複数件なら、一致の件数を返す",
			edits:       []model.BodyEdit{{OldStr: "bar", NewStr: "x"}},
			wantField:   "edits/0/old_str",
			wantMessage: "2件あります",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-draft-body-edit-mis-"+string(rune('a'+i)), memberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").Build()
			draftID := testutil.NewDraftPageBuilderDB(t, db).
				WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
				WithTitle("下書き").WithBody("foo bar bar").Build()
			before := draftPageDigest(t, db, draftID, f.space.ID)

			output, err := newCreateAPIDraftPageBodyEditUC(db).Execute(t.Context(), CreateAPIDraftPageBodyEditInput{
				Principal:       f.principal(memberRole, draftPageWriteScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				Edits:           tt.edits,
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("Execute()のエラー = %v、期待値 = ValidationError", err)
			}
			if len(ve.Fields) != 1 || len(ve.GetFieldErrors(tt.wantField)) != 1 {
				t.Fatalf("Fields = %v、期待値 = %s だけのエラー", ve.Fields, tt.wantField)
			}
			if msg := ve.GetFieldErrors(tt.wantField)[0]; !strings.Contains(msg, tt.wantMessage) {
				t.Errorf("メッセージ = %q、期待値 = %q を含む", msg, tt.wantMessage)
			}
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			if draftPageDigest(t, db, draftID, f.space.ID) != before {
				t.Error("下書きが更新された")
			}
			if n := countRows(t, db, "draft_page_revisions", f.space.ID, ""); n != 0 {
				t.Errorf("リビジョンの数 = %d、期待値 = 0", n)
			}
			if n := countRows(t, db, "pages", f.space.ID, ""); n != 1 {
				t.Errorf("ページの数 = %d、期待値 = 1 (Wikiリンク先も作らない)", n)
			}
		})
	}
}

func TestCreateAPIDraftPageBodyEditUsecase_Execute_Precondition(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole

	tests := []struct {
		name       string
		hasIfMatch bool
		digests    func(current string) []string
		any        bool
		wantFailed bool
	}{
		{name: "If-Matchが無ければ版を照合せずに置換する"},
		{name: "今の版と一致すれば置換する", hasIfMatch: true, digests: func(current string) []string { return []string{current} }},
		{name: "`*` なら版を問わず置換する", hasIfMatch: true, any: true},
		{name: "今の版と一致しなければ412", hasIfMatch: true, digests: func(string) []string { return []string{"古い版"} }, wantFailed: true},
		{name: "If-Matchがあって候補が無ければ412", hasIfMatch: true, wantFailed: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, "api-draft-body-edit-pre-"+string(rune('a'+i)), memberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("ページ").Build()
			draftID := testutil.NewDraftPageBuilderDB(t, db).
				WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
				WithTitle("下書き").WithBody("本文").Build()
			current := draftPageDigest(t, db, draftID, f.space.ID)
			var digests []string
			if tt.digests != nil {
				digests = tt.digests(current)
			}

			output, err := newCreateAPIDraftPageBodyEditUC(db).Execute(t.Context(), CreateAPIDraftPageBodyEditInput{
				Principal:       f.principal(memberRole, draftPageWriteScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				HasIfMatch:      tt.hasIfMatch,
				IfMatchDigests:  digests,
				IfMatchAny:      tt.any,
				Edits:           []model.BodyEdit{{OldStr: "本文", NewStr: "新しい本文"}},
			})

			if !tt.wantFailed {
				if err != nil {
					t.Fatalf("Execute()のエラー = %v", err)
				}
				if output.DraftPage.Body != "新しい本文" {
					t.Errorf("本文 = %q、期待値 = 新しい本文", output.DraftPage.Body)
				}
				return
			}
			if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
				t.Errorf("Execute()のエラー = %v、期待値 = AppErrCodePreconditionFailed", err)
			}
			if draftPageDigest(t, db, draftID, f.space.ID) != current {
				t.Error("下書きが置換された")
			}
		})
	}
}

func TestCreateAPIDraftPageBodyEditUsecase_Execute_NotFoundAndForbidden(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-body-edit-denied", memberRole)
	ownPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("自分の下書きのページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(ownPageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).WithBody("本文").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).WithTitle("下書きの無いページ").WithBody("本文").Build()
	otherUserID := testutil.NewUserBuilderDB(t, db).WithAtname("api_draft_body_edit_other").WithEmail("api_draft_body_edit_other@example.com").Build()
	otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(f.space.ID).WithUserID(otherUserID).Build()
	otherPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(3).WithTitle("他のメンバーの下書きのページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(otherPageID).WithSpaceMemberID(otherMemberID).WithTopicID(f.topicIDs[1]).WithBody("本文").Build()

	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		pageNumber  int32
		wantCode    model.AppErrorCode
	}{
		{name: "下書きの無いページは未存在", memberRole: memberRole, tokenScopes: draftPageWriteScopes, pageNumber: 2, wantCode: model.AppErrCodeResourceNotFound},
		{name: "ほかのメンバーの下書きは未存在", memberRole: memberRole, tokenScopes: draftPageWriteScopes, pageNumber: 3, wantCode: model.AppErrCodeResourceNotFound},
		{name: "draft_page:writeの無いトークンは置換できない", memberRole: memberRole, tokenScopes: []model.Scope{model.ScopeDraftPageRead, model.ScopePageWrite}, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
		{name: "下書きを保存できないロールは置換できない", memberRole: model.SpaceRoleViewer, tokenScopes: draftPageWriteScopes, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := newCreateAPIDraftPageBodyEditUC(db).Execute(t.Context(), CreateAPIDraftPageBodyEditInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      tt.pageNumber,
				Edits:           []model.BodyEdit{{OldStr: "本文", NewStr: "置換できない本文"}},
			})
			if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
				t.Fatalf("Execute()のエラー = %v、期待値 = %v", err, tt.wantCode)
			}
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
		})
	}

	if n := countRows(t, db, "draft_pages", f.space.ID, "body = '置換できない本文'"); n != 0 {
		t.Errorf("書き換えられた下書きの数 = %d、期待値 = 0", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, "body = '置換できない本文'"); n != 0 {
		t.Errorf("書き換えられたページの数 = %d、期待値 = 0", n)
	}
}

func TestCreateAPIDraftPageBodyEditUsecase_Execute_ConcurrentWebSave(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	tests := []struct {
		name       string
		identifier string
		webBody    string
		wantBody   string
		wantFailed bool
	}{
		{
			name:       "Webの変更を保持した本文へ置換を適用し直す",
			identifier: "api-draft-body-edit-concurrent-ok",
			webBody:    "置換対象\nWebの補足",
			wantBody:   "[[確定リンク先]]\nWebの補足",
		},
		{
			name:       "Webが置換対象を消したら422でWebの保存だけが残る",
			identifier: "api-draft-body-edit-concurrent-missing",
			webBody:    "Webが書き換えた本文",
			wantBody:   "Webが書き換えた本文",
			wantFailed: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAPIPageFixture(t, db, tt.identifier, memberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()
			draftID := testutil.NewDraftPageBuilderDB(t, db).
				WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
				WithTitle("下書き").WithBody("置換対象\n[[試行だけのリンク先]]").Build()

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			webTx, webPID := beginWebTx(t, db)
			q := query.New(db).WithTx(webTx)
			// Webの保存を未コミットにして下書きをロックする。APIの通常の取得には変更前の本文が見え、
			// リンク先の作成を終えて下書きをロックする時点でWebのコミットを待つ。
			webDraft, err := repository.NewDraftPageRepository(q).Update(ctx, repository.UpdateDraftPageInput{
				ID: draftID, SpaceID: f.space.ID, TopicID: f.topicIDs[1],
				Title: ptr("Webの下書き"), Body: tt.webBody, ModifiedAt: time.Now(),
			})
			if err != nil {
				t.Fatalf("Web側の下書きの更新: %v", err)
			}
			if _, err := repository.NewDraftPageRevisionRepository(q).Create(ctx, repository.CreateDraftPageRevisionInput{
				DraftPageID: draftID, SpaceID: f.space.ID, SpaceMemberID: f.spaceMemberID,
				Title: "Webの下書き", Body: tt.webBody,
			}); err != nil {
				t.Fatalf("Web側のリビジョンの作成: %v", err)
			}

			type result struct {
				output *UpdateAPIDraftPageOutput
				err    error
			}
			results := make(chan result, 1)
			go func() {
				output, err := newCreateAPIDraftPageBodyEditUC(db).Execute(ctx, CreateAPIDraftPageBodyEditInput{
					Principal: f.principal(memberRole, draftPageWriteScopes), SpaceIdentifier: f.space.Identifier,
					PageNumber: 1, Edits: []model.BodyEdit{{OldStr: "置換対象", NewStr: "[[確定リンク先]]"}},
				})
				results <- result{output: output, err: err}
			}()

			// 時間の経過だけに頼らず、APIが置換とリンク先の作成を終えてWebの下書きを待ったことを確認する。
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for blocked := false; !blocked; {
				err := db.QueryRowContext(ctx, `SELECT EXISTS (
					SELECT 1 FROM pg_stat_activity
					WHERE datname = current_database()
					  AND $1 = ANY(pg_blocking_pids(pid))
					  AND query LIKE '%FindDraftPageByIDForUpdate%'
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
					t.Fatalf("APIがWebの下書きを待たなかった: %v", ctx.Err())
				case <-ticker.C:
				}
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
			wantRevisions := 2
			wantPages := 2
			if tt.wantFailed {
				ve := model.AsValidationError(r.err)
				if ve == nil || len(ve.Fields) != 1 || len(ve.GetFieldErrors("edits/0/old_str")) != 1 {
					t.Fatalf("APIのエラー = %v、期待値 = edits/0/old_strのValidationError (422)", r.err)
				}
				if r.output != nil {
					t.Errorf("output = %+v、期待値 = nil", r.output)
				}
				if got := draftPageDigest(t, db, draftID, f.space.ID); got != webDraft.ContentDigest() {
					t.Errorf("下書きの版 = %q、期待値 = Webの保存した版 %q", got, webDraft.ContentDigest())
				}
				wantRevisions = 1
				wantPages = 1
			} else {
				if r.err != nil {
					t.Fatalf("Execute()のエラー = %v", r.err)
				}
				if r.output == nil || r.output.DraftPage == nil {
					t.Fatal("置換後の下書きが返らなかった")
				}
				if draft := r.output.DraftPage; draft.Title == nil || *draft.Title != "Webの下書き" || draft.Body != tt.wantBody {
					t.Errorf("下書き = %+v、期待値 = Webのタイトルと補足を保持した本文 %q", draft, tt.wantBody)
				}
				if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = 'Webの下書き' AND body = $3", draftID, tt.wantBody); n != 1 {
					t.Errorf("置換後のリビジョンの数 = %d、期待値 = 1", n)
				}
				if n := countRows(t, db, "pages", f.space.ID, "title = '確定リンク先' AND published_at IS NULL"); n != 1 {
					t.Errorf("確定したリンク先の数 = %d、期待値 = 1", n)
				}
			}
			if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2 AND title = 'Webの下書き' AND body = $3", draftID, tt.wantBody); n != 1 {
				t.Errorf("最終本文を保存した下書きの数 = %d、期待値 = 1", n)
			}
			if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != wantRevisions {
				t.Errorf("リビジョンの数 = %d、期待値 = %d (失敗した試行は残さない)", n, wantRevisions)
			}
			if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2 AND title = 'Webの下書き' AND body = $3", draftID, tt.webBody); n != 1 {
				t.Errorf("Webのリビジョンの数 = %d、期待値 = 1", n)
			}
			if n := countRows(t, db, "pages", f.space.ID, "title = '試行だけのリンク先'"); n != 0 {
				t.Errorf("失敗した試行のリンク先の数 = %d、期待値 = 0", n)
			}
			if n := countRows(t, db, "pages", f.space.ID, ""); n != wantPages {
				t.Errorf("ページの数 = %d、期待値 = %d", n, wantPages)
			}
		})
	}
}
