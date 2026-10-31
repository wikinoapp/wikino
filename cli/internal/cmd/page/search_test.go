package page

import (
	"net/http"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/cmdtest"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestSearch_Query(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"1つのキーワード", []string{"議事録"}, "議事録"},
		{"引用符でまとめたキーワード", []string{"議事録 予算"}, "議事録 予算"},
		{"並べたキーワード", []string{"議事録", "予算"}, "議事録 予算"},
		{"全角の空白と連続する空白は1つの空白に詰める", []string{"  議事録　　予算 ", "定例"}, "議事録 予算 定例"},
		{"記号はそのまま送る", []string{"100%", "a_b"}, "100% a_b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
				return listResponse([]map[string]any{}, nil)
			})

			if err := e.Run(NewCmdPage, append([]string{"page", "search"}, tt.args...)...); err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			requests := e.Requests()
			if len(requests) != 1 {
				t.Fatalf("要求の数 = %d、期待値 = 1", len(requests))
			}
			if got := requests[0].URL.Query().Get("q"); got != tt.want {
				t.Errorf("q = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestSearch_Params(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{}, nil)
	})

	err := e.Run(NewCmdPage, "page", "search", "議事録", "--topic", "3", "--status", "all", "--limit", "50", "--cursor", "abc")
	if err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	requests := e.Requests()
	if len(requests) != 1 {
		t.Fatalf("要求の数 = %d、期待値 = 1", len(requests))
	}
	query := requests[0].URL.Query()
	want := map[string]string{
		"q":            "議事録",
		"topic_number": "3",
		"status":       "all",
		"limit":        "50",
		"cursor":       "abc",
	}
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Errorf("クエリの %s = %q、期待値 = %q", key, got, value)
		}
	}
	for _, key := range []string{"title", "modified_since"} {
		if query.Has(key) {
			t.Errorf("クエリに %s がある: %v", key, query)
		}
	}
}

func TestSearch_JSON(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{testListPage(12, ptr("議事録"), "# 議事録\n\n予算の確認\n", "# 議事録 予算の確認")}, ptr("cursor-2"))
	})

	if err := e.Run(NewCmdPage, "page", "search", "予算"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	var got listOutput
	cmdtest.Decode(t, e.Stdout.String(), &got)
	if len(got.Items) != 1 {
		t.Fatalf("件数 = %d、期待値 = 1", len(got.Items))
	}
	item := got.Items[0]
	if _, ok := item["body"]; ok {
		t.Errorf("一覧の既定の出力に本文が含まれる: %v", item)
	}
	if item["excerpt"] != "# 議事録 予算の確認" || item["url"] != "https://wikino.example.com/s/team/pages/12" {
		t.Errorf("1件目 = %v", item)
	}
	if got.NextCursor == nil || *got.NextCursor != "cursor-2" {
		t.Errorf("next_cursor = %v、期待値 = cursor-2", got.NextCursor)
	}
	if !strings.Contains(e.Stderr.String(), "--cursor cursor-2") {
		t.Errorf("標準エラーに次のページの取り方が無い: %q", e.Stderr.String())
	}
	// 本文を出さないため、APIに本文を求めない。
	e.AssertQuery(t, "include")
}

func TestSearch_Human(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return listResponse([]map[string]any{testListPage(12, ptr("議事録"), longBody, longExcerpt), testListPage(7, ptr("予算案"), "", "")}, nil)
	})

	if err := e.RunTTY(NewCmdPage, "page", "search", "議事録"); err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}

	cmdtest.AssertGolden(t, "search", e.Stdout.String())
	if e.Stderr.Len() > 0 {
		t.Errorf("標準エラーに出力がある: %q", e.Stderr.String())
	}
}

func TestSearch_Empty(t *testing.T) {
	t.Parallel()

	t.Run("JSONでは空の一覧を出し、ヒントを出さない", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
			return listResponse([]map[string]any{}, nil)
		})

		if err := e.Run(NewCmdPage, "page", "search", "存在しない語"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if got, want := e.Stdout.String(), `{"items":[],"next_cursor":null}`+"\n"; got != want {
			t.Errorf("標準出力 = %q、期待値 = %q", got, want)
		}
		if e.Stderr.Len() > 0 {
			t.Errorf("標準エラーに出力がある: %q", e.Stderr.String())
		}
	})

	t.Run("人間向けの表示では標準エラーに条件を緩めるヒントを出す", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
			return listResponse([]map[string]any{}, nil)
		})

		if err := e.RunTTY(NewCmdPage, "page", "search", "存在しない語"); err != nil {
			t.Fatalf("エラーを期待しなかったが %v だった", err)
		}
		if e.Stdout.Len() > 0 {
			t.Errorf("標準出力に出力がある: %q", e.Stdout.String())
		}
		want := "該当するものはありません。\nキーワードを減らすか、`--topic` を外すか、`--status all` を付けて検索し直してください。\n"
		if got := e.Stderr.String(); got != want {
			t.Errorf("標準エラー = %q、期待値 = %q", got, want)
		}
	})
}

func TestSearch_InvalidArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"キーワードが無い", nil},
		{"空白だけのキーワード", []string{" 　 "}},
		{"トピック番号が0", []string{"議事録", "--topic", "0"}},
		{"未知の公開状態", []string{"議事録", "--status", "draft"}},
		{"件数が0", []string{"議事録", "--limit", "0"}},
		{"全件とカーソルの併用", []string{"議事録", "--all", "--cursor", "abc"}},
		{"未知の項目", []string{"議事録", "--fields", "number,content"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := newEnv(t)
			err := e.Run(NewCmdPage, append([]string{"page", "search"}, tt.args...)...)
			_ = cmdtest.AssertCLIError(t, err, output.CodeUsage, output.ExitUsage)
			assertNoRequest(t, e)
		})
	}
}

func TestSearch_TooManyKeywords(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	e.Handle(http.MethodGet, pagesPath, func(req *http.Request) *http.Response {
		return cmdtest.Problem(http.StatusUnprocessableEntity, "キーワードは10個まで指定できます。")
	})

	err := e.Run(NewCmdPage, "page", "search", "1 2 3 4 5 6 7 8 9 10 11")
	cliErr := cmdtest.AssertCLIError(t, err, output.CodeValidationFailed, output.ExitValidation)
	if cliErr.Status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d、期待値 = 422", cliErr.Status)
	}
}
