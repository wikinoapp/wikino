package cmdutil

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apigen"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// 一覧のページ送りのフラグの名前。一覧を返すすべてのコマンドで同じ意味に使う。
const (
	FlagLimit  = "limit"
	FlagCursor = "cursor"
	FlagAll    = "all"
)

// ListFlagsは、一覧のページ送りのフラグの値。
type ListFlags struct {
	// Limitは1ページの件数。0ならAPIの既定の件数にする。
	Limit int32
	// Cursorは取得するページのカーソル。空なら最初のページを取得する。
	Cursor string
	// Allは、`next_cursor` を辿って全件を取得するか。
	All bool
}

// AddListFlagsは、一覧のページ送りのフラグをcmdに足す。
func AddListFlags(cmd *cobra.Command, loc *i18n.Localizer) *ListFlags {
	flags := &ListFlags{}
	cmd.Flags().Int32Var(&flags.Limit, FlagLimit, 0, loc.T("help_limit_flag"))
	cmd.Flags().StringVar(&flags.Cursor, FlagCursor, "", loc.T("help_cursor_flag"))
	cmd.Flags().BoolVar(&flags.All, FlagAll, false, loc.T("help_all_flag"))

	return flags
}

// Validateは、要求を送る前にページ送りのフラグの組み合わせと値を検査する。
func (l *ListFlags) Validate(loc *i18n.Localizer, cmd *cobra.Command) error {
	if cmd.Flags().Changed(FlagLimit) && l.Limit < 1 {
		return NewUsageError(loc, cmd, loc.T("usage_error_invalid_limit", map[string]any{"Value": l.Limit}))
	}
	// `--all` は最初のページから辿るため、途中のカーソルとは組み合わせない。
	if l.All && cmd.Flags().Changed(FlagCursor) {
		return NewUsageError(loc, cmd, loc.T("usage_error_flags_exclusive", map[string]any{"Flags": "--all, --cursor"}))
	}

	return nil
}

// LimitParamは、要求に載せる1ページの件数を返す。指定が無ければnil (APIの既定の件数) を返す。
func (l *ListFlags) LimitParam() *int32 {
	if l.Limit < 1 {
		return nil
	}

	return &l.Limit
}

// CursorParamは、最初のページで要求に載せるカーソルを返す。指定が無ければnilを返す。
func (l *ListFlags) CursorParam() *string {
	if l.Cursor == "" {
		return nil
	}

	return &l.Cursor
}

// ListIncludeは、一覧の要求に載せる `include` を返す。本文 (body) を出すときだけ本文を求め、出さないならnilを返す。
// 本文を求めると応答が本文の長さに比例して大きくなるため、出さない本文は受け取らない。
func ListInclude(printer *output.Printer) *apigen.ListInclude {
	if !printer.Selects("body") {
		return nil
	}

	return &apigen.ListInclude{apigen.Body}
}

// Listは、一覧を返すコマンドの取得と表示の方法。
type List[T any] struct {
	// Fetchは、cursorのページを取得する。件数や絞り込みはコマンドの側で載せる。
	Fetch apiclient.PageFetcher[T]
	// Itemは、APIの1件をJSONで出す1件にする。
	Item func(T) any
	// Tableは、人間向けの表示で1ページ分を表にして出す。itemsは1件以上ある。
	Table func(items []T) error
	// EmptyHintは、人間向けの表示で0件のときに、案内に続けて標準エラーに出すヒント。空なら出さない。
	EmptyHint string
	// NotFoundHintは、404のときのヒント。空ならAPIのエラーの変換の共通のヒントにする。
	// ページを起点にした一覧で、起点の番号を確かめるコマンドを案内するために使う。
	NotFoundHint string
}

func (l List[T]) convertError(loc *i18n.Localizer, err error) error {
	if l.NotFoundHint == "" {
		return apiutil.ConvertError(loc, err)
	}

	return ConvertNotFound(loc, err, l.NotFoundHint)
}

// listOutputは一覧のJSONの形。APIの一覧の応答と同じ形にする。
type listOutput struct {
	Items      []any   `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// RunListは、flagsに従って一覧を取得して出す。
//
//   - `--all` なら `next_cursor` を辿って全件を取得し、1行1件のJSON Linesで出す。
//     全件をメモリに溜めずに流すため、人間向けの表示にはしない
//   - それ以外は1ページだけを取得し、APIの一覧の形のJSONか、人間向けの表で出す。
//     次のページがあれば、標準出力を汚さないよう標準エラーに取り方を出す
//   - 0件なら、JSONでは空の一覧を出す。人間向けの表示では標準出力に何も出さず、標準エラーに案内とヒントを出す
func RunList[T any](ctx context.Context, f *Factory, printer *output.Printer, flags *ListFlags, l List[T]) error {
	loc := f.Localizer

	if flags.All {
		for item, err := range apiclient.All(ctx, l.Fetch) {
			if err != nil {
				return l.convertError(loc, err)
			}
			if err := printer.PrintLine(l.Item(item)); err != nil {
				return err
			}
		}

		return nil
	}

	items, next, err := l.Fetch(ctx, flags.CursorParam())
	if err != nil {
		return l.convertError(loc, err)
	}
	if next != nil && *next == "" {
		next = nil
	}

	switch {
	case printer.IsJSON():
		out := listOutput{Items: make([]any, 0, len(items)), NextCursor: next}
		for _, item := range items {
			out.Items = append(out.Items, l.Item(item))
		}
		if err := printer.PrintList(out); err != nil {
			return err
		}
	case len(items) == 0:
		if _, err := fmt.Fprintln(f.IOStreams.ErrOut, loc.T("list_empty")); err != nil {
			return err
		}
		if l.EmptyHint != "" {
			if _, err := fmt.Fprintln(f.IOStreams.ErrOut, l.EmptyHint); err != nil {
				return err
			}
		}
	default:
		if err := l.Table(items); err != nil {
			return err
		}
	}

	if next != nil {
		if _, err := fmt.Fprintln(f.IOStreams.ErrOut, loc.T("list_next_page", map[string]any{"Cursor": *next})); err != nil {
			return err
		}
	}

	return nil
}
