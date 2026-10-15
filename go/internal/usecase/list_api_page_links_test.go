package usecase

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// apiPageLinkFixtureはリンク一覧・バックリンク一覧のテストのスペースと、番号ごとのページのID
type apiPageLinkFixture struct {
	apiTopicFixture
	pageIDs map[model.PageNumber]model.PageID
}

// setupAPIPageLinkFixtureは、setupAPITopicFixtureのトピックに次のページを作る。
// 番号が小さいほど更新日時が新しい (番号nのページの更新日時は基準日時のn時間前)
//   - 1: 公開トピックの公開済みのページ。2〜8と自身にリンクし、2・4〜8と自身からリンクされる
//   - 2: 公開トピックの公開済みのページ
//   - 3: 公開トピックの未公開のページ (Wikiリンクで自動作成されたページ)
//   - 4: 公開トピックのゴミ箱のページ
//   - 5: 公開トピックの廃棄済みのページ
//   - 6: 参加している非公開トピックのページ
//   - 7: 参加していない非公開トピックのページ (一般メンバーもトークンが `topic:read` を持てば見られる)
//   - 8: 廃棄済みのトピックのページ
//   - 9: 公開トピックの公開済みのページ。1とはメンバーの下書きでだけ互いにリンクする
func setupAPIPageLinkFixture(t *testing.T, tx *sql.Tx, identifier string) apiPageLinkFixture {
	t.Helper()

	f := apiPageLinkFixture{
		apiTopicFixture: setupAPITopicFixture(t, tx, identifier, apiTopicRegularMemberRole),
		pageIDs:         map[model.PageNumber]model.PageID{},
	}
	pages := []struct {
		number      model.PageNumber
		topicNumber int32
		modify      func(b *testutil.PageBuilder) *testutil.PageBuilder
	}{
		{number: 1, topicNumber: 1},
		{number: 2, topicNumber: 1},
		{number: 3, topicNumber: 1, modify: (*testutil.PageBuilder).WithUnpublished},
		{number: 4, topicNumber: 1, modify: (*testutil.PageBuilder).WithTrashed},
		{number: 5, topicNumber: 1, modify: (*testutil.PageBuilder).WithDiscarded},
		{number: 6, topicNumber: 2},
		{number: 7, topicNumber: 3},
		{number: 8, topicNumber: 4},
		{number: 9, topicNumber: 1},
	}
	for _, p := range pages {
		b := testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithTopicID(f.topicIDs[p.topicNumber]).
			WithNumber(p.number).
			WithTitle(fmt.Sprintf("ページ%d", p.number)).
			WithModifiedAt(apiPageBaseTime.Add(-time.Duration(p.number) * time.Hour))
		if p.modify != nil {
			b = p.modify(b)
		}
		f.pageIDs[p.number] = b.Build()
	}

	f.setLinkedPages(t, tx, 1, 1, 2, 3, 4, 5, 6, 7, 8)
	for _, number := range []model.PageNumber{2, 4, 5, 6, 7, 8} {
		f.setLinkedPages(t, tx, number, 1)
	}
	for _, link := range []struct{ from, to model.PageNumber }{{from: 1, to: 9}, {from: 9, to: 1}} {
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithPageID(f.pageIDs[link.from]).
			WithSpaceMemberID(f.spaceMemberID).
			WithTopicID(f.topicIDs[1]).
			WithLinkedPageIDs([]model.PageID{f.pageIDs[link.to]}).
			Build()
	}
	return f
}

// setLinkedPagesはページの保存済みのリンク先を書き換える。
// 互いにリンクするページはIDが決まってからでないと作れないため、作成後に書き換える
func (f apiPageLinkFixture) setLinkedPages(t *testing.T, tx *sql.Tx, from model.PageNumber, to ...model.PageNumber) {
	t.Helper()

	ids := make([]string, len(to))
	for i, number := range to {
		ids[i] = string(f.pageIDs[number])
	}
	if _, err := tx.ExecContext(t.Context(), "UPDATE pages SET linked_page_ids = $1 WHERE id = $2 AND space_id = $3", pq.Array(ids), string(f.pageIDs[from]), string(f.space.ID)); err != nil {
		t.Fatalf("リンク先の書き換えに失敗: %v", err)
	}
}

func newListAPIPageLinksUC(tx *sql.Tx) *ListAPIPageLinksUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewListAPIPageLinksUsecase(repository.NewPageRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

func TestListAPIPageLinksUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPIPageLinkFixture(t, tx, "api-page-links")
	uc := newListAPIPageLinksUC(tx)

	// サブテストは親のトランザクションを共有するため、並列にしない

	tests := []struct {
		name        string
		tokenScopes []model.Scope
		pageNumber  int32
		want        []model.PageNumber
	}{
		{
			// 2・3・6・7は保存済みの本文のリンク先。自身へのリンクも含め、下書きにだけあるリンク先 (9) は含めない。
			// ゴミ箱・廃棄済みのページ (4・5) と廃棄済みのトピックのページ (8) は除く
			name:        "保存済みの本文のリンク先のうち見られるページを返す",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			pageNumber:  1,
			want:        []model.PageNumber{1, 2, 3, 6, 7},
		},
		{
			name:        "topic:readの無いトークンでは非公開トピックのページを含めない",
			tokenScopes: []model.Scope{model.ScopePageRead},
			pageNumber:  1,
			want:        []model.PageNumber{1, 2, 3},
		},
		{
			name:        "下書きにだけリンクがあるページは空の一覧を返す",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			pageNumber:  9,
			want:        []model.PageNumber{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(t.Context(), ListAPIPageLinksInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      tt.pageNumber,
				Limit:           20,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			assertPageNumbers(t, output.Pages, tt.want)
			if output.HasNext {
				t.Error("HasNext = true、期待値 = false")
			}
			for _, page := range output.Pages {
				if output.Topics[page.TopicID] == nil {
					t.Errorf("ページ%dのトピックがありません", page.Number)
				}
			}
		})
	}

	t.Run("リンク先の未公開のページを未公開のまま返す", func(t *testing.T) {
		output, err := uc.Execute(t.Context(), ListAPIPageLinksInput{
			Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopePageRead}),
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			Limit:           20,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		for _, page := range output.Pages {
			if page.Number == 3 && page.IsPublished() {
				t.Error("ページ3が公開済みになっています、期待値 = 未公開")
			}
		}
	})

	t.Run("カーソルで全件を重複なく辿れる", func(t *testing.T) {
		var got []model.PageNumber
		var after *APIPageCursor
		for range 10 {
			output, err := uc.Execute(t.Context(), ListAPIPageLinksInput{
				Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopePageRead, model.ScopeTopicRead}),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				After:           after,
				Limit:           3,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			for _, page := range output.Pages {
				got = append(got, page.Number)
			}
			if !output.HasNext {
				break
			}
			last := output.Pages[len(output.Pages)-1]
			after = &APIPageCursor{ModifiedAt: last.ModifiedAt, PageID: last.ID}
		}
		assertPageNumberSlice(t, got, []model.PageNumber{1, 2, 3, 6, 7})
	})

	assertAPIPageLinksNotFound(t, func(principal *model.APIPrincipal, spaceIdentifier model.SpaceIdentifier, pageNumber int32) error {
		_, err := uc.Execute(t.Context(), ListAPIPageLinksInput{
			Principal:       principal,
			SpaceIdentifier: spaceIdentifier,
			PageNumber:      pageNumber,
			Limit:           20,
		})
		return err
	}, f)
}

// assertAPIPageLinksNotFoundは、起点のページが存在しない・見られない場合に未存在のエラーを返すことを確かめる
func assertAPIPageLinksNotFound(t *testing.T, execute func(principal *model.APIPrincipal, spaceIdentifier model.SpaceIdentifier, pageNumber int32) error, f apiPageLinkFixture) {
	t.Helper()

	tests := []struct {
		name            string
		memberRole      model.SpaceRole
		tokenScopes     []model.Scope
		spaceIdentifier model.SpaceIdentifier
		pageNumber      int32
	}{
		{name: "存在しないページ", memberRole: apiTopicRegularMemberRole, tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead}, pageNumber: 99},
		// 管理者はWeb画面でゴミ箱を開けるが、APIではゴミ箱のページを返さない
		{name: "ゴミ箱のページ", memberRole: apiTopicAdminMemberRole, tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead}, pageNumber: 4},
		{name: "廃棄済みのページ", memberRole: apiTopicRegularMemberRole, tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead}, pageNumber: 5},
		{name: "topic:readの無いトークンでの非公開トピックのページ", memberRole: apiTopicRegularMemberRole, tokenScopes: []model.Scope{model.ScopePageRead}, pageNumber: 6},
		{name: "廃棄済みのトピックのページ", memberRole: apiTopicRegularMemberRole, tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead}, pageNumber: 8},
		{name: "トークンが属さないスペース", memberRole: apiTopicRegularMemberRole, tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead}, spaceIdentifier: "api-page-links-other", pageNumber: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name+"は未存在のエラーを返す", func(t *testing.T) {
			spaceIdentifier := f.space.Identifier
			if tt.spaceIdentifier != "" {
				spaceIdentifier = tt.spaceIdentifier
			}
			err := execute(f.principal(tt.memberRole, tt.tokenScopes), spaceIdentifier, tt.pageNumber)
			if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
				t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
			}
		})
	}
}
