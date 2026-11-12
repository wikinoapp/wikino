package usecase

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestGetSearchShowUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	uc := NewGetSearchShowUsecase(
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewPageRepository(queries),
		repository.NewTopicRepository(queries),
		repository.NewTopicMemberRepository(queries),
		validator.NewSearchShowValidator(),
	)

	// 検索はすべてのスペースをまたぐため、他のテストのページに一致しないキーワードを使う
	const keyword = "検索ユースケース"

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("search-show@example.com").
		WithAtname("search_show").
		Build()

	joinedSpaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("search-show-joined").WithName("参加しているスペース").Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(joinedSpaceID).WithUserID(userID).WithRole(model.SpaceRoleViewer).Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("search-show-other").WithName("参加していないスペース").Build()

	newTopic := func(space model.SpaceID, number int32, visibility model.TopicVisibility) model.TopicID {
		return testutil.NewTopicBuilder(t, tx).WithSpaceID(space).WithNumber(number).
			WithName(fmt.Sprintf("トピック%d", number)).WithVisibility(int32(visibility)).Build()
	}
	joinedPublic := newTopic(joinedSpaceID, 1, model.TopicVisibilityPublic)
	joinedPrivate := newTopic(joinedSpaceID, 2, model.TopicVisibilityPrivate)
	otherPublic := newTopic(otherSpaceID, 1, model.TopicVisibilityPublic)
	otherPrivate := newTopic(otherSpaceID, 2, model.TopicVisibilityPrivate)

	base := time.Now().Add(-time.Hour)
	newPage := func(space model.SpaceID, topic model.TopicID, number model.PageNumber, title string) model.PageID {
		return testutil.NewPageBuilder(t, tx).WithSpaceID(space).WithTopicID(topic).WithNumber(number).
			WithTitle(title).WithModifiedAt(base.Add(-time.Duration(number) * time.Minute)).Build()
	}
	joinedPublicPage := newPage(joinedSpaceID, joinedPublic, 1, keyword+" 公開")
	joinedPrivatePage := newPage(joinedSpaceID, joinedPrivate, 2, keyword+" 非公開")
	otherPublicPage := newPage(otherSpaceID, otherPublic, 3, keyword+" 別のスペースの公開")
	newPage(otherSpaceID, otherPrivate, 4, keyword+" 別のスペースの非公開")
	newPage(joinedSpaceID, joinedPublic, 5, "一致しないタイトル")

	// 上限を確かめるため、別のキーワードのページを上限より1件多く作る
	const limitKeyword = "検索上限ユースケース"
	for i := range searchResultLimit + 1 {
		newPage(joinedSpaceID, joinedPublic, model.PageNumber(100+i), fmt.Sprintf("%s %d", limitKeyword, i))
	}

	pageIDs := func(pages []*model.Page) []model.PageID {
		ids := make([]model.PageID, len(pages))
		for i, pg := range pages {
			ids[i] = pg.ID
		}
		return ids
	}

	tests := []struct {
		name         string
		query        string
		wantSearched bool
		wantPages    []model.PageID
	}{
		{
			name:         "参加しているスペースの開けるトピックと、すべてのスペースの公開トピックのページを返す",
			query:        keyword,
			wantSearched: true,
			wantPages:    []model.PageID{joinedPublicPage, joinedPrivatePage, otherPublicPage},
		},
		{
			name:         "すべてのキーワードを含むページに絞る",
			query:        keyword + " 非公開",
			wantSearched: true,
			wantPages:    []model.PageID{joinedPrivatePage},
		},
		{
			name:         "space:で参加しているスペースに絞る",
			query:        "space:search-show-joined " + keyword,
			wantSearched: true,
			wantPages:    []model.PageID{joinedPublicPage, joinedPrivatePage},
		},
		{
			name:         "space:を大文字で指定しても、参加しているスペースの非公開トピックのページを返す",
			query:        "space:SEARCH-SHOW-JOINED " + keyword,
			wantSearched: true,
			wantPages:    []model.PageID{joinedPublicPage, joinedPrivatePage},
		},
		{
			name:         "space:で参加していないスペースに絞ると、公開トピックのページだけを返す",
			query:        "space:search-show-other " + keyword,
			wantSearched: true,
			wantPages:    []model.PageID{otherPublicPage},
		},
		{
			name:         "space:の指定だけなら検索しない",
			query:        "space:search-show-joined",
			wantSearched: false,
			wantPages:    []model.PageID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(ctx, GetSearchShowInput{UserID: userID, Query: tt.query})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.Searched != tt.wantSearched {
				t.Errorf("Searched = %v、期待値 = %v", output.Searched, tt.wantSearched)
			}
			if got := pageIDs(output.Pages); !slices.Equal(got, tt.wantPages) {
				t.Errorf("ページ = %v、期待値 = %v", got, tt.wantPages)
			}
			for _, pg := range output.Pages {
				if topic := output.TopicMap[pg.TopicID]; topic == nil || topic.Space == nil || topic.Space.Identifier == "" {
					t.Errorf("ページ%vのトピックとスペースが解決されていない", pg.ID)
				}
			}
		})
	}

	t.Run("25件まで返す", func(t *testing.T) {
		output, err := uc.Execute(ctx, GetSearchShowInput{UserID: userID, Query: limitKeyword})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if len(output.Pages) != searchResultLimit {
			t.Errorf("件数 = %d、期待値 = %d", len(output.Pages), searchResultLimit)
		}
	})

	t.Run("キーワードの誤りはValidationErrorを返す", func(t *testing.T) {
		_, err := uc.Execute(ctx, GetSearchShowInput{UserID: userID, Query: "a"})
		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("q") {
			t.Fatalf("qのValidationErrorを期待したが、%vだった", err)
		}
	})
}

// 組み込みのロールはどれもスペースでtopic:readを持つため、開けない非公開トピックはUseCaseの
// テストでは作れない。topic:readを持たないスコープの組み合わせで、検索が使う判定を確かめる。
func TestTopicAccess_visibleTopics(t *testing.T) {
	t.Parallel()

	publicTopic := &model.Topic{ID: "public", Visibility: model.TopicVisibilityPublic}
	privateTopic := &model.Topic{ID: "private", Visibility: model.TopicVisibilityPrivate}
	joinedPrivateTopic := &model.Topic{ID: "joined-private", Visibility: model.TopicVisibilityPrivate}

	access := &topicAccess{
		authorize: func(topicMember *model.TopicMember) policy.Authorizer {
			var topicScopes []model.Scope
			if topicMember != nil {
				topicScopes = topicMember.Role.Scopes()
			}
			return policy.NewMemberPolicy([]model.Scope{model.ScopePageRead}, topicScopes)
		},
		topics: []*model.Topic{publicTopic, privateTopic, joinedPrivateTopic},
		memberByTopic: map[model.TopicID]*model.TopicMember{
			joinedPrivateTopic.ID: {TopicID: joinedPrivateTopic.ID, Role: model.TopicRoleViewer},
		},
	}

	var got []model.TopicID
	for _, topic := range access.visibleTopics() {
		got = append(got, topic.ID)
	}
	want := []model.TopicID{publicTopic.ID, joinedPrivateTopic.ID}
	if !slices.Equal(got, want) {
		t.Errorf("トピック = %v、期待値 = %v", got, want)
	}
}
