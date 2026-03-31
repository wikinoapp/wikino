# コードレビュー: usecase-6-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-31                                           |
| 対象ブランチ               | usecase-6-1                                          |
| ベースブランチ             | usecase-5a-2                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 3 ファイル                                           |
| 変更行数（実装）           | +28 / -1 行                                          |
| 変更行数（テスト）         | +0 / -0 行                                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@CLAUDE.md#コメントのガイドライン](/workspace/CLAUDE.md) - コメントのガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/usecase-guide.md](/workspace/go/docs/usecase-guide.md) - ユースケースガイド

## 変更ファイル一覧

### ドキュメント

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/usecase-guide.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。各ファイルの変更内容は以下の通りです：

- **`docs/plans/1_doing/usecase-orchestration-refactor.md`**: タスク 6-1 のチェックボックスにチェックを入れただけ。問題なし。
- **`go/docs/architecture-guide.md`**: 採用しなかった方針 F（Worker でテンプレートをレンダリングし UseCase に渡す）と G（メールテンプレートを独立パッケージに分離する）を追加。作業計画書の採用しなかった方針 B, C と内容・理由が一致しており、既存セクション（A〜E）のフォーマットとも一貫している。
- **`go/docs/usecase-guide.md`**: 採用しなかった方針 C（Read UseCase を廃止し UseCase を1つに統合する）を追加。作業計画書の採用しなかった方針 A と内容が一致しており、既存セクション（A, B）のフォーマットとも一貫している。

## 設計改善の提案

### 仕様書への反映範囲: email パッケージの Sender パターンに関する仕様が未記載

**ステータス**: 対応済み

**現状**:

タスク 6-1 の定義は「作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する」だが、今回の差分は「採用しなかった方針」3 件の追加のみ。

作業計画書の設計判断のうち、以下の項目が仕様書に十分反映されていない：

1. **email パッケージの Sender パターン**（検討事項 7）:
   - `ConfirmationSender`, `PasswordResetSender` の具体的な設計パターン
   - UseCase が email パッケージの interface に依存する構造
   - `SendRaw` 廃止と `Send` への統一
   - email-layer の depguard ルール

`architecture-guide.md` には既にメール送信関連の簡潔な記載（Presentation 層ヘルパーとしての位置づけ、Worker/UseCase セクションでの言及）があるが、email パッケージ内の Sender パターンの詳細な仕様は記載されていない。

**提案**:

`architecture-guide.md` の email パッケージに関する記述を拡充し、以下を追加する：

- email パッケージの構成（`Sender` インターフェース、`ConfirmationSender`, `PasswordResetSender`）
- UseCase → email の interface 依存パターン（テンプレートレンダリングは email パッケージに閉じる）
- email-layer の depguard ルールの概要

**メリット**:

- 作業計画書の検討事項 7 の設計判断が仕様書に残る
- email パッケージの設計意図が将来の開発者にも伝わる

**トレードオフ**:

- 既存の仕様書に十分な情報があるとも解釈できる（「メール種別ごとの Sender でテンプレートレンダリングと i18n 件名取得を担当」等の記載は既にある）
- 作業計画書自体が詳細な設計記録として残っている

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [x] 提案通り email パッケージの詳細仕様を `architecture-guide.md` に追加する
- [ ] 現状の記載で十分と判断する（理由を回答欄に記入）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Comment

**総評**:

今回の差分は、作業計画書の「採用しなかった方針」3 件（A: Read UseCase 統合、B: Worker でのテンプレートレンダリング、C: メールテンプレート独立パッケージ化）を仕様書に転記するドキュメント更新であり、内容・フォーマットともに問題ない。

仕様書の大部分はフェーズ 1〜5a の実装タスクの中で既に更新されており、今回の差分で追加された「採用しなかった方針」も作業計画書と正確に一致している。

設計改善の提案として email パッケージの Sender パターン仕様の拡充を提案したが、これは必須ではなく、開発者の判断に委ねる。
