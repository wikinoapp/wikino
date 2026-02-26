# Rate Limiter へのアドバイザリロック導入 設計書

<!--
このテンプレートの使い方:
1. このファイルを `docs/designs/2_todo/` ディレクトリにコピー
   例: cp docs/designs/template.md docs/designs/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残しておくことを推奨

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 実装ガイドラインの参照

<!--
**重要**: 設計書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
設計書作成の段階でガイドラインに準拠していることを確認してください。
-->

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 概要

<!--
ガイドライン:
- この機能が「何を」実現するのかを簡潔に説明
- ユーザーにとっての価値や背景を記述
- 2-3段落程度で簡潔に
-->

現在の Rate Limiter 実装に PostgreSQL のアドバイザリロック（`pg_advisory_xact_lock`）を導入し、高負荷時の競合状態（race condition）を防止します。

現在の UPSERT ベースの実装は PostgreSQL の行レベルロックで競合を処理していますが、同一キーへの大量の同時リクエストがあった場合、カウントが正確に反映されない可能性があります。アドバイザリロックを導入することで、同一キーへのアクセスを直列化し、正確なカウントを保証します。

**目的**:

- 高負荷時の Rate Limiting の正確性を向上させる
- 競合状態によるカウントの不整合を防止する

**背景**:

- 現在の実装は [Neon の Rate Limiting ガイド](https://neon.com/guides/rate-limiting) と比較して、アドバイザリロックが導入されていない
- 通常の負荷では問題ないが、高負荷時に競合状態が発生する可能性がある
- YAGNI 原則に従い、実際に問題が発生した場合に導入を検討する

**関連仕様書**:

- [Go への移行 (新規ユーザー登録機能編)](./1_doing/go-sign-up.md) - Rate Limiter の初期実装（タスク 7-1）

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

<!--
「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
箇条書きで簡潔に
-->

- システムは Rate Limit チェック時にアドバイザリロックを取得し、同一キーへのアクセスを直列化する
- アドバイザリロックはトランザクションスコープで自動的に解放される
- 既存の API（`Check()`, `Allow()`）は変更しない

### 非機能要件

<!--
必要に応じて以下のような項目を追加してください：
- セキュリティ（認証、認可、暗号化、監査ログなど）
- パフォーマンス（応答時間、スループット、リソース使用量など）
- ユーザビリティ（UX）（使いやすさ、わかりやすさ、アクセシビリティなど）
- 可用性・信頼性（稼働率、障害時の挙動、エラーハンドリングなど）
- 保守性（テストのしやすさ、コードの読みやすさ、ドキュメントなど）

不要な場合はこのセクション全体を削除してください。
-->

#### パフォーマンス

- アドバイザリロックのオーバーヘッドは最小限に抑える
- 異なるキーへのリクエストは引き続き並行処理される

#### 可用性・信頼性

- ロック取得のタイムアウトを設定し、デッドロックを防止する
- ロック取得に失敗した場合は、フォールバックとして現在の動作（ロックなし）を維持するか、エラーを返すか検討する

## 設計

<!--
ガイドライン:
- 技術的な実装の詳細を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - 技術スタック（使用するライブラリ、フレームワーク、ツールなど）
  - アーキテクチャ（システム全体の構成、コンポーネント間の関係など）
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - セキュリティ設計（認証・認可、トークン管理、Rate Limitingなど）
  - コード設計（パッケージ構成、主要な構造体、インターフェースなど）
  - テスト戦略（単体テスト、統合テスト、E2Eテストの方針）
  - マイグレーション管理（データベースマイグレーションの方針）
  - 実装方針（特記事項、既存システムとの関係、制約など）

不要な場合はこのセクション全体を削除してください。
-->

### アドバイザリロックの仕組み

PostgreSQL のアドバイザリロックは、アプリケーションレベルで排他制御を行うための機能です。

| 関数                             | スコープ         | 解放タイミング                                        |
| -------------------------------- | ---------------- | ----------------------------------------------------- |
| `pg_advisory_lock(key)`          | セッション       | 明示的に `pg_advisory_unlock` を呼ぶまで              |
| `pg_advisory_xact_lock(key)`     | トランザクション | トランザクション終了時に自動解放                      |
| `pg_try_advisory_lock(key)`      | セッション       | ロック取得を試行、取得できなければ即座に false を返す |
| `pg_try_advisory_xact_lock(key)` | トランザクション | ロック取得を試行、取得できなければ即座に false を返す |

Rate Limiting では `pg_advisory_xact_lock` を使用します。トランザクション終了時に自動的にロックが解放されるため、ロック解放漏れのリスクがありません。

### SQL クエリの変更

現在の `IncrementRateLimit` クエリを修正し、アドバイザリロックを追加します。

**現在の実装**:

```sql
-- name: IncrementRateLimit :one
INSERT INTO rate_limits (key, window_start, count, created_at, updated_at)
VALUES ($1, $2, 1, NOW(), NOW())
ON CONFLICT (key, window_start)
DO UPDATE SET
    count = rate_limits.count + 1,
    updated_at = NOW()
RETURNING *;
```

**変更後の実装**:

```sql
-- name: IncrementRateLimitWithLock :one
-- アドバイザリロックを取得してからカウンターをインクリメントする
-- hashtext() でキーを整数に変換してロックIDとして使用
SELECT pg_advisory_xact_lock(hashtext(@key::text));

INSERT INTO rate_limits (key, window_start, count, created_at, updated_at)
VALUES (@key, @window_start, 1, NOW(), NOW())
ON CONFLICT (key, window_start)
DO UPDATE SET
    count = rate_limits.count + 1,
    updated_at = NOW()
RETURNING *;
```

**注意**: sqlc は複数のステートメントを 1 つのクエリとして扱えないため、以下のいずれかの方法を検討する必要があります：

1. **Go コード側でロック取得とインクリメントを分ける**
2. **PostgreSQL 関数を作成し、関数内でロック取得とインクリメントを行う**
3. **CTE（Common Table Expression）を使用して 1 つのクエリにまとめる**

#### 方法 1: Go コード側での実装

```go
func (l *Limiter) CheckWithLock(ctx context.Context, input CheckInput) (*CheckResult, error) {
    // トランザクション開始
    tx, err := l.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    q := l.q.WithTx(tx)

    // アドバイザリロックを取得
    _, err = q.AcquireAdvisoryLock(ctx, input.Key)
    if err != nil {
        return nil, err
    }

    // カウンターをインクリメント
    result, err := q.IncrementRateLimit(ctx, ...)
    if err != nil {
        return nil, err
    }

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    // 結果を返す
    return &CheckResult{...}, nil
}
```

#### 方法 2: PostgreSQL 関数を使用

```sql
CREATE OR REPLACE FUNCTION increment_rate_limit_with_lock(
    p_key TEXT,
    p_window_start TIMESTAMPTZ
) RETURNS rate_limits AS $$
DECLARE
    result rate_limits;
BEGIN
    -- アドバイザリロックを取得
    PERFORM pg_advisory_xact_lock(hashtext(p_key));

    -- カウンターをインクリメント
    INSERT INTO rate_limits (key, window_start, count, created_at, updated_at)
    VALUES (p_key, p_window_start, 1, NOW(), NOW())
    ON CONFLICT (key, window_start)
    DO UPDATE SET
        count = rate_limits.count + 1,
        updated_at = NOW()
    RETURNING * INTO result;

    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

### 推奨: 方法 1（Go コード側での実装）

- sqlc との親和性が高い
- テストが容易
- ロックの取得・解放が明示的でデバッグしやすい

### テスト戦略

- **競合状態のテスト**: 複数の goroutine から同時にリクエストを送り、カウントが正確に反映されることを確認
- **ロックタイムアウトのテスト**: ロック取得待ちが長時間続いた場合の動作を確認

## タスクリスト

<!--
ガイドライン:
- フェーズごとに段階的な実装計画を記述
- チェックボックスで進捗を管理
- **重要**: 1タスク = 1 Pull Request の粒度で作成してください
- **重要**: 各タスクには想定ファイル数と想定行数を明記してください（PRサイズの見積もりのため）
- 想定ファイル数は「実装」と「テスト」に分けて記載してください
- 想定行数も「実装」と「テスト」に分けて記載してください
- 依存関係を明確に
- Pull Requestのガイドラインは CLAUDE.md を参照（変更ファイル数20以下、変更行数300行以下）

タスク番号の付け方:
- 各タスクには階層的な番号を付与します（例: 1-1, 1-2, 2-1, 2-2）
- フォーマット: **フェーズ番号-タスク番号**: タスク名
- **フェーズ番号は半角英数字とハイフンのみで表記**してください（ブランチ名に使用するため）
  - 例: フェーズ 1, フェーズ 2, フェーズ 5a（フェーズ 5 と 6 の間に追加する場合）
  - NG: フェーズ 5.5（ドットは使用不可）
- タスクの前に別のタスクを追加する場合は、サブ番号を使用します
  - 例: タスク 2-1 の前にタスクを追加する場合 → 2-0
  - 例: タスク 2-0 の前にタスクを追加する場合 → 2-0-1
- この番号はブランチ名の一部として使用されます（例: feature-1-1, feature-2-0）

プラットフォームプレフィックス:
- Go版またはRails版の修正を行うタスクには、タスク名の先頭にプラットフォームを示すプレフィックスを付けてください
- フォーマット: **フェーズ番号-タスク番号**: [Go] タスク名 または **フェーズ番号-タスク番号**: [Rails] タスク名
- Go版とRails版の両方を修正する場合は、別々のタスクに分けてください
- 例:
  - `- [ ] **1-1**: [Go] マイグレーション作成`
  - `- [ ] **1-2**: [Rails] モデルへのコールバック追加`
-->

### フェーズ 1: アドバイザリロックの導入

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
Go版/Rails版の両方を修正する場合は別タスクに分けてください
-->

- [ ] **1-1**: [Go] アドバイザリロック取得用クエリの追加
  - `db/queries/rate_limits.sql` にロック取得クエリを追加
  - sqlc コード生成
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 20 行（実装 20 行 + テスト 0 行）

- [ ] **1-2**: [Go] Limiter へのアドバイザリロック機能の追加
  - `internal/ratelimit/limiter.go` に `CheckWithLock()` メソッドを追加
  - トランザクション管理の実装
  - 競合状態のテストを追加
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 150 行（実装 50 行 + テスト 100 行）

### フェーズ 2: ハンドラーへの適用

- [ ] **2-1**: [Go] メール確認コード送信ハンドラーでアドバイザリロック版を使用
  - `internal/handler/email_confirmation/create.go` の更新
  - `Check()` から `CheckWithLock()` への変更
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 50 行（実装 10 行 + テスト 40 行）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **ロック取得タイムアウト**: 現時点では無期限待機とし、問題が発生した場合に導入を検討
- **ロックなしへのフォールバック**: ロック取得に失敗した場合はエラーを返し、フォールバックは行わない
- **スキーマの変更（1キー1行方式への移行）**: 現在のスキーマを維持し、互換性を保つ

## 導入の判断基準

この設計書は「将来的に問題が発生した場合」に備えたものです。以下の状況が観測された場合に導入を検討してください：

1. **カウントの不整合**: Rate Limit を超えたリクエストが許可される、または許可されるべきリクエストが拒否される
2. **高負荷時のエラー増加**: 同一キーへの同時リクエストによるデッドロックや競合エラー
3. **監視ログでの競合検出**: PostgreSQL のログで競合に関する警告が増加

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Neon - Rate Limiting with Postgres](https://neon.com/guides/rate-limiting) - PostgreSQL を使った Rate Limiting の実装ガイド
- [PostgreSQL Advisory Locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS) - PostgreSQL 公式ドキュメント
- [Go への移行 (新規ユーザー登録機能編)](./1_doing/go-sign-up.md) - Rate Limiter の初期実装
