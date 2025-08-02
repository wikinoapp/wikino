# タスク完了時のチェックリスト

タスクを完了する前に、以下の手順を実行してください：

## 1. 包括的なチェック（推奨）

```bash
bin/check
```

このコマンドは以下を順番に実行します：

1. `bin/rails zeitwerk:check` - オートローディングの確認
2. `bin/rails sorbet:update` - Sorbet型定義の更新
3. `pnpm prettier . --write` - コード整形
4. `pnpm eslint . --fix` - ESLintによる修正
5. `pnpm tsc` - TypeScript型チェック
6. `bin/erb_lint --lint-all` - ERBの検証
7. `bin/standardrb` - Rubyコードの検証
8. `bin/srb tc` - Sorbet型チェック
9. `bin/rspec` - テストの実行

## 2. エラーが出た場合の対処

### Ruby関連

- Standard違反: `bin/standardrb` で詳細確認
- ERB Lint違反: `bin/erb_lint --lint-all` で確認
- Sorbet型エラー: `bin/srb tc` で確認、必要に応じて `bin/rails sorbet:update`

### JavaScript/TypeScript関連

- ESLintエラー: `pnpm eslint . --fix` で自動修正
- Prettierフォーマット: `pnpm prettier . --write` で自動修正
- TypeScriptエラー: `pnpm tsc` で詳細確認

### テスト失敗

- `bin/rspec` で失敗したテストを確認
- 特定のテストのみ実行: `bin/rspec spec/path/to/spec.rb`

## 3. 最終確認

- すべてのチェックがパスすることを確認
- 新規ファイルには適切なマジックコメントがあることを確認
- I18nの翻訳が日英両方更新されていることを確認
