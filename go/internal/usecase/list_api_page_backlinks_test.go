package usecase

import (
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newListAPIPageBacklinksUC(tx *sql.Tx) *ListAPIPageBacklinksUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewListAPIPageBacklinksUsecase(repository.NewPageRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

func TestListAPIPageBacklinksUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPIPageLinkFixture(t, tx, "api-page-backlinks")
	uc := newListAPIPageBacklinksUC(tx)

	// サブテストは親のトランザクションを共有するため、並列にしない

	tests := []struct {
		name        string
		tokenScopes []model.Scope
		pageNumber  int32
		want        []model.PageNumber
	}{
		{
			// 2・6・7は保存済みの本文でリンクしているページ。自身からのリンクも含め、下書きにだけあるリンク元 (9) は含めない。
			// ゴミ箱・廃棄済みのページ (4・5) と廃棄済みのトピックのページ (8) は除く
			name:        "保存済みの本文でリンクしているページのうち見られるページを返す",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			pageNumber:  1,
			want:        []model.PageNumber{1, 2, 6, 7},
		},
		{
			name:        "topic:readの無いトークンでは非公開トピックのページを含めない",
			tokenScopes: []model.Scope{model.ScopePageRead},
			pageNumber:  1,
			want:        []model.PageNumber{1, 2},
		},
		{
			name:        "下書きからだけリンクされているページは空の一覧を返す",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			pageNumber:  9,
			want:        []model.PageNumber{},
		},
		{
			name:        "未公開のページのバックリンクを返す",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			pageNumber:  3,
			want:        []model.PageNumber{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(t.Context(), ListAPIPageBacklinksInput{
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

	t.Run("カーソルで全件を重複なく辿れる", func(t *testing.T) {
		var got []model.PageNumber
		var after *APIPageCursor
		for range 10 {
			output, err := uc.Execute(t.Context(), ListAPIPageBacklinksInput{
				Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopePageRead, model.ScopeTopicRead}),
				SpaceIdentifier: f.space.Identifier,
				PageNumber:      1,
				After:           after,
				Limit:           2,
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
		assertPageNumberSlice(t, got, []model.PageNumber{1, 2, 6, 7})
	})

	assertAPIPageLinksNotFound(t, func(principal *model.APIPrincipal, spaceIdentifier model.SpaceIdentifier, pageNumber int32) error {
		_, err := uc.Execute(t.Context(), ListAPIPageBacklinksInput{
			Principal:       principal,
			SpaceIdentifier: spaceIdentifier,
			PageNumber:      pageNumber,
			Limit:           20,
		})
		return err
	}, f)
}
