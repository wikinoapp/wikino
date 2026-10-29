package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// stubAttachmentContentsは、添付ファイルの中身を読むユースケースの代わり。受け取った入力を記録し、
// contentとcontentTypeを中身として返す (contentがnilなら中身を返さない)。
// panicValueを指定すると、画像の取得のpanicから回復する経路を確かめるためにpanicする
type stubAttachmentContents struct {
	got         *usecase.GetAPIAttachmentContentInput
	content     []byte
	contentType string
	err         error
	panicValue  any
}

func (s stubAttachmentContents) Execute(_ context.Context, input usecase.GetAPIAttachmentContentInput) (*usecase.GetAPIAttachmentContentOutput, error) {
	if s.panicValue != nil {
		panic(s.panicValue)
	}
	if s.got != nil {
		*s.got = input
	}
	if s.err != nil {
		return nil, s.err
	}
	return &usecase.GetAPIAttachmentContentOutput{
		Attachment:  &model.Attachment{ID: input.AttachmentID},
		Content:     s.content,
		ContentType: s.contentType,
	}, nil
}

// attachmentAPIResponseは、偽の公開APIが返す添付ファイルの情報
const attachmentAPIResponse = `{"id":"01234567-89ab-cdef-0123-456789abcdef","filename":"diagram.png","content_type":"image/png","byte_size":4,"download_url":"https://storage.example/diagram.png","download_url_expires_at":"2026-10-28T00:05:00Z"}`

// attachmentArgumentsは、getAttachmentの引数
const attachmentArguments = `{"attachment_id":"01234567-89ab-cdef-0123-456789abcdef"}`

// attachmentAPIは、添付ファイルの情報を返す偽の公開APIを返す
func attachmentAPI(got *recordedAPIRequest) http.Handler {
	return fakeAPI(got, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(attachmentAPIResponse))
	})
}

// callAttachmentToolは、添付ファイルの中身をcontentsで読むMCPサーバーのgetAttachmentを、apiへ委譲して呼ぶ
func callAttachmentTool(t *testing.T, api http.Handler, contents stubAttachmentContents) toolCallResult {
	t.Helper()

	server := newTestServerWith(t, stubAuthenticator{scopes: []model.Scope{model.ScopeAttachmentRead}}, api, contents)
	return callServerTool(t, server, attachmentToolName, attachmentArguments, i18n.LangJa)
}

func TestCallTool_添付ファイルの画像は中身を画像として返す(t *testing.T) {
	t.Parallel()

	var apiReq recordedAPIRequest
	var got usecase.GetAPIAttachmentContentInput
	png := []byte("\x89PNG")
	result := callAttachmentTool(t, attachmentAPI(&apiReq), stubAttachmentContents{got: &got, content: png, contentType: "image/png"})

	if apiReq.uri != "/api/v1/spaces/my-space/attachments/01234567-89ab-cdef-0123-456789abcdef" {
		t.Errorf("公開APIへの要求 = %s、期待値 = 添付ファイルの取得", apiReq.uri)
	}
	if got.AttachmentID != "01234567-89ab-cdef-0123-456789abcdef" || got.SpaceIdentifier != "my-space" || got.Principal == nil {
		t.Errorf("中身の取得の入力 = %+v、期待値 = 引数の添付ファイル・トークンのスペース・呼び出し主体", got)
	}
	if got.MaxBytes != attachmentImageMaxBytes || !slices.Equal(got.ContentTypes, attachmentImageContentTypes) {
		t.Errorf("中身の取得の条件 = (%v, %d)、期待値 = (%v, %d)", got.ContentTypes, got.MaxBytes, attachmentImageContentTypes, attachmentImageMaxBytes)
	}

	if result.IsError || len(result.Content) != 2 {
		t.Fatalf("結果 = %+v、画像とファイルの情報の2つを期待", result)
	}
	if image := result.Content[0]; image.Type != "image" || image.MIMEType != "image/png" || !bytes.Equal(image.Data, png) {
		t.Errorf("1つ目 = %+v、期待値 = PNGの画像", image)
	}
	if info := result.Content[1]; info.Type != "text" || info.Text != attachmentAPIResponse {
		t.Errorf("2つ目 = %+v、期待値 = 公開APIの応答のテキスト", info)
	}
	if result.StructuredContent["download_url"] != "https://storage.example/diagram.png" {
		t.Errorf("structuredContent = %v、期待値 = 公開APIの応答", result.StructuredContent)
	}
}

func TestCallTool_添付ファイルの中身を返せないときは公開APIの応答のまま返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contents stubAttachmentContents
		// wantErrorLogは、中身の取得の失敗をErrorのログに残すときのエラーの文面。残さないなら空
		wantErrorLog string
	}{
		{name: "画像でないか上限を超える (中身を返さない)", contents: stubAttachmentContents{}},
		{
			name:         "中身の取得に失敗する",
			contents:     stubAttachmentContents{err: errors.New("ストレージに接続できない")},
			wantErrorLog: "ストレージに接続できない",
		},
		{
			name:     "公開APIを呼んだ後に見られなくなった",
			contents: stubAttachmentContents{err: &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: "見つかりません"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var apiReq recordedAPIRequest
			var logs bytes.Buffer
			server := newTestServerWith(t, stubAuthenticator{scopes: []model.Scope{model.ScopeAttachmentRead}}, attachmentAPI(&apiReq), tt.contents)
			server.logger = slog.New(slog.NewJSONHandler(&logs, nil))
			result := callServerTool(t, server, attachmentToolName, attachmentArguments, i18n.LangJa)

			if result.IsError {
				t.Fatalf("isError = true、成功を期待 (結果 = %v)", result.texts())
			}
			if len(result.Content) != 1 || result.Content[0].Type != "text" || result.Content[0].Text != attachmentAPIResponse {
				t.Errorf("結果 = %+v、公開APIの応答のテキストだけを期待", result.Content)
			}

			var errorRecords []map[string]any
			for _, record := range logRecords(t, &logs) {
				if record["level"] == "ERROR" {
					errorRecords = append(errorRecords, record)
				}
			}
			if tt.wantErrorLog == "" {
				if len(errorRecords) != 0 {
					t.Errorf("Errorのログ = %v、残さないことを期待", errorRecords)
				}
				return
			}
			if len(errorRecords) != 1 || errorRecords[0]["msg"] != "MCPのツールで添付ファイルの中身の取得に失敗しました" ||
				errorRecords[0]["error"] != tt.wantErrorLog {
				t.Errorf("Errorのログ = %v、中身の取得の失敗 (%q) の1件を期待", errorRecords, tt.wantErrorLog)
			}
		})
	}
}

func TestCallTool_添付ファイルの公開APIがエラーなら中身を読まない(t *testing.T) {
	t.Parallel()

	var apiReq recordedAPIRequest
	api := fakeAPI(&apiReq, func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"about:blank","title":"Not Found","status":404,"detail":null}`))
	})
	var got usecase.GetAPIAttachmentContentInput
	result := callAttachmentTool(t, api, stubAttachmentContents{got: &got, content: []byte("\x89PNG"), contentType: "image/png"})

	if !result.IsError {
		t.Errorf("isError = false、エラーを期待 (結果 = %+v)", result)
	}
	if got.AttachmentID != "" {
		t.Errorf("中身の取得の入力 = %+v、公開APIがエラーなら読まないことを期待", got)
	}
}

func TestCallTool_画像の取得のpanicは回復して公開APIの結果を返す(t *testing.T) {
	t.Parallel()

	var apiReq recordedAPIRequest
	attachments := attachmentAPI(&apiReq)
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/spaces/my-space" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"ホーム"}`))
			return
		}
		attachments.ServeHTTP(w, r)
	})
	var logs bytes.Buffer
	const panicValue = "画像の取得のpanic"
	server := newTestServerWith(t, stubAuthenticator{scopes: []model.Scope{model.ScopeAttachmentRead}}, api,
		stubAttachmentContents{panicValue: panicValue})
	server.logger = slog.New(slog.NewJSONHandler(&logs, nil))

	result := callServerTool(t, server, attachmentToolName, attachmentArguments, i18n.LangJa)

	if result.IsError || len(result.Content) != 1 || result.Content[0].Type != "text" || result.Content[0].Text != attachmentAPIResponse {
		t.Fatalf("結果 = %+v、公開APIの成功のテキストだけを期待", result)
	}
	if result.StructuredContent["download_url"] != "https://storage.example/diagram.png" {
		t.Errorf("structuredContent = %v、署名付きURLを保持した公開APIの応答を期待", result.StructuredContent)
	}

	var panicRecord map[string]any
	for _, record := range logRecords(t, &logs) {
		if record["msg"] == "MCPのツールで添付ファイルの画像の取得中にpanicしました" {
			panicRecord = record
		}
	}
	if panicRecord == nil || panicRecord["level"] != "ERROR" || panicRecord["operation_id"] != attachmentToolName || panicRecord["panic"] != panicValue {
		t.Errorf("panicのログ = %v、ErrorレベルでoperationIdとpanicの値を期待", panicRecord)
	}
	if stack, ok := panicRecord["stack"].(string); !ok || stack == "" {
		t.Errorf("panicのスタック = %v、スタックの記録を期待", panicRecord["stack"])
	}
	for _, value := range []string{"diagram.png", "https://storage.example/diagram.png"} {
		if strings.Contains(logs.String(), value) {
			t.Errorf("ログにファイルの内容%qが含まれている", value)
		}
	}

	// 同じサーバーで別のツールを続けて呼べることも確認する
	result = callServerTool(t, server, "getSpace", `{}`, i18n.LangJa)
	if result.IsError || result.StructuredContent["name"] != "ホーム" {
		t.Errorf("継続した呼び出しの結果 = %+v、getSpaceの成功を期待", result)
	}
}
