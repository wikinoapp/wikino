package usecase

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// apiPageBaseTimeはページのテストで更新日時の基準にする日時。
// pages.modified_atはマイクロ秒までのUTCで保存するため、その精度に収まる値にする
var apiPageBaseTime = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// setupAPIPageFixtureは、setupAPITopicFixtureのトピックに次のページを作る。
// 番号が小さいほど更新日時が新しい (番号nのページの更新日時は基準日時のn時間前)
//   - 1, 2: 公開トピックの公開済みのページ
//   - 3: 公開トピックの未公開のページ
//   - 4: 公開トピックのゴミ箱のページ
//   - 5: 公開トピックの廃棄済みのページ
//   - 6: 参加している非公開トピックのページ
//   - 7: 参加していない非公開トピックのページ
//   - 8: 廃棄済みのトピックのページ
func setupAPIPageFixture(t *testing.T, tx *sql.Tx, identifier string, memberRole model.SpaceRole) apiTopicFixture {
	t.Helper()

	f := setupAPITopicFixture(t, tx, identifier, memberRole)
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
		b.Build()
	}
	return f
}

func newListAPIPagesUC(tx *sql.Tx) *ListAPIPagesUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewListAPIPagesUsecase(repository.NewPageRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

func TestListAPIPagesUsecase_Execute_Filters(t *testing.T) {
	t.Parallel()

	int32Ptr := func(v int32) *int32 { return &v }
	timePtr := func(v time.Time) *time.Time { return &v }

	tests := []struct {
		name          string
		memberRole    model.SpaceRole
		tokenScopes   []model.Scope
		topicNumber   *int32
		modifiedSince *time.Time
		wantNumbers   []model.PageNumber
	}{
		{
			name:        "一般メンバーでトークンがtopic:readを持てば、参加していない非公開トピックのページも返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			wantNumbers: []model.PageNumber{1, 2, 6, 7},
		},
		{
			name:        "一般メンバーでトークンがtopic:readを持たなければ、公開トピックのページだけを返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead},
			wantNumbers: []model.PageNumber{1, 2},
		},
		{
			name:        "管理者でトークンがtopic:readを持てば、すべての非公開トピックのページを返す",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			wantNumbers: []model.PageNumber{1, 2, 6, 7},
		},
		{
			name:        "管理者でもトークンがtopic:readを持たなければ、公開トピックのページだけを返す",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite},
			wantNumbers: []model.PageNumber{1, 2},
		},
		{
			name:        "topic_numberでトピックを絞り込む",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber: int32Ptr(2),
			wantNumbers: []model.PageNumber{6},
		},
		{
			name:        "トークンがtopic:readを持たず開けない非公開トピックで絞り込むと空の一覧",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead},
			topicNumber: int32Ptr(3),
			wantNumbers: []model.PageNumber{},
		},
		{
			name:        "廃棄済みのトピックで絞り込むと空の一覧",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber: int32Ptr(4),
			wantNumbers: []model.PageNumber{},
		},
		{
			name:        "存在しないトピックで絞り込むと空の一覧",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber: int32Ptr(99),
			wantNumbers: []model.PageNumber{},
		},
		{
			name:          "modified_sinceより前に更新したページを除き、ちょうどの日時は含める",
			memberRole:    apiTopicAdminMemberRole,
			tokenScopes:   []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			modifiedSince: timePtr(apiPageBaseTime.Add(-2 * time.Hour)),
			wantNumbers:   []model.PageNumber{1, 2},
		},
		{
			name:          "modified_sinceはUTC以外のオフセットでも同じ時点として扱う",
			memberRole:    apiTopicAdminMemberRole,
			tokenScopes:   []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			modifiedSince: timePtr(apiPageBaseTime.Add(-2 * time.Hour).In(time.FixedZone("JST", 9*60*60))),
			wantNumbers:   []model.PageNumber{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIPageFixture(t, tx, "api-page-filters", tt.memberRole)

			output, err := newListAPIPagesUC(tx).Execute(t.Context(), ListAPIPagesInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     tt.topicNumber,
				ModifiedSince:   tt.modifiedSince,
				Limit:           20,
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			assertPageNumbers(t, output.Pages, tt.wantNumbers)
			if output.HasNext {
				t.Error("HasNext = true、期待値 = false")
			}
			for _, page := range output.Pages {
				topic, ok := output.Topics[page.TopicID]
				if !ok || topic.ID != page.TopicID {
					t.Errorf("ページ%dの所属トピックがTopicsにありません", page.Number)
				}
			}
		})
	}
}

func TestListAPIPagesUsecase_Execute_Cursor(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPIPageFixture(t, tx, "api-page-cursor", apiTopicAdminMemberRole)
	// 更新日時が同じページを並べ、主キーで順序が決まり重複も読み飛ばしも起きないことを確かめる
	sameModifiedAt := apiPageBaseTime.Add(-30 * time.Minute)
	for number := model.PageNumber(11); number <= 14; number++ {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithTopicID(f.topicIDs[1]).
			WithNumber(number).
			WithTitle(fmt.Sprintf("同時刻のページ%d", number)).
			WithModifiedAt(sameModifiedAt).
			Build()
	}
	principal := f.principal(apiTopicAdminMemberRole, []model.Scope{model.ScopePageRead, model.ScopeTopicRead})
	uc := newListAPIPagesUC(tx)

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("カーソルで全件を重複なく辿れる", func(t *testing.T) {
		seen := map[model.PageNumber]bool{}
		var got []model.PageNumber
		var after *APIPageCursor
		for range 20 {
			output, err := uc.Execute(t.Context(), ListAPIPagesInput{
				Principal:       principal,
				SpaceIdentifier: f.space.Identifier,
				After:           after,
				Limit:           3,
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(output.Pages) > 3 {
				t.Fatalf("1ページの件数 = %d、期待値 = 3以下", len(output.Pages))
			}
			for _, page := range output.Pages {
				if seen[page.Number] {
					t.Fatalf("ページ%dを重複して返しました (辿ったページ = %v)", page.Number, got)
				}
				seen[page.Number] = true
				got = append(got, page.Number)
			}
			if !output.HasNext {
				break
			}
			last := output.Pages[len(output.Pages)-1]
			after = &APIPageCursor{ModifiedAt: last.ModifiedAt, PageID: last.ID}
		}

		if len(got) != 8 {
			t.Fatalf("辿ったページ = %v、期待値 = 11〜14 (順不同) と1・2・6・7の8件", got)
		}
		// 同じ更新日時の4件が先頭に並び、その後は更新日時の新しい順になる
		for _, number := range got[:4] {
			if number < 11 || number > 14 {
				t.Fatalf("辿ったページ = %v、期待値 = 先頭の4件が11〜14", got)
			}
		}
		assertPageNumberSlice(t, got[4:], []model.PageNumber{1, 2, 6, 7})
	})

	t.Run("残りがちょうどlimit件なら次のページは無い", func(t *testing.T) {
		after := &APIPageCursor{ModifiedAt: apiPageBaseTime.Add(-time.Hour), PageID: "ffffffff-ffff-ffff-ffff-ffffffffffff"}
		output, err := uc.Execute(t.Context(), ListAPIPagesInput{
			Principal:       principal,
			SpaceIdentifier: f.space.Identifier,
			After:           after,
			Limit:           4,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		assertPageNumbers(t, output.Pages, []model.PageNumber{1, 2, 6, 7})
		if output.HasNext {
			t.Error("HasNext = true、期待値 = false")
		}
	})
}

func TestListAPIPagesUsecase_Execute_OtherSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPIPageFixture(t, tx, "api-page-bound", apiTopicAdminMemberRole)
	setupAPIPageFixture(t, tx, "api-page-other", apiTopicAdminMemberRole)

	output, err := newListAPIPagesUC(tx).Execute(t.Context(), ListAPIPagesInput{
		Principal:       f.principal(apiTopicAdminMemberRole, []model.Scope{model.ScopePageRead}),
		SpaceIdentifier: "api-page-other",
		Limit:           20,
	})
	if output != nil {
		t.Errorf("output = %+v、期待値 = nil", output)
	}
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
	}
}

func assertPageNumbers(t *testing.T, pages []*model.Page, want []model.PageNumber) {
	t.Helper()

	got := make([]model.PageNumber, len(pages))
	for i, page := range pages {
		got[i] = page.Number
	}
	assertPageNumberSlice(t, got, want)
}

func assertPageNumberSlice(t *testing.T, got, want []model.PageNumber) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("ページの番号 = %v、期待値 = %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ページの番号 = %v、期待値 = %v", got, want)
		}
	}
}
