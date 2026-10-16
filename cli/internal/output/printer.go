// Package outputは、コマンドの結果とエラーの出し方 (人間向けの表示・JSON・JSON Lines) と、
// 出力を絞るフラグ (`--json`・`--fields`・`--jq`)、終了コードを扱う。
//
// 標準出力には結果だけを出し、進捗や警告は標準エラーに出す。
// エージェントが標準出力をそのままJSONとして読めるようにするためである。
package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/spf13/pflag"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
)

// 出力を選ぶフラグの名前。すべてのコマンドで同じ意味に使う。
const (
	FlagJSON   = "json"
	FlagFields = "fields"
	FlagJQ     = "jq"
)

// AddFlagsは、出力を選ぶフラグをfsに足す。
func AddFlags(fs *pflag.FlagSet, loc *i18n.Localizer) {
	fs.Bool(FlagJSON, false, loc.T("help_json_flag"))
	fs.StringSlice(FlagFields, nil, loc.T("help_fields_flag"))
	fs.String(FlagJQ, "", loc.T("help_jq_flag"))
}

// IsJSONRequestedは、fsで `--json` が指定されたかを返す。fsにフラグが無ければfalseを返す。
func IsJSONRequested(fs *pflag.FlagSet) bool {
	flag := fs.Lookup(FlagJSON)

	return flag != nil && flag.Value.String() == "true"
}

// Specは、コマンドが出す項目の定義。
type Spec struct {
	// Fieldsは `--fields` で選べる項目の名前。
	Fields []string
	// Defaultは `--fields` が無いときに出す項目の名前。nilならすべての項目を出す。
	// 一覧で本文 (body) を既定で出さないときなどに使う。
	Default []string
}

// Printerはコマンドの結果を出す。
type Printer struct {
	ios *iostreams.IOStreams
	loc *i18n.Localizer

	json   bool
	fields []string
	jq     *gojq.Code
}

// NewPrinterは、fsで指定された出力のフラグとspecからPrinterを作る。
//
// `--fields` の未知の項目名と `--jq` の構文の誤りは、ここで使い方の誤りとして返す。
// コマンドはAPIへ要求を送る前にNewPrinterを呼び、誤った出力の指定で書き込みだけが
// 済んでしまうことを防ぐ。
func NewPrinter(ios *iostreams.IOStreams, loc *i18n.Localizer, fs *pflag.FlagSet, spec Spec) (*Printer, error) {
	p := &Printer{
		ios:    ios,
		loc:    loc,
		json:   IsJSONRequested(fs) || !ios.IsStdoutTTY(),
		fields: spec.Default,
	}

	if fs.Changed(FlagFields) {
		fields, err := fs.GetStringSlice(FlagFields)
		if err != nil {
			return nil, err
		}
		if err := validateFields(loc, fields, spec.Fields); err != nil {
			return nil, err
		}
		p.fields = fields
		// 項目を選んだ出力は機械が読むためのものなので、TTYでもJSONにする。
		p.json = true
	}

	if fs.Changed(FlagJQ) {
		expr, err := fs.GetString(FlagJQ)
		if err != nil {
			return nil, err
		}
		code, err := compileJQ(loc, expr)
		if err != nil {
			return nil, err
		}
		p.jq = code
		p.json = true
	}

	return p, nil
}

// IsJSONは、結果をJSONで出すかを返す。falseなら、コマンドは人間向けの表示を出す。
func (p *Printer) IsJSON() bool {
	return p.json
}

// PrintItemは1件の結果をJSONで出す。`--fields` はその項目に、`--jq` はその値に適用する。
func (p *Printer) PrintItem(v any) error {
	value, err := toJSONValue(v)
	if err != nil {
		return err
	}

	return p.write(p.selectFields(value), p.ios.IsStdoutTTY())
}

// PrintListは一覧の結果 (`{"items": [...], "next_cursor": ...}`) をJSONで出す。
// `--fields` は `items` の各要素に適用し、`--jq` は一覧全体に適用する。
func (p *Printer) PrintList(v any) error {
	value, err := toJSONValue(v)
	if err != nil {
		return err
	}

	if list, ok := value.(map[string]any); ok {
		if items, ok := list["items"].([]any); ok {
			for i, item := range items {
				items[i] = p.selectFields(item)
			}
		}
	}

	return p.write(value, p.ios.IsStdoutTTY())
}

// PrintLineは、全件の取得 (`--all`) の1件をJSON Linesの1行として出す。
// `--fields` と `--jq` はその1件に適用する。全件をメモリに溜めずに流せ、
// 途中で止めても壊れた行が残らないよう、1件ごとに1行で書く。
func (p *Printer) PrintLine(v any) error {
	value, err := toJSONValue(v)
	if err != nil {
		return err
	}

	return p.write(p.selectFields(value), false)
}

func (p *Printer) selectFields(value any) any {
	object, ok := value.(map[string]any)
	if !ok || p.fields == nil {
		return value
	}

	selected := make(map[string]any, len(p.fields))
	for _, name := range p.fields {
		if v, ok := object[name]; ok {
			selected[name] = v
		}
	}

	return selected
}

// writeはvalueに `--jq` を適用して標準出力に書く。
func (p *Printer) write(value any, indent bool) error {
	var buf bytes.Buffer

	if p.jq == nil {
		if err := encodeJSON(&buf, value, indent); err != nil {
			return err
		}
	} else if err := p.runJQ(&buf, value, indent); err != nil {
		return err
	}

	_, err := buf.WriteTo(p.ios.Out)

	return err
}

// runJQは `--jq` の結果を1つずつ書く。文字列はgh CLIと同じく引用符を付けずにそのまま書き、
// 本文などをシェルの他のコマンドへ渡しやすくする。引用符の無い文字列は制御文字もエスケープされないため、
// 標準出力がTTYのときだけ本文の表示と同じく取り除く。
func (p *Printer) runJQ(w io.Writer, value any, indent bool) error {
	iter := p.jq.Run(value)
	for {
		v, ok := iter.Next()
		if !ok {
			return nil
		}

		if err, ok := v.(error); ok {
			var halt *gojq.HaltError
			if errors.As(err, &halt) && halt.Value() == nil && halt.ExitCode() == 0 {
				return nil
			}

			// 出力はコマンドの処理を終えてから行うため、ここでの失敗は書き込みなどの成否を表さない。
			// エージェントが書き込みを再送しないよう、コマンド自体は完了していることを伝える。
			return &Error{
				Code:     CodeJQFailed,
				Message:  p.loc.T("error_jq_failed", map[string]any{"Detail": err.Error()}),
				Hint:     p.loc.T("error_jq_failed_hint"),
				ExitCode: ExitError,
				Err:      err,
			}
		}

		if s, ok := v.(string); ok {
			if _, err := io.WriteString(w, terminalText(p.ios, s)+"\n"); err != nil {
				return err
			}
			continue
		}

		if err := encodeJSON(w, v, indent); err != nil {
			return err
		}
	}
}

func validateFields(loc *i18n.Localizer, fields []string, selectable []string) error {
	for _, name := range fields {
		if !slices.Contains(selectable, name) {
			return &Error{
				Code:     CodeUsage,
				Message:  loc.T("usage_error_unknown_field", map[string]any{"Field": name}),
				Hint:     loc.T("usage_error_unknown_field_hint", map[string]any{"Fields": strings.Join(selectable, ",")}),
				ExitCode: ExitUsage,
			}
		}
	}

	return nil
}

func compileJQ(loc *i18n.Localizer, expr string) (*gojq.Code, error) {
	query, err := gojq.Parse(expr)
	if err == nil {
		var code *gojq.Code
		code, err = gojq.Compile(query)
		if err == nil {
			return code, nil
		}
	}

	return nil, &Error{
		Code:     CodeUsage,
		Message:  loc.T("usage_error_invalid_jq", map[string]any{"Detail": err.Error()}),
		Hint:     loc.T("usage_error_invalid_jq_hint"),
		ExitCode: ExitUsage,
		Err:      err,
	}
}

// toJSONValueは、vをJSONの値 (map[string]any・[]any・文字列・json.Number・真偽値・nil) に変換する。
// 構造体のままでは項目を選べず、gojqにも渡せないためである。
// 数値はjson.Numberで保ち、float64を経由して大きな整数の精度を落とさない (gojqもjson.Numberを扱える)。
func toJSONValue(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("出力をJSONに変換できません: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("出力のJSONを読み直せません: %w", err)
	}

	return value, nil
}
