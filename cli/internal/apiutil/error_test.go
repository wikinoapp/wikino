package apiutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestConvertError_APIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		resp         *http.Response
		wantCode     string
		wantExit     int
		wantMessage  string
		wantHintPart string
	}{
		{
			name:         "400",
			resp:         problemResponse(400, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"リクエスト本文をJSONとして読めません。"}`),
			wantCode:     output.CodeBadRequest,
			wantExit:     output.ExitValidation,
			wantMessage:  "リクエスト本文をJSONとして読めません。",
			wantHintPart: "JSON",
		},
		{
			name:         "401",
			resp:         problemResponse(401, `{"type":"about:blank","title":"Unauthorized","status":401,"detail":"トークンが無効です。"}`, "WWW-Authenticate", `Bearer error="invalid_token"`),
			wantCode:     output.CodeUnauthorized,
			wantExit:     output.ExitAuth,
			wantMessage:  "トークンが無効です。",
			wantHintPart: "wikino auth login",
		},
		{
			name:         "403で要るスコープが分かる",
			resp:         problemResponse(403, `{"type":"about:blank","title":"Forbidden","status":403,"detail":null}`, "WWW-Authenticate", `Bearer error="insufficient_scope", scope="draft_page:delete"`),
			wantCode:     output.CodeForbidden,
			wantExit:     output.ExitForbidden,
			wantMessage:  "Forbidden",
			wantHintPart: "スコープ draft_page:delete",
		},
		{
			name:         "403で要るスコープが分からない",
			resp:         problemResponse(403, `{"type":"about:blank","title":"Forbidden","status":403,"detail":null}`),
			wantCode:     output.CodeForbidden,
			wantExit:     output.ExitForbidden,
			wantMessage:  "Forbidden",
			wantHintPart: "wikino auth status",
		},
		{
			name:         "404",
			resp:         problemResponse(404, `{"type":"about:blank","title":"Not Found","status":404,"detail":"ページが見つかりません。"}`),
			wantCode:     output.CodeNotFound,
			wantExit:     output.ExitNotFound,
			wantMessage:  "ページが見つかりません。",
			wantHintPart: "--space",
		},
		{
			name:         "409",
			resp:         problemResponse(409, `{"type":"https://example.com/api/problems/already-exists","title":"Conflict","status":409,"detail":"下書きは既にあります。"}`),
			wantCode:     output.CodeConflict,
			wantExit:     output.ExitConflict,
			wantMessage:  "下書きは既にあります。",
			wantHintPart: "`get`",
		},
		{
			name:         "412",
			resp:         problemResponse(412, `{"type":"about:blank","title":"Precondition Failed","status":412,"detail":"ページはその後に更新されています。"}`),
			wantCode:     output.CodePreconditionFailed,
			wantExit:     output.ExitConflict,
			wantMessage:  "ページはその後に更新されています。",
			wantHintPart: "--if-match",
		},
		{
			name:         "428",
			resp:         problemResponse(428, `{"type":"about:blank","title":"Precondition Required","status":428,"detail":null}`),
			wantCode:     output.CodePreconditionRequired,
			wantExit:     output.ExitConflict,
			wantMessage:  "Precondition Required",
			wantHintPart: "--if-match",
		},
		{
			name:         "422は項目ごとの問題をメッセージに並べる",
			resp:         problemResponse(422, `{"type":"about:blank","title":"Unprocessable Entity","status":422,"detail":"入力が正しくありません。","errors":[{"detail":"重複しています","pointer":"#/title","parameter":null},{"detail":"多すぎます","pointer":null,"parameter":"q"}]}`),
			wantCode:     output.CodeValidationFailed,
			wantExit:     output.ExitValidation,
			wantMessage:  "入力が正しくありません。 (#/title: 重複しています; q: 多すぎます)",
			wantHintPart: "入力を直して",
		},
		{
			name:         "429で再試行までの時間が分かる",
			resp:         problemResponse(429, `{"type":"about:blank","title":"Too Many Requests","status":429,"detail":null}`, "Retry-After", "42"),
			wantCode:     output.CodeRateLimited,
			wantExit:     output.ExitRateLimited,
			wantMessage:  "Too Many Requests",
			wantHintPart: "42秒",
		},
		{
			name:         "429で再試行までの時間が分からない",
			resp:         problemResponse(429, `{"type":"about:blank","title":"Too Many Requests","status":429,"detail":null}`),
			wantCode:     output.CodeRateLimited,
			wantExit:     output.ExitRateLimited,
			wantMessage:  "Too Many Requests",
			wantHintPart: "しばらく待って",
		},
		{
			name: "5xxで本文がProblem Detailsでない",
			resp: &http.Response{
				StatusCode: 503,
				Header:     http.Header{"Content-Type": {"text/html"}},
				Body:       io.NopCloser(strings.NewReader("<html>unavailable</html>")),
			},
			wantCode:     output.CodeServerError,
			wantExit:     output.ExitError,
			wantMessage:  "APIがエラーを返しました (HTTP 503)",
			wantHintPart: "読み取りのコマンドで確かめて",
		},
		{
			name:         "分類していない4xx",
			resp:         problemResponse(415, `{"type":"about:blank","title":"Unsupported Media Type","status":415,"detail":null}`),
			wantCode:     output.CodeAPIError,
			wantExit:     output.ExitError,
			wantMessage:  "Unsupported Media Type",
			wantHintPart: "--help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loc := i18n.NewLocalizer(i18n.LangJa)
			c := newTestClient(t, loc, nil, func(*http.Request) (*http.Response, error) {
				return tt.resp, nil
			})

			_, err := c.GetPageWithResponse(context.Background(), "my-space", 12, nil)

			var cliErr *output.Error
			if !errors.As(ConvertError(loc, err), &cliErr) {
				t.Fatalf("CLIのエラーになりません: %v", err)
			}
			if cliErr.Code != tt.wantCode || cliErr.ExitCode != tt.wantExit || cliErr.Status != tt.resp.StatusCode {
				t.Errorf("分類が違います: code=%q exit=%d status=%d", cliErr.Code, cliErr.ExitCode, cliErr.Status)
			}
			if cliErr.Message != tt.wantMessage {
				t.Errorf("メッセージが違います:\n実際  %q\n期待 %q", cliErr.Message, tt.wantMessage)
			}
			if !strings.Contains(cliErr.Hint, tt.wantHintPart) {
				t.Errorf("ヒントが違います: %q に %q が含まれません", cliErr.Hint, tt.wantHintPart)
			}
		})
	}
}

// 英語でも翻訳が揃っていることを、翻訳キーがそのまま出ないことで確かめる。
func TestConvertError_English(t *testing.T) {
	t.Parallel()

	loc := i18n.NewLocalizer(i18n.LangEn)
	for _, status := range []int{400, 401, 403, 404, 409, 412, 415, 422, 428, 429, 500} {
		err := ConvertError(loc, &apiclient.Error{StatusCode: status})

		var cliErr *output.Error
		if !errors.As(err, &cliErr) {
			t.Fatalf("%d: CLIのエラーになりません", status)
		}
		if strings.Contains(cliErr.Hint, "error_api_") || strings.Contains(cliErr.Message, "error_api_") {
			t.Errorf("%d: 翻訳がありません: %+v", status, cliErr)
		}
	}
}

func TestConvertError_RetryAfterSeconds(t *testing.T) {
	t.Parallel()

	loc := i18n.NewLocalizer(i18n.LangEn)
	err := ConvertError(loc, &apiclient.Error{StatusCode: 429, RetryAfter: 90 * time.Second, HasRetryAfter: true})

	var cliErr *output.Error
	if !errors.As(err, &cliErr) || !strings.Contains(cliErr.Hint, "Wait 90 seconds") {
		t.Errorf("ヒントが違います: %+v", cliErr)
	}
}

func TestConvertError_Network(t *testing.T) {
	t.Parallel()

	loc := i18n.NewLocalizer(i18n.LangJa)
	c := newTestClient(t, loc, nil, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})

	_, err := c.GetPageWithResponse(context.Background(), "my-space", 12, nil)

	var cliErr *output.Error
	if !errors.As(ConvertError(loc, err), &cliErr) {
		t.Fatalf("CLIのエラーになりません: %v", err)
	}
	if cliErr.Code != output.CodeNetwork || cliErr.ExitCode != output.ExitError {
		t.Errorf("通信の失敗の分類が違います: %+v", cliErr)
	}
	if !strings.Contains(cliErr.Message, "connection refused") {
		t.Errorf("原因がメッセージにありません: %q", cliErr.Message)
	}
}

func TestConvertError_InputError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason apiclient.InputErrorReason
		want   string
	}{
		{apiclient.ReasonEmpty, "値が空です"},
		{apiclient.ReasonControlCharacter, "制御文字"},
		{apiclient.ReasonDotSegment, "`..`"},
		{apiclient.ReasonReservedCharacter, "URLやクエリを付けず"},
		{apiclient.ReasonInvalidFormat, "形式が誤っています"},
		{apiclient.ReasonInsecureScheme, "HTTPS"},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			t.Parallel()

			loc := i18n.NewLocalizer(i18n.LangJa)
			err := ConvertError(loc, &apiclient.InputError{Name: "--space", Value: "x", Reason: tt.reason})

			var cliErr *output.Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("CLIのエラーになりません: %v", err)
			}
			if cliErr.Code != output.CodeUsage || cliErr.ExitCode != output.ExitUsage {
				t.Errorf("使い方の誤りになりません: %+v", cliErr)
			}
			if !strings.Contains(cliErr.Message, "--space") || !strings.Contains(cliErr.Message, tt.want) {
				t.Errorf("メッセージが違います: %q", cliErr.Message)
			}
			if !strings.Contains(cliErr.Hint, "要求はまだ送っていません") {
				t.Errorf("ヒントが違います: %q", cliErr.Hint)
			}
		})
	}
}

func TestConvertError_Passthrough(t *testing.T) {
	t.Parallel()

	loc := i18n.NewLocalizer(i18n.LangJa)
	if ConvertError(loc, nil) != nil {
		t.Error("nilを変換するとnilになるはずです")
	}

	for _, passthrough := range []error{ErrDryRun, &output.Error{Code: output.CodeUsage, ExitCode: output.ExitUsage}} {
		if got := ConvertError(loc, passthrough); got != passthrough {
			t.Errorf("既存のCLIエラーかdry-runのエラーが変わっています: %v", got)
		}
	}

	other := errors.New("別のエラー")
	if got := ConvertError(loc, other); got != other {
		t.Errorf("変換の対象でないエラーが変わっています: %v", got)
	}

	// 送信後のリダイレクト拒否は `*url.Error` に包まれても、未送信の使い方の誤りと区別し、専用のメッセージにする。
	wrapped := &url.Error{Op: "Get", URL: "http://example.com", Err: apiclient.ErrInsecureRedirect}
	var cliErr *output.Error
	if !errors.As(ConvertError(loc, wrapped), &cliErr) || cliErr.Code != output.CodeNetwork || cliErr.Message != loc.T("error_insecure_redirect") {
		t.Errorf("リダイレクトの拒否が送信後の失敗になりません: %+v", cliErr)
	}
}

// エラーのメッセージ・ヒント・JSONに、送ったトークンが含まれないことを確かめる。
func TestConvertError_NoSecret(t *testing.T) {
	t.Parallel()

	for _, status := range []int{400, 401, 403, 404, 409, 412, 422, 428, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			t.Parallel()

			loc := i18n.NewLocalizer(i18n.LangJa)
			c := newTestClient(t, loc, nil, func(*http.Request) (*http.Response, error) {
				return problemResponse(status, fmt.Sprintf(`{"type":"about:blank","title":"t","status":%d,"detail":null}`, status),
					"WWW-Authenticate", `Bearer error="insufficient_scope", scope="page:write"`, "Retry-After", "1"), nil
			})

			_, err := c.GetPageWithResponse(context.Background(), "my-space", 12, nil)

			var buf bytes.Buffer
			if werr := output.WriteError(&buf, ConvertError(loc, err), true); werr != nil {
				t.Fatal(werr)
			}
			if werr := output.WriteError(&buf, ConvertError(loc, err), false); werr != nil {
				t.Fatal(werr)
			}
			if strings.Contains(buf.String(), testToken) {
				t.Errorf("エラーの出力にトークンが含まれています: %s", buf.String())
			}
		})
	}
}

func TestConvertError_RedirectAfterWrite(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()
			loc := i18n.NewLocalizer(lang)
			sent := 0
			// 既定のクライアントのリダイレクト制限を通すため、トランスポートだけを差し替える。
			rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				sent++
				if req.Method != http.MethodPost || req.Header.Get("Authorization") != "Bearer "+testToken {
					t.Errorf("送信された要求が違います: %s %v", req.Method, req.Header)
				}
				return &http.Response{
					StatusCode: http.StatusTemporaryRedirect,
					Header:     http.Header{"Location": {"http://example.com/elsewhere"}},
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    req,
				}, nil
			})
			c, err := NewClient(loc, ClientOptions{Host: testHost, Token: apiclient.StaticToken(testToken), HTTPClient: apiclient.NewHTTPClient(rt)})
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.CreatePageWithResponse(context.Background(), "my-space", apigen.NewPage{TopicNumber: 1, Title: "ページ", Body: "本文"})
			if !errors.Is(err, apiclient.ErrInsecureRedirect) {
				t.Fatalf("リダイレクトの拒否になりません: %v", err)
			}
			var cliErr *output.Error
			if !errors.As(ConvertError(loc, err), &cliErr) {
				t.Fatalf("CLIのエラーになりません: %v", err)
			}
			if sent != 1 || cliErr.ExitCode != output.ExitError || cliErr.Code != output.CodeNetwork {
				t.Errorf("送信後の拒否の扱いが違います: 送信回数=%d エラー=%+v", sent, cliErr)
			}
			// 翻訳した文言だけを出し、SDKの日本語の説明を埋め込まない。
			if cliErr.Message != loc.T("error_insecure_redirect") {
				t.Errorf("リダイレクトの拒否のメッセージが違います: %q", cliErr.Message)
			}
			if cliErr.Hint != loc.T("error_network_hint") || strings.Contains(cliErr.Message+cliErr.Hint, testToken) {
				t.Errorf("再送前の確認のヒントが違います: %+v", cliErr)
			}
		})
	}
}

type interruptedBody struct{ closed bool }

func (*interruptedBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (b *interruptedBody) Close() error           { b.closed = true; return nil }

func TestConvertError_ResponseAfterWrite(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
		for _, broken := range []bool{false, true} {
			name := lang + "/不正JSON"
			if broken {
				name = lang + "/本文の途中切断"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				loc := i18n.NewLocalizer(lang)
				sent := 0
				interrupted := &interruptedBody{}
				c := newTestClient(t, loc, nil, func(*http.Request) (*http.Response, error) {
					sent++
					body := io.NopCloser(strings.NewReader(`{"id":`))
					if broken {
						body = interrupted
					}
					return &http.Response{StatusCode: http.StatusCreated, Header: http.Header{"Content-Type": {"application/json"}}, Body: body}, nil
				})
				_, err := c.CreatePageWithResponse(context.Background(), "my-space", apigen.NewPage{TopicNumber: 1, Title: "ページ", Body: "本文"})
				var responseErr *apigen.ResponseError
				if !errors.As(err, &responseErr) || responseErr.StatusCode != http.StatusCreated {
					t.Fatalf("応答解析の型付きエラーになりません: %v", err)
				}
				if broken && (!errors.Is(err, io.ErrUnexpectedEOF) || !interrupted.closed) {
					t.Errorf("原因のエラーか本文のCloseが欠けています: %v 閉じた=%t", err, interrupted.closed)
				}
				var cliErr *output.Error
				if !errors.As(ConvertError(loc, err), &cliErr) {
					t.Fatalf("CLIのエラーになりません: %v", err)
				}
				if sent != 1 || cliErr.ExitCode != output.ExitError || cliErr.Code != output.CodeResponse || cliErr.Status != http.StatusCreated {
					t.Errorf("応答解析の失敗の扱いが違います: 送信回数=%d エラー=%+v", sent, cliErr)
				}
				if cliErr.Message != loc.T("error_response") || cliErr.Hint != loc.T("error_response_hint") {
					t.Errorf("翻訳された確認の案内がありません: %+v", cliErr)
				}
				var buf bytes.Buffer
				if err := output.WriteError(&buf, cliErr, true); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(buf.String(), `"hint"`) || strings.Contains(buf.String(), testToken) {
					t.Errorf("エラーのJSONが違います: %s", buf.String())
				}
			})
		}
	}
}

func TestConvertError_Download(t *testing.T) {
	t.Parallel()

	loc := i18n.NewLocalizer(i18n.LangJa)

	tests := []struct {
		name       string
		err        error
		wantCode   string
		wantStatus int
	}{
		{name: "署名付きURLの失敗の応答", err: &apiclient.DownloadError{StatusCode: http.StatusForbidden}, wantCode: output.CodeDownloadFailed, wantStatus: http.StatusForbidden},
		{name: "安全でない署名付きURL", err: apiclient.ErrInsecureDownloadURL, wantCode: output.CodeResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cliErr *output.Error
			if !errors.As(ConvertError(loc, tt.err), &cliErr) {
				t.Fatalf("CLIのエラーになりません: %v", tt.err)
			}
			if cliErr.Code != tt.wantCode || cliErr.ExitCode != output.ExitError || cliErr.Status != tt.wantStatus {
				t.Errorf("分類が違います: %+v", cliErr)
			}
			if cliErr.Hint == "" {
				t.Errorf("ヒントがありません: %+v", cliErr)
			}
		})
	}
}

func TestConvertError_DownloadNetworkRedactsSignature(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()
			const secret = "secret-signature"
			rt := roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"https://storage.example.com/%zz?X-Amz-Signature=" + secret}}, Body: io.NopCloser(strings.NewReader(""))}, nil
			})
			client, err := apiclient.New(apiclient.Options{Host: "https://wikino.example.com", HTTPClient: apiclient.NewHTTPClient(rt)})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Download(context.Background(), "https://storage.example.com/key?X-Amz-Signature="+secret)
			if err == nil {
				t.Fatal("不正なLocationのエラーを期待しました")
			}
			converted := ConvertError(i18n.NewLocalizer(lang), err)
			var cliErr *output.Error
			if !errors.As(converted, &cliErr) || cliErr.Code != output.CodeNetwork || cliErr.ExitCode != output.ExitError {
				t.Fatalf("通信の失敗の分類が違います: %v", converted)
			}
			if strings.Contains(converted.Error(), secret) {
				t.Fatalf("変換後のエラーに署名が含まれます: %v", converted)
			}
			for _, asJSON := range []bool{true, false} {
				var buf bytes.Buffer
				if err := output.WriteError(&buf, converted, asJSON); err != nil {
					t.Fatal(err)
				}
				if strings.Contains(buf.String(), secret) {
					t.Fatalf("表示に署名が含まれます: %s", buf.String())
				}
				if !strings.Contains(buf.String(), i18n.NewLocalizer(lang).T("error_download_network")) {
					t.Fatalf("安全な翻訳が表示されません: %s", buf.String())
				}
			}
		})
	}
}

func TestConvertError_DownloadInsecureRedirect(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{i18n.LangJa, i18n.LangEn} {
		t.Run(lang, func(t *testing.T) {
			t.Parallel()
			const secret = "secret-signature"
			rt := roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"http://storage.example.com/key?X-Amz-Signature=" + secret}}, Body: io.NopCloser(strings.NewReader(""))}, nil
			})
			client, err := apiclient.New(apiclient.Options{Host: "https://wikino.example.com", HTTPClient: apiclient.NewHTTPClient(rt)})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Download(context.Background(), "https://storage.example.com/key?X-Amz-Signature="+secret)
			if !errors.Is(err, apiclient.ErrInsecureRedirect) {
				t.Fatalf("リダイレクトの拒否を期待しました: %v", err)
			}

			loc := i18n.NewLocalizer(lang)
			var cliErr *output.Error
			if !errors.As(ConvertError(loc, err), &cliErr) || cliErr.Code != output.CodeNetwork || cliErr.ExitCode != output.ExitError {
				t.Fatalf("通信の失敗の分類が違います: %+v", cliErr)
			}
			// APIへの要求のリダイレクトの拒否と違い、書き込みの反映の確認ではなく接続先の確認を案内する。
			if cliErr.Message != loc.T("error_insecure_download_redirect") || cliErr.Hint != loc.T("error_insecure_download_url_hint") {
				t.Errorf("署名付きURL向けの案内になりません: %+v", cliErr)
			}
			if strings.Contains(cliErr.Message+cliErr.Hint, secret) {
				t.Errorf("表示に署名が含まれます: %+v", cliErr)
			}
		})
	}
}
