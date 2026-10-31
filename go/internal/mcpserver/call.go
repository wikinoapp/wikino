package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// apiPathPrefixは、OpenAPI記述のパスの前に付ける公開APIのパス (記述の `servers`)
const apiPathPrefix = model.APIBasePath

// problemMediaTypeは、公開APIがエラーを返すProblem Details (RFC 9457) のメディアタイプ
const problemMediaType = "application/problem+json"

// callToolは、ツールの呼び出しを、ツールの引数から組み立てた公開APIへの要求にして、照合済みの呼び出し主体
// principalと要求の表示言語localeを持たせて公開APIのハンドラーへプロセス内で委譲する。
// リクエストの検証・権限の確認・レート制限は公開APIと同じものを通り、応答をツールの結果に変える
func (s *Server) callTool(tool *operationTool, principal *model.APIPrincipal, locale string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// SDKがHTTPの要求のコンテキストを引き継がない場合も、引数のエラーの文言と公開APIの応答を
		// 要求の表示言語にするため、ここで設定する
		ctx = i18n.SetLocale(ctx, locale)
		apiReq, err := tool.newAPIRequest(ctx, principal, req.Params.Arguments)
		if err != nil {
			if ae := model.AsAppError(err); ae != nil {
				s.logger.ErrorContext(ctx, ae.LogString(), "operation_id", tool.tool.Name)
			}
			return toolError(err.Error()), nil
		}
		apiReq = apiReq.WithContext(middleware.SetAPIPrincipalToContext(apiReq.Context(), principal))

		recorder := s.serveAPI(ctx, tool, apiReq)

		logAttrs := []any{
			"operation_id", tool.tool.Name,
			"space_id", principal.Space.ID,
			"space_member_id", principal.SpaceMember.ID,
			"status", recorder.status,
		}
		if principal.OAuthApplicationID != nil {
			logAttrs = append(logAttrs, "oauth_application_id", *principal.OAuthApplicationID)
		}
		s.logger.InfoContext(ctx, "MCPのツールを呼び出しました", logAttrs...)

		result := recorder.toolResult(ctx)
		if tool.tool.Name == attachmentToolName && !result.IsError {
			s.addAttachmentImage(ctx, principal, req.Params.Arguments, result)
		}
		return result, nil
	}
}

// serveAPIは、公開APIのハンドラーへ要求reqを委譲し、受け取った応答を返す。SDKはツールのハンドラーを
// 別のgoroutineで動かし、HTTPの要求のgoroutineにある全体のRecovererはそこで起きたpanicを捕まえられない。
// panicでサーバーのプロセスが落ちないよう、ここで回復してログに残し、500の応答にする
func (s *Server) serveAPI(ctx context.Context, tool *operationTool, req *http.Request) (recorder *responseRecorder) {
	recorder = newResponseRecorder()
	defer func() {
		rvr := recover()
		if rvr == nil {
			return
		}
		s.logger.ErrorContext(ctx, "MCPのツールの呼び出しで公開APIのハンドラーがpanicしました",
			"operation_id", tool.tool.Name, "panic", rvr, "stack", string(debug.Stack()))

		// panicの前に書かれた途中の応答は捨て、公開APIの500と同じくステータスの説明だけを返す
		recorder = newResponseRecorder()
		recorder.header.Set("Content-Type", "text/plain; charset=utf-8")
		recorder.WriteHeader(http.StatusInternalServerError)
		_, _ = recorder.Write([]byte(http.StatusText(http.StatusInternalServerError)))
	}()
	s.apiHandler.ServeHTTP(recorder, req)
	return recorder
}

// newAPIRequestは、ツールの引数argumentsから、トークンが属するスペースの公開APIへの要求を組み立てる。
// 引数が足りない・定義していない・形が合わない場合は、ツールのエラーとして返す文面のエラーを返す。
// 引数によらない内部の失敗は、文面を翻訳した `*model.AppError` にし、内部のエラーはログ用に持たせる。
// 値の形 (型・長さ・列挙値など) は、公開APIのリクエストの検証に任せる
func (t *operationTool) newAPIRequest(ctx context.Context, principal *model.APIPrincipal, arguments json.RawMessage) (*http.Request, error) {
	args := map[string]any{}
	if len(arguments) > 0 && string(arguments) != "null" {
		decoder := json.NewDecoder(bytes.NewReader(arguments))
		decoder.UseNumber()
		// argumentsはJSON-RPCの要求として解析済みのJSONのため、失敗するのはオブジェクトでない場合だけである
		if err := decoder.Decode(&args); err != nil {
			return nil, errors.New(i18n.T(ctx, "validation_mcp_arguments_object"))
		}
	}

	for name := range args {
		if !slices.ContainsFunc(t.params, func(p toolParam) bool { return p.property == name }) && !slices.Contains(t.bodyProperties, name) {
			return nil, errors.New(i18n.T(ctx, "validation_mcp_argument_unknown", map[string]any{"Name": name}))
		}
	}

	path := strings.Replace(t.path, "{"+spaceIdentifierParam+"}", url.PathEscape(string(principal.Space.Identifier)), 1)
	query := url.Values{}
	header := http.Header{}
	for _, param := range t.params {
		value, ok := args[param.property]
		if !ok {
			if param.in == "path" {
				return nil, errors.New(i18n.T(ctx, "validation_mcp_argument_required", map[string]any{"Name": param.property}))
			}
			continue
		}
		values, err := paramValues(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", i18n.T(ctx, "validation_mcp_argument_name", map[string]any{"Name": param.property}), err)
		}
		switch param.in {
		case "path":
			if len(values) != 1 {
				return nil, errors.New(i18n.T(ctx, "validation_mcp_argument_single_value", map[string]any{"Name": param.property}))
			}
			path = strings.Replace(path, "{"+param.name+"}", url.PathEscape(values[0]), 1)
		case "query":
			query[param.name] = values
		case "header":
			header[http.CanonicalHeaderKey(param.name)] = values
		}
	}

	target := apiPathPrefix + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var body []byte
	if t.bodyMediaType != "" {
		fields := map[string]any{}
		for _, name := range t.bodyProperties {
			if value, ok := args[name]; ok {
				fields[name] = value
			}
		}
		var err error
		body, err = json.Marshal(fields)
		if err != nil {
			return nil, &model.AppError{
				Code:     model.AppErrCodeInternal,
				UserMsg:  i18n.T(ctx, "mcp_call_body_error"),
				Internal: fmt.Errorf("リクエスト本文の組み立てに失敗: %w", err),
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, t.method, target, bytes.NewReader(body))
	if err != nil {
		return nil, &model.AppError{
			Code:     model.AppErrCodeInternal,
			UserMsg:  i18n.T(ctx, "mcp_call_request_error"),
			Internal: fmt.Errorf("公開APIへの要求の組み立てに失敗: %w", err),
		}
	}
	req.Header = header
	req.Header.Set("Accept", "application/json")
	if t.bodyMediaType != "" {
		req.Header.Set("Content-Type", t.bodyMediaType)
	}
	return req, nil
}

// paramValuesは、パス・クエリ・ヘッダーに置く引数の値を文字列にする。配列は複数の値にする
func paramValues(ctx context.Context, value any) ([]string, error) {
	switch v := value.(type) {
	case string:
		return []string{v}, nil
	case json.Number:
		return []string{v.String()}, nil
	case bool:
		return []string{strconv.FormatBool(v)}, nil
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			itemValues, err := paramValues(ctx, item)
			if err != nil || len(itemValues) != 1 {
				return nil, errors.New(i18n.T(ctx, "validation_mcp_array_item_scalar"))
			}
			values = append(values, itemValues...)
		}
		return values, nil
	default:
		return nil, errors.New(i18n.T(ctx, "validation_mcp_argument_scalar"))
	}
}

// responseRecorderは、公開APIのハンドラーの応答をメモリ上で受け取るhttp.ResponseWriter
type responseRecorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{header: http.Header{}}
}

// Headerはhttp.ResponseWriterを満たす
func (r *responseRecorder) Header() http.Header {
	return r.header
}

// WriteHeaderはhttp.ResponseWriterを満たす。最初に書いたステータスだけを残す
func (r *responseRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}

// Writeはhttp.ResponseWriterを満たす
func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

// toolResultは、受け取った応答をツールの結果にする。
//
//   - 2xxのJSONの応答は、オブジェクトなら `structuredContent` に入れ、同じJSONをテキストでも返す
//     (`structuredContent` を読まないクライアントのため)。`ETag` があれば、更新のツールの `if_match` に
//     渡せるようテキストで添える
//   - 2xxの本文の無い応答は、成功したことを表すテキストにする
//   - それ以外 (Problem Detailsの応答など) は `isError` の結果にし、ステータスとエラーの内容をテキストで返す。
//     `Retry-After` があれば、呼び直すまで待つ時間の手がかりとして添える
func (r *responseRecorder) toolResult(ctx context.Context) *mcp.CallToolResult {
	status := r.status
	if status == 0 {
		status = http.StatusOK
	}
	if status < 200 || status >= 300 {
		return toolError(r.errorText(ctx, status))
	}

	result := &mcp.CallToolResult{}
	body := bytes.TrimSpace(r.body.Bytes())
	if len(body) == 0 {
		result.Content = []mcp.Content{&mcp.TextContent{Text: i18n.T(ctx, "mcp_call_success", map[string]any{"Status": status})}}
	} else {
		result.Content = []mcp.Content{&mcp.TextContent{Text: string(body)}}
		var structured map[string]any
		if isJSONMediaType(r.header.Get("Content-Type")) && json.Unmarshal(body, &structured) == nil {
			result.StructuredContent = structured
		}
	}
	if etag := r.header.Get("ETag"); etag != "" {
		result.Content = append(result.Content, &mcp.TextContent{Text: "ETag: " + etag})
	}
	return result
}

// problemDetailsは、公開APIのエラーの応答 (Problem Details) のうち、ツールのエラーの文面に使う項目
type problemDetails struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Errors []struct {
		Detail    string  `json:"detail"`
		Pointer   *string `json:"pointer"`
		Parameter *string `json:"parameter"`
	} `json:"errors"`
}

// errorTextは、エラーの応答を、ステータスとエラーの内容 (Problem Detailsなら `title`・`detail`・
// 項目ごとのエラー) のテキストにする。429の待ち時間はProblem Detailsに無く `Retry-After` にしか無いため、
// ヘッダーがあればその値も添える
func (r *responseRecorder) errorText(ctx context.Context, status int) string {
	lines := []string{i18n.T(ctx, "mcp_call_status", map[string]any{"Status": status})}
	var problem problemDetails
	mediaType, _, _ := mime.ParseMediaType(r.header.Get("Content-Type"))
	if mediaType == problemMediaType && json.Unmarshal(r.body.Bytes(), &problem) == nil {
		if problem.Title != "" {
			lines = append(lines, problem.Title)
		}
		if problem.Detail != "" {
			lines = append(lines, problem.Detail)
		}
		for _, e := range problem.Errors {
			switch {
			case e.Pointer != nil:
				lines = append(lines, "- "+*e.Pointer+": "+e.Detail)
			case e.Parameter != nil:
				lines = append(lines, "- "+*e.Parameter+": "+e.Detail)
			default:
				lines = append(lines, "- "+e.Detail)
			}
		}
	} else if body := strings.TrimSpace(r.body.String()); body != "" {
		lines = append(lines, body)
	}
	if retryAfter := r.header.Get("Retry-After"); retryAfter != "" {
		lines = append(lines, "Retry-After: "+retryAfter)
	}
	return strings.Join(lines, "\n")
}

// isJSONMediaTypeは、Content-Typeの値がJSON (`application/json` か `+json` の構造化構文) を表すかを返す
func isJSONMediaType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

// toolErrorは、文面textのツールのエラーの結果を返す
func toolError(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		IsError: true,
	}
}
