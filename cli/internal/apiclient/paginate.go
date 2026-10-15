package apiclient

import (
	"context"
	"errors"
	"iter"
)

// ErrCursorLoopは、次のページのカーソルが前に辿ったカーソルと同じになったことを表す。
// サーバーの不具合で同じページを取り続け、`--all` が終わらなくなるのを防ぐ。
var ErrCursorLoop = errors.New("apiclient: next_cursorが既に辿ったカーソルと同じです")

// PageFetcherは、cursorのページ (nilなら最初のページ) を取得し、その要素と次のページのカーソルを返す。
// 次のページが無ければnextはnilか空文字列にする。
type PageFetcher[T any] func(ctx context.Context, cursor *string) (items []T, next *string, err error)

// Allは、fetchで `next_cursor` を辿り、すべての要素を順に返す。
// 全件をメモリに溜めず、1ページずつ取得して流す。失敗したらそのエラーを返して終える。
func All[T any](ctx context.Context, fetch PageFetcher[T]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		var cursor *string
		seen := map[string]bool{}

		for {
			items, next, err := fetch(ctx, cursor)
			if err != nil {
				yield(zero, err)
				return
			}

			for _, item := range items {
				if !yield(item, nil) {
					return
				}
			}

			if next == nil || *next == "" {
				return
			}
			if seen[*next] {
				yield(zero, ErrCursorLoop)
				return
			}
			seen[*next] = true
			cursor = next
		}
	}
}
