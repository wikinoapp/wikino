# E2EテストのCIハング修正 作業計画書

## 仕様書

- 該当なし（テストインフラの改善であり、機能仕様の変更ではない）

## 概要

Go版E2Eテスト（Playwright）がCI環境で10分のタイムアウトを超過してハングする問題を修正する。

ローカルで再現・調査した結果、以下の複数の原因が判明した：

1. **Turnstile検証がE2Eサインインをブロック**: `op run` が1PasswordからTurnstileシークレットキーを解決するため、E2Eテストでもbot検証が有効になり、サインインPOSTが422で失敗
2. **サインイン後のリダイレクトループ**: サインイン成功後、`/` → `/home` → Rails（プロキシ） → Railsがセッションを認識せずサインインにリダイレクトという循環が発生。`waitForURL`が完了しない
3. **フィーチャーフラグ未設定**: ページ編集URL（`/s/{space}/pages/{number}/edit`）は `go_page_edit` フィーチャーフラグが必要。テストユーザーにフラグが未設定のため、全リクエストがRailsにプロキシされる
4. **フィーチャーフラグのチェックが失敗**: フラグをDBに作成しても、Playwrightの `storageState` に `user_session_tokens` クッキーが保存されておらず、リバースプロキシの `isFeatureFlagEnabled` が常にfalseを返す
5. **テストデータの競合**: 並行テストで `topics.number`/`pages.number` のユニーク制約違反が発生
6. **プロセスリーク**: `go run` の子プロセスがポート4201を保持し続ける

## 要件

### 機能要件

- E2Eテストがローカル環境とCI環境の両方で安定して動作すること
- 認証セットアップが正常に完了し、テストユーザーのセッションが後続テストで使えること
- ページ編集URL（`/s/{space}/pages/{number}/edit`）がGo版で処理されること（Railsにプロキシされないこと）
- テストの並行実行時にデータ競合が発生しないこと

### 非機能要件

- CIの10分タイムアウト内にテスト全体が完了すること
- テスト実行後にプロセスやポートが残留しないこと

## 実装ガイドラインの参照

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 設計

### 原因と対策の一覧

| #   | 原因                                                         | 対策                                                                           | 状態 |
| --- | ------------------------------------------------------------ | ------------------------------------------------------------------------------ | ---- |
| 1   | Turnstile検証がE2Eをブロック                                 | `WIKINO_TURNSTILE_ENABLED` 環境変数でTurnstileの有効/無効を明示的に制御        | 完了 |
| 2   | サインイン後のリダイレクトループ                             | `waitForURL` → `waitForResponse` に変更。302レスポンスをキャプチャ             | 完了 |
| 3   | フィーチャーフラグ未設定                                     | `auth.setup.ts` で `createTestFeatureFlag(user.id, "go_page_edit")` を呼び出す | 完了 |
| 4   | `user_session_tokens` クッキーが `storageState` に含まれない | 原因1の修正（Turnstile無効化）でサインインが成功するようになり解消             | 完了 |
| 5   | テストデータの競合                                           | `queryWithRetry` + アトミックなサブクエリで `number` 採番を安全に              | 完了 |
| 6   | プロセスリーク                                               | `run-e2e.sh` の `cleanup` に `fuser -k` を追加                                 | 完了 |

### 原因 4 の詳細調査

**現象**: `auth.setup.ts` でサインインが成功（302レスポンス確認済み）し、`storageState` でクッキーを保存するが、保存されたファイルに `user_session_tokens` クッキーが含まれていない。

**保存されたクッキー** (`playwright/.auth/user.json`):

- `wikino_csrf_token` (domain: localhost) ← Go版が設定
- `wikino_flash` (domain: localhost) ← Go版が設定
- `_wikino_session` (domain: localhost) ← Rails版が設定（プロキシ経由）

**含まれないクッキー**:

- `user_session_tokens` ← Go版のセッションクッキー（**フィーチャーフラグ判定に必要**）

**フローの分析**:

1. POST /sign_in → Go版が処理 → `user_session_tokens` クッキー付きの302レスポンスを返す
2. ブラウザが302に従い GET `/` にリダイレクト
3. GET `/` → Go版のリバースプロキシがRailsに転送
4. Rails側がレスポンスを返す際に追加のクッキー (`_wikino_session`) を設定
5. `auth.setup.ts` で `storageState` を呼び出すが、`user_session_tokens` が含まれない

**考えられる原因**:

- **クッキードメインの不一致**: `WIKINO_COOKIE_DOMAIN` が1Passwordから解決されるが、その値が `localhost` でない場合、ブラウザは `localhost:4201` へのリクエストでクッキーを保存しない。ただし `wikino_csrf_token` と `wikino_flash` は同じ `CookieDomain` 設定を使用しているにもかかわらず `domain: localhost` で保存されているため、この仮説は矛盾する
- **`HttpOnly` 属性の影響**: `user_session_tokens` は `HttpOnly: true` だが、`storageState` は HttpOnly クッキーも保存するため、これは原因ではないはず
- **サインインハンドラーが `SetSessionCookie` を呼ぶ前にエラーが発生**: `create.go:95` で呼ばれているが、二要素認証チェック等で分岐している可能性がある
- **セッションクッキーの `Domain` 属性が空でない値に設定されている**: `CookieDomain` が空文字列なら Domain 属性は省略され、ブラウザはリクエスト先のホスト（localhost）をデフォルトにする。非空の値（例: `.example.dev`）だとブラウザがドメイン不一致で拒否する

**次の調査ステップ**:

1. E2Eサーバー起動時の `WIKINO_COOKIE_DOMAIN` の実際の値を確認する
2. サインインPOSTの302レスポンスヘッダーに含まれる `Set-Cookie` の内容を確認する
3. 必要に応じて `run-e2e.sh` で `WIKINO_COOKIE_DOMAIN=localhost` を上書きする

### 対策案

**案A: `run-e2e.sh` で `WIKINO_COOKIE_DOMAIN` を上書き** (推奨)

Turnstileキーと同様に、`op run` の後で空文字または `localhost` に上書きする。

```bash
APP_ENV=test_e2e op run --env-file=".env" -- env \
  WIKINO_TURNSTILE_SECRET_KEY= \
  WIKINO_TURNSTILE_SITE_KEY= \
  WIKINO_COOKIE_DOMAIN=localhost \
  go run cmd/server/main.go &
```

ただし `WIKINO_COOKIE_DOMAIN` が空の場合は必須環境変数のバリデーションでエラーになるため、`localhost` を設定する必要がある。

**案B: E2E環境では `isFeatureFlagEnabled` を常にtrueにする**

E2E環境でのみフィーチャーフラグチェックをバイパスする。しかし本番に近い環境でテストする意義が薄れるため非推奨。

**案C: `storageState` を使わず、各テストでサインインする**

認証セットアップパターンを廃止して各テストで個別にサインインする。テスト実行時間が大幅に増加するため非推奨。

## 採用しなかった方針

### `waitForURL` によるサインイン完了の検出

当初は `page.waitForURL((url) => !url.pathname.includes("/sign_in"))` でサインイン完了を検知していたが、サインイン後のリダイレクトチェーンがRailsプロキシ経由で循環するため、URLベースの検知は不可能だった。代わりに `page.waitForResponse()` でPOSTの302レスポンスを直接キャプチャする方式を採用した。

### テストデータの採番を `SELECT MAX(number) + 1` で行う方式

当初はSELECTとINSERTを別クエリで実行していたが、並行テストで競合が発生した。アトミックなサブクエリ (`INSERT ... VALUES ($1, (SELECT COALESCE(MAX(number), 0) + 1 FROM ... WHERE ...), ...)`) に変更し、さらに `queryWithRetry` でユニーク制約違反時のリトライを追加した。

## タスクリスト

### フェーズ 1: 完了済みの修正

- [x] **1-1**: [Go] `WIKINO_TURNSTILE_ENABLED` 環境変数の導入とプロセスクリーンアップ
  - `config.go` に `TurnstileEnabled` フィールドを追加（デフォルト: `true`、`false` で無効化）
  - `turnstile/client.go` で `enabled` パラメータによる制御に変更
  - `run-e2e.sh` で `WIKINO_TURNSTILE_ENABLED=false` を設定
  - `cleanup` 関数に `fuser -k "${E2E_PORT}/tcp"` を追加
  - **想定ファイル数**: 5ファイル（config.go, client.go, main.go, run-e2e.sh, .env.example）
  - **想定行数**: 約 15行

- [x] **1-2**: [Go] E2Eテストデータの競合修正
  - `queryWithRetry` 関数の追加（ユニーク制約違反時のリトライ）
  - `createTestTopic`/`createTestPage` でアトミックなサブクエリを使用
  - **想定ファイル数**: 1ファイル（実装 1）
  - **想定行数**: 約 30行（実装 30行）

- [x] **1-3**: [Go] 認証セットアップの改善
  - `waitForURL` → `waitForResponse` に変更
  - `createTestFeatureFlag` の追加と `auth.setup.ts` での呼び出し
  - `cleanupTestData` に `feature_flags` の削除を追加
  - **想定ファイル数**: 2ファイル（実装 2）
  - **想定行数**: 約 25行（実装 25行）

- [x] **1-4**: Dockerfile.devにデバッグツールを追加
  - `net-tools`（netstat）と `psmisc`（fuser）を追加
  - **想定ファイル数**: 1ファイル（実装 1）
  - **想定行数**: 約 2行（実装 2行）

### フェーズ 2: セッションクッキーの問題修正

- [x] **2-1**: [Go] `user_session_tokens` クッキーが `storageState` に保存されない問題の調査
  - 原因1の修正（`WIKINO_TURNSTILE_ENABLED=false`）によりサインインが成功するようになり、クッキーが正常にSet-Cookieされることを確認
  - `WIKINO_COOKIE_DOMAIN=localhost` は1Passwordで正しく設定済み
  - `Set-Cookie: user_session_tokens=...; Domain=localhost; HttpOnly; SameSite=Lax` が返ることを確認
  - 追加のコード変更は不要
  - **想定ファイル数**: 0ファイル
  - **想定行数**: 0行

### フェーズ 3: コミットとCI確認

- [ ] **3-1**: [Go] 全修正をコミットしCIで動作確認
  - フォーマットチェック（`make -C /workspace fmt`）
  - 全変更を適切な粒度でコミット
  - CIで10分以内にE2Eテストが完了することを確認
  - **想定ファイル数**: 約 4ファイル（既存の変更）
  - **想定行数**: 約 60行（既存の変更の合計）

### 実装しない機能（スコープ外）

- **E2E環境専用の `.env.test_e2e` ファイル**: 現在は1Passwordで管理されている `test_e2e` 環境変数を、E2E専用のenvファイルに分離すること。今回は `run-e2e.sh` での上書きで対応する
- **Rails版セッションとの統合テスト**: GoとRailsのセッション相互運用のE2Eテストは今回のスコープ外
