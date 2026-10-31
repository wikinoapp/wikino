package usecase

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// 置き換えの付け替え後、コミット前に作った編集提案には、古いIDが残りうる。
// その提案を反映するときに本文からリンクを解決し直し、公開したページが置き換え先を指すことを確かめる
func TestApplySuggestion_CreationDuringReplacement(t *testing.T) {
	t.Parallel()
	db := testutil.GetTestDB()
	f := setupCreateAPIPageFixture(t, db, "apply-create-race-"+uuid.NewString()[:8], model.SpaceRoleEditor)
	spaceID, topicID := f.space.ID, f.topicIDs[1]
	topicMemberID := testutil.NewTopicMemberBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).
		WithSpaceMemberID(f.spaceMemberID).WithRole(model.TopicRoleEditor).Build()
	oldX := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).
		WithTitle("X").WithBody("").WithUnpublished().Build()
	replacement := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(2).
		WithTitle("元のX").WithBody("本文").Build()
	pageP := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(3).
		WithTitle("P").WithBody("公開済みの本文").Build()
	draftID := testutil.NewDraftPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithSpaceMemberID(f.spaceMemberID).
		WithPageID(pageP).WithTitle("P").WithBody("[[X]]").WithLinkedPageIDs([]model.PageID{oldX}).Build()
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	draftRepo := repository.NewDraftPageRepository(q)
	create := NewCreateSuggestionUsecase(db, repository.NewSpaceRepository(q), repository.NewSpaceMemberRepository(q),
		repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q), repository.NewSuggestionRepository(q),
		repository.NewSuggestionPageRepository(q), repository.NewSuggestionPageRevisionRepository(q), draftRepo,
		repository.NewPageRevisionRepository(q), validator.NewSuggestionCreateValidator(draftRepo, pageRepo))
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	blocker, blockerPID := beginWebTx(t, db)
	if _, err := blocker.ExecContext(ctx, "SELECT id FROM topic_members WHERE id = $1 AND space_id = $2 FOR NO KEY UPDATE", topicMemberID, spaceID); err != nil {
		t.Fatal(err)
	}
	results := make(chan concurrentPublicationResult, 2)
	go func() {
		_, err := newConcurrentPublishPageUC(db).Execute(ctx, PublishPageInput{
			SpaceIdentifier: f.space.Identifier, PageNumber: 2, UserID: f.userID, Title: "X", Body: "本文",
		})
		results <- concurrentPublicationResult{replacement, err}
	}()
	waitForPublicationsBlocked(t, ctx, db, blockerPID, results, "UpdateTopicMemberLastPageModifiedAt")
	var suggestionID model.SuggestionID
	go func() {
		output, err := create.Execute(ctx, CreateSuggestionInput{
			SpaceIdentifier: f.space.Identifier, TopicNumber: 1, UserID: f.userID, Title: "提案", DraftPageIDs: []model.DraftPageID{draftID},
		})
		if output != nil {
			suggestionID = output.Suggestion.ID
		}
		results <- concurrentPublicationResult{pageP, err}
	}()
	waitForPublicationsBlocked(t, ctx, db, blockerPID, results, "UpdateTopicMemberLastPageModifiedAt", "UpdateDraftPageSuggestionPageID")
	if err := blocker.Rollback(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if r := <-results; r.err != nil {
			t.Fatalf("並行処理に失敗: %v", r.err)
		}
	}
	draft, err := draftRepo.FindByID(ctx, draftID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	sp, err := repository.NewSuggestionPageRepository(q).ListBySuggestionID(ctx, suggestionID, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(draft.LinkedPageIDs, []model.PageID{replacement}) {
		t.Fatalf("付け替え後の下書きの記録 = %v、期待値 = [%v]", draft.LinkedPageIDs, replacement)
	}
	if len(sp) != 1 || !slices.Equal(sp[0].LinkedPageIDs, []model.PageID{oldX}) {
		t.Fatalf("編集提案が置き換えと重なって作成されていません: %v", sp)
	}
	if _, err := newConcurrentApplySuggestionUC(db).Execute(ctx, ApplySuggestionInput{
		SpaceIdentifier: f.space.Identifier, SuggestionNumber: 1, UserID: f.userID,
	}); err != nil {
		t.Fatal(err)
	}
	backlinks, err := pageRepo.FindBacklinkedByPageID(ctx, replacement, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(backlinks, func(p *model.Page) bool { return p.ID == pageP }) {
		t.Errorf("バックリンクに反映先のページが含まれていません: %v", pageP)
	}
	p := findPageForTest(t, db, spaceID, 3)
	if !slices.Equal(p.LinkedPageIDs, []model.PageID{replacement}) {
		t.Errorf("反映後の公開ページのリンク = %v、期待値 = [%v]", p.LinkedPageIDs, replacement)
	}
}

// 取得からスペースのロックまでの間に反映先のページが改名されても、編集提案のタイトルを書いてからリンクを解決し、
// 元のタイトルへのリンクで同じタイトルのページを自動作成しない
func TestApplySuggestion_RenamedBeforeLock(t *testing.T) {
	t.Parallel()
	db := testutil.GetTestDB()
	f := setupCreateAPIPageFixture(t, db, "apply-renamed-before-lock-"+uuid.NewString()[:8], model.SpaceRoleEditor)
	spaceID, topicID := f.space.ID, f.topicIDs[1]
	testutil.NewTopicMemberBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).
		WithSpaceMemberID(f.spaceMemberID).WithRole(model.TopicRoleEditor).Build()
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).
		WithTitle("ページ").WithBody("公開済みの本文").Build()
	suggestionID := testutil.NewSuggestionBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).
		WithCreatedSpaceMemberID(f.spaceMemberID).WithStatus(model.SuggestionStatusOpen).Build()
	testutil.NewSuggestionPageBuilderDB(t, db).WithSpaceID(spaceID).WithSuggestionID(suggestionID).WithPageID(pageID).
		WithTitle("ページ").WithBody("目次: [[ページ]]").WithLinkedPageIDs([]model.PageID{pageID}).Build()
	uc := newConcurrentApplySuggestionUC(db)
	data, err := uc.fetchData(t.Context(), ApplySuggestionInput{SpaceIdentifier: f.space.Identifier, SuggestionNumber: 1, UserID: f.userID})
	if err != nil {
		t.Fatal(err)
	}
	execForTest(t, db, "UPDATE pages SET title = '改名したページ' WHERE id = $1 AND space_id = $2", pageID, spaceID)
	if _, err := uc.applySuggestion(t.Context(), data, nil); err != nil {
		t.Fatalf("反映のエラー = %v", err)
	}

	page := findPageForTest(t, db, spaceID, 1)
	if derefOr(page.Title, "") != "ページ" || !slices.Equal(page.LinkedPageIDs, []model.PageID{pageID}) {
		t.Errorf("反映後のタイトルとリンク = %q / %v、期待値 = ページ / [%v]", derefOr(page.Title, ""), page.LinkedPageIDs, pageID)
	}
	if n := countRows(t, db, "pages", spaceID, "discarded_at IS NULL"); n != 1 {
		t.Errorf("未廃棄のページ数 = %d、期待値 = 1 (同じタイトルのページを作らない)", n)
	}
}

// 同じ提案で変えるすべてのタイトルを解決に使い、自分自身とほかの反映対象へのリンクを保つ
func TestApplySuggestion_ResolveLinksAfterRenamingAllPages(t *testing.T) {
	t.Parallel()
	db := testutil.GetTestDB()
	f := setupCreateAPIPageFixture(t, db, "apply-rename-links-"+uuid.NewString()[:8], model.SpaceRoleEditor)
	spaceID, topicID := f.space.ID, f.topicIDs[1]
	testutil.NewTopicMemberBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).
		WithSpaceMemberID(f.spaceMemberID).WithRole(model.TopicRoleEditor).Build()
	pageA := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("元のA").Build()
	pageB := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(2).WithTitle("元のB").Build()
	oldA := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(3).
		WithTitle("新しいA").WithBody("").WithUnpublished().Build()
	oldB := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(4).
		WithTitle("新しいB").WithBody("").WithUnpublished().Build()
	suggestionID := testutil.NewSuggestionBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).
		WithCreatedSpaceMemberID(f.spaceMemberID).WithStatus(model.SuggestionStatusOpen).Build()
	for _, page := range []struct {
		id    model.PageID
		title string
	}{{id: pageA, title: "新しいA"}, {id: pageB, title: "新しいB"}} {
		testutil.NewSuggestionPageBuilderDB(t, db).WithSpaceID(spaceID).WithSuggestionID(suggestionID).WithPageID(page.id).
			WithTitle(page.title).WithBody("[[新しいA]] [[新しいB]]").WithLinkedPageIDs([]model.PageID{oldA, oldB}).Build()
	}
	if _, err := newConcurrentApplySuggestionUC(db).Execute(t.Context(), ApplySuggestionInput{
		SpaceIdentifier: f.space.Identifier, SuggestionNumber: 1, UserID: f.userID,
	}); err != nil {
		t.Fatal(err)
	}
	for _, number := range []model.PageNumber{1, 2} {
		page := findPageForTest(t, db, spaceID, number)
		if !slices.Equal(page.LinkedPageIDs, []model.PageID{pageA, pageB}) {
			t.Errorf("ページ%dのリンク = %v、期待値 = [%v, %v]", number, page.LinkedPageIDs, pageA, pageB)
		}
	}
	if n := countRows(t, db, "pages", spaceID, "discarded_at IS NULL"); n != 2 {
		t.Errorf("未廃棄のページ数 = %d、期待値 = 2 (余分なリンク先を作らない)", n)
	}
}
