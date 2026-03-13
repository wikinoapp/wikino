# コードレビュー: handler-usecase-refactor-5-3

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-12                                     |
| 対象ブランチ               | handler-usecase-refactor-5-3                   |
| ベースブランチ             | handler-usecase-refactor-5-2                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 9 ファイル                                     |
| 変更行数（実装）           | +192 / -217 行                                 |
| 変更行数（テスト）         | +190 / -12 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/get_page_move_data.go`
- [x] `go/internal/handler/page_move/handler.go`
- [x] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/get_page_move_data_test.go`
- [x] `go/internal/handler/page_move/new_test.go`
- [x] `go/internal/handler/page_move/create_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/get_page_move_data.go`: `availableTopicsForMove` の認可ロジックに関するコメントの欠落

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/coding-guide.md#コメントのガイドライン](/workspace/go/docs/coding-guide.md) - コメントのガイドライン

**問題点・改善提案**:

- **[@go/docs/coding-guide.md#コメントのガイドライン]**: `availableTopicsForMove` メソッドの旧実装（Handler 内）には、認可ロジックが暗黙的に満たされている理由を説明する「なぜ」のコメントがあったが、UseCase への移動時に省略されている

  旧実装のコメント:

  ```go
  // スペースオーナーは同スペース内の全トピックにCanCreatePageが真であり、
  // 非オーナーはListJoinedBySpaceMemberが所属トピックのみを返すため、
  // いずれの場合もリスト取得の段階で権限が暗黙的に満たされています。
  ```

  現在のコメント:

  ```go
  // availableTopicsForMove は移動先候補のトピック一覧を取得する。
  // スペースオーナーは全アクティブトピック、それ以外は所属トピックのみ返す。
  // 現在のトピックは除外する。
  ```

  **修正案**:

  認可が暗黙的に満たされている理由を「なぜ」のコメントとして追加する:

  ```go
  // availableTopicsForMove は移動先候補のトピック一覧を取得する。
  // スペースオーナーは全アクティブトピック、それ以外は所属トピックのみ返す。
  // 現在のトピックは除外する。
  // スペースオーナーは同スペース内の全トピックにCanCreatePageが真であり、
  // 非オーナーはListJoinedBySpaceMemberが所属トピックのみを返すため、
  // いずれの場合もリスト取得の段階で権限が暗黙的に満たされている。
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りコメントを追加する
  - [ ] 現状のまま（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 5-3 の要件通り、page_move ハンドラーから Repository への直接依存が正しく除去され、`GetPageMoveDataUsecase` 経由に統一されている。

**良い点**:

- Handler 構造体から 5 つの Repository フィールドが削除され、UseCase 1 つに集約された。Handler がシンプルになっている
- 既存の UseCase パターン（命名規則、nil 返却、エラーラップ）との一貫性が保たれている
- `availableTopicsForMove` のロジックが UseCase のプライベートメソッドとして適切に移動されている
- `policy.NewTopicPolicy` による認可チェックも UseCase 内に移動され、Handler から認可の関心が分離されている
- UseCase テストで正常系・異常系（存在しないスペース、非メンバー、存在しないページ、権限なし）が網羅されている
- Handler テストが UseCase 経由に更新され、既存のテストケースがすべて維持されている
- `renderMoveForm` の引数が個別のモデルから `*usecase.GetPageMoveDataOutput` に集約され、シグネチャが簡潔になっている

**指摘事項**:

- 1 件（推奨）: `availableTopicsForMove` の認可ロジックに関する「なぜ」のコメントが移動時に省略されている。セキュリティ上の判断根拠を残すため、追加を推奨
