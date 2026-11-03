package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/session"
	errpages "github.com/wikinoapp/wikino/go/internal/templates/pages/errors"
)

// DefaultMaxBodyBytesはリクエストボディのデフォルトサイズ上限 (4 MiB)。
// 上限の文字数 (model.PageBodyMaxLength) のページ本文が、どの送り方でもこの上限で弾かれず、
// 必ずアプリのバリデーションに届く大きさにする。
// multipartではUTF-8のまま送るため1字が最大4バイト、JSONでは制御文字や `<`・`>`・`&` が
// `\u00XX` の6バイトになりうるため、50万字で最大3MBになる。
// そのうえで、意図的な大容量送信によるメモリ枯渇を防ぐ。
const DefaultMaxBodyBytes = 4 << 20

// BodyLimitはリクエストボディのサイズを [DefaultMaxBodyBytes] に制限するミドルウェア。
// 上限を超過したリクエストには413 Payload Too Largeを返す。
//
// r.ParseForm / r.FormValue / r.PostFormValueを呼ぶハンドラー・ミドルウェアより前に
// 配置する必要がある(これらは呼び出しと同時にボディを読み込むため、事前にラップしておく)。
//
// net/http標準の [http.MaxBytesHandler] はr.Bodyをラップするだけで自動413応答は
// 行わないため、ここではミドルウェア側でボディを先読みしてエラーを検出し、413を明示的に返す。
// 下流ハンドラーには先読み済みのボディを [io.NopCloser] でラップして渡すため、既存の
// r.FormValue / r.ParseFormの挙動は維持される。
func BodyLimit(next http.Handler) http.Handler {
	return NewBodyLimit(nil)(next)
}

// NewBodyLimitはAPIのエラーをProblem Detailsで返す本文サイズ制限ミドルウェアを作成する。
// problemsがnilの場合は従来のHTML画面向けの応答を返す
func NewBodyLimit(problems *apierror.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ボディがないリクエスト(GET/HEAD/OPTIONS等)はそのまま通す
			if r.Body == nil || r.Body == http.NoBody {
				next.ServeHTTP(w, r)
				return
			}

			// Content-Lengthが上限を超えていることが事前に分かる場合は、ボディを読まずに413を返す。
			// chunked転送等でContentLengthが不明な場合は -1になるため、ここでは拒否せず
			// 後段のMaxBytesReaderに判定を委ねる。
			if r.ContentLength > DefaultMaxBodyBytes {
				slog.WarnContext(r.Context(), "リクエストボディサイズ上限超過(Content-Length)",
					"path", r.URL.Path,
					"method", r.Method,
					"content_length", r.ContentLength,
					"max", int64(DefaultMaxBodyBytes),
				)
				writeBodyLimitError(w, r, problems, http.StatusRequestEntityTooLarge)
				return
			}

			limited := http.MaxBytesReader(w, r.Body, DefaultMaxBodyBytes)
			buf, err := io.ReadAll(limited)
			if err != nil {
				var maxBytesErr *http.MaxBytesError
				if errors.As(err, &maxBytesErr) {
					slog.WarnContext(r.Context(), "リクエストボディサイズ上限超過(読み込み中)",
						"path", r.URL.Path,
						"method", r.Method,
						"limit", maxBytesErr.Limit,
					)
					writeBodyLimitError(w, r, problems, http.StatusRequestEntityTooLarge)
					return
				}
				slog.ErrorContext(r.Context(), "リクエストボディの読み込み失敗",
					"error", err,
					"path", r.URL.Path,
					"method", r.Method,
				)
				writeBodyLimitError(w, r, problems, http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(buf))
			r.ContentLength = int64(len(buf))
			next.ServeHTTP(w, r)
		})
	}
}

func writeBodyLimitError(w http.ResponseWriter, r *http.Request, problems *apierror.Writer, status int) {
	// このミドルウェアは本文を読む前に置くため、言語を決めるミドルウェアより前で動く。
	// 応答の文言を利用者の言語にするため、ここで言語を決める
	r = r.WithContext(i18n.SetLocale(r.Context(), i18n.DetectLanguage(r)))

	if problems != nil && session.IsAPIPath(r.URL.Path) {
		if status == http.StatusRequestEntityTooLarge {
			problems.ContentTooLarge(w, r)
		} else {
			problems.BadRequest(w, r, i18n.T(r.Context(), "api_error_request_body_unreadable"))
		}
		return
	}
	if status == http.StatusRequestEntityTooLarge {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		// テンプレートのレンダリングエラーはレスポンス書き込み後なので無視
		_ = errpages.ContentTooLargePage(fmt.Sprintf("%dMB", DefaultMaxBodyBytes>>20)).Render(r.Context(), w)
	} else {
		http.Error(w, "Bad Request", status)
	}
}
