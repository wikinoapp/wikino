# 開発作業フローとメモリ活用

## 作業開始時のフロー

### 1. タスク理解フェーズ
1. タスクの要件を理解
2. 関連するメモリを確認:
   - `coding_standards_checklist` - 必ず確認
   - `project_specific_rules` - 一般的な慣習との違いを確認
   - タスクに応じて他のメモリも確認

### 2. 実装前の準備
1. 既存コードの調査
2. `coding_standards_checklist` を開いてチェックリストとして使用
3. 特に以下の点を意識:
   - プライベートメソッドは `private def`
   - `includes` ではなく `preload` / `eager_load`
   - `T.must` ではなく `not_nil!`

### 3. 実装フェーズ
1. チェックリストを見ながらコーディング
2. 新しいメソッドを書くたびに規約を確認
3. 特にプロジェクト固有のルールに注意

### 4. レビュー前の確認
1. `coding_standards_checklist` の全項目を再確認
2. 型チェック: `bin/srb tc`
3. テスト実行
4. コード整形確認

## メモリ使用の優先順位

1. **最優先で確認**
   - `coding_standards_checklist`
   - `project_specific_rules`

2. **タスクに応じて確認**
   - `activerecord_query_methods` - DB操作時
   - `service_class_guidelines` - サービスクラス作成時
   - `Rails クラス設計ガイドライン` - 新規クラス作成時

3. **参考として確認**
   - `project_overview`
   - `suggested_commands`

## よくある間違いを防ぐために

### コード作成前の自問
- [ ] プライベートメソッドの定義方法は確認したか？
- [ ] ActiveRecordのクエリメソッドは適切か？
- [ ] プロジェクト固有のルールを確認したか？
- [ ] 一般的なRuby/Railsの慣習に引きずられていないか？

### 危険信号
- 「一般的にはこう書く」と思ったら → `project_specific_rules` を確認
- 新しい種類のコードを書くとき → 関連メモリを必ず確認
- リファクタリング時 → 既存の規約を維持することを意識