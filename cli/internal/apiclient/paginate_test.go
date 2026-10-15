package apiclient

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func ptr(s string) *string {
	return &s
}

// pagesは、カーソルごとのページを返すPageFetcherを作る。キーの "" は最初のページ。
func pages(t *testing.T, byCursor map[string]struct {
	items []int
	next  *string
}, calls *[]string) PageFetcher[int] {
	t.Helper()

	return func(_ context.Context, cursor *string) ([]int, *string, error) {
		key := ""
		if cursor != nil {
			key = *cursor
		}
		*calls = append(*calls, key)

		p, ok := byCursor[key]
		if !ok {
			return nil, nil, errors.New("無いページです: " + key)
		}

		return p.items, p.next, nil
	}
}

func TestAll(t *testing.T) {
	t.Parallel()

	var calls []string
	fetch := pages(t, map[string]struct {
		items []int
		next  *string
	}{
		"":   {items: []int{1, 2}, next: ptr("c2")},
		"c2": {items: []int{}, next: ptr("c3")},
		"c3": {items: []int{3}, next: nil},
	}, &calls)

	var got []int
	for item, err := range All(context.Background(), fetch) {
		if err != nil {
			t.Fatalf("失敗しました: %v", err)
		}
		got = append(got, item)
	}

	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("要素が違います: %v", got)
	}
	if !slices.Equal(calls, []string{"", "c2", "c3"}) {
		t.Errorf("辿ったカーソルが違います: %v", calls)
	}
}

func TestAll_StopsEarly(t *testing.T) {
	t.Parallel()

	var calls []string
	fetch := pages(t, map[string]struct {
		items []int
		next  *string
	}{
		"":   {items: []int{1, 2}, next: ptr("c2")},
		"c2": {items: []int{3}, next: nil},
	}, &calls)

	for item := range All(context.Background(), fetch) {
		if item == 1 {
			break
		}
	}

	if len(calls) != 1 {
		t.Errorf("途中で止めたのに次のページを取得しました: %v", calls)
	}
}

func TestAll_Error(t *testing.T) {
	t.Parallel()

	var calls []string
	fetch := pages(t, map[string]struct {
		items []int
		next  *string
	}{
		"": {items: []int{1}, next: ptr("missing")},
	}, &calls)

	var got []int
	var gotErr error
	for item, err := range All(context.Background(), fetch) {
		if err != nil {
			gotErr = err
			break
		}
		got = append(got, item)
	}

	if gotErr == nil {
		t.Fatal("取得の失敗が返りません")
	}
	if !slices.Equal(got, []int{1}) {
		t.Errorf("失敗の前の要素が違います: %v", got)
	}
}

func TestAll_CursorLoop(t *testing.T) {
	t.Parallel()

	var calls []string
	fetch := pages(t, map[string]struct {
		items []int
		next  *string
	}{
		"":   {items: []int{1}, next: ptr("c2")},
		"c2": {items: []int{2}, next: ptr("c2")},
	}, &calls)

	var gotErr error
	for _, err := range All(context.Background(), fetch) {
		if err != nil {
			gotErr = err
		}
	}

	if !errors.Is(gotErr, ErrCursorLoop) {
		t.Errorf("同じカーソルを辿り続けています: %v (呼び出し=%v)", gotErr, calls)
	}
}
