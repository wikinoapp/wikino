package usecase

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newCreateAPIDraftPagePublicationUC(db *sql.DB) *CreateAPIDraftPagePublicationUsecase {
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	return NewCreateAPIDraftPagePublicationUsecase(
		db,
		repository.NewSpaceRepository(q),
		pageRepo,
		repository.NewPageRevisionRepository(q),
		repository.NewPageEditorRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
		repository.NewPageAttachmentReferenceRepository(q),
		validator.NewAPIPageUpdateValidator(pageRepo),
	)
}

// draftPagePublishScopesは下書きを公開できるトークンのスコープ
var draftPagePublishScopes = []model.Scope{model.ScopeDraftPageWrite, model.ScopePageWrite}

func TestCreateAPIDraftPagePublicationUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-publish", memberRole)
	publishedAt := time.Now().Add(-24 * time.Hour)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
		WithTitle("ページ").WithBody("ページの本文").WithPublishedAt(publishedAt).Build()
	draftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
		WithTitle("新しいタイトル").WithBody("下書きの本文 [[リンク先]]").Build()
	testutil.NewDraftPageRevisionBuilderDB(t, db).WithDraftPageID(draftID).WithSpaceID(f.space.ID).WithSpaceMemberID(f.spaceMemberID).Build()
	uc := newCreateAPIDraftPagePublicationUC(db)
	digest := draftPageDigest(t, db, draftID, f.space.ID)

	// サブテストは同じ下書きを順に扱うため、並列にしない。
	t.Run("古い版を前提にした公開は412で、ページと下書きを変えない", func(t *testing.T) {
		output, err := uc.Execute(t.Context(), CreateAPIDraftPagePublicationInput{
			Principal:       f.principal(memberRole, draftPagePublishScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{"古い版"},
		})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodePreconditionFailed {
			t.Fatalf("Execute()のエラー = %v、期待値 = AppErrCodePreconditionFailed (412)", err)
		}
		if output != nil {
			t.Errorf("output = %+v、期待値 = nil", output)
		}
		if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", draftID); n != 1 {
			t.Errorf("残った下書きの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "pages", f.space.ID, "id = $2 AND title = 'ページ' AND body = 'ページの本文'", pageID); n != 1 {
			t.Errorf("元のままのページの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("下書きの内容でページを公開し直し、下書きを編集履歴ごと削除する", func(t *testing.T) {
		output, err := uc.Execute(t.Context(), CreateAPIDraftPagePublicationInput{
			Principal:       f.principal(memberRole, draftPagePublishScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{digest},
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Page.ID != pageID || output.Page.Title == nil || *output.Page.Title != "新しいタイトル" || output.Page.Body != "下書きの本文 [[リンク先]]" {
			t.Errorf("ページ = %+v、期待値 = 下書きのタイトルと本文にしたページ1", output.Page)
		}
		if output.Topic == nil || output.Topic.ID != f.topicIDs[1] {
			t.Errorf("トピック = %+v、期待値 = ページのトピック", output.Topic)
		}
		if output.Page.PublishedAt == nil || !output.Page.PublishedAt.After(publishedAt) {
			t.Errorf("公開日時 = %v、期待値 = 元の公開日時 (%v) より後", output.Page.PublishedAt, publishedAt)
		}
		if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", draftID); n != 0 {
			t.Errorf("残った下書きの数 = %d、期待値 = 0", n)
		}
		if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != 0 {
			t.Errorf("残った下書きのリビジョンの数 = %d、期待値 = 0", n)
		}
		if n := countRows(t, db, "page_revisions", f.space.ID, "page_id = $2 AND title = '新しいタイトル'", pageID); n != 1 {
			t.Errorf("公開した内容のページのリビジョンの数 = %d、期待値 = 1", n)
		}
		if n := countRows(t, db, "pages", f.space.ID, "title = 'リンク先' AND published_at IS NULL"); n != 1 {
			t.Errorf("未公開で作られたリンク先のページの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("公開済みの下書きをもう一度公開すると未存在", func(t *testing.T) {
		_, err := uc.Execute(t.Context(), CreateAPIDraftPagePublicationInput{
			Principal:       f.principal(memberRole, draftPagePublishScopes),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			IfMatchDigests:  []string{digest},
		})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Fatalf("Execute()のエラー = %v、期待値 = AppErrCodeResourceNotFound (404)", err)
		}
	})
}

// TestCreateAPIDraftPagePublicationUsecase_Execute_Unpublishedは、Wikiリンクで作られたタイトルの無い未公開のページを
// 下書きのタイトルで公開できることを確かめる
func TestCreateAPIDraftPagePublicationUsecase_Execute_Unpublished(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-publish-unpublished", memberRole)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
		WithNilTitle().WithBody("").WithUnpublished().Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
		WithTitle("新しいページ").WithBody("本文").Build()

	output, err := newCreateAPIDraftPagePublicationUC(db).Execute(t.Context(), CreateAPIDraftPagePublicationInput{
		Principal:       f.principal(memberRole, draftPagePublishScopes),
		SpaceIdentifier: f.space.Identifier,
		PageNumber:      1,
		IfMatchAny:      true,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if !output.Page.IsPublished() || output.Page.Title == nil || *output.Page.Title != "新しいページ" {
		t.Errorf("ページ = %+v、期待値 = 「新しいページ」として公開したページ", output.Page)
	}
}

func TestCreateAPIDraftPagePublicationUsecase_Execute_Validation(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-publish-invalid", memberRole)
	untitledPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
		WithNilTitle().WithBody("").WithUnpublished().Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(untitledPageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
		WithNilTitle().WithBody("タイトルの無い下書き").Build()
	duplicatePageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(2).WithTitle("ページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(duplicatePageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
		WithTitle("重複").WithBody("重複するタイトルの下書き").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(3).WithTitle("重複").Build()

	tests := []struct {
		name       string
		pageNumber int32
	}{
		{name: "タイトルの無い下書きは422", pageNumber: 1},
		{name: "トピック内の他のページと同じタイトルの下書きは422", pageNumber: 2},
	}
	// 下書きが残ることを全ケースの後に確かめるため、サブテストを並列にしない。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := newCreateAPIDraftPagePublicationUC(db).Execute(t.Context(), CreateAPIDraftPagePublicationInput{
				Principal:       f.principal(memberRole, draftPagePublishScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      tt.pageNumber,
				IfMatchAny:      true,
			})
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("title") {
				t.Errorf("エラー = %v、期待値 = titleのValidationError", err)
			}
		})
	}

	if n := countRows(t, db, "draft_pages", f.space.ID, ""); n != 2 {
		t.Errorf("残った下書きの数 = %d、期待値 = 2", n)
	}
}

// ページの移動との読み取りのずれは取得からやり直して解消するため、やり直しても所属トピックが一致しない
// 下書きは上限まで試した後に失敗し、ページを下書きのトピックへ移さないことを確かめる
func TestCreateAPIDraftPagePublicationUsecase_Execute_TopicMismatch(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-publish-topic-mismatch", memberRole)
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
		WithTitle("ページ").WithBody("ページの本文").Build()
	otherTopicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(f.space.ID).WithNumber(4).WithName("別の公開").Build()
	draftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(otherTopicID).
		WithTitle("下書きのタイトル").WithBody("下書きの本文").Build()
	before := findPageForTest(t, db, f.space.ID, 1)

	output, err := newCreateAPIDraftPagePublicationUC(db).Execute(t.Context(), CreateAPIDraftPagePublicationInput{
		Principal:       f.principal(memberRole, draftPagePublishScopes),
		SpaceIdentifier: f.space.Identifier,
		PageNumber:      1,
		IfMatchAny:      true,
	})
	if output != nil {
		t.Errorf("output = %+v、期待値 = nil", output)
	}
	if !errors.Is(err, errAPIDraftPageChanged) || model.AsAppError(err) != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = やり直しの上限に達したerrAPIDraftPageChanged", err)
	}
	if after := findPageForTest(t, db, f.space.ID, 1); !reflect.DeepEqual(after, before) {
		t.Errorf("公開後のページ = %+v、期待値 = 公開前のページ %+v", after, before)
	}
	if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", draftID); n != 1 {
		t.Errorf("残った下書きの数 = %d、期待値 = 1", n)
	}
}

func TestCreateAPIDraftPagePublicationUsecase_Execute_NotFoundAndForbidden(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-draft-publish-denied", memberRole)
	ownPageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("自分の下書きのページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(f.space.ID).WithPageID(ownPageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).Build()
	otherUserID := testutil.NewUserBuilderDB(t, db).WithAtname("api_draft_publish_other").WithEmail("api_draft_publish_other@example.com").Build()
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
		{name: "ほかのメンバーの下書きは未存在", memberRole: memberRole, tokenScopes: draftPagePublishScopes, pageNumber: 2, wantCode: model.AppErrCodeResourceNotFound},
		{name: "page:writeの無いトークンは公開できない", memberRole: memberRole, tokenScopes: draftPageWriteScopes, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
		{name: "draft_page:writeの無いトークンは公開できない", memberRole: memberRole, tokenScopes: []model.Scope{model.ScopeDraftPageRead, model.ScopePageWrite}, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
		{name: "ページを更新できないロールは公開できない", memberRole: model.SpaceRoleViewer, tokenScopes: draftPagePublishScopes, pageNumber: 1, wantCode: model.AppErrCodeForbidden},
	}
	// 下書きが残ることを全ケースの後に確かめるため、サブテストを並列にしない。
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newCreateAPIDraftPagePublicationUC(db).Execute(t.Context(), CreateAPIDraftPagePublicationInput{
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

// 公開の前提を照合した後にWeb側の変更をコミットし、ページに書き込んだ内容まで取り消すことを確かめる。
// 下書きのロック待ちを観測して同期するため、処理速度に依存せずコミット直前の再照合を通る
func TestCreateAPIDraftPagePublicationUsecase_Execute_ConcurrentDraftChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		delete     bool
		wantCode   model.AppErrorCode
	}{
		{name: "下書きの更新が割り込むと412で公開を取り消す", identifier: "draft-publish-web-update", wantCode: model.AppErrCodePreconditionFailed},
		{name: "下書きの削除が割り込むと404で公開を取り消す", identifier: "draft-publish-web-delete", delete: true, wantCode: model.AppErrCodeResourceNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.GetTestDB()
			f := setupCreateAPIPageFixture(t, db, tt.identifier, apiTopicRegularMemberRole)
			pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).
				WithTitle("公開前のタイトル").WithBody("公開前の本文").WithPublishedAt(time.Now().Add(-24 * time.Hour)).Build()
			draftID := testutil.NewDraftPageBuilderDB(t, db).
				WithSpaceID(f.space.ID).WithPageID(pageID).WithSpaceMemberID(f.spaceMemberID).WithTopicID(f.topicIDs[1]).
				WithTitle("下書きのタイトル").WithBody("下書きの本文 [[公開で作るリンク先]]").Build()
			testutil.NewDraftPageRevisionBuilderDB(t, db).
				WithDraftPageID(draftID).WithSpaceID(f.space.ID).WithSpaceMemberID(f.spaceMemberID).Build()
			digest := draftPageDigest(t, db, draftID, f.space.ID)
			before := findPageForTest(t, db, f.space.ID, 1)
			rowCounts := make(map[string]int)
			for _, table := range []string{"pages", "page_revisions", "page_editors"} {
				rowCounts[table] = countRows(t, db, table, f.space.ID, "")
			}

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			webTx, webPID := beginWebTx(t, db)
			q := query.New(db).WithTx(webTx)
			draftRepo := repository.NewDraftPageRepository(q)
			revisionRepo := repository.NewDraftPageRevisionRepository(q)
			if tt.delete {
				if err := revisionRepo.DeleteByDraftPageID(ctx, draftID, f.space.ID); err != nil {
					t.Fatalf("Web側の下書き編集履歴の削除: %v", err)
				}
				if err := draftRepo.Delete(ctx, draftID, f.space.ID); err != nil {
					t.Fatalf("Web側の下書きの削除: %v", err)
				}
			} else {
				if _, err := draftRepo.Update(ctx, repository.UpdateDraftPageInput{
					ID: draftID, SpaceID: f.space.ID, TopicID: f.topicIDs[1],
					Title: strPtr("Web側のタイトル"), Body: "Web側で保存した本文", ModifiedAt: time.Now(),
				}); err != nil {
					t.Fatalf("Web側の下書きの更新: %v", err)
				}
				if _, err := revisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
					DraftPageID: draftID, SpaceID: f.space.ID, SpaceMemberID: f.spaceMemberID,
					Title: "Web側のタイトル", Body: "Web側で保存した本文",
				}); err != nil {
					t.Fatalf("Web側の下書き編集履歴の作成: %v", err)
				}
			}

			result := make(chan error, 1)
			go func() {
				_, err := newCreateAPIDraftPagePublicationUC(db).Execute(ctx, CreateAPIDraftPagePublicationInput{
					Principal:       f.principal(apiTopicRegularMemberRole, draftPagePublishScopes),
					SpaceIdentifier: f.space.Identifier, PageNumber: 1, IfMatchDigests: []string{digest},
				})
				result <- err
			}()
			waitForDraftDeletionBlocked(t, ctx, db, webPID, "FindDraftPageByIDForUpdate", result)
			if err := webTx.Commit(); err != nil {
				t.Fatalf("Web側トランザクションのコミット: %v", err)
			}
			select {
			case err := <-result:
				if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
					t.Fatalf("公開のエラー = %v、期待値 = %v", err, tt.wantCode)
				}
			case <-ctx.Done():
				t.Fatalf("競合解除後も公開が終了しなかった: %v", ctx.Err())
			}

			if after := findPageForTest(t, db, f.space.ID, 1); !reflect.DeepEqual(after, before) {
				t.Errorf("公開後のページ = %+v、期待値 = 公開前のページ %+v", after, before)
			}
			for table, want := range rowCounts {
				if got := countRows(t, db, table, f.space.ID, ""); got != want {
					t.Errorf("%sの行数 = %d、期待値 = 公開前の行数 %d", table, got, want)
				}
			}
			wantDrafts, wantRevisions := 1, 2
			if tt.delete {
				wantDrafts, wantRevisions = 0, 0
			} else if n := countRows(t, db, "draft_pages", f.space.ID,
				"id = $2 AND title = $3 AND body = $4", draftID, "Web側のタイトル", "Web側で保存した本文"); n != 1 {
				t.Errorf("Web側で更新した内容が残った下書きの数 = %d、期待値 = 1", n)
			}
			if n := countRows(t, db, "draft_pages", f.space.ID, "id = $2", draftID); n != wantDrafts {
				t.Errorf("残った下書きの数 = %d、期待値 = %d", n, wantDrafts)
			}
			if n := countRows(t, db, "draft_page_revisions", f.space.ID, "draft_page_id = $2", draftID); n != wantRevisions {
				t.Errorf("残った下書き編集履歴の数 = %d、期待値 = %d", n, wantRevisions)
			}
		})
	}
}
