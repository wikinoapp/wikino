package model_test

import (
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestApplyBodyEdits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		edits       []model.BodyEdit
		want        string
		wantFailure *model.BodyEditFailure
	}{
		{
			name:  "一致がちょうど1件なら置き換える",
			body:  "- [ ] **6b-1**\n- [ ] **6b-2**",
			edits: []model.BodyEdit{{OldStr: "- [ ] **6b-1**", NewStr: "- [x] **6b-1**"}},
			want:  "- [x] **6b-1**\n- [ ] **6b-2**",
		},
		{
			name: "複数の編集を配列の順に、前の編集の結果へ適用する",
			body: "あいう",
			edits: []model.BodyEdit{
				{OldStr: "い", NewStr: "えお"},
				{OldStr: "えお", NewStr: "か"},
			},
			want: "あかう",
		},
		{
			name:  "new_strが空なら一致した箇所を削除する",
			body:  "残す消す残す",
			edits: []model.BodyEdit{{OldStr: "消す", NewStr: ""}},
			want:  "残す残す",
		},
		{
			name:  "replace_allなら一致した箇所をすべて置き換える",
			body:  "foo bar foo",
			edits: []model.BodyEdit{{OldStr: "foo", NewStr: "baz", ReplaceAll: true}},
			want:  "baz bar baz",
		},
		{
			name:  "replace_allでも一致が1件なら置き換える",
			body:  "foo bar",
			edits: []model.BodyEdit{{OldStr: "foo", NewStr: "baz", ReplaceAll: true}},
			want:  "baz bar",
		},
		{
			name:        "一致が0件なら失敗する",
			body:        "本文",
			edits:       []model.BodyEdit{{OldStr: "無い", NewStr: "x"}},
			wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 0},
		},
		{
			name:        "replace_allでも一致が0件なら失敗する",
			body:        "本文",
			edits:       []model.BodyEdit{{OldStr: "無い", NewStr: "x", ReplaceAll: true}},
			wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 0},
		},
		{
			name:        "replace_allでなく一致が複数件なら失敗する",
			body:        "foo bar foo foo",
			edits:       []model.BodyEdit{{OldStr: "foo", NewStr: "baz"}},
			wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 3},
		},
		{
			name:        "大文字小文字を区別する",
			body:        "Wikino",
			edits:       []model.BodyEdit{{OldStr: "wikino", NewStr: "x"}},
			wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 0},
		},
		{
			name: "前の編集で一致しなくなった編集は、その位置で失敗する",
			body: "あいう",
			edits: []model.BodyEdit{
				{OldStr: "あ", NewStr: "か"},
				{OldStr: "い", NewStr: "き"},
				{OldStr: "あ", NewStr: "さ"},
				{OldStr: "う", NewStr: "く"},
			},
			wantFailure: &model.BodyEditFailure{Index: 2, MatchCount: 0},
		},
		{
			name: "前の編集で一致が増えた編集は、その位置で失敗する",
			body: "あいう",
			edits: []model.BodyEdit{
				{OldStr: "う", NewStr: "あ"},
				{OldStr: "あ", NewStr: "か"},
			},
			wantFailure: &model.BodyEditFailure{Index: 1, MatchCount: 2},
		},
		{
			name:        "空のold_strは失敗する",
			body:        "本文",
			edits:       []model.BodyEdit{{OldStr: "", NewStr: "x"}},
			wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 0},
		},
		{
			name:  "本文のCRLFとold_strのLFを同じ改行として照合し、LFに揃える",
			body:  "1行目\r\n2行目\r\n3行目",
			edits: []model.BodyEdit{{OldStr: "1行目\n2行目", NewStr: "一行目\n二行目"}},
			want:  "一行目\n二行目\n3行目",
		},
		{
			name:  "old_strとnew_strのCRLFもLFに揃える",
			body:  "1行目\n2行目",
			edits: []model.BodyEdit{{OldStr: "1行目\r\n", NewStr: "一行目\r\n"}},
			want:  "一行目\n2行目",
		},
		{
			name:  "重なり合う一致は重ならない数で数える",
			body:  "aaa",
			edits: []model.BodyEdit{{OldStr: "aa", NewStr: "b"}},
			want:  "ba",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, failure := model.ApplyBodyEdits(tt.body, tt.edits)
			if tt.wantFailure != nil {
				if failure == nil || *failure != *tt.wantFailure {
					t.Errorf("failure = %+v、期待値 = %+v", failure, tt.wantFailure)
				}
				if got != "" {
					t.Errorf("本文 = %q、期待値 = 失敗したときは空", got)
				}
				return
			}
			if failure != nil {
				t.Fatalf("failure = %+v、期待値 = nil", failure)
			}
			if got != tt.want {
				t.Errorf("本文 = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestApplyBodyEdits_BodySize(t *testing.T) {
	t.Parallel()
	limit := model.BodyEditMaxBodyBytes
	doubling := make([]model.BodyEdit, 21)
	for i := range doubling {
		doubling[i] = model.BodyEdit{OldStr: "a", NewStr: "aa", ReplaceAll: true}
	}
	tests := []struct {
		name        string
		body        string
		edits       []model.BodyEdit
		wantBytes   int
		wantFailure *model.BodyEditFailure
	}{
		{name: "上限ちょうどなら適用する", body: "a", edits: []model.BodyEdit{{OldStr: "a", NewStr: strings.Repeat("b", limit)}}, wantBytes: limit},
		{name: "上限より1バイト少なければ適用する", body: "a", edits: []model.BodyEdit{{OldStr: "a", NewStr: strings.Repeat("b", limit-1)}}, wantBytes: limit - 1},
		{name: "残す本文を含めて上限を超えれば拒否する", body: "ab", edits: []model.BodyEdit{{OldStr: "a", NewStr: strings.Repeat("c", limit)}}, wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 1, SizeExceeded: true}},
		{name: "文字数ではなくUTF-8のバイト数で制限する", body: "a", edits: []model.BodyEdit{{OldStr: "a", NewStr: strings.Repeat("あ", limit/3+1)}}, wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 1, SizeExceeded: true}},
		{name: "改行コードをLFに揃えた後のバイト数で判定する", body: "a", edits: []model.BodyEdit{{OldStr: "a", NewStr: strings.Repeat("\r\n", limit)}}, wantBytes: limit},
		{name: "連続した倍増を途中で拒否する", body: "a", edits: doubling, wantFailure: &model.BodyEditFailure{Index: 20, MatchCount: limit, SizeExceeded: true}},
		{name: "後の編集で小さくなる場合も途中の上限超過を拒否する", body: "a", edits: []model.BodyEdit{{OldStr: "a", NewStr: strings.Repeat("b", limit+1)}, {OldStr: "b", NewStr: "", ReplaceAll: true}}, wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 1, SizeExceeded: true}},
		{name: "巨大な置換結果を乗算や確保の前に拒否する", body: strings.Repeat("a", limit), edits: []model.BodyEdit{{OldStr: "a", NewStr: strings.Repeat("b", limit), ReplaceAll: true}}, wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: limit, SizeExceeded: true}},
		{name: "既存の大きい本文も上限以下に縮めれば適用する", body: strings.Repeat("a", limit+1), edits: []model.BodyEdit{{OldStr: "a", NewStr: "", ReplaceAll: true}}, wantBytes: 0},
		{name: "一致部分を削除しても残る本文が上限を超えれば拒否する", body: "a" + strings.Repeat("b", limit+1), edits: []model.BodyEdit{{OldStr: "a", NewStr: ""}}, wantFailure: &model.BodyEditFailure{Index: 0, MatchCount: 1, SizeExceeded: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body, failure := model.ApplyBodyEdits(tt.body, tt.edits)
			if tt.wantFailure != nil {
				if failure == nil || *failure != *tt.wantFailure {
					t.Errorf("failure = %+v、期待値 = %+v", failure, tt.wantFailure)
				}
				if body != "" {
					t.Errorf("本文のバイト数 = %d、期待値 = 0 (失敗時は結果を返さない)", len(body))
				}
				return
			}
			if failure != nil {
				t.Fatalf("failure = %+v、期待値 = nil", failure)
			}
			if len(body) != tt.wantBytes {
				t.Errorf("本文のバイト数 = %d、期待値 = %d", len(body), tt.wantBytes)
			}
		})
	}
}
