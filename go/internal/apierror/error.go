// Package apierrorは公開Web APIのエラーレスポンスを、RFC 9457のProblem Detailsで書き込む。
// HTML画面のエラーを描画する `internal/httperror` のAPI版にあたる。
package apierror

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// ContentTypeはProblem DetailsのContent-Type。APIの規約に揃えてcharsetを付けない
const ContentType = "application/problem+json"

// 問題タイプのうち、ステータスコード以上の情報を持つもののパス (Writerの基準URLからの相対)。
// 公開した値はクライアントが分岐に使う契約になるため、変えない
const (
	// ProblemTypeValidationFailedはリクエストの内容がスキーマや業務ルールに反することを表す
	ProblemTypeValidationFailed = "validation-failed"
)

// problemTypeAboutBlankはステータスコード以上の情報を持たない問題タイプ (RFC 9457 §4.2.1)
const problemTypeAboutBlank = "about:blank"

// Problemはレスポンス本文のProblem Details
type Problem struct {
	Type   string  `json:"type"`
	Title  string  `json:"title"`
	Status int     `json:"status"`
	Detail *string `json:"detail"`
	// Errorsはフィールド単位の問題。`validation-failed` のときだけ含める (空でも `[]` として含める)
	Errors *[]FieldError `json:"errors,omitempty"`
}

// FieldErrorはバリデーションで見つかった1つの問題
type FieldError struct {
	Detail string `json:"detail"`
	// Pointerはリクエスト本文の問題の箇所を指すJSON Pointer (URIフラグメント形式。例: `#/title`)
	Pointer *string `json:"pointer"`
	// Parameterは問題のあったパラメーター (パス・クエリ・ヘッダー) の名前
	Parameter *string `json:"parameter"`
}

// ParameterErrorは、パラメーター (パス・クエリ・ヘッダー) の内容をハンドラーが受け付けられないことを表すエラー。
// OpenAPI記述のスキーマでは表せない形式 (カーソルなど) の検証に使い、WriteErrorが422に変換する
type ParameterError struct {
	// Parameterは問題のあったパラメーターの名前
	Parameter string
	// Detailは問題の説明
	Detail string
}

func (e *ParameterError) Error() string {
	return "パラメーター " + e.Parameter + " を受け付けられません: " + e.Detail
}

// ErrPreconditionRequiredは、`If-Match` の要る更新のリクエストに `If-Match` が無いことを表すエラー。
// HandlerがUseCaseを呼ぶ前に返し、WriteErrorが428に変換する。
// OpenAPI記述で `If-Match` を必須にすると、欠落がリクエスト検証の422になるため、記述では任意にしてここで扱う
var ErrPreconditionRequired = errors.New("If-Matchヘッダーがありません")

// BearerChallengeは401・403で返す `WWW-Authenticate: Bearer` の内容 (RFC 6750 §3)
type BearerChallenge struct {
	// Errorはエラーコード (`invalid_token`・`insufficient_scope` など)。
	// 認証情報の無いリクエストには付けない (RFC 6750 §3.1)
	Error string
	// Scopeはリクエストに要るスコープ。`insufficient_scope` のときに付ける
	Scope []string
	// ResourceMetadataは保護リソースのメタデータのURL (RFC 9728 §5.1)。
	// WriterのUnauthorized・Forbiddenがリクエストのパスから決めるため、呼び出し側は指定しない
	ResourceMetadata string
}

// Stringは `WWW-Authenticate` ヘッダーの値を返す
func (c BearerChallenge) String() string {
	var params []string
	if c.Error != "" {
		params = append(params, `error="`+c.Error+`"`)
	}
	if len(c.Scope) > 0 {
		params = append(params, `scope="`+strings.Join(c.Scope, " ")+`"`)
	}
	if c.ResourceMetadata != "" {
		params = append(params, `resource_metadata="`+c.ResourceMetadata+`"`)
	}
	if len(params) == 0 {
		return "Bearer"
	}
	return "Bearer " + strings.Join(params, ", ")
}

// WriterはProblem Detailsのレスポンスを書き込む
type Writer struct {
	appURL             string
	problemTypeBaseURL string
}

// NewWriterは新しいWriterを作成する。
// appURLは問題タイプのURI (`{appURL}/api/problems/{名前}`) と、保護リソースのメタデータのURLの基準に使う
func NewWriter(appURL string) *Writer {
	appURL = strings.TrimSuffix(appURL, "/")
	return &Writer{
		appURL:             appURL,
		problemTypeBaseURL: appURL + "/api/problems/",
	}
}

// BadRequestは400を書き込む。リクエスト本文がJSONとして読めないなど、形式が壊れているときに使う。
// 形式は正しいが内容がスキーマや業務ルールに反する場合はValidationFailedを使う
func (pw *Writer) BadRequest(w http.ResponseWriter, r *http.Request, detail string) {
	pw.write(w, r, Problem{
		Type:   problemTypeAboutBlank,
		Title:  http.StatusText(http.StatusBadRequest),
		Status: http.StatusBadRequest,
		Detail: &detail,
	})
}

// Unauthorizedは401を書き込む。トークンが無い・無効・期限切れのときに使う
func (pw *Writer) Unauthorized(w http.ResponseWriter, r *http.Request, challenge BearerChallenge) {
	challenge.ResourceMetadata = pw.resourceMetadataURL(r)
	w.Header().Set("WWW-Authenticate", challenge.String())
	pw.writeStatus(w, r, http.StatusUnauthorized)
}

// Forbiddenは403を書き込む。トークンのスコープが足りないときに使う。
// メンバー権限が足りない場合は、リソースの存在を秘匿するためNotFoundを使う
func (pw *Writer) Forbidden(w http.ResponseWriter, r *http.Request, challenge BearerChallenge) {
	challenge.ResourceMetadata = pw.resourceMetadataURL(r)
	w.Header().Set("WWW-Authenticate", challenge.String())
	pw.writeStatus(w, r, http.StatusForbidden)
}

// resourceMetadataURLは、401・403の `WWW-Authenticate` の `resource_metadata` に載せる、
// リクエストのパスのスペースの保護リソースのメタデータのURLを返す。
// トークンはスペースに束縛され、保護リソース (RFC 8707の `resource`) はスペースのAPIのため、
// スペースに属さないパス (`/api/v1/openapi.yaml` など) では指す先が無く、空を返して付けない
func (pw *Writer) resourceMetadataURL(r *http.Request) string {
	identifier, ok := model.SpaceIdentifierFromAPIPath(r.URL.EscapedPath())
	if !ok {
		return ""
	}
	return model.SpaceAPIResourceMetadataURL(pw.appURL, identifier)
}

// NotFoundは404を書き込む
func (pw *Writer) NotFound(w http.ResponseWriter, r *http.Request) {
	pw.writeStatus(w, r, http.StatusNotFound)
}

// MethodNotAllowedは405を書き込む。allowにはそのパスで使えるメソッドを渡す
func (pw *Writer) MethodNotAllowed(w http.ResponseWriter, r *http.Request, allow []string) {
	w.Header().Set("Allow", strings.Join(allow, ", "))
	pw.writeStatus(w, r, http.StatusMethodNotAllowed)
}

// Conflictは409を書き込む
func (pw *Writer) Conflict(w http.ResponseWriter, r *http.Request, detail string) {
	pw.write(w, r, Problem{
		Type:   problemTypeAboutBlank,
		Title:  http.StatusText(http.StatusConflict),
		Status: http.StatusConflict,
		Detail: &detail,
	})
}

// PreconditionFailedは412を書き込む。`If-Match` が現在のETagと一致しないときに使う。
// detailには、クライアントが次に取れる対処 (取得し直してから更新するなど) を書く
func (pw *Writer) PreconditionFailed(w http.ResponseWriter, r *http.Request, detail string) {
	pw.write(w, r, Problem{
		Type:   problemTypeAboutBlank,
		Title:  http.StatusText(http.StatusPreconditionFailed),
		Status: http.StatusPreconditionFailed,
		Detail: &detail,
	})
}

// ContentTooLargeは本文のサイズ上限を超えたときに413を書き込む
func (pw *Writer) ContentTooLarge(w http.ResponseWriter, r *http.Request) {
	pw.writeStatus(w, r, http.StatusRequestEntityTooLarge)
}

// UnsupportedMediaTypeは415を書き込む。acceptedにはリクエスト本文として受け付けるメディアタイプを渡す。
// PATCHには `Accept-Patch` (RFC 5789 §3.1)、それ以外には `Accept-Post` で受け付ける形式を示す
func (pw *Writer) UnsupportedMediaType(w http.ResponseWriter, r *http.Request, accepted []string) {
	header := "Accept-Post"
	if r.Method == http.MethodPatch {
		header = "Accept-Patch"
	}
	w.Header().Set(header, strings.Join(accepted, ", "))
	pw.writeStatus(w, r, http.StatusUnsupportedMediaType)
}

// ValidationFailedは422を書き込む。リクエストの内容がスキーマや業務ルールに反するときに使う
func (pw *Writer) ValidationFailed(w http.ResponseWriter, r *http.Request, errs []FieldError) {
	if errs == nil {
		errs = []FieldError{}
	}
	pw.write(w, r, Problem{
		Type:   pw.problemTypeBaseURL + ProblemTypeValidationFailed,
		Title:  "Validation Failed",
		Status: http.StatusUnprocessableEntity,
		Errors: &errs,
	})
}

// PreconditionRequiredは428を書き込む。`If-Match` の要るリクエストに無いときに使う
func (pw *Writer) PreconditionRequired(w http.ResponseWriter, r *http.Request) {
	pw.writeStatus(w, r, http.StatusPreconditionRequired)
}

// TooManyRequestsは429を書き込む。retryAfterには次に受け付けるまでの時間を渡す
func (pw *Writer) TooManyRequests(w http.ResponseWriter, r *http.Request, retryAfter time.Duration) {
	w.Header().Set("Retry-After", retryAfterSeconds(retryAfter))
	pw.writeStatus(w, r, http.StatusTooManyRequests)
}

// InternalServerErrorは500を書き込む
func (pw *Writer) InternalServerError(w http.ResponseWriter, r *http.Request) {
	pw.writeStatus(w, r, http.StatusInternalServerError)
}

// ServiceUnavailableは503を書き込む。retryAfterには再開の見込みまでの時間を渡す
func (pw *Writer) ServiceUnavailable(w http.ResponseWriter, r *http.Request, retryAfter time.Duration) {
	w.Header().Set("Retry-After", retryAfterSeconds(retryAfter))
	pw.writeStatus(w, r, http.StatusServiceUnavailable)
}

// WriteErrorはUseCaseが返したエラーをProblem Detailsに変換して書き込む。
//   - *ParameterError・*model.ValidationError: 422
//   - ErrPreconditionRequired: 428
//   - *model.AppError: 未存在と権限不足は404 (リソースの存在を秘匿する)、競合は409、前提の不一致は412 (どちらもUserMsgをdetailにする)、それ以外は500
//   - それ以外のエラー: 500
func (pw *Writer) WriteError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	var pe *ParameterError
	if errors.As(err, &pe) {
		parameter := pe.Parameter
		pw.ValidationFailed(w, r, []FieldError{{Detail: pe.Detail, Parameter: &parameter}})
		return
	}

	if ve := model.AsValidationError(err); ve != nil {
		pw.ValidationFailed(w, r, validationFieldErrors(ve))
		return
	}

	if errors.Is(err, ErrPreconditionRequired) {
		pw.PreconditionRequired(w, r)
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
			pw.NotFound(w, r)
		case model.AppErrCodeConflict:
			pw.Conflict(w, r, ae.UserMsg)
		case model.AppErrCodePreconditionFailed:
			pw.PreconditionFailed(w, r, ae.UserMsg)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			pw.InternalServerError(w, r)
		}
		return
	}

	slog.ErrorContext(ctx, "APIのリクエストの処理中に予期しないエラーが発生しました", "error", err)
	pw.InternalServerError(w, r)
}

// writeStatusはステータスコード以上の情報を持たない問題 (`about:blank`) を書き込む
func (pw *Writer) writeStatus(w http.ResponseWriter, r *http.Request, status int) {
	pw.write(w, r, Problem{
		Type:   problemTypeAboutBlank,
		Title:  http.StatusText(status),
		Status: status,
	})
}

func (pw *Writer) write(w http.ResponseWriter, r *http.Request, problem Problem) {
	body, err := json.Marshal(problem)
	if err != nil {
		slog.ErrorContext(r.Context(), "Problem Detailsの変換に失敗しました", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(problem.Status)
	// ヘッダー送信後の書き込みエラーはクライアントの切断などで、返せる応答が無いため無視する
	_, _ = w.Write(body)
}

// validationFieldErrorsは *model.ValidationErrorを、フィールド名の順に並べたFieldErrorへ変換する。
// フィールド名はリクエスト本文のキーとしてJSON Pointerにする
func validationFieldErrors(ve *model.ValidationError) []FieldError {
	errs := make([]FieldError, 0, len(ve.Global)+len(ve.Fields))
	for _, message := range ve.Global {
		errs = append(errs, FieldError{Detail: message})
	}

	fields := make([]string, 0, len(ve.Fields))
	for field := range ve.Fields {
		fields = append(fields, field)
	}
	slices.Sort(fields)
	for _, field := range fields {
		pointer := JSONPointer(field)
		for _, message := range ve.Fields[field] {
			errs = append(errs, FieldError{Detail: message, Pointer: &pointer})
		}
	}
	return errs
}

// JSONPointerはリクエスト本文の中の位置を、URIフラグメント形式のJSON Pointer (RFC 6901 §6) で表す
func JSONPointer(tokens ...string) string {
	var b strings.Builder
	b.WriteString("#")
	for _, token := range tokens {
		b.WriteString("/")
		token = strings.ReplaceAll(token, "~", "~0")
		token = strings.ReplaceAll(token, "/", "~1")
		b.WriteString(token)
	}
	return b.String()
}

// retryAfterSecondsは `Retry-After` の秒数 (RFC 9110 §10.2.3のdelay-seconds) を返す。
// 1秒未満は切り上げ、0秒で即時の再試行を促さないようにする
func retryAfterSeconds(d time.Duration) string {
	seconds := int64((d + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return strconv.FormatInt(seconds, 10)
}
