# コードレビュー: usecase-1-3

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-1-3                                          |
| ベースブランチ             | usecase-1-2                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 2 ファイル                                           |
| 変更行数（実装）           | +27 / -6 行（go/.golangci.yml）                      |
| 変更行数（テスト）         | +0 / -0 行                                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャの依存関係ルール
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - golangci-lint 設定

## 変更ファイル一覧

### 実装ファイル

- [ ] `go/.golangci.yml`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/.golangci.yml`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - レイヤー間の依存関係
- [作業計画書 検討事項 6](/workspace/docs/plans/1_doing/usecase-orchestration-refactor.md) - depguard ルールの更新表

**問題点・改善提案**:

- **[Dispatcher 層コメントの不正確さ]**: Dispatcher 層ルールのコメント「上位層や同レイヤーの他パッケージに依存しない」は deny リストの実態と合っていない。deny リストでは `model` と `repository`（同レイヤー）を禁止していないが、コメントは「同レイヤーの他パッケージに依存しない」としている。

  ```yaml
  # 現在のコメント
  # Dispatcher層のルール: ジョブキューへの投入を抽象化（Domain/Infrastructure層）
  # 上位層や同レイヤーの他パッケージに依存しない
  ```

  Dispatcher の実装を確認すると、実際に `model` や `repository` には依存しておらず、deny リストに含めなくても問題ない。ただしコメントが実態と乖離している。

  **修正案**:

  コメントを実態に合わせて修正する:

  ```yaml
  # Dispatcher層のルール: ジョブキューへの投入を抽象化（Domain/Infrastructure層）
  # 上位層（Application層、Presentation層）および関連しないパッケージに依存しない
  ```

  **対応方針**:
  - [ ] コメントを修正する
  - [x] deny リストに `model` と `repository` を追加して完全に禁止する
  - [ ] 現状のまま（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[Worker の desc メッセージが現行アーキテクチャと不一致]**: Dispatcher から Worker への依存禁止の desc メッセージが「DispatcherはPresentation層に依存できません。」となっているが、Worker は現在 Application 層（[@go/CLAUDE.md](/workspace/go/CLAUDE.md) の「主要なパッケージ」表を参照）。フェーズ 4 で Presentation 層に移動する予定だが、現時点では不正確。

  ```yaml
  # 現在
  - pkg: github.com/wikinoapp/wikino/go/internal/worker
    desc: "DispatcherはPresentation層に依存できません。"
  ```

  **修正案**:

  層名ではなくパッケージ名を使用する（`policy` や `query` の desc と同様のパターン）:

  ```yaml
  - pkg: github.com/wikinoapp/wikino/go/internal/worker
    desc: "DispatcherはWorkerに依存できません。"
  ```

  **対応方針**:
  - [x] パッケージ名ベースの desc に修正する
  - [ ] 現状のまま（フェーズ 4 で正確になるため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 作業計画書（検討事項 6）との整合確認

| 変更対象               | 計画              | 実装                                     | 結果 |
| ---------------------- | ----------------- | ---------------------------------------- | ---- |
| UseCase → Policy       | 禁止 → 許可       | deny ルール削除                          | OK   |
| UseCase → templates    | 禁止 → 例外許可   | deny ルール削除 + コメント追加           | OK   |
| UseCase → Validator    | 暗黙的許可 → 許可 | 変更不要（元から deny なし）             | OK   |
| UseCase → Dispatcher   | 未存在 → 許可     | 変更不要（新パッケージは deny 未設定）   | OK   |
| Dispatcher → UseCase   | 新設 → 禁止       | deny ルール追加                          | OK   |
| Dispatcher → Handler   | 新設 → 禁止       | deny ルール追加                          | OK   |
| Dispatcher → Worker    | 新設 → 禁止       | deny ルール追加                          | OK   |
| Dispatcher → Validator | 新設 → 禁止       | deny ルール追加                          | OK   |
| Dispatcher → Policy    | 新設 → 禁止       | deny ルール追加                          | OK   |
| Handler → Policy       | 許可 → 禁止       | フェーズ 3a-2 に延期（計画書に記載あり） | OK   |
| Handler → Validator    | 許可 → 禁止       | フェーズ 3a-2 に延期（計画書に記載あり） | OK   |
| Worker → templates     | 例外許可 → 禁止   | フェーズ 4-4 に延期（計画書に記載あり）  | OK   |
| Worker → UseCase       | 未存在 → 許可     | 変更不要（元から deny なし）             | OK   |
| Worker → Dispatcher    | 未存在 → 許可     | 変更不要（新パッケージは deny 未設定）   | OK   |

全項目が計画と一致。延期された禁止ルール（Handler → Policy/Validator、Worker → templates）は新タスク 3a-2 と 4-4 として適切に追跡されている。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 1-3 の要件（UseCase → Policy/templates の禁止ルール削除、Dispatcher 層の依存ルール新設）は正確に実装されている。作業計画書の検討事項 6 の全項目と照合した結果、すべて一致しており、延期された禁止ルールも新タスクとして適切に管理されている。

指摘は 2 件とも desc メッセージ / コメントの正確性に関するものであり、depguard ルール自体の機能には影響しない。修正は任意。
