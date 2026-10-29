package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// recordedAPIRequestは、委譲先の偽の公開APIが受け取った要求
type recordedAPIRequest struct {
	method      string
	uri         string
	header      http.Header
	body        string
	principal   *model.APIPrincipal
	locale      string
	contentType string
}

// fakeAPIは、受け取った要求を記録し、respondの応答を返す偽の公開APIのハンドラーを返す
func fakeAPI(got *recordedAPIRequest, respond func(w http.ResponseWriter)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*got = recordedAPIRequest{
			method:      r.Method,
			uri:         r.URL.RequestURI(),
			header:      r.Header,
			body:        string(body),
			principal:   middleware.APIPrincipalFromContext(r.Context()),
			locale:      i18n.GetLocale(r.Context()),
			contentType: r.Header.Get("Content-Type"),
		}
		respond(w)
	})
}

// toolCallResultはtools/callの応答の結果
type toolCallResult struct {
	Content []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Data     []byte `json:"data"`
		MIMEType string `json:"mimeType"`
	} `json:"content"`
	StructuredContent map[string]any `json:"structuredContent"`
	IsError           bool           `json:"isError"`
}

// textsは結果のテキストを並べる
func (r toolCallResult) texts() []string {
	texts := make([]string, len(r.Content))
	for i, c := range r.Content {
		texts[i] = c.Text
	}
	return texts
}

// callToolは、MCPサーバーのツールnameを引数argumentsで呼び、結果を返す
func callTool(t *testing.T, authenticator stubAuthenticator, apiHandler http.Handler, name, arguments string) toolCallResult {
	t.Helper()
	return callToolInLocale(t, authenticator, apiHandler, name, arguments, i18n.LangEn)
}

// callToolInLocaleは、HTTPのAccept-Languageで表示言語を指定してcallToolと同じくツールを呼ぶ。
// SDKを経由しても、MCP独自の文言が要求の表示言語に従うことを確かめるために使う
func callToolInLocale(t *testing.T, authenticator stubAuthenticator, apiHandler http.Handler, name, arguments, locale string) toolCallResult {
	t.Helper()
	return callServerTool(t, newTestServer(t, authenticator, apiHandler), name, arguments, locale)
}

// callServerToolは、MCPサーバーserverのツールnameを、表示言語localeと引数argumentsで呼び、結果を返す。
// 同じサーバーで続けて呼ぶときや、ロガーを差し替えたサーバーで呼ぶときに使う
func callServerTool(t *testing.T, server *Server, name, arguments, locale string) toolCallResult {
	t.Helper()

	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, arguments)
	rr := serveMCPServer(t, server, http.MethodPost, body, testMCPToken, http.Header{"Accept-Language": {locale}})
	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d (本文 = %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
	var resp struct {
		Result *toolCallResult `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("本文のJSONの解析に失敗: %v (本文 = %s)", err, rr.Body.String())
	}
	if resp.Result == nil {
		t.Fatalf("ツールの結果が無い (本文 = %s)", rr.Body.String())
	}
	return *resp.Result
}

func TestCallTool_引数から公開APIへの要求を組み立て照合済みの呼び出し主体を渡す(t *testing.T) {
	t.Parallel()

	var got recordedAPIRequest
	api := fakeAPI(&got, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte(`{"number":42,"title":"ホーム"}`))
	})
	authenticator := stubAuthenticator{scopes: []model.Scope{model.ScopePageWrite}}

	result := callTool(t, authenticator, api, "updatePage", `{"page_number":42,"if_match":"\"v0\"","title":"ホーム"}`)

	if got.method != http.MethodPatch || got.uri != "/api/v1/spaces/my-space/pages/42" {
		t.Errorf("要求 = %s %s、期待値 = PATCH /api/v1/spaces/my-space/pages/42", got.method, got.uri)
	}
	if v := got.header.Get("If-Match"); v != `"v0"` {
		t.Errorf("If-Match = %q、期待値 = %q", v, `"v0"`)
	}
	if got.contentType != "application/merge-patch+json" || got.body != `{"title":"ホーム"}` {
		t.Errorf("本文 = %s (%s)、期待値 = {\"title\":\"ホーム\"} (application/merge-patch+json)", got.body, got.contentType)
	}
	if got.principal == nil || got.principal.SpaceMember.ID != "space-member-1" {
		t.Errorf("呼び出し主体 = %+v、照合で得た主体を期待", got.principal)
	}
	if got.locale != i18n.LangEn {
		t.Errorf("表示言語 = %q、期待値 = %q (MCPの要求の表示言語)", got.locale, i18n.LangEn)
	}

	if result.IsError {
		t.Errorf("isError = true、成功を期待 (結果 = %v)", result.texts())
	}
	if result.StructuredContent["number"] != float64(42) {
		t.Errorf("structuredContent = %v、期待値 = 応答のJSON", result.StructuredContent)
	}
	if texts := result.texts(); len(texts) != 2 || texts[0] != `{"number":42,"title":"ホーム"}` || texts[1] != `ETag: "v1"` {
		t.Errorf("テキスト = %q、期待値 = 応答のJSONとETag", texts)
	}
}

func TestCallTool_クエリの引数と本文の無い応答(t *testing.T) {
	t.Parallel()

	var got recordedAPIRequest
	api := fakeAPI(&got, func(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) })
	authenticator := stubAuthenticator{scopes: []model.Scope{model.ScopeDraftPageDelete}}

	result := callTool(t, authenticator, api, "deleteDraftPage", `{"page_number":7}`)
	if got.method != http.MethodDelete || got.uri != "/api/v1/spaces/my-space/pages/7/draft_page" || got.body != "" {
		t.Errorf("要求 = %s %s (本文 %q)、期待値 = DELETE /api/v1/spaces/my-space/pages/7/draft_page (本文なし)", got.method, got.uri, got.body)
	}
	if result.IsError || len(result.Content) != 1 || !strings.Contains(result.Content[0].Text, "204") {
		t.Errorf("結果 = %+v、成功を表すテキストを期待", result)
	}

	api = fakeAPI(&got, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"next_cursor":null}`))
	})
	callTool(t, stubAuthenticator{}, api, "listPages", `{"topic_number":3,"q":"Go 移行","status":"all"}`)
	if got.method != http.MethodGet || got.uri != "/api/v1/spaces/my-space/pages?q=Go+%E7%A7%BB%E8%A1%8C&status=all&topic_number=3" {
		t.Errorf("要求 = %s %s、期待値 = クエリを付けたGET", got.method, got.uri)
	}
}

func TestCallTool_ProblemDetailsの応答はisErrorにして内容を返す(t *testing.T) {
	t.Parallel()

	var got recordedAPIRequest
	api := fakeAPI(&got, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"type":"about:blank","title":"入力が正しくありません","status":422,"detail":"送られた値を受け付けられません",` +
			`"errors":[{"detail":"同じタイトルのページがあります","pointer":"#/title","parameter":null},{"detail":"1以上にしてください","pointer":null,"parameter":"topic_number"}]}`))
	})
	authenticator := stubAuthenticator{scopes: []model.Scope{model.ScopePageWrite}}

	result := callTool(t, authenticator, api, "createPage", `{"topic_number":0,"title":"ホーム","body":""}`)

	if !result.IsError {
		t.Fatalf("isError = false、エラーを期待 (結果 = %v)", result.texts())
	}
	want := "Status 422\n入力が正しくありません\n送られた値を受け付けられません\n- #/title: 同じタイトルのページがあります\n- topic_number: 1以上にしてください"
	if texts := result.texts(); len(texts) != 1 || texts[0] != want {
		t.Errorf("テキスト = %q、期待値 = %q", texts, want)
	}
}

func TestCallTool_引数の誤りは公開APIを呼ばずにisErrorにする(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		arguments string
		want      string
	}{
		{name: "定義していない引数", arguments: `{"page_number":1,"space_identifier":"other-space"}`, want: "space_identifier"},
		{name: "パスの引数が無い", arguments: `{}`, want: "page_number"},
		{name: "パスの引数がオブジェクト", arguments: `{"page_number":{"n":1}}`, want: "page_number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			api := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

			result := callTool(t, stubAuthenticator{}, api, "getPage", tt.arguments)

			if called {
				t.Error("公開APIを呼んだ")
			}
			if texts := result.texts(); !result.IsError || len(texts) != 1 || !strings.Contains(texts[0], tt.want) {
				t.Errorf("結果 = %+v、%qを含むエラーを期待", result, tt.want)
			}
		})
	}
}

func TestCallTool_要求の表示言語で独自の文言を返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tool        string
		arguments   string
		scopes      []model.Scope
		status      int
		contentType string
		response    string
		etag        string
		retryAfter  string
		wantJa      string
		wantEn      string
		wantError   bool
	}{
		{
			name: "引数がオブジェクトでない", tool: "getSpace", arguments: `[]`, wantError: true,
			wantJa: "引数がJSONのオブジェクトではありません", wantEn: "Arguments must be a JSON object",
		},
		{
			name: "定義していない引数", tool: "getPage", arguments: `{"page_number":1,"space_identifier":"other-space"}`, wantError: true,
			wantJa: "引数space_identifierはこのツールに定義していません", wantEn: "Argument space_identifier is not defined for this tool",
		},
		{
			name: "必須のパスの引数が無い", tool: "getPage", arguments: `{}`, wantError: true,
			wantJa: "引数page_numberがありません", wantEn: "Argument page_number is missing",
		},
		{
			name: "パスの引数がオブジェクト", tool: "getPage", arguments: `{"page_number":{}}`, wantError: true,
			wantJa: "引数page_number: 文字列・数値・真偽値を指定してください", wantEn: "Argument page_number: Specify a string, number, or boolean",
		},
		{
			name: "配列の要素がオブジェクト", tool: "listPages", arguments: `{"q":[{}]}`, wantError: true,
			wantJa: "引数q: 配列の要素には文字列・数値・真偽値を指定してください", wantEn: "Argument q: Array items must be strings, numbers, or booleans",
		},
		{
			name: "パスの引数が複数の値", tool: "getPage", arguments: `{"page_number":[1,2]}`, wantError: true,
			wantJa: "引数page_numberには1つの値を指定してください", wantEn: "Specify a single value for argument page_number",
		},
		{
			name: "本文の無い成功", tool: "deleteDraftPage", arguments: `{"page_number":7,"if_match":"*"}`,
			scopes: []model.Scope{model.ScopeDraftPageDelete}, status: http.StatusNoContent,
			wantJa: "成功しました (ステータス 204)", wantEn: "Succeeded (status 204)",
		},
		{
			name: "Problem Detailsの前置き", tool: "getPage", arguments: `{"page_number":7}`, wantError: true,
			status: http.StatusNotFound, contentType: "application/problem+json",
			response: `{"title":"Not Found","detail":"The page was not found.","errors":[{"parameter":"page_number","detail":"No matching page."}]}`,
			wantJa:   "ステータス 404\nNot Found\nThe page was not found.\n- page_number: No matching page.",
			wantEn:   "Status 404\nNot Found\nThe page was not found.\n- page_number: No matching page.",
		},
		{
			name: "Problem Details以外の前置き", tool: "getPage", arguments: `{"page_number":7}`, wantError: true,
			status: http.StatusServiceUnavailable, contentType: "text/plain", response: "Unavailable",
			wantJa: "ステータス 503\nUnavailable", wantEn: "Status 503\nUnavailable",
		},
		{
			name: "429のRetry-After", tool: "getPage", arguments: `{"page_number":7}`, wantError: true,
			status: http.StatusTooManyRequests, contentType: "application/problem+json", retryAfter: "30",
			response: `{"type":"about:blank","title":"Too Many Requests","status":429}`,
			wantJa:   "ステータス 429\nToo Many Requests\nRetry-After: 30",
			wantEn:   "Status 429\nToo Many Requests\nRetry-After: 30",
		},
		{
			name: "JSONとETagは書き換えない", tool: "getSpace", arguments: `{}`,
			status: http.StatusOK, contentType: "application/json", response: `{"name":"ホーム"}`, etag: `"v1"`,
			wantJa: `{"name":"ホーム"}`, wantEn: `{"name":"ホーム"}`,
		},
	}
	for _, tt := range tests {
		for _, locale := range []string{i18n.LangJa, i18n.LangEn} {
			t.Run(tt.name+"/"+locale, func(t *testing.T) {
				t.Parallel()

				called := false
				api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called = true
					if tt.status == 0 {
						t.Error("引数のエラーなのに公開APIを呼んだ")
						return
					}
					if got := i18n.GetLocale(r.Context()); got != locale {
						t.Errorf("公開APIの表示言語 = %q、期待値 = %q", got, locale)
					}
					w.Header().Set("Content-Type", tt.contentType)
					w.Header().Set("ETag", tt.etag)
					w.Header().Set("Retry-After", tt.retryAfter)
					w.WriteHeader(tt.status)
					if tt.response != "" {
						_, _ = w.Write([]byte(tt.response))
					}
				})
				result := callToolInLocale(t, stubAuthenticator{scopes: tt.scopes}, api, tt.tool, tt.arguments, locale)

				if called != (tt.status != 0) {
					t.Errorf("公開APIの呼び出し = %v、期待値 = %v", called, tt.status != 0)
				}
				if result.IsError != tt.wantError {
					t.Errorf("isError = %v、期待値 = %v", result.IsError, tt.wantError)
				}
				wantCount := 1
				if tt.etag != "" {
					wantCount++
				}
				texts := result.texts()
				if len(texts) != wantCount {
					t.Fatalf("テキスト = %q、%d件を期待", texts, wantCount)
				}
				want := tt.wantJa
				if locale == i18n.LangEn {
					want = tt.wantEn
				}
				if texts[0] != want {
					t.Errorf("テキスト = %q、期待値 = %q", texts[0], want)
				}
				if tt.etag != "" && texts[1] != "ETag: "+tt.etag {
					t.Errorf("ETagのテキスト = %q、期待値 = %q", texts[1], "ETag: "+tt.etag)
				}
				if tt.contentType == "application/json" && result.StructuredContent["name"] != "ホーム" {
					t.Errorf("structuredContent = %v、APIのJSONを期待", result.StructuredContent)
				}
			})
		}
	}
}

// logRecordsは、JSONのハンドラーのロガーがlogsに書いたログを1件ずつ解析して返す
func logRecords(t *testing.T, logs *bytes.Buffer) []map[string]any {
	t.Helper()

	var records []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	for decoder.More() {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			t.Fatalf("ログの解析に失敗: %v (ログ = %s)", err, logs.String())
		}
		records = append(records, record)
	}
	return records
}

func TestCallTool_公開APIのハンドラーのpanicは回復して500のisErrorにする(t *testing.T) {
	t.Parallel()

	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/spaces/my-space/pages/7" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"number":7}`))
			panic("公開APIのハンドラーのpanic")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"ホーム"}`))
	})
	var logs bytes.Buffer
	server := newTestServer(t, stubAuthenticator{}, api)
	server.logger = slog.New(slog.NewJSONHandler(&logs, nil))

	result := callServerTool(t, server, "getPage", `{"page_number":7}`, i18n.LangEn)

	if !result.IsError {
		t.Fatalf("isError = false、エラーを期待 (結果 = %v)", result.texts())
	}
	// panicの前に書かれた途中の応答は返さない
	want := "Status 500\nInternal Server Error"
	if texts := result.texts(); len(texts) != 1 || texts[0] != want || result.StructuredContent != nil {
		t.Errorf("結果 = %+v、テキスト %qだけを期待", result, want)
	}

	var panicRecord, callRecord map[string]any
	for _, record := range logRecords(t, &logs) {
		switch record["msg"] {
		case "MCPのツールの呼び出しで公開APIのハンドラーがpanicしました":
			panicRecord = record
		case "MCPのツールを呼び出しました":
			callRecord = record
		}
	}
	if panicRecord == nil || panicRecord["level"] != "ERROR" || panicRecord["operation_id"] != "getPage" ||
		panicRecord["panic"] != "公開APIのハンドラーのpanic" || panicRecord["stack"] == "" {
		t.Errorf("panicのログ = %v、エラーのレベルでoperationId・panicの値・スタックを期待", panicRecord)
	}
	if callRecord == nil || callRecord["status"] != float64(http.StatusInternalServerError) {
		t.Errorf("呼び出しのログ = %v、ステータス500を期待", callRecord)
	}

	// 回復したあとも、同じサーバーで続けてツールを呼べる
	result = callServerTool(t, server, "getSpace", `{}`, i18n.LangEn)
	if result.IsError || result.StructuredContent["name"] != "ホーム" {
		t.Errorf("結果 = %+v、getSpaceの成功を期待", result)
	}
}

func TestCallTool_呼び出しをログに残し引数と本文の値は残さない(t *testing.T) {
	t.Parallel()

	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"number":42,"title":"応答の本文の値"}`))
	})
	var logs bytes.Buffer
	server := newTestServer(t, stubAuthenticator{scopes: []model.Scope{model.ScopePageWrite}}, api)
	server.logger = slog.New(slog.NewJSONHandler(&logs, nil))

	callServerTool(t, server, "updatePage", `{"page_number":42,"if_match":"\"etag-value\"","title":"引数の値"}`, i18n.LangJa)

	records := logRecords(t, &logs)
	if len(records) != 1 {
		t.Fatalf("ログ = %d件、1件を期待 (ログ = %s)", len(records), logs.String())
	}
	want := map[string]any{
		"level":                "INFO",
		"msg":                  "MCPのツールを呼び出しました",
		"operation_id":         "updatePage",
		"space_id":             "space-1",
		"space_member_id":      "space-member-1",
		"oauth_application_id": string(testOAuthApplicationID),
		"status":               float64(http.StatusOK),
	}
	for key, value := range want {
		if records[0][key] != value {
			t.Errorf("ログの%s = %v、期待値 = %v", key, records[0][key], value)
		}
	}
	for _, value := range []string{"引数の値", "etag-value", "応答の本文の値"} {
		if strings.Contains(logs.String(), value) {
			t.Errorf("ログに%qが含まれている (ログ = %s)", value, logs.String())
		}
	}
}
