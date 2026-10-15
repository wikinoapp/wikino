package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apigen"
)

func TestAPIRouter_CreateDraftPageBodyEdit(t *testing.T) {
	t.Parallel()

	r := newDraftPageWriteTestRouter(t, "api-http-draft-body-edit")
	const draftPath = "/api/v1/spaces/api-http-draft-body-edit/pages/1/draft_page"
	target := draftPath + "/body_edits"
	edit := func(token, ifMatch, body string) *httptest.ResponseRecorder {
		return requestDraftPage(t, r, http.MethodPost, target, token, ifMatch, "application/json", body)
	}
	// currentDraftは下書きを取得し、応答の本文とETagを返す
	currentDraft := func(t *testing.T) (apigen.DraftPage, string) {
		t.Helper()
		rr := requestDraftPage(t, r, http.MethodGet, draftPath, "wkp_draft_page_read", "", "", "")
		if rr.Code != http.StatusOK {
			t.Fatalf("取得のステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var draft apigen.DraftPage
		if err := json.Unmarshal(rr.Body.Bytes(), &draft); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		return draft, rr.Header().Get("ETag")
	}
	// assertFieldProblemは422の問題が、pointerの指すリクエスト本文の項目についてのものであることを確かめ、detailを返す
	assertFieldProblem := func(t *testing.T, rr *httptest.ResponseRecorder, pointer string) string {
		t.Helper()
		assertProblem(t, rr, http.StatusUnprocessableEntity)
		var problem struct {
			Errors []struct {
				Detail  string  `json:"detail"`
				Pointer *string `json:"pointer"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(problem.Errors) == 0 || problem.Errors[0].Pointer == nil || *problem.Errors[0].Pointer != pointer {
			t.Errorf("errors = %s、期待値 = pointerが %s の問題", rr.Body.String(), pointer)
			return ""
		}
		return problem.Errors[0].Detail
	}

	// サブテストは同じ下書きを更新するため、並列にしない。
	t.Run("If-Match無しで本文を置換し、下書きのURLとETagを付けて200を返す", func(t *testing.T) {
		rr := edit("wkp_draft_page_write", "", `{"edits": [{"old_str": "の本文", "new_str": "の本文 本文 本文"}]}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q、期待値 = application/json", got)
		}
		if got := rr.Header().Get("Content-Location"); got != draftPath {
			t.Errorf("Content-Location = %q、期待値 = %q", got, draftPath)
		}
		var body apigen.DraftPage
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.PageNumber != 1 || body.Title == nil || *body.Title != "下書き" || body.Body != "下書きの本文 本文 本文" {
			t.Errorf("下書き = %+v、期待値 = 本文を置換した下書き", body)
		}
		if want := "https://example.com/s/api-http-draft-body-edit/pages/1/edit"; body.Url != want {
			t.Errorf("url = %q、期待値 = 編集画面の %q", body.Url, want)
		}
		if _, etag := currentDraft(t); rr.Header().Get("ETag") != etag {
			t.Errorf("ETag = %q、期待値 = 取得と同じ %q", rr.Header().Get("ETag"), etag)
		}
	})

	t.Run("今のETagをIf-Matchで送れば置換し、置換前のETagでは412", func(t *testing.T) {
		_, etag := currentDraft(t)
		rr := edit("wkp_draft_page_write", etag, `{"edits": [{"old_str": "下書き", "new_str": "したがき"}]}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}

		assertProblem(t, edit("wkp_draft_page_write", etag, `{"edits": [{"old_str": "したがき", "new_str": "x"}]}`), http.StatusPreconditionFailed)
	})

	t.Run("どのETagとも一致しないIf-Match (弱いETag) は、省略と区別して412", func(t *testing.T) {
		_, etag := currentDraft(t)
		assertProblem(t, edit("wkp_draft_page_write", "W/"+etag, `{"edits": [{"old_str": "したがき", "new_str": "x"}]}`), http.StatusPreconditionFailed)
	})

	t.Run("一致しない編集は、何番目かを示して422にし、前の編集も適用しない", func(t *testing.T) {
		before, etag := currentDraft(t)
		rr := edit("wkp_draft_page_write", "", `{"edits": [{"old_str": "したがき", "new_str": "x"}, {"old_str": "無い", "new_str": "y"}]}`)
		assertFieldProblem(t, rr, "#/edits/1/old_str")
		if after, afterETag := currentDraft(t); after.Body != before.Body || afterETag != etag {
			t.Errorf("本文 = %q、期待値 = %q のまま", after.Body, before.Body)
		}
	})

	t.Run("replace_allでなく複数件に一致する編集は、一致の件数を示して422", func(t *testing.T) {
		detail := assertFieldProblem(t, edit("wkp_draft_page_write", "", `{"edits": [{"old_str": "本文", "new_str": "x"}]}`), "#/edits/0/old_str")
		if !strings.Contains(detail, "3件") {
			t.Errorf("detail = %q、期待値 = 一致の件数 (3件) を含む", detail)
		}
	})

	t.Run("old_strが空なら422", func(t *testing.T) {
		assertProblem(t, edit("wkp_draft_page_write", "", `{"edits": [{"old_str": "", "new_str": "x"}]}`), http.StatusUnprocessableEntity)
	})

	t.Run("draft_page:writeの無いトークンは、page:writeを持っていても403", func(t *testing.T) {
		rr := edit("wkp_page_write", "", `{"edits": [{"old_str": "したがき", "new_str": "x"}]}`)
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="draft_page:write", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-draft-body-edit"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("下書きの無いページと存在しないページは404", func(t *testing.T) {
		for _, pageNumber := range []string{"2", "99"} {
			rr := requestDraftPage(t, r, http.MethodPost, "/api/v1/spaces/api-http-draft-body-edit/pages/"+pageNumber+"/draft_page/body_edits",
				"wkp_draft_page_write", "", "application/json", `{"edits": [{"old_str": "a", "new_str": "b"}]}`)
			assertProblem(t, rr, http.StatusNotFound)
		}
	})
}
