# コードレビュー: suggestion-12-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-26                       |
| 対象ブランチ               | suggestion-12-1                  |
| ベースブランチ             | suggestion-fix3                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 7 ファイル                       |
| 変更行数（実装）           | +34 / -0 行                      |
| 変更行数（テスト）         | +277 / -0 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 認可チェック（Policy）
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`

### テストファイル

- [x] `go/internal/policy/topic_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに従っています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 12-1「編集提案・コメント編集の権限（Policy）追加」が仕様通りに実装されている。

**良い点**:

- `TopicPolicy` インターフェースに `CanUpdateSuggestion` と `CanUpdateSuggestionComment` が追加され、4つのポリシー実装（owner, admin, member, guest）すべてで適切に実装されている
- 仕様の「スペースメンバー（アクティブ）であれば編集可能」を正しく反映: owner はスペースレベル、admin/member はトピックレベル、guest はアクティブ + オープンのみでチェックし、いずれのパスでもアクティブなスペースメンバーであれば編集可能
- 「オープン状態の編集提案のみ編集可能」の制約が全ポリシーで `suggestion.Status == model.SuggestionStatusOpen` として統一的に実装されている
- 既存の `CanApplySuggestion`/`CanCloseSuggestion` と一貫したパターンで実装されており、可読性が高い
- テストが各ロール（SpaceOwner, TopicAdmin, TopicMember, Guest）に対して正常系・異常系（クローズ状態、別スペース/別トピック、非アクティブ）を網羅的にカバーしている
- `CanUpdateSuggestion` と `CanUpdateSuggestionComment` を常にペアでテストしており、漏れがない
- Policy は model のみに依存しており、アーキテクチャガイドの依存関係ルールに従っている
