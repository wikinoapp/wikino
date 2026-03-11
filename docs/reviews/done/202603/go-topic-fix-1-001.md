# コードレビュー: go-topic-fix-1

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-09                                    |
| 対象ブランチ               | go-topic-fix-1                                |
| ベースブランチ             | go-topic                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md |
| 変更ファイル数             | 9 ファイル                                    |
| 変更行数（実装）           | +2 / -237 行                                  |
| 変更行数（テスト）         | +0 / -160 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版の開発ガイド
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `rails/app/controllers/topics/show_controller.rb`（削除）
- [x] `rails/app/views/topics/show_view.rb`（削除）
- [x] `rails/app/views/topics/show_view.html.erb`（削除）
- [x] `rails/app/views/topics/show_view/header_component.rb`（削除）
- [x] `rails/app/views/topics/show_view/header_component.html.erb`（削除）

### テストファイル

- [x] `rails/spec/requests/topics/show_spec.rb`（削除）
- [x] `rails/spec/system/global_hotkey_spec.rb`（修正）

### 設定・その他

- [ ] `Dockerfile.dev`（修正）
- [x] `docs/plans/1_doing/topic-show-go-migration.md`（修正）

## ファイルごとのレビュー結果

### `Dockerfile.dev`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md](/workspace/CLAUDE.md) - コメントのガイドライン

**問題点・改善提案**:

- **タスクと無関係な変更**: Dockerfile.dev のコメント変更（`net-tools` と `psmisc` への「Claude Codeで使用」追加）は、トピック詳細画面の Rails コード削除とは無関係な変更です。PR の目的を明確に保つため、この変更は別のコミット/PR に分けることを推奨します。

  ```dockerfile
  # 変更前（元のコメント）
  net-tools                 # netstat (ネットワークデバッグ用)
  psmisc                    # fuser (プロセス特定用)

  # 変更後（このPRでの変更）
  net-tools                 # Claude Codeで使用 netstat (ネットワークデバッグ用)
  psmisc                    # Claude Codeで使用 fuser (プロセス特定用)
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] この PR から Dockerfile.dev の変更を除外し、別の PR で対応する
  - [x] 軽微な変更のため、このまま含める
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 作業計画書との整合性

作業計画書（フェーズ 3a-1）に記載された要件との整合性を確認しました。

| 要件                                | 状態 | 備考                                                              |
| ----------------------------------- | ---- | ----------------------------------------------------------------- |
| show_controller.rb の削除           | ✅   | 削除済み                                                          |
| show_view.rb の削除                 | ✅   | 削除済み                                                          |
| show_view.html.erb の削除           | ✅   | 削除済み                                                          |
| header_component.rb の削除          | ✅   | 削除済み                                                          |
| header_component.html.erb の削除    | ✅   | 削除済み                                                          |
| show_spec.rb の削除                 | ✅   | 削除済み                                                          |
| routes.rb の topic named route 維持 | ✅   | `topic_path` が 8+ ファイルで使用中のため維持                     |
| 共有コンポーネント・モデルの維持    | ✅   | 削除対象外のファイルは変更なし                                    |
| Sorbet 型定義の更新                 | ✅   | 関連する RBI ファイルが存在しないため更新不要                     |
| global_hotkey_spec.rb のテスト削除  | ✅   | 計画書には未記載だが、Go に移行したページのテストとして適切な対応 |

### Go 版でのテストカバレッジ

Rails 版で削除されたテストケースに対応するテストが Go 版（`go/internal/handler/topic/show_test.go`）に存在することを確認しました:

- 未ログイン + 非公開トピック → 404 ✅
- 別スペースメンバー + 非公開トピック → 404 ✅
- 公開トピックの閲覧 ✅
- トピックメンバーの非公開トピック閲覧 ✅
- スペースオーナーの非公開トピック閲覧 ✅
- ページ一覧の表示 ✅

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

Go 版へのトピック詳細画面移行に伴う Rails 版の不要コード削除が適切に行われています。

- 削除対象のファイルは作業計画書と一致しており、漏れはありません
- `config/routes.rb` の `topic` named route は他のコントローラー・コンポーネントで使用されているため正しく維持されています
- `global_hotkey_spec.rb` のトピック関連テスト削除は計画書に未記載ですが、Go 版が該当ページを処理するようになったため妥当な対応です
- Go 版のハンドラーテスト（`show_test.go`、9 テストケース）で削除された Rails テストのカバレッジが担保されています
- 唯一の指摘は `Dockerfile.dev` のコメント変更がタスクと無関係な点ですが、軽微な変更です
