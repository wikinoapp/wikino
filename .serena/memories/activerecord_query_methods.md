# ActiveRecord クエリメソッドの使い分け

## preload と eager_load の使い分け

### 基本的な方針

- **`includes` は使用しない** - 挙動が予測しにくいため避ける
- 代わりに `preload` または `eager_load` を明示的に使用する

### preload を使う場合

- 関連データを別クエリで取得したい場合（N+1問題の解決）
- 関連テーブルの条件でフィルタリングしない場合
- パフォーマンスを重視する場合（別クエリの方が効率的なことが多い）

```ruby
# 良い例
PageAttachmentReferenceRecord
  .preload(page_record: :topic_record)
  .where(attachment_record:)
```

### eager_load を使う場合

- LEFT OUTER JOINで一度に取得したい場合
- 関連テーブルの条件でフィルタリングする場合
- JOINが必要な複雑なクエリの場合

```ruby
# 良い例（関連テーブルでフィルタリング）
PageRecord
  .eager_load(:topic_record)
  .where(topics: { visibility: 0 })
```

### includes を避ける理由

- Rails が自動的に `preload` か `eager_load` を選択するため、挙動が不明瞭
- 条件によって生成されるSQLが変わり、予期しない動作をすることがある
- 明示的に `preload` か `eager_load` を使うことで意図が明確になる

## その他のクエリメソッド

### joins

- 関連テーブルでフィルタリングだけしたい場合（関連データは取得しない）
- INNER JOINを使用

### left_joins

- LEFT OUTER JOINが必要な場合
- 関連がない場合も含めて取得したい場合

## 注意点

- N+1問題を防ぐため、関連データを使用する場合は必ず `preload` か `eager_load` を使用する
- パフォーマンスを考慮して適切なメソッドを選択する
- 基本的には `preload` を優先し、JOINが必要な場合のみ `eager_load` を使用する
