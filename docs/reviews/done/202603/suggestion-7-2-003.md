# コードレビュー: suggestion-7-2 (編集提案反映のハンドラーとPolicy)

## レビュー情報

| 項目                       | 内容                                       |
| -------------------------- | ------------------------------------------ |
| レビュー日                 | 2026-03-19                                 |
| 対象ブランチ               | suggestion-7-2                             |
| ベースブランチ             | suggestion-7-1c                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md (7-2)     |
| 変更ファイル数             | 20 ファイル（うちドキュメント・レビュー3） |
| 変更行数（実装）           | +227 行                                    |
| 変更行数（テスト）         | +498 行                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/handler/suggestion_apply/create.go`
- [x] `go/internal/handler/suggestion_apply/handler.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/show_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_apply/create_test.go`
- [x] `go/internal/handler/suggestion_apply/main_test.go`
- [x] `go/internal/policy/topic_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-7-2-001.md`
- [x] `docs/reviews/suggestion-7-2-002.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書タスク **7-2** の要件と実装の対応:

| 要件                                                                 | 実装状況 |
| -------------------------------------------------------------------- | -------- |
| `TopicPolicy` インターフェースに `CanApplySuggestion` メソッド追加   | ✅       |
| 各ポリシー実装に `CanApplySuggestion` を実装                         | ✅       |
| `suggestion_apply/handler.go` に Handler 構造体を定義                | ✅       |
| `suggestion_apply/create.go` に `Create` メソッドを実装              | ✅       |
| POST /s/{space}/suggestions/{suggestion_number}/apply のルーティング | ✅       |
| 反映ボタンをテンプレートに追加（オープン+権限ありの場合のみ）        | ✅       |
| `cmd/server/main.go` にルーティング登録                              | ✅       |
| 翻訳ファイル（ja.toml, en.toml）にメッセージ追加                     | ✅       |
| べき等性: 反映済みの場合は何もせず成功                               | ✅       |
| 反映権限: スペースオーナーまたはトピック管理者のみ                   | ✅       |

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 7-2（編集提案反映のハンドラーとPolicy）が作業計画書の要件通りに実装されている。

**良い点**:

- 既存の `suggestion_comment` ハンドラーと一貫性のあるパターンで実装されている
- `TopicPolicy` インターフェースへの `CanApplySuggestion` の追加が自然で、既存の認可パターン（`CanCreatePage`, `CanUpdatePage`, `CanUpdateDraftPage`）と一貫している
- べき等性が正しく実装されている（反映済み → 何もせず成功、クローズ済み → エラー）
- ポリシーの権限設計が仕様通り: スペースオーナーは `spaceID` で判定、トピック管理者は `topicID` で判定、一般メンバー・ゲストは不可
- テストカバレッジが十分（未認証、404、非メンバー403、一般メンバー403、オーナー成功、べき等性）
- セキュリティ対策が適切（CSRF トークン、認証チェック、認可チェック）
- 翻訳が日英両方追加されている
- PRサイズが適切（実装 227 行、テスト 498 行）
