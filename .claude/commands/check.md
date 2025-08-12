---
description: "変更点のチェック"
argument-hint: "ベースブランチ名 (default: main)"
---

以下のコマンドを実行し、異常終了したらエラー内容をもとに修正してください。

**注意:** StandardRBの既存のRSpec警告（RSpec/AnyInstance、RSpec/NoExpectationExample、RSpec/MessageChain）も異常終了として扱います。これらの警告が出た場合は、rubocop:disableコメントを使わずに、以下の方法で修正してください：
- RSpec/AnyInstance: スタブヘルパーメソッドを作成し、個別のインスタンスに対してスタブを設定
- RSpec/NoExpectationExample: 大きなテストを複数の小さなテストに分割
- RSpec/MessageChain: doubleを使用してメソッドチェーンを回避

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
