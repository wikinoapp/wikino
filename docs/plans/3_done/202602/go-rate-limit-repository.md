# Rate Limiter の Repository 層追加 設計書

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

- [@go/CLAUDE.md](/wikino/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/wikino/go/docs/architecture-guide.md) - アーキテクチャガイド

## 概要

<!--
ガイドライン:
- この機能が「何を」実現するのかを簡潔に説明
- ユーザーにとっての価値や背景を記述
- 2-3段落程度で簡潔に
-->

`ratelimit.Limiter` が `internal/query` に直接依存している現在の実装を、`repository.RateLimitRepository` を経由する形に修正します。これにより、アーキテクチャガイドの「Query への依存は Repository のみ」というルールに完全準拠します。

**目的**:

- アーキテクチャガイドへの完全準拠
- データアクセスロジックの一貫性向上
- テスト容易性の向上

**背景**:

- 現在の `ratelimit.Limiter` は `internal/query.Queries` に直接依存している
- アーキテクチャガイドでは「Query への依存は Repository のみ」と規定されている
- Mewst では同様の修正を完了済み（参考実装として活用可能）

**関連実装**:

- [Mewst の RateLimitRepository 実装](https://github.com/mewstcom/mewst/go/internal/repository/rate_limit_repository.go) - 参考実装

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

- `RateLimitRepository` を作成し、Rate Limit 関連のデータアクセスを Repository に集約する
- `ratelimit.Limiter` が `RateLimitRepository` を経由してデータアクセスを行う
- 既存の API（`Check()`, `Allow()`, `CleanupOldRecords()`）は変更しない（後方互換性を維持）

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

#### 保守性

- 既存のテストがすべてパスすること
- Repository のテストを追加すること

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

### コード設計

#### RateLimitRepository の構造

```go
// internal/repository/rate_limit_repository.go
package repository

import (
    "context"
    "database/sql"
    "time"

    "github.com/wikinoapp/wikino/go/internal/query"
)

// RateLimitRepository はRate Limitのリポジトリ
type RateLimitRepository struct {
    queries *query.Queries
}

// NewRateLimitRepository はRateLimitRepositoryを生成する
func NewRateLimitRepository(db query.DBTX) *RateLimitRepository {
    return &RateLimitRepository{
        queries: query.New(db),
    }
}

// WithTx はトランザクションを設定したRateLimitRepositoryを返す
func (r *RateLimitRepository) WithTx(tx *sql.Tx) *RateLimitRepository {
    return &RateLimitRepository{
        queries: r.queries.WithTx(tx),
    }
}

// IncrementParams はRate Limitカウンターインクリメントのパラメータ
type IncrementParams struct {
    Key         string
    WindowStart time.Time
}

// IncrementResult はRate Limitカウンターインクリメントの結果
type IncrementResult struct {
    Count int32
}

// Increment はRate Limitカウンターをインクリメントする
func (r *RateLimitRepository) Increment(ctx context.Context, params IncrementParams) (*IncrementResult, error) {
    row, err := r.queries.IncrementRateLimit(ctx, query.IncrementRateLimitParams{
        Key:         params.Key,
        WindowStart: params.WindowStart,
    })
    if err != nil {
        return nil, err
    }

    return &IncrementResult{
        Count: row.Count,
    }, nil
}

// DeleteOldRecords は指定された時刻より古いRate Limitレコードを削除する
func (r *RateLimitRepository) DeleteOldRecords(ctx context.Context, cutoff time.Time) error {
    return r.queries.DeleteOldRateLimits(ctx, cutoff)
}
```

#### Limiter の変更

```go
// internal/ratelimit/limiter.go（変更後）
package ratelimit

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "time"

    "github.com/wikinoapp/wikino/go/internal/repository"
)

// Limiter はPostgreSQLベースのRate Limiter
type Limiter struct {
    repo *repository.RateLimitRepository
}

// NewLimiter は新しいLimiterを作成する
func NewLimiter(repo *repository.RateLimitRepository) *Limiter {
    return &Limiter{repo: repo}
}

// WithTx はトランザクションを使用する新しいLimiterを返す
func (l *Limiter) WithTx(tx *sql.Tx) *Limiter {
    return &Limiter{repo: l.repo.WithTx(tx)}
}

// Check はRate Limitをチェックし、カウンターをインクリメントする
func (l *Limiter) Check(ctx context.Context, input CheckInput) (*CheckResult, error) {
    // ... バリデーション ...

    // カウンターをインクリメント（Repository経由）
    result, err := l.repo.Increment(ctx, repository.IncrementParams{
        Key:         input.Key,
        WindowStart: windowStart,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to increment rate limit: %w", err)
    }

    // ... 結果を返す ...
}
```

### 変更が必要な箇所

1. **新規作成**: `internal/repository/rate_limit_repository.go`
2. **新規作成**: `internal/repository/rate_limit_repository_test.go`
3. **修正**: `internal/ratelimit/limiter.go`
4. **修正**: `internal/ratelimit/limiter_test.go`
5. **修正**: `ratelimit.Limiter` を使用しているハンドラー（呼び出し元の修正）

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

### フェーズ 1: Repository 層の追加

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
Go版/Rails版の両方を修正する場合は別タスクに分けてください
-->

- [x] **1-1**: [Go] RateLimitRepository の作成と Limiter の修正

  - `internal/repository/rate_limit_repository.go` を新規作成
  - `internal/repository/rate_limit_repository_test.go` を新規作成
  - `internal/ratelimit/limiter.go` を修正（Repository 経由に変更）
  - `internal/ratelimit/limiter_test.go` を修正（Repository を使用するように変更）
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 200 行（実装 70 行 + テスト 130 行）

### フェーズ 2: 呼び出し元の修正

- [x] **2-1**: [Go] Limiter 使用箇所の修正

  - `ratelimit.NewLimiter()` の呼び出し箇所を修正
  - Repository を生成して渡すように変更
  - **想定ファイル数**: 約 2-4 ファイル（実装のみ、呼び出し箇所の数による）
  - **想定行数**: 約 20-50 行（呼び出し箇所の数による）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **Model の追加**: Rate Limit は単純なカウンターであり、ドメインモデルとして抽象化する必要性が低いため、Repository の結果型（`IncrementResult`）のみを使用
- **アドバイザリロックの導入**: 別の設計書（[go-rate-limit-advisory-lock.md](./go-rate-limit-advisory-lock.md)）で対応

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Mewst の RateLimitRepository 実装](https://github.com/mewstcom/mewst) - 同様の修正を先行実装
- [@go/docs/architecture-guide.md](/wikino/go/docs/architecture-guide.md) - 「Query への依存は Repository のみ」ルール
