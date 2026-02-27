# フォーマッタ 仕様書

<!--
このテンプレートの使い方:
1. 操作対象のモデルに対応するディレクトリを `docs/specs/` 配下に作成（例: `docs/specs/page/`）
2. このファイルをそのディレクトリにコピー（例: cp docs/specs/template.md docs/specs/page/create.md）
3. [機能名] などのプレースホルダーを実際の内容に置き換え
4. 各セクションのガイドラインに従って記述
5. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**ファイルの配置ルール**:
- 仕様書は操作対象のモデル（名詞）ごとにディレクトリを分け、機能（動詞）をファイル名にする
  - 例: `docs/specs/user/sign-up.md`、`docs/specs/page/create.md`
- モデルに分類しにくい横断的な機能は、その機能自体を名詞としてディレクトリにする
  - 例: `docs/specs/search/full-text.md`
- モデルの定義・状態遷移・他モデルとの関係を記述する場合は `overview.md` を作成する
  - `overview.md` はモデルの静的な性質（「これは何か」）を書く場所
  - 操作に紐づく仕様（バリデーション、権限など）は各機能の仕様書に書く
- 詳細は [@docs/README.md](/workspace/docs/README.md) を参照

**仕様書の性質**:
- 仕様書は「現在のシステムの状態」を記述するドキュメントです
- 実装が完了したら、仕様書を最新の状態に更新してください
- 過去の状態はGit履歴で参照できるため、仕様書には常に現在の状態のみを記述します

**作業計画書との関係**:
- 新しい機能の場合: `docs/plans/` の作業計画書に概要・要件・設計を記述し、タスク完了後にこの仕様書を作成します
- 既存機能の変更の場合: `docs/plans/` の作業計画書に変更内容を記述し、タスク完了後にこの仕様書を更新します

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 概要

<!--
ガイドライン:
- この機能が現在「どのように動いているか」を簡潔に説明
- なぜこの仕組みになっているかの背景も記述
- 2-3段落程度で簡潔に
-->

プロジェクト全体のコードフォーマッタとして [Oxfmt](https://oxc.rs/docs/guide/usage/formatter) を使用している。Oxfmt はプロジェクトルートにインストールされ、JavaScript, TypeScript, CSS, YAML, Markdown, TOML, JSON ファイルを統一的にフォーマットする。

**目的**:

- プロジェクト全体で一貫したコードフォーマットを維持する
- Go プロジェクト、Rails プロジェクト、ルートレベルのドキュメントを含む全ファイルを対象とする

**背景**:

- Oxfmt は Rust 製の高速フォーマッタで、Prettier との互換性が高い
- Tailwind CSS クラスソートをプラグインなしでネイティブサポートしている
- JavaScript, TypeScript, CSS, YAML, Markdown, TOML, JSON をプラグインなしでサポートしている

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

- `pnpm fmt`（または `make fmt`）を実行して、プロジェクト全体の対象ファイルをフォーマットできる
- `pnpm fmt:check`（または `make fmt-check`）でフォーマットチェックを実行できる
- フォーマット対象: JavaScript, TypeScript, CSS, YAML, Markdown, TOML, JSON ファイル
- フォーマット設定: printWidth: 120, tabWidth: 2, ダブルクォート, トレイリングカンマあり
- `go/**` 配下のファイルでは Tailwind CSS クラスソートが有効
- CI（GitHub Actions）で push 時にフォーマットチェックが自動実行される

## 設計

<!--
ガイドライン:
- 現在の技術的な実装の詳細を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - 技術スタック（使用するライブラリ、フレームワーク、ツールなど）
  - アーキテクチャ（システム全体の構成、コンポーネント間の関係など）
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - セキュリティ設計（認証・認可、トークン管理、Rate Limitingなど）
  - コード設計（パッケージ構成、主要な構造体、インターフェースなど）
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### ディレクトリ構成

```
/workspace/
├── package.json          # oxfmt の依存関係、scripts（fmt / fmt:check）
├── pnpm-lock.yaml        # pnpm のロックファイル
├── Makefile              # make fmt / make fmt-check
├── .oxfmtrc.json         # Oxfmt 設定ファイル
├── mise.toml             # Node.js / pnpm のバージョン管理
└── .github/
    └── workflows/
        └── fmt-ci.yml    # フォーマットチェック CI
```

### 設定ファイル（`.oxfmtrc.json`）

```json
{
  "$schema": "./node_modules/oxfmt/configuration_schema.json",
  "printWidth": 120,
  "semi": true,
  "singleQuote": false,
  "tabWidth": 2,
  "trailingComma": "all",
  "sortPackageJson": false,
  "overrides": [
    {
      "files": ["go/**"],
      "options": {
        "sortTailwindcss": {}
      }
    }
  ],
  "ignorePatterns": [
    "**/node_modules/",
    "**/vendor/",
    "rails/db/",
    "rails/app/assets/builds/",
    "rails/sorbet/",
    "go/internal/query/*.go",
    "go/internal/templates/*_templ.go",
    "go/static/"
  ]
}
```

### 実行方法

| コマンド         | 説明                             |
| ---------------- | -------------------------------- |
| `pnpm fmt`       | プロジェクト全体をフォーマット   |
| `pnpm fmt:check` | フォーマットチェック（差分検出） |
| `make fmt`       | `pnpm fmt` のラッパー            |
| `make fmt-check` | `pnpm fmt:check` のラッパー      |

### CI ワークフロー

`fmt-ci.yml` でフォーマット対象のファイルが push されたときに自動実行される。対象拡張子: `.js`, `.ts`, `.tsx`, `.css`, `.json`, `.yaml`, `.yml`, `.md`, `.toml`

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### dprint の採用

Rust 製の高速フォーマッタ dprint も検討した。プラグイン方式で柔軟性が高いが、Oxfmt は Prettier との互換性が高く移行コマンドが用意されている点、Tailwind CSS クラスソートが内蔵されている点で Oxfmt を選定した。

### Rails ディレクトリのみで Oxfmt を使用する方針

Rails の `package.json` に oxfmt を追加する方針も検討したが、プロジェクト全体で統一的にフォーマットを適用するためルートレベルに配置する方針とした。

### ERB テンプレートの Tailwind クラスソートを Oxfmt で行う方針

Oxfmt は HTML の Tailwind クラスソートをネイティブサポートしているが、ERB テンプレートは標準的な HTML ではなく Ruby コードが混在するため、対応状況が不明。ERB のフォーマットは `erb_lint` が担当しているため、ERB を Oxfmt の対象外とした。

### `.oxfmtignore` ファイルを使用する方針

除外設定を `.oxfmtignore` ファイルで管理する方針も検討したが、設定ファイルの数を最小限に保つため `.oxfmtrc.json` 内の `ignorePatterns` で管理する方針とした。

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Oxfmt 公式ドキュメント - Usage](https://oxc.rs/docs/guide/usage/formatter)
- [Oxfmt Beta アナウンス](https://oxc.rs/blog/2026-02-24-oxfmt-beta)
