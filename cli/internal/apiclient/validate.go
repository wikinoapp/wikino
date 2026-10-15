package apiclient

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// InputErrorReasonは、入力の値を受け付けない理由。呼び出し側が理由ごとに文言を選ぶために使う。
type InputErrorReason string

const (
	// ReasonEmptyは値が空であること。
	ReasonEmpty InputErrorReason = "empty"
	// ReasonControlCharacterは値に制御文字 (改行・タブなど) が含まれること。
	ReasonControlCharacter InputErrorReason = "control_character"
	// ReasonDotSegmentは値に `..` が含まれ、パスの階層を移りうること。
	ReasonDotSegment InputErrorReason = "dot_segment"
	// ReasonReservedCharacterは値にURLで意味を持つ文字 (`/`・`\`・`?`・`#`・`%`) が含まれること。
	// URLやクエリを付けた値、エンコード済みの値を渡した場合に当たる。
	ReasonReservedCharacter InputErrorReason = "reserved_character"
	// ReasonInvalidFormatは値の形式が誤っていること。
	ReasonInvalidFormat InputErrorReason = "invalid_format"
	// ReasonInsecureSchemeは接続先がHTTPSでもループバックのHTTPでもないこと。
	ReasonInsecureScheme InputErrorReason = "insecure_scheme"
	// ReasonNotAPIPathは値がAPIの基底 (/api/v1) からのパスでないこと。
	// URL (スキームやホストを含む値)・`/api/v1` から始まるパス・`#` や `\` を含む値が当たる。
	ReasonNotAPIPath InputErrorReason = "not_api_path"
)

// InputErrorは、要求を組み立てる前に入力の値を受け付けなかったことを表す。要求は送っていない。
type InputError struct {
	// Nameは値の名前。呼び出し側が渡したもの (フラグ名など) をそのまま持つ。
	Name string
	// Valueは受け付けなかった値。
	Value string
	// Reasonは受け付けなかった理由。
	Reason InputErrorReason
}

func (e *InputError) Error() string {
	return fmt.Sprintf("%sの値 %q が不正です: %s", e.Name, e.Value, e.Reason)
}

// spaceIdentifierRegexpはスペースの識別子の文字種 (サーバーのスペースの作成の検証と同じ)。
var spaceIdentifierRegexp = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// uuidRegexpはハイフン区切りのUUID。uuid.Parseは波括弧や `urn:uuid:` の形も受け付けるため、先に形を絞る。
var uuidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateHostは、接続先のURLを検証し、スキームとホストだけにしたURLを返す。
// トークンを平文で送らないよう、HTTPSか、開発環境のためのループバックのHTTPだけを受け付ける。
// nameはエラーに載せる値の名前。
func ValidateHost(name, raw string) (*url.URL, error) {
	invalid := func(reason InputErrorReason) error {
		return &InputError{Name: name, Value: raw, Reason: reason}
	}

	if raw == "" {
		return nil, invalid(ReasonEmpty)
	}
	if containsControl(raw) {
		return nil, invalid(ReasonControlCharacter)
	}

	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Opaque != "" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, invalid(ReasonInvalidFormat)
	}
	if !isAllowedURLScheme(u) {
		return nil, invalid(ReasonInsecureScheme)
	}

	return &url.URL{Scheme: strings.ToLower(u.Scheme), Host: u.Host}, nil
}

// isAllowedURLSchemeは、uがHTTPSか、ループバックのホストへのHTTPかを返す。
func isAllowedURLScheme(u *url.URL) bool {
	switch strings.ToLower(u.Scheme) {
	case "https":
		return true
	case "http":
		return isLoopbackHost(u.Hostname())
	default:
		return false
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

// ValidateSpaceIdentifierは、パスに埋め込むスペースの識別子を検証する。
func ValidateSpaceIdentifier(name, value string) error {
	if err := validatePathValue(name, value); err != nil {
		return err
	}
	if !spaceIdentifierRegexp.MatchString(value) {
		return &InputError{Name: name, Value: value, Reason: ReasonInvalidFormat}
	}

	return nil
}

// ParseNumberは、パスに埋め込むページ番号・トピック番号 (1以上の整数) を読む。
// 符号・空白・クエリの付いた値 (`12?x=1` など) は受け付けない。
func ParseNumber(name, value string) (int32, error) {
	if err := validatePathValue(name, value); err != nil {
		return 0, err
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, &InputError{Name: name, Value: value, Reason: ReasonInvalidFormat}
		}
	}

	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil || n < 1 {
		return 0, &InputError{Name: name, Value: value, Reason: ReasonInvalidFormat}
	}

	return int32(n), nil
}

// ParseUUIDは、パスに埋め込むID (リビジョンのID・添付ファイルのID) を、ハイフン区切りのUUIDとして読む。
func ParseUUID(name, value string) (uuid.UUID, error) {
	if err := validatePathValue(name, value); err != nil {
		return uuid.UUID{}, err
	}
	if !uuidRegexp.MatchString(value) {
		return uuid.UUID{}, &InputError{Name: name, Value: value, Reason: ReasonInvalidFormat}
	}

	return uuid.Parse(value)
}

// apiBasePrefixは、APIの基底のパスを重ねて指定したことを見分けるための接頭辞。
var apiBasePrefix = basePath + "/"

// ParseAPIPathは、APIの基底 (/api/v1) からのパス (クエリを含められる) を検証し、
// パスとクエリだけを持つ相対のURLにして返す。先頭の "/" は省略できる。
// 基底の外 (別のホストや基底より上の階層) を指す値は受け付けない。
// クエリは、エンコードしていない値 (日本語など) も送れるようエンコードし直す。
func ParseAPIPath(name, raw string) (*url.URL, error) {
	invalid := func(reason InputErrorReason) error {
		return &InputError{Name: name, Value: raw, Reason: reason}
	}

	switch {
	case raw == "":
		return nil, invalid(ReasonEmpty)
	case containsControl(raw):
		return nil, invalid(ReasonControlCharacter)
	case strings.ContainsAny(raw, `#\`):
		return nil, invalid(ReasonNotAPIPath)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, invalid(ReasonNotAPIPath)
	}
	if u.Scheme != "" || u.Host != "" || u.Opaque != "" || u.User != nil || strings.HasPrefix(raw, "//") {
		return nil, invalid(ReasonNotAPIPath)
	}

	escaped := u.EscapedPath()
	if !strings.HasPrefix(escaped, "/") {
		escaped = "/" + escaped
	}
	if escaped == basePath || strings.HasPrefix(escaped, apiBasePrefix) {
		return nil, invalid(ReasonNotAPIPath)
	}

	// `%2e%2e` のようにエンコードした `..` も、送り先で解釈されると階層を移りうるため拒否する。
	for _, segment := range strings.Split(escaped, "/") {
		unescaped, err := url.PathUnescape(segment)
		if err != nil {
			return nil, invalid(ReasonNotAPIPath)
		}
		if unescaped == "." || unescaped == ".." {
			return nil, invalid(ReasonDotSegment)
		}
	}

	ref, err := url.Parse(escaped)
	if err != nil {
		return nil, invalid(ReasonNotAPIPath)
	}
	if u.RawQuery != "" {
		query, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, invalid(ReasonNotAPIPath)
		}
		ref.RawQuery = query.Encode()
	}

	return ref, nil
}

// validatePathValueは、パスの1つのセグメントに埋め込む値に共通の検査を行う。
// エージェントが生成しがちな誤り (改行の混入・パストラバーサル・URLやクエリの付いた値・
// エンコード済みの値) を、形式の誤りより具体的な理由で返す。
func validatePathValue(name, value string) error {
	invalid := func(reason InputErrorReason) error {
		return &InputError{Name: name, Value: value, Reason: reason}
	}

	switch {
	case value == "":
		return invalid(ReasonEmpty)
	case containsControl(value):
		return invalid(ReasonControlCharacter)
	case strings.Contains(value, ".."):
		return invalid(ReasonDotSegment)
	case strings.ContainsAny(value, `/\?#%`):
		return invalid(ReasonReservedCharacter)
	}

	return nil
}

func containsControl(s string) bool {
	return strings.ContainsFunc(s, unicode.IsControl)
}
