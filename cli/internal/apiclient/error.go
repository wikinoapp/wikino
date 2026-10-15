package apiclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// maxErrorBodyBytesは、失敗の応答の本文を読む上限。Problem Detailsは小さく、
// 想定外の大きな本文 (プロキシのエラーページなど) を読み切らないようにする。
const maxErrorBodyBytes = 1 << 20

// Errorは、APIが失敗の応答 (4xx・5xx) を返したことを表す。
// Problem Details (RFC 9457) の項目と、対処に使うヘッダーの値を持つ。
type Error struct {
	// StatusCodeはHTTPステータスコード。
	StatusCode int
	// TypeはProblem Detailsの `type` (問題の種類を表すURI)。Problem Detailsでない応答では空。
	Type string
	// TitleはProblem Detailsの `title`。
	Title string
	// DetailはProblem Detailsの `detail`。
	Detail string
	// FieldErrorsはProblem Detailsの `errors` (入力の検証で見つかった問題)。
	FieldErrors []FieldError
	// RequiredScopesは、403の `WWW-Authenticate` の `scope` が示す、操作に要るスコープ。
	RequiredScopes []string
	// RetryAfterは、429の `Retry-After` が示す再試行までの時間。HasRetryAfterがfalseなら値は無い。
	RetryAfter    time.Duration
	HasRetryAfter bool
}

// FieldErrorは、入力の検証で見つかった1つの問題。
type FieldError struct {
	Detail    string `json:"detail"`
	Pointer   string `json:"pointer"`
	Parameter string `json:"parameter"`
}

func (e *Error) Error() string {
	text := e.Detail
	if text == "" {
		text = e.Title
	}
	if text == "" {
		text = http.StatusText(e.StatusCode)
	}

	return fmt.Sprintf("APIエラー %d: %s", e.StatusCode, text)
}

type problemDetails struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Detail *string      `json:"detail"`
	Errors []FieldError `json:"errors"`
}

// newErrorは失敗の応答をErrorにする。本文はProblem Detailsとして読めた場合だけ使う。
func newError(resp *http.Response) *Error {
	e := &Error{StatusCode: resp.StatusCode}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if strings.Contains(resp.Header.Get("Content-Type"), "json") {
		var p problemDetails
		if json.Unmarshal(body, &p) == nil {
			e.Type = p.Type
			e.Title = p.Title
			if p.Detail != nil {
				e.Detail = *p.Detail
			}
			e.FieldErrors = p.Errors
		}
	}

	if resp.StatusCode == http.StatusForbidden {
		if scope, ok := parseBearerChallenge(resp.Header.Get("WWW-Authenticate"))["scope"]; ok {
			e.RequiredScopes = strings.Fields(scope)
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		e.RetryAfter, e.HasRetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	}

	return e
}

// parseRetryAfterは、Retry-After (秒数かHTTPの日時) を再試行までの時間にする。
func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}

	if at, err := http.ParseTime(value); err == nil {
		return max(at.Sub(now), 0), true
	}

	return 0, false
}

// parseBearerChallengeは、WWW-AuthenticateのBearerのチャレンジ (RFC 6750) の
// パラメーターを返す。Bearerのチャレンジが無ければ空のmapを返す。
//
// 例: `Bearer error="insufficient_scope", scope="topic:read"`
func parseBearerChallenge(header string) map[string]string {
	params := map[string]string{}

	rest := strings.TrimSpace(header)
	for rest != "" {
		var scheme string
		scheme, rest = readToken(rest)
		rest = strings.TrimLeft(rest, " ")

		var p map[string]string
		p, rest = readParams(rest)
		if strings.EqualFold(scheme, "Bearer") {
			return p
		}
		if scheme == "" {
			// 解釈できない文字に当たったら、続きを読まずに諦める。
			return params
		}
	}

	return params
}

// readParamsは、`key=value` か `key="quoted"` をカンマ区切りで読み、次のチャレンジの手前で止まる。
func readParams(s string) (map[string]string, string) {
	params := map[string]string{}

	for {
		s = strings.TrimLeft(s, " ,")
		key, after := readToken(s)
		after = strings.TrimLeft(after, " ")
		if key == "" || !strings.HasPrefix(after, "=") {
			// `=` の続かないトークンは次のチャレンジのスキームのため、読まずに返す。
			return params, s
		}
		after = strings.TrimLeft(after[1:], " ")

		var value string
		if strings.HasPrefix(after, `"`) {
			value, after = readQuoted(after)
		} else {
			value, after = readToken(after)
		}
		params[strings.ToLower(key)] = value
		s = after
	}
}

// readTokenは、先頭からトークン (RFC 9110のtchar) の文字が続く部分を読む。
func readToken(s string) (string, string) {
	i := 0
	for i < len(s) && isTokenChar(s[i]) {
		i++
	}

	return s[:i], s[i:]
}

func isTokenChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	default:
		return strings.IndexByte("!#$%&'*+-.^_`|~/:", c) >= 0
	}
}

// readQuotedは、先頭の引用符で囲んだ文字列を読み、`\` のエスケープを戻す。
func readQuoted(s string) (string, string) {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			}
		case '"':
			return b.String(), s[i+1:]
		default:
			b.WriteByte(s[i])
		}
	}

	return b.String(), ""
}
