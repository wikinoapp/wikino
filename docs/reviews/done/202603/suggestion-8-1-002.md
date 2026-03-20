# コードレビュー: suggestion-8-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-19                       |
| 対象ブランチ               | suggestion-8-1                   |
| ベースブランチ             | suggestion-7-2                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 20 ファイル                      |
| 変更行数（実装）           | 約 +310 / -20 行                 |
| 変更行数（テスト）         | 約 +754 行                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/handler/suggestion_close/create.go`
- [x] `go/internal/handler/suggestion_close/handler.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/show_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/close_suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion_close/create_test.go`
- [x] `go/internal/handler/suggestion_close/main_test.go`
- [x] `go/internal/policy/topic_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/suggestion-8-1-001.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載。問題のないファイルは変更ファイル一覧のチェックボックスにチェック済み。

（問題なし — 全ファイル問題ありませんでした）

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 8-1「編集提案クローズのUseCase・ハンドラー」の実装として、過不足なく完成度の高い実装です。

**良い点**:

- **`suggestion_apply` との高い一貫性**: ハンドラー構造体、パス生成、テスト構造が `suggestion_apply` と同じパターンで統一されており、コードの見通しが良い
- **Policy設計の適切さ**: `CanCloseSuggestion` の権限モデルが「作成者 OR スペースオーナー OR トピック管理者」を各ポリシーで正しく実装。特に `topicGuestPolicy` でも作成者本人のクローズを許可している点が正確
- **べき等性の実装**: クローズ済み→成功、反映済み→エラー、オープン→クローズ実行 と、設計通りのべき等挙動を実装
- **セキュリティ**: `space_id` によるクエリスコープ（`FindByID`、`UpdateStatus` の両方）、CSRF対策、認証・認可チェックすべて適切
- **テストカバレッジ**: 8つのテストケース（未ログイン、存在しない提案、非メンバー、権限なし一般メンバー、作成者、オーナー、管理者、べき等性）で権限パターンを網羅。Policy単体テストも全ロール分追加
- **i18n**: 日英両方のメッセージが適切に追加されている
- **アーキテクチャ準拠**: Handler→UseCase→Repository の依存方向、認可チェックはHandlerで実行、UseCase内でトランザクション管理、すべてガイドライン通り
