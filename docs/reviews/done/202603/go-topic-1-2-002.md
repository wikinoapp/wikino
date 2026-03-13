# コードレビュー: go-topic-1-2

## レビュー情報

| 項目                       | 内容                                                       |
| -------------------------- | ---------------------------------------------------------- |
| レビュー日                 | 2026-03-08                                                 |
| 対象ブランチ               | go-topic-1-2                                               |
| ベースブランチ             | go-topic                                                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md（タスク1-2） |
| 変更ファイル数             | 4 ファイル                                                 |
| 変更行数（実装）           | +9 / -5 行                                                 |
| 変更行数（テスト）         | +92 / -0 行                                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/viewmodel/page.go`

### テストファイル

- [x] `go/internal/viewmodel/page_test.go`

### 設定・その他

- [x] `.claude/commands/review.md`
- [x] `docs/plans/1_doing/topic-show-go-migration.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

## 設計改善の提案

### `go/internal/viewmodel/page_test.go`: TestNewCardLinkPage で TopicName と TopicIcon の検証追加

**ステータス**: 対応済み

**現状**:

`TestNewCardLinkPage` は今回新規追加されたテスト関数であり、`CardImageURL` の検証が主目的ですが、`NewCardLinkPage` 関数全体のテストでもあります。現在 `Title`、`Number`、`CardImageURL`、`Pinned` を検証していますが、`TopicName` と `TopicIcon` の検証が含まれていません。

```go
// 現在の検証項目
if got.Title != tt.wantTitle { ... }
if got.Number != tt.wantNumber { ... }
if got.CardImageURL != tt.wantCardImageURL { ... }
if got.Pinned != tt.wantPinned { ... }
// TopicName, TopicIcon の検証がない
```

**提案**:

テストケースに `wantTopicName` と `wantTopicIcon` を追加し、`NewCardLinkPage` の全フィールドを網羅的に検証する。

```go
tests := []struct {
    name             string
    page             *model.Page
    topicMap         map[model.TopicID]*model.Topic
    wantTitle        string
    wantNumber       int32
    wantCardImageURL string
    wantPinned       bool
    wantTopicName    string
    wantTopicIcon    viewmodel.IconName
}{
    // ... 既存テストケースに wantTopicName, wantTopicIcon を追加
}
```

**メリット**:

- `NewCardLinkPage` の全出力フィールドが網羅され、テストの信頼性が向上する
- トピック情報が `topicMap` から正しく取得されることを明示的に検証できる

**トレードオフ**:

- テストコードが若干増える（各テストケースに2フィールド追加）
- `TopicName`/`TopicIcon` は今回の変更対象ではないため、この PR のスコープを超える可能性がある

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [x] 提案通り変更する
- [ ] 現状のまま（理由を回答欄に記入）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク1-2「ページカード用 ViewModel のアイキャッチ画像対応」の実装は、作業計画書の仕様通りに正確に実装されています。

**良かった点**:

- 変更が最小限に抑えられており、`NewCardLinkPage` コンストラクタへの `CardImageURL` 設定処理の追加のみという明確なスコープ
- テストがテーブル駆動テストで書かれており、アイキャッチ画像あり・なし・タイトルnil の3パターンを適切にカバー
- 既存のコードパターン（`/attachments/{id}` のURL形式）と一貫性がある
- アーキテクチャガイドの依存関係ルール（ViewModel → Model のみ）に従っている
- `fmt.Sprintf` による URL 構築は、プロジェクト内の他の添付ファイルURLパターンと統一されている

**設計との整合性**: 作業計画書に記載された「`NewCardLinkPage` コンストラクタでアイキャッチ画像 URL（`CardImageURL`）を設定する処理を追加」「`CardImageURL` フィールドとテンプレート側の表示ロジックは既に存在するため、コンストラクタの修正のみ」という要件を満たしています。
