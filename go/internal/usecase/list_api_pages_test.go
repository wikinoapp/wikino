package usecase

import (
	"database/sql"
	"fmt"
	"strings"
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
				t.Fatalf("Execute()のエラー = %v", err)
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

// setupAPIUnpublishedPageFixtureは、setupAPIPageFixtureのページに加えて次のページを作る。
// 番号nのページの更新日時は、setupAPIPageFixtureと同じく基準日時のn時間前にする
//   - 21: 公開トピックの未公開のページ。公開トピックの公開済みのページ30からリンクされている
//   - 22: 公開トピックの未公開のページ。参加していない非公開トピックの公開済みのページ31からだけリンクされている
//   - 23: 公開トピックの未公開のページ。メンバー自身の下書きからだけリンクされている
//   - 24: 公開トピックの未公開のページ。他のメンバーの下書きからだけリンクされている
//   - 26: 参加している非公開トピックの未公開のページ。公開トピックの公開済みのページ30からリンクされている
//   - 30: 公開トピックの公開済みのページ (タイトルは "Source")
//   - 31: 参加していない非公開トピックの公開済みのページ
//   - 33: 公開トピックの未公開のページ。どこからもリンクされておらず、下書きを置くページにする
func setupAPIUnpublishedPageFixture(t *testing.T, tx *sql.Tx, identifier string, memberRole model.SpaceRole) apiTopicFixture {
	t.Helper()

	f := setupAPIPageFixture(t, tx, identifier, memberRole)
	buildPage := func(number model.PageNumber, topicNumber int32, title string, published bool, linkedPageIDs ...model.PageID) model.PageID {
		b := testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithTopicID(f.topicIDs[topicNumber]).
			WithNumber(number).
			WithTitle(title).
			WithLinkedPageIDs(linkedPageIDs).
			WithModifiedAt(apiPageBaseTime.Add(-time.Duration(number) * time.Hour))
		if !published {
			b = b.WithUnpublished()
		}
		return b.Build()
	}

	page21 := buildPage(21, 1, "ページ21", false)
	page22 := buildPage(22, 1, "ページ22", false)
	page23 := buildPage(23, 1, "ページ23", false)
	page24 := buildPage(24, 1, "ページ24", false)
	page26 := buildPage(26, 2, "ページ26", false)
	buildPage(30, 1, "Source", true, page21, page26)
	buildPage(31, 3, "ページ31", true, page22)
	draftHost := buildPage(33, 1, "ページ33", false)

	otherAtname := strings.ReplaceAll(identifier, "-", "_") + "_other"
	otherUserID := testutil.NewUserBuilder(t, tx).WithAtname(otherAtname).WithEmail(otherAtname + "@example.com").Build()
	otherSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(f.space.ID).WithUserID(otherUserID).Build()
	drafts := []struct {
		spaceMemberID model.SpaceMemberID
		linkedPageID  model.PageID
	}{
		{spaceMemberID: f.spaceMemberID, linkedPageID: page23},
		{spaceMemberID: otherSpaceMemberID, linkedPageID: page24},
	}
	for _, d := range drafts {
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithTopicID(f.topicIDs[1]).
			WithPageID(draftHost).
			WithSpaceMemberID(d.spaceMemberID).
			WithLinkedPageIDs([]model.PageID{d.linkedPageID}).
			Build()
	}
	return f
}

func TestListAPIPagesUsecase_Execute_StatusAndTitle(t *testing.T) {
	t.Parallel()

	int32Ptr := func(v int32) *int32 { return &v }
	stringPtr := func(v string) *string { return &v }
	withTopicRead := []model.Scope{model.ScopePageRead, model.ScopeTopicRead}

	tests := []struct {
		name        string
		tokenScopes []model.Scope
		status      APIPageStatusFilter
		topicNumber *int32
		title       *string
		wantNumbers []model.PageNumber
	}{
		{
			name:        "statusを指定しなければ公開済みのページだけを返す",
			tokenScopes: withTopicRead,
			wantNumbers: []model.PageNumber{1, 2, 6, 7, 30, 31},
		},
		{
			name:        "publishedは公開済みのページだけを返す",
			tokenScopes: withTopicRead,
			status:      APIPageStatusFilterPublished,
			wantNumbers: []model.PageNumber{1, 2, 6, 7, 30, 31},
		},
		{
			name:        "unpublishedは、見られるトピックの公開済みのページか自分の下書きからリンクされた未公開のページだけを返す",
			tokenScopes: withTopicRead,
			status:      APIPageStatusFilterUnpublished,
			wantNumbers: []model.PageNumber{21, 22, 23, 26},
		},
		{
			name:        "allは公開済みのページとリンクされた未公開のページを返す",
			tokenScopes: withTopicRead,
			status:      APIPageStatusFilterAll,
			wantNumbers: []model.PageNumber{1, 2, 6, 7, 21, 22, 23, 26, 30, 31},
		},
		{
			name:        "見られないトピックからだけリンクされた未公開のページは含まない",
			tokenScopes: []model.Scope{model.ScopePageRead},
			status:      APIPageStatusFilterUnpublished,
			wantNumbers: []model.PageNumber{21, 23},
		},
		{
			name:        "topic_numberで絞り込んでも、リンク元は見られるトピック全体から探す",
			tokenScopes: withTopicRead,
			status:      APIPageStatusFilterUnpublished,
			topicNumber: int32Ptr(2),
			wantNumbers: []model.PageNumber{26},
		},
		{
			name:        "titleはタイトル全体が一致するページだけを返す",
			tokenScopes: withTopicRead,
			status:      APIPageStatusFilterAll,
			title:       stringPtr("ページ2"),
			wantNumbers: []model.PageNumber{2},
		},
		{
			name:        "titleで未公開のリンク先を探せる",
			tokenScopes: withTopicRead,
			status:      APIPageStatusFilterAll,
			topicNumber: int32Ptr(1),
			title:       stringPtr("ページ21"),
			wantNumbers: []model.PageNumber{21},
		},
		{
			name:        "titleは大文字と小文字を区別しない",
			tokenScopes: withTopicRead,
			title:       stringPtr("SOURCE"),
			wantNumbers: []model.PageNumber{30},
		},
		{
			name:        "titleで絞り込んでも、statusの対象でないページは返さない",
			tokenScopes: withTopicRead,
			title:       stringPtr("ページ21"),
			wantNumbers: []model.PageNumber{},
		},
		{
			name:        "titleで絞り込んでも、リンクされていない未公開のページは返さない",
			tokenScopes: withTopicRead,
			status:      APIPageStatusFilterAll,
			title:       stringPtr("ページ3"),
			wantNumbers: []model.PageNumber{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIUnpublishedPageFixture(t, tx, "api-page-status", apiTopicRegularMemberRole)

			output, err := newListAPIPagesUC(tx).Execute(t.Context(), ListAPIPagesInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     tt.topicNumber,
				Status:          tt.status,
				Title:           tt.title,
				Limit:           20,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			assertPageNumbers(t, output.Pages, tt.wantNumbers)
		})
	}
}

// setupAPIPageKeywordFixtureは、setupAPIPageFixtureのページに加えて、キーワードの検索を確かめる次のページを作る。
// 番号nのページの更新日時は、setupAPIPageFixtureと同じく基準日時のn時間前にする
//   - 40: 公開トピックの公開済みのページ。タイトルだけに「りんご」を含み、47・50にリンクしている
//   - 41: 公開トピックの公開済みのページ。本文だけに「りんご」と「みかん」を含み、タイトルは「果物」
//   - 42: 公開トピックの公開済みのページ。タイトルに「みかん」、本文に「果物」を含む
//   - 43: 公開トピックの公開済みのページ。タイトルに「Wikino」、本文に「wikino」「50%」「a_c」を含む
//   - 44: 公開トピックの公開済みのページ。本文の「500円」「abc」は、エスケープしなければ「50%」「a_c」に一致する
//   - 45: 参加していない非公開トピックの公開済みのページ。本文に「りんご」を含む
//   - 46: 公開トピックのゴミ箱のページ。本文に「りんご」を含む
//   - 47: 公開トピックの未公開のページ。タイトルに「りんご」を含み、40からリンクされている
//   - 48: 公開トピックの未公開のページ。タイトルに「りんご」を含むが、どこからもリンクされていない
//   - 49: 参加している非公開トピックの公開済みのページ。本文に「りんご」を含む
//   - 50: 公開トピックのタイトルの無い未公開のページ。本文に「りんご」を含み、40からリンクされている
func setupAPIPageKeywordFixture(t *testing.T, tx *sql.Tx, identifier string, memberRole model.SpaceRole) apiTopicFixture {
	t.Helper()

	f := setupAPIPageFixture(t, tx, identifier, memberRole)
	newPage := func(number model.PageNumber, topicNumber int32, body string) *testutil.PageBuilder {
		return testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithTopicID(f.topicIDs[topicNumber]).
			WithNumber(number).
			WithBody(body).
			WithModifiedAt(apiPageBaseTime.Add(-time.Duration(number) * time.Hour))
	}

	page47 := newPage(47, 1, "ジャムの作り方").WithTitle("りんごジャム").WithUnpublished().Build()
	page50 := newPage(50, 1, "りんごの皮").WithNilTitle().WithUnpublished().Build()
	newPage(40, 1, "土と水").WithTitle("りんごの育て方").WithLinkedPageIDs([]model.PageID{page47, page50}).Build()
	newPage(41, 1, "りんごとみかんを買った").WithTitle("果物").Build()
	newPage(42, 1, "冬の果物").WithTitle("みかん").Build()
	newPage(43, 1, "wikinoで50%の力を出す。a_cの値").WithTitle("Wikino入門").Build()
	newPage(44, 1, "500円でabcを買う").WithTitle("買い物").Build()
	newPage(45, 3, "りんごの品種").WithTitle("品種").Build()
	newPage(46, 1, "りんごの在庫").WithTitle("在庫").WithTrashed().Build()
	newPage(48, 1, "").WithTitle("りんご飴").WithUnpublished().Build()
	newPage(49, 2, "りんごの値段").WithTitle("値段").Build()
	return f
}

func TestListAPIPagesUsecase_Execute_Keywords(t *testing.T) {
	t.Parallel()

	int32Ptr := func(v int32) *int32 { return &v }
	stringPtr := func(v string) *string { return &v }
	timePtr := func(v time.Time) *time.Time { return &v }
	withTopicRead := []model.Scope{model.ScopePageRead, model.ScopeTopicRead}

	tests := []struct {
		name          string
		tokenScopes   []model.Scope
		keywords      []string
		status        APIPageStatusFilter
		topicNumber   *int32
		title         *string
		modifiedSince *time.Time
		wantNumbers   []model.PageNumber
	}{
		{
			name:        "キーワードをタイトルか本文に含むページを返し、ゴミ箱のページは含まない",
			tokenScopes: withTopicRead,
			keywords:    []string{"りんご"},
			wantNumbers: []model.PageNumber{40, 41, 45, 49},
		},
		{
			name:        "見られないトピックのページは含まない",
			tokenScopes: []model.Scope{model.ScopePageRead},
			keywords:    []string{"りんご"},
			wantNumbers: []model.PageNumber{40, 41},
		},
		{
			name:        "複数のキーワードはすべてを含むページだけを返す",
			tokenScopes: withTopicRead,
			keywords:    []string{"りんご", "みかん"},
			wantNumbers: []model.PageNumber{41},
		},
		{
			name:        "キーワードごとにタイトルと本文のどちらに含んでもよい",
			tokenScopes: withTopicRead,
			keywords:    []string{"みかん", "果物"},
			wantNumbers: []model.PageNumber{41, 42},
		},
		{
			name:        "大文字と小文字を区別しない",
			tokenScopes: withTopicRead,
			keywords:    []string{"WIKINO"},
			wantNumbers: []model.PageNumber{43},
		},
		{
			name:        "%は1つの文字として探す",
			tokenScopes: withTopicRead,
			keywords:    []string{"50%"},
			wantNumbers: []model.PageNumber{43},
		},
		{
			name:        "_は1つの文字として探す",
			tokenScopes: withTopicRead,
			keywords:    []string{"a_c"},
			wantNumbers: []model.PageNumber{43},
		},
		{
			name:        "一致するページが無ければ空の一覧",
			tokenScopes: withTopicRead,
			keywords:    []string{"ぶどう"},
			wantNumbers: []model.PageNumber{},
		},
		{
			name:        "statusと組み合わせると、リンクされた未公開のページもタイトルの有無によらず探す",
			tokenScopes: withTopicRead,
			keywords:    []string{"りんご"},
			status:      APIPageStatusFilterAll,
			wantNumbers: []model.PageNumber{40, 41, 45, 47, 49, 50},
		},
		{
			name:        "topic_numberと組み合わせられる",
			tokenScopes: withTopicRead,
			keywords:    []string{"りんご"},
			topicNumber: int32Ptr(2),
			wantNumbers: []model.PageNumber{49},
		},
		{
			name:        "titleと組み合わせられる",
			tokenScopes: withTopicRead,
			keywords:    []string{"みかん"},
			title:       stringPtr("果物"),
			wantNumbers: []model.PageNumber{41},
		},
		{
			name:          "modified_sinceと組み合わせられる",
			tokenScopes:   withTopicRead,
			keywords:      []string{"りんご"},
			modifiedSince: timePtr(apiPageBaseTime.Add(-41 * time.Hour)),
			wantNumbers:   []model.PageNumber{40, 41},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIPageKeywordFixture(t, tx, "api-page-keywords", apiTopicRegularMemberRole)

			output, err := newListAPIPagesUC(tx).Execute(t.Context(), ListAPIPagesInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     tt.topicNumber,
				Status:          tt.status,
				Title:           tt.title,
				Keywords:        tt.keywords,
				ModifiedSince:   tt.modifiedSince,
				Limit:           20,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			assertPageNumbers(t, output.Pages, tt.wantNumbers)
		})
	}
}

func TestListAPIPagesUsecase_Execute_KeywordsCursor(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPIPageKeywordFixture(t, tx, "api-page-keywords-cursor", apiTopicRegularMemberRole)
	principal := f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopePageRead, model.ScopeTopicRead})
	uc := newListAPIPagesUC(tx)

	var got []model.PageNumber
	var after *APIPageCursor
	for range 10 {
		output, err := uc.Execute(t.Context(), ListAPIPagesInput{
			Principal:       principal,
			SpaceIdentifier: f.space.Identifier,
			Keywords:        []string{"りんご"},
			After:           after,
			Limit:           1,
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
	assertPageNumberSlice(t, got, []model.PageNumber{40, 41, 45, 49})
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
				t.Fatalf("Execute()のエラー = %v", err)
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
			t.Fatalf("Execute()のエラー = %v", err)
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
