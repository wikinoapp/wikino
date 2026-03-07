# コードレビュー: sidebar-sync-2-1

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-07                                      |
| 対象ブランチ               | sidebar-sync-2-1                                |
| ベースブランチ             | sidebar-sync                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/sidebar-sync.md（タスク2-1） |
| 変更ファイル数             | 3 ファイル                                      |
| 変更行数（実装）           | +24 / -0 行                                     |
| 変更行数（テスト）         | +123 / -0 行                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版コーディング規約
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド
- [@rails/docs/testing-guide.md](/workspace/rails/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `rails/app/repositories/draft_page_repository.rb`

### テストファイル

- [x] `rails/spec/repositories/draft_page_repository_spec.rb`

### 設定・その他

- [x] `docs/plans/1_doing/sidebar-sync.md`

## ファイルごとのレビュー結果

### `rails/spec/repositories/draft_page_repository_spec.rb`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@rails/CLAUDE.md#RSpec](/workspace/rails/CLAUDE.md) - FactoryBot変数の命名規則
- [@rails/docs/testing-guide.md](/workspace/rails/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **[@rails/CLAUDE.md#RSpec]**: FactoryBotで作成したレコードの変数名に `_record` サフィックスが付いていない箇所がある

  該当箇所（「非アクティブなスペースメンバーの下書きは含まれないこと」テスト）:

  ```ruby
  # 問題のあるコード
  active_member = FactoryBot.create(:space_member_record, ...)
  inactive_member = FactoryBot.create(:space_member_record, ...)
  active_topic = FactoryBot.create(:topic_record, ...)
  inactive_topic = FactoryBot.create(:topic_record, ...)
  active_page = FactoryBot.create(:page_record, ...)
  inactive_page = FactoryBot.create(:page_record, ...)
  ```

  該当箇所（「他のユーザーの下書きは含まれないこと」テスト）:

  ```ruby
  # 問題のあるコード
  my_page = FactoryBot.create(:page_record, ...)
  other_page = FactoryBot.create(:page_record, ...)
  ```

  **修正案**:

  ```ruby
  # 「非アクティブなスペースメンバーの下書きは含まれないこと」
  active_member_record = FactoryBot.create(:space_member_record, ...)
  inactive_member_record = FactoryBot.create(:space_member_record, ...)
  active_topic_record = FactoryBot.create(:topic_record, ...)
  inactive_topic_record = FactoryBot.create(:topic_record, ...)
  active_page_record = FactoryBot.create(:page_record, ...)
  inactive_page_record = FactoryBot.create(:page_record, ...)

  # 「他のユーザーの下書きは含まれないこと」
  my_page_record = FactoryBot.create(:page_record, ...)
  other_page_record = FactoryBot.create(:page_record, ...)
  ```

  **対応方針**:
  - [x] 修正案の通り `_record` サフィックスを付ける
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書のタスク2-1の要件をすべて確認:

- [x] `find_for_sidebar(user_record:, limit:)` メソッドを追加 → 実装済み
- [x] `DraftPageRecord` を `space_record`, `page_record`, `topic_record` で preload → `preload(:space_record, page_record: [:space_record, topic_record: :space_record])` で実装済み
- [x] `modified_at DESC` でソート → `.order(modified_at: :desc)` で実装済み
- [x] テストを作成 → 6つのテストケースで正常系・異常系を網羅

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

作業計画書のタスク2-1の要件を正確に実装しており、設計との乖離はない。実装コードは既存のRepositoryパターン（`preload`の使用、`to_model`の呼び出し、Sorbet型注釈）と一貫しており、品質が高い。テストも正常系（ソート順、件数制限）と異常系（非アクティブメンバー、他ユーザーの下書き、空結果）を網羅している。

唯一の指摘は、テスト内のFactoryBot変数名に `_record` サフィックスが欠けている箇所がある点（コーディング規約違反）。軽微な命名規則の問題であり、修正は任意。
