# タスク完了時のチェックリスト

## 必須確認事項

### Rubyファイル編集後

1. `bin/standardrb` - コードフォーマット
2. `bin/erb_lint --lint-all` - ERBファイルのLint
3. `bin/srb tc` - 型チェック
4. `bin/rails sorbet:update` - 必要に応じて型定義更新
5. `bin/rails zeitwerk:check` - オートローディング確認
6. `bin/rspec path/to/xxx_spec.rb` - 関連テスト実行

### JavaScript/TypeScriptファイル編集後

1. `pnpm prettier . --write` - コードフォーマット
2. `pnpm eslint . --fix` - Lint実行・修正
3. `pnpm tsc` - 型チェック

### Markdownファイル編集後

1. `pnpm prettier . --write` - フォーマット

### 全体確認

- `bin/check` - 全ての検証を一括実行

## 完了報告前の確認

- テストが全て通ること
- コンパイルエラーがないこと
- Linterエラーがないこと
- 明らかなランタイムエラーがないこと

## リトライポリシー

- 問題発生時は自動で最大5回まで再試行
- 5回失敗した場合のみユーザーに報告
