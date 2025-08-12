# コーディング規約チェックリスト

## 新しいコードを書く前に必ず確認

### Ruby全般

- [ ] 文字列はダブルクオート `"` を使用
- [ ] プライベートメソッドは `private def` で定義（**重要: 一般的なRubyと異なる**）
- [ ] 後置ifは使用しない
- [ ] ハッシュの省略記法を使用（`user: user` → `user:`）
- [ ] `T.must` ではなく `#not_nil!` を使用

### ActiveRecord

- [ ] `includes` は使用せず、`preload` または `eager_load` を明示的に使用（**重要**）
- [ ] マイグレーションのIDは `generate_ulid()` を使用

### RSpec

- [ ] `context` ブロックは使用しない
- [ ] `let`, `let!` は使用しない
- [ ] `described_class` は使用しない

### ファイル作成時

- [ ] マジックコメントを追加:
  ```ruby
  # typed: strict
  # frozen_string_literal: true
  ```
- [ ] 最終行に改行を入れる

## コードレビュー前の最終確認

1. このチェックリストの全項目を確認
2. `bin/srb tc` で型チェック
3. 関連するテストを実行
4. 1行90文字以内を確認
