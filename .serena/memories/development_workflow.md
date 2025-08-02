# 開発ワークフロー

## 環境セットアップ

1. `docker compose up` - Docker環境起動
2. `mise install` - 依存関係インストール
3. `bin/setup` - 初期セットアップ
4. `bin/dev` - 開発環境起動
5. `bin/rails server` - サーバー起動

## 開発フロー

### 1. 新機能開発時

- 適切なディレクトリにファイルを作成（app/の構造に従う）
- Rubyファイルには必ずマジックコメントを追加
- クラス間の依存関係ルールを守る

### 2. コード記述時の注意点

- Rubyは文字列をダブルクオートで
- 後置ifは使わない
- プライベートメソッドは `private def` で定義
- CSSはTailwind CSSを使用
- RSpecでは`context`や`let`を使わない

### 3. データベース変更時

- マイグレーションでIDは `generate_ulid()` を使用
- `bin/rails db:migrate` でマイグレーション実行

### 4. I18n対応

- 新しいテキストは必ず翻訳ファイルに追加
- 日本語（.ja.yml）と英語（.en.yml）の両方を更新
- 適切なファイルに分類（forms, messages, meta, nouns）

### 5. コミット前の確認

- `bin/check` を実行してすべてのチェックをパス
- エラーがある場合は個別コマンドで修正
- テストが通ることを確認

## デバッグ・トラブルシューティング

- Sorbetエラー: `bin/rails sorbet:update` で型定義更新
- オートローディングエラー: `bin/rails zeitwerk:check`
- フォーマットエラー: `pnpm prettier . --write`
- Lintエラー: 各種Lintコマンドで修正

## 特記事項

- ViewComponentパターンを採用
- Hotwireでインタラクティブな機能を実装
- CodeMirror 6でリッチなエディタ機能を提供
