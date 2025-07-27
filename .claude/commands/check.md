---
description: "変更点のチェック"
argument-hint: "ベースブランチ名 (default: main)"
---

以下のコマンドを実行し、異常終了したらエラー内容をもとに修正してください。

```bash
bin/rails zeitwerk:check
bin/rails sorbet:update
pnpm prettier . --write
pnpm eslint . --fix
pnpm tsc
bin/erb_lint --lint-all
bin/srb tc
bin/standardrb
bin/rspec
```
