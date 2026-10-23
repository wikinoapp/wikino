package page

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// pageCursorはページの一覧のカーソルの中身。更新日時は複数のページで重なりうるため、主キーと組にして位置を決める
type pageCursor struct {
	ModifiedAt *time.Time    `json:"m"`
	PageID     *model.PageID `json:"i"`
}

// ListPagesはトークンが属するスペースのページの一覧を返す (GET /api/v1/spaces/{space_identifier}/pages)
func (h *Handler) ListPages(ctx context.Context, request apigen.ListPagesRequestObject) (apigen.ListPagesResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	// カーソルはハンドラーで検証するため、その前にトークンが属するスペースかを確かめ、属さないスペースでは未存在として扱う。
	// スキーマで表せるパラメーターの違反 (`limit` など) は、ハンドラーより前のリクエスト検証が422を返す。
	// UseCase側でも同じくトークンが属するスペースかを検証し、認可をHandlerだけに委ねない。
	if !principal.IsBoundTo(model.SpaceIdentifier(request.SpaceIdentifier)) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	after, err := DecodePageCursor(ctx, request.Params.Cursor)
	if err != nil {
		return nil, err
	}
	keywords, err := splitPageQuery(ctx, request.Params.Q)
	if err != nil {
		return nil, err
	}

	output, err := h.listAPIPagesUC.Execute(ctx, usecase.ListAPIPagesInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		TopicNumber:     request.Params.TopicNumber,
		Status:          toUsecasePageStatusFilter(request.Params.Status),
		Title:           request.Params.Title,
		Keywords:        keywords,
		ModifiedSince:   request.Params.ModifiedSince,
		After:           after,
		Limit:           apipagination.Limit(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	body, link, err := ToAPIPageList(ctx, h.appURL, principal.Space, output.Pages, output.Topics, output.HasNext)
	if err != nil {
		return nil, err
	}
	return apigen.ListPages200JSONResponse{Body: body, Headers: apigen.ListPages200ResponseHeaders{Link: link}}, nil
}

// ToAPIPageListは更新日時の新しい順に並べたページを、APIのページの一覧と次のページを示す `Link` ヘッダーの値に変換する。
// 次のページが無ければ `next_cursor` と `Link` ヘッダーの値をnilにする。
// ページのリンク一覧・バックリンク一覧も同じ並び順とカーソルで返すため、それらのハンドラーからも使う。
// appURLとspaceはToAPIPageに渡す
func ToAPIPageList(ctx context.Context, appURL string, space *model.Space, pages []*model.Page, topics map[model.TopicID]*model.Topic, hasNext bool) (apigen.PageList, *string, error) {
	items := make([]apigen.Page, len(pages))
	for i, page := range pages {
		var err error
		items[i], err = ToAPIPage(appURL, space, page, topics[page.TopicID])
		if err != nil {
			return apigen.PageList{}, nil, err
		}
	}

	list := apigen.PageList{Items: items}
	if !hasNext {
		return list, nil, nil
	}
	last := pages[len(pages)-1]
	modifiedAt := last.ModifiedAt.UTC()
	nextCursor, err := apipagination.EncodeCursor(pageCursor{ModifiedAt: &modifiedAt, PageID: &last.ID})
	if err != nil {
		return apigen.PageList{}, nil, err
	}
	list.NextCursor = &nextCursor
	if link := apipagination.NextLink(ctx, nextCursor); link != "" {
		return list, &link, nil
	}
	return list, nil, nil
}

// toUsecasePageStatusFilterは一覧の `status` パラメーターをユースケースの値に変換する。
// 省略時は公開済みのページだけを返す。値の検証はハンドラーより前のリクエスト検証が行う
func toUsecasePageStatusFilter(status *apigen.ListPagesParamsStatus) usecase.APIPageStatusFilter {
	if status == nil {
		return usecase.APIPageStatusFilterPublished
	}
	switch *status {
	case apigen.ListPagesParamsStatusUnpublished:
		return usecase.APIPageStatusFilterUnpublished
	case apigen.ListPagesParamsStatusAll:
		return usecase.APIPageStatusFilterAll
	default:
		return usecase.APIPageStatusFilterPublished
	}
}

// maxPageQueryKeywordsは一覧の `q` に指定できるキーワードの数の上限。
// 本文にはインデックスが無く、キーワードごとにスペースのページの本文を走査し直すため、数で費用の上限を決める
const maxPageQueryKeywords = 10

// splitPageQueryは一覧の `q` パラメーターを空白 (全角の空白を含む) で区切り、キーワードの一覧にする。
// `q` が無ければnilを返す。空白だけの `q` は、絞り込みの無い一覧を返して指定の誤りに気付けなくなるため拒否する。
// 空文字列と長さの上限はハンドラーより前のリクエスト検証が拒否する
func splitPageQuery(ctx context.Context, q *string) ([]string, error) {
	if q == nil {
		return nil, nil
	}
	keywords := strings.Fields(*q)
	if len(keywords) == 0 {
		return nil, &apierror.ParameterError{Parameter: "q", Detail: i18n.T(ctx, "api_error_blank_page_query")}
	}
	if len(keywords) > maxPageQueryKeywords {
		return nil, &apierror.ParameterError{
			Parameter: "q",
			Detail:    i18n.T(ctx, "api_error_too_many_page_query_keywords", map[string]any{"Max": maxPageQueryKeywords}),
		}
	}
	return keywords, nil
}

// DecodePageCursorはリクエストのカーソルから、前のページの最後のページの位置を取り出す。
// カーソルが無ければnilを返す。ページのリンク一覧・バックリンク一覧のハンドラーからも使う
func DecodePageCursor(ctx context.Context, cursor *string) (*usecase.APIPageCursor, error) {
	if cursor == nil {
		return nil, nil
	}

	invalid := &apierror.ParameterError{Parameter: "cursor", Detail: i18n.T(ctx, "api_error_invalid_cursor")}
	var c pageCursor
	if err := apipagination.DecodeCursor(*cursor, &c); err != nil || c.ModifiedAt == nil || c.PageID == nil {
		return nil, invalid
	}
	// IDはSQLでUUIDとして扱うため、UUIDとして読めないものはここで拒否する
	if _, err := uuid.Parse(string(*c.PageID)); err != nil {
		return nil, invalid
	}
	return &usecase.APIPageCursor{ModifiedAt: *c.ModifiedAt, PageID: *c.PageID}, nil
}
