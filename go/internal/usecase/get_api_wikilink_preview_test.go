package usecase

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetAPIWikilinkPreviewUC(q *query.Queries) *GetAPIWikilinkPreviewUsecase {
	return NewGetAPIWikilinkPreviewUsecase(repository.NewPageRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

// wikilinkPreviewSummaryはプレビューの1つのリンクを、テストで比べやすい形にしたもの
type wikilinkPreviewSummary struct {
	raw        string
	topicName  string
	pageTitle  string
	resolution WikilinkResolution
	// pageNumberは返したページの番号。ページを返さなければ0
	pageNumber model.PageNumber
}

func summarizeWikilinkPreview(output *GetAPIWikilinkPreviewOutput) []wikilinkPreviewSummary {
	summaries := make([]wikilinkPreviewSummary, len(output.Links))
	for i, link := range output.Links {
		summaries[i] = wikilinkPreviewSummary{
			raw:        link.Key.Raw,
			topicName:  link.Key.TopicName,
			pageTitle:  link.Key.PageTitle,
			resolution: link.Resolution,
		}
		if link.Page != nil {
			summaries[i].pageNumber = link.Page.Number
		}
	}
	return summaries
}

func TestGetAPIWikilinkPreviewUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	// トピックは1が公開、2が参加している非公開、3が参加していない非公開、4が廃棄済み
	f := setupAPITopicFixture(t, tx, "api-wikilink-preview", apiTopicRegularMemberRole)
	pages := []struct {
		number      model.PageNumber
		topicNumber int32
		title       string
		modify      func(b *testutil.PageBuilder) *testutil.PageBuilder
	}{
		{number: 1, topicNumber: 1, title: "既存"},
		{number: 2, topicNumber: 1, title: "未公開", modify: (*testutil.PageBuilder).WithUnpublished},
		{number: 3, topicNumber: 1, title: "ゴミ箱", modify: (*testutil.PageBuilder).WithTrashed},
		{number: 4, topicNumber: 1, title: "廃棄済みのページ", modify: (*testutil.PageBuilder).WithDiscarded},
		{number: 5, topicNumber: 3, title: "秘密"},
	}
	for _, p := range pages {
		b := testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithTopicID(f.topicIDs[p.topicNumber]).
			WithNumber(p.number).
			WithTitle(p.title)
		if p.modify != nil {
			b = p.modify(b)
		}
		b.Build()
	}
	uc := newGetAPIWikilinkPreviewUC(testutil.QueriesWithTx(tx))

	body := "[[既存]] [[未公開]] [[ゴミ箱]] [[廃棄済みのページ]] [[新規]] [[公開/既存]] [[参加している非公開/新規]]\n" +
		"[[無いトピック/ページ]] [[廃棄済み/ページ]] [[参加していない非公開/秘密]] [[参加していない非公開/無いページ]]\n" +
		"[[既存]] [[ 新規 ]] `[[コード]]` \\[[エスケープ]]"

	// サブテストは親のトランザクションを共有するため、並列にしない

	tests := []struct {
		name        string
		tokenScopes []model.Scope
		topicNumber int32
		want        []wikilinkPreviewSummary
	}{
		{
			// 同じリンク先の2つ目以降 (前後の空白を除くと同じものを含む) と、コード・エスケープの [[...]] は含めない。
			// ゴミ箱・廃棄済みのページは既存のページとして扱い、ページは返さない。
			// topic:readの無いトークンでは非公開トピックを見られないため、参加している非公開トピックへのリンクも、
			// 参加していない非公開トピックのページの有無も、トピックが無いリンクと区別しない
			name:        "topic:readの無いトークンでは非公開トピックへのリンクをトピックが無いものとして返す",
			tokenScopes: []model.Scope{model.ScopePageRead},
			topicNumber: 1,
			want: []wikilinkPreviewSummary{
				{raw: "既存", topicName: "公開", pageTitle: "既存", resolution: WikilinkResolutionExistingPage, pageNumber: 1},
				{raw: "未公開", topicName: "公開", pageTitle: "未公開", resolution: WikilinkResolutionExistingPage, pageNumber: 2},
				{raw: "ゴミ箱", topicName: "公開", pageTitle: "ゴミ箱", resolution: WikilinkResolutionExistingPage},
				{raw: "廃棄済みのページ", topicName: "公開", pageTitle: "廃棄済みのページ", resolution: WikilinkResolutionExistingPage},
				{raw: "新規", topicName: "公開", pageTitle: "新規", resolution: WikilinkResolutionNewPage},
				{raw: "参加している非公開/新規", topicName: "参加している非公開", pageTitle: "新規", resolution: WikilinkResolutionTopicNotFound},
				{raw: "無いトピック/ページ", topicName: "無いトピック", pageTitle: "ページ", resolution: WikilinkResolutionTopicNotFound},
				{raw: "廃棄済み/ページ", topicName: "廃棄済み", pageTitle: "ページ", resolution: WikilinkResolutionTopicNotFound},
				{raw: "参加していない非公開/秘密", topicName: "参加していない非公開", pageTitle: "秘密", resolution: WikilinkResolutionTopicNotFound},
				{raw: "参加していない非公開/無いページ", topicName: "参加していない非公開", pageTitle: "無いページ", resolution: WikilinkResolutionTopicNotFound},
			},
		},
		{
			// [[公開/既存]] は現在のトピックを省いた [[既存]] と別のリンク先として解決する
			name:        "topic:readを持つトークンでは非公開トピックのページの有無を返し、現在のトピックを非公開トピックにできる",
			tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead},
			topicNumber: 3,
			want: []wikilinkPreviewSummary{
				{raw: "既存", topicName: "参加していない非公開", pageTitle: "既存", resolution: WikilinkResolutionNewPage},
				{raw: "未公開", topicName: "参加していない非公開", pageTitle: "未公開", resolution: WikilinkResolutionNewPage},
				{raw: "ゴミ箱", topicName: "参加していない非公開", pageTitle: "ゴミ箱", resolution: WikilinkResolutionNewPage},
				{raw: "廃棄済みのページ", topicName: "参加していない非公開", pageTitle: "廃棄済みのページ", resolution: WikilinkResolutionNewPage},
				{raw: "新規", topicName: "参加していない非公開", pageTitle: "新規", resolution: WikilinkResolutionNewPage},
				{raw: "公開/既存", topicName: "公開", pageTitle: "既存", resolution: WikilinkResolutionExistingPage, pageNumber: 1},
				{raw: "参加している非公開/新規", topicName: "参加している非公開", pageTitle: "新規", resolution: WikilinkResolutionNewPage},
				{raw: "無いトピック/ページ", topicName: "無いトピック", pageTitle: "ページ", resolution: WikilinkResolutionTopicNotFound},
				{raw: "廃棄済み/ページ", topicName: "廃棄済み", pageTitle: "ページ", resolution: WikilinkResolutionTopicNotFound},
				{raw: "参加していない非公開/秘密", topicName: "参加していない非公開", pageTitle: "秘密", resolution: WikilinkResolutionExistingPage, pageNumber: 5},
				{raw: "参加していない非公開/無いページ", topicName: "参加していない非公開", pageTitle: "無いページ", resolution: WikilinkResolutionNewPage},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(t.Context(), GetAPIWikilinkPreviewInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     tt.topicNumber,
				Body:            body,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}

			got := summarizeWikilinkPreview(output)
			if len(got) != len(tt.want) {
				t.Fatalf("リンク = %+v、期待値 = %+v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("リンク[%d] = %+v、期待値 = %+v", i, got[i], tt.want[i])
				}
			}
			// 返したページの所属トピックを応答に組み立てられるよう、トピックも返す
			for _, link := range output.Links {
				if link.Page != nil && output.Topics[link.Page.TopicID] == nil {
					t.Errorf("ページ %d の所属トピックが返されていない", link.Page.Number)
				}
				if link.Page != nil && link.Page.Body != "" {
					t.Errorf("リンク先ページ %d の不要な本文を取得している", link.Page.Number)
				}
			}
		})
	}

	t.Run("Wikiリンクの無い本文は空の一覧を返す", func(t *testing.T) {
		output, err := uc.Execute(t.Context(), GetAPIWikilinkPreviewInput{
			Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopePageRead}),
			SpaceIdentifier: f.space.Identifier,
			TopicNumber:     1,
			Body:            "リンクの無い本文",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Links == nil || len(output.Links) != 0 {
			t.Errorf("Links = %#v、期待値 = 空の一覧", output.Links)
		}
	})

	topicErrorTests := []struct {
		name        string
		tokenScopes []model.Scope
		topicNumber int32
	}{
		{name: "存在しないトピック", tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead}, topicNumber: 99},
		{name: "廃棄済みのトピック", tokenScopes: []model.Scope{model.ScopePageRead, model.ScopeTopicRead}, topicNumber: 4},
		{name: "topic:readの無いトークンから見た非公開トピック", tokenScopes: []model.Scope{model.ScopePageRead}, topicNumber: 3},
	}
	for _, tt := range topicErrorTests {
		t.Run(tt.name+"はtopic_numberの422", func(t *testing.T) {
			_, err := uc.Execute(t.Context(), GetAPIWikilinkPreviewInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     tt.topicNumber,
				Body:            "[[既存]]",
			})
			var ve *model.ValidationError
			if !errors.As(err, &ve) || len(ve.Fields) != 1 || len(ve.Fields["topic_number"]) == 0 {
				t.Errorf("エラー = %v、期待値 = topic_numberのバリデーションエラー", err)
			}
		})
	}

	t.Run("トークンが属さないスペースは404", func(t *testing.T) {
		_, err := uc.Execute(t.Context(), GetAPIWikilinkPreviewInput{
			Principal:       f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopePageRead}),
			SpaceIdentifier: "other-space",
			TopicNumber:     1,
			Body:            "[[既存]]",
		})
		var appErr *model.AppError
		if !errors.As(err, &appErr) || appErr.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}

// TestGetAPIWikilinkPreviewUsecase_ExecuteMatchesWriteは、プレビューで作られるとしたリンク先のページだけが、
// 同じ本文の書き込みで作られることを確かめる。見られないトピックへのリンクは、プレビューでトピックが無いものとして
// 返し、書き込みでもページを作らない
func TestGetAPIWikilinkPreviewUsecase_ExecuteMatchesWrite(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	// トピックは1が公開、2がトピック編集者として参加している非公開、3が参加していない非公開
	f := setupCreateAPIPageFixture(t, db, "api-wikilink-preview-write", memberRole)
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[1]).WithNumber(1).WithTitle("既存").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[3]).WithNumber(2).WithTitle("秘密").Build()

	tests := []struct {
		name        string
		tokenScopes []model.Scope
		// wantCreatedは書き込みで作られるリンク先の「トピック番号/タイトル」
		wantCreated []string
	}{
		{
			name:        "topic:readの無いトークンでは公開トピックにだけリンク先を作る",
			tokenScopes: []model.Scope{model.ScopePageWrite},
			wantCreated: []string{"1/新規A"},
		},
		{
			name:        "topic:readを持つトークンでは非公開トピックにもリンク先を作る",
			tokenScopes: []model.Scope{model.ScopePageWrite, model.ScopeTopicRead},
			wantCreated: []string{"1/新規B", "2/新規B", "3/新規B"},
		},
	}
	topicNames := map[int32]string{1: "公開", 2: "参加している非公開", 3: "参加していない非公開"}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// サブテストは同じスペースにページを作るため、リンク先のタイトルをサブテストごとに変え、並列にしない
			suffix := string(rune('A' + i))
			body := "[[既存]] [[新規" + suffix + "]] [[参加している非公開/新規" + suffix + "]] [[参加していない非公開/新規" + suffix + "]] [[参加していない非公開/秘密]] [[無いトピック/新規" + suffix + "]]"
			principal := f.principal(memberRole, tt.tokenScopes)

			preview, err := newGetAPIWikilinkPreviewUC(query.New(db)).Execute(t.Context(), GetAPIWikilinkPreviewInput{
				Principal:       principal,
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     1,
				Body:            body,
			})
			if err != nil {
				t.Fatalf("プレビューのエラー = %v", err)
			}
			var previewed []string
			for _, link := range preview.Links {
				if link.Resolution != WikilinkResolutionNewPage {
					continue
				}
				for number, name := range topicNames {
					if name == link.Key.TopicName {
						previewed = append(previewed, topicPageKey(number, link.Key.PageTitle))
					}
				}
			}

			before := countRows(t, db, "pages", f.space.ID, "")
			if _, err := newCreateAPIPageUC(db).Execute(t.Context(), CreateAPIPageInput{
				Principal:       principal,
				SpaceIdentifier: f.space.Identifier,
				TopicNumber:     1,
				Title:           "書き込み" + suffix,
				Body:            body,
			}); err != nil {
				t.Fatalf("ページの作成のエラー = %v", err)
			}

			// 作成したページ自身の1件と、リンク先として作られたページだけが増える
			if got := countRows(t, db, "pages", f.space.ID, "") - before; got != len(tt.wantCreated)+1 {
				t.Errorf("増えたページの数 = %d、期待値 = %d", got, len(tt.wantCreated)+1)
			}
			if len(previewed) != len(tt.wantCreated) {
				t.Fatalf("プレビューで作られるリンク先 = %v、期待値 = %v", previewed, tt.wantCreated)
			}
			for _, key := range tt.wantCreated {
				if !slices.Contains(previewed, key) {
					t.Errorf("プレビューで作られるリンク先 = %v、期待値に %s を含む", previewed, key)
				}
			}
			for number := range topicNames {
				title := "新規" + suffix
				want := 0
				if slices.Contains(tt.wantCreated, topicPageKey(number, title)) {
					want = 1
				}
				if got := countRows(t, db, "pages", f.space.ID, "topic_id = $2 AND title = $3 AND published_at IS NULL", f.topicIDs[number], title); got != want {
					t.Errorf("トピック%dに作られた「%s」の数 = %d、期待値 = %d", number, title, got, want)
				}
			}
		})
	}
}

// TestCreateAPIDraftPageUsecase_ExecuteSkipsInvisibleLinkTopicは、下書きの作成でも、トークンで見られない
// トピックへのWikiリンクの先を作らず、既存のページへのリンクは記録することを確かめる
func TestCreateAPIDraftPageUsecase_ExecuteSkipsInvisibleLinkTopic(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	memberRole := apiTopicRegularMemberRole
	f := setupCreateAPIPageFixture(t, db, "api-wikilink-draft-invisible", memberRole)
	secretID := testutil.NewPageBuilderDB(t, db).WithSpaceID(f.space.ID).WithTopicID(f.topicIDs[3]).WithNumber(1).WithTitle("秘密").Build()

	topicNumber := int32(1)
	body := "[[参加していない非公開/秘密]] [[参加していない非公開/新規]] [[新規]]"
	output, err := newCreateAPIDraftPageUC(db).Execute(t.Context(), CreateAPIDraftPageInput{
		Principal:       f.principal(memberRole, []model.Scope{model.ScopePageWrite, model.ScopeDraftPageWrite}),
		SpaceIdentifier: f.space.Identifier,
		TopicNumber:     &topicNumber,
		Body:            &body,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if n := countRows(t, db, "pages", f.space.ID, "topic_id = $2 AND title = $3", f.topicIDs[3], "新規"); n != 0 {
		t.Errorf("見られないトピックに作られたページの数 = %d、期待値 = 0", n)
	}
	if n := countRows(t, db, "pages", f.space.ID, "topic_id = $2 AND title = $3", f.topicIDs[1], "新規"); n != 1 {
		t.Errorf("公開トピックに作られたページの数 = %d、期待値 = 1", n)
	}
	if ids := output.DraftPage.LinkedPageIDs; len(ids) != 2 || ids[0] != secretID {
		t.Errorf("LinkedPageIDs = %v、期待値 = 見られないトピックの既存のページと公開トピックに作ったページ", ids)
	}
}

// topicPageKeyはリンク先を「トピック番号/タイトル」の形で表す
func topicPageKey(topicNumber int32, title string) string {
	return fmt.Sprintf("%d/%s", topicNumber, title)
}
