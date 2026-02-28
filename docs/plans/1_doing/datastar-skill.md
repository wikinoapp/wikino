# Datastar スキル作成 作業計画書

<!--
このテンプレートの使い方:
1. このファイルを `docs/plans/2_todo/` ディレクトリにコピー
   例: cp docs/plans/template.md docs/plans/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**作業計画書の性質**:
- 作業計画書は「何をどう変えるか」という変更内容を記述するドキュメントです
- 新しい機能の場合は、概要・要件・設計もこのドキュメントに記述します
- 現在のシステムの状態は `docs/specs/` の仕様書に記述されています
- タスク完了後は、仕様書を新しい状態に更新してください（設計判断や採用しなかった方針も含める）

**仕様書との関係**:
- 新しい機能の場合: タスク完了後に `docs/specs/` に仕様書を作成する
- 既存機能の変更の場合: 「仕様書」セクションに対応する仕様書へのリンクを記載し、タスク完了後に仕様書を更新する

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 仕様書

<!--
- 既存機能を変更する場合: 変更対象の仕様書へのリンクを記載してください
- 新しい機能の場合: タスク完了後に作成予定の仕様書のパスを記載してください
-->

- 仕様書の作成は不要（開発ツールの追加であり、アプリケーション機能ではないため）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

Go 版 Wikino では、フロントエンドのインタラクションに [Datastar](https://data-star.dev/)（v1.0.0-RC.7）を採用している。Datastar はサーバードリブンな hypermedia フレームワークであり、HTML の `data-*` 属性と Server-Sent Events（SSE）を使って、JavaScript を最小限に抑えた宣言的な UI を構築できる。

現在の Datastar の利用はフォーム送信時のボタン無効化（`$isSubmitting` シグナル）に限定されている。ページ編集画面のリンク一覧更新（タスク 8b-1）など、本来 Datastar の SSE + フラグメント更新パターンで実装すべき機能が、plain JavaScript の `fetch` + `innerHTML` で実装されてしまっている。これは Datastar の知見が不足していることが原因である。

この問題を解決するため、以下の 2 つを行う:

1. **Datastar Go SDK の導入**: 公式の Go SDK（`github.com/starfederation/datastar-go`）を導入し、SSE エンドポイントの実装を標準化する。併せて Datastar JS を SDK と互換性のあるバージョンに更新する
2. **Datastar スキルの作成**: Datastar に特化した Claude Code スキルを個人プラグインリポジトリ（`shimbaco-skills`）に作成し、SDK の使い方、API リファレンス、ソースコード等を含め、Wikino プロジェクトからはプラグインとして利用する

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

- Datastar Go SDK を使って SSE エンドポイントを実装できる
- Claude Code が `shimbaco-skills:datastar` スキルを実行すると、Datastar の実装ガイドが読み込まれ、Datastar を使った機能の実装・リファクタリングを正確に行える
- スキルには以下の知識が含まれる:
  - Datastar の公式ドキュメント（ガイド、属性リファレンス、アクションリファレンス等）
  - SSE（Server-Sent Events）によるサーバードリブン UI 更新パターン
  - Datastar Go SDK を使った SSE エンドポイント実装パターン（`NewSSE`, `PatchElementTempl`, `MarshalAndPatchSignals` 等）
  - Datastar JS / Go SDK のソースコード（最終手段としてのリファレンス）
- スキルは引数として実装対象の機能説明を受け取り、Datastar を使った実装方針を提示できる
- Wikino プロジェクト固有の情報（ベンダーファイルのパス、templ テンプレートとの統合方法、既存の使用パターン）は、Wikino リポジトリ側の `CLAUDE.md` またはプロジェクトスキルに記載する

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

- **保守性**: Datastar のバージョンアップ時にスキルの内容を更新しやすい構成にする
- **正確性**: Datastar の公式ドキュメントに基づいた正確な API 情報を記載する（推測や不正確な情報を含めない）

## 実装ガイドラインの参照

<!--
**重要**: 作業計画書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
作業計画書作成の段階でガイドラインに準拠していることを確認してください。
-->

### Go 版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン（**ファイル名は標準の 9 種類のみ**）
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 設計

<!--
ガイドライン:
- 技術的な実装の設計を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - UI設計（画面構成、ユーザーフローなど）
  - セキュリティ設計（認証・認可、トークン管理など）
  - コード設計（パッケージ構成、主要な構造体など）

**重要: 設計は実装中に更新する**:
- 作業計画書内の設計は初期の方針であり、完璧ではない
- 実装中により良いアプローチが見つかった場合は、設計を積極的に更新する
- 設計に固執して実装の質を下げるよりも、実装で得た知見を設計に反映する方が重要
- 変更した場合は「採用しなかった方針」セクションに変更前の方針と変更理由を記録する
-->

### スキルの管理方針

Datastar スキルは **個人プラグインリポジトリ**（`shimbaco-skills`）で管理し、Wikino プロジェクトからはプラグインとして利用する。

#### なぜプラグイン方式を採用するか

- **ライセンスの分離**: 外部ライブラリのソースコードを Wikino プロジェクトのリポジトリに含めるとライセンス上の懸念がある。プライベートリポジトリで管理することでリスクを低減する
- **再利用性**: Datastar は Wikino 固有の技術ではなく、他の個人プロジェクトでも使う可能性がある。プラグインとして分離することで複数プロジェクトで共有できる
- **外部リソースの一元管理**: 今後 Datastar と同様に外部リソースを伴うスキルを作成する場合も、同じリポジトリで管理できる

#### プラグインリポジトリの構成

```
github.com/shimbaco/shimbaco-skills/  (private)
├── .claude-plugin/
│   └── plugin.json    # プラグインマニフェスト（name, description, version 等）
└── skills/
    └── datastar/
        ├── SKILL.md           # メインのスキルファイル（概要、Go SDK 統合パターン、実装チェックリスト）
        ├── reference.md       # Datastar 公式ドキュメント（不要セクション除外済み）
        ├── datastar-src-1.0.0-RC.7/  # Datastar JS ソースコード（最終手段としてのリファレンス）
        │   ├── engine/        # コアエンジン（engine.ts, signals.ts, consts.ts, types.ts）
        │   ├── plugins/       # プラグイン（attributes/, actions/, watchers/）
        │   └── bundles/       # バンドル定義（datastar.ts 等）
        └── datastar-go-src-1.1.0/  # Datastar Go SDK ソースコード（最終手段としてのリファレンス）
            ├── sse.go         # SSE 初期化
            ├── elements.go    # PatchElements, RemoveElementByID
            ├── elements-sugar.go  # PatchElementTempl
            ├── signals.go     # ReadSignals, MarshalAndPatchSignals
            └── consts.go      # 定数定義
```

- スキルは `shimbaco-skills:datastar` の名前空間でアクセスされる
- `SKILL.md` にはフロントマター（`name`, `description`, `argument-hint`）、スキルの概要、Go SDK 統合パターン、実装チェックリストを記載する
- `reference.md` には Datastar 公式ドキュメント（`tmp/datastar-docs.md`）から不要なセクション（Pro 機能、Rocket、Go 以外の SDK）を除外した内容を配置する。`SKILL.md` から参照される
- `datastar-src-1.0.0-RC.7/` には Datastar JS のコアソースコード（`tmp/datastar-1.0.0-RC.7/library/src/`）を配置する。ドキュメントで不明な点がある場合の最終手段として、実装を直接確認できる（39 ファイル、約 4,044 行）。ディレクトリ名にバージョンを含めることで、他プロジェクトで異なるバージョンを使用する場合にも対応可能
- `datastar-go-src-1.1.0/` には Go SDK のソースコード（`tmp/datastar-go-1.1.0/datastar/`）を配置する。SDK の内部動作を確認する最終手段として使用（約 1,579 行）
- この分離により、`SKILL.md` を簡潔に保ちつつ、詳細な API 情報やソースコードは必要な場合にのみ読み込まれる
- `reference.md` のソース: `tmp/datastar-docs.md`（4,088 行）から以下を除外して約 2,700 行に圧縮:
  - Pro Attributes / Pro Actions（有料機能）
  - Rocket（Web Components フレームワーク）
  - Go 以外の SDK セクション（PHP, Python, Ruby, C#, Java 等 12 言語）

**重要**: `tmp/` 配下のファイルは一時的なものであり、いずれ削除される可能性がある。タスク 2-2 の実施時に、有用なファイルを `shimbaco-skills` リポジトリ配下にコピーすること

#### Wikino プロジェクト側の設定

**プラグインの導入方法**:

`shimbaco-skills` リポジトリをホスト側でクローンし、`docker-compose.override.yml`（非公開）でコンテナ内にマウントする:

```yaml
# docker-compose.override.yml（Git管理外）
services:
  app:
    volumes:
      - ~/Dev/src/github.com/shimbaco/shimbaco-skills:/workspace/shimbaco-skills
```

Claude Code 起動時に `--plugin-dir` オプションで指定する:

```bash
claude --plugin-dir ./shimbaco-skills
```

これにより `shimbaco-skills:datastar` の名前空間でスキルにアクセスできる。マーケットプレイスへの公開は不要。

**プロジェクト固有の情報**:

プロジェクト固有の情報（ベンダーファイルのパス、templ 統合パターン、既存の Datastar 使用箇所）は Wikino リポジトリ側で管理する。具体的な記載場所はタスク 2-2 の実施時に決定する（`go/CLAUDE.md` への追記、または `.claude/skills/datastar-wikino/SKILL.md` としてプロジェクトスキルを作成）。

### SKILL.md の内容構成

`SKILL.md`（`shimbaco-skills` リポジトリ内）には以下のセクションを含める:

#### 1. フロントマターとスキルの概要

- `name`: `datastar`
- `description`: Datastar を使った機能の実装・リファクタリングを支援する旨の説明（Claude が自動的にスキルを読み込むタイミングの判断に使用される）
- `argument-hint`: `[実装対象の機能説明]`
- 引数: `$ARGUMENTS` で実装対象の機能説明を受け取る

#### 2. Datastar リファレンス（`reference.md`）

Datastar 公式ドキュメント（`tmp/datastar-docs.md`）をベースに、不要なセクションを除外した包括的なリファレンスを `reference.md` として配置する。以下の内容が含まれる:

- **Guide**: Getting Started, Patching Elements, Reactive Signals, Data Attributes, Frontend Reactivity, Patching Signals, Datastar Expressions, SSE Events 等
- **属性リファレンス**: 全 `data-*` 属性の詳細仕様（`data-on`, `data-signals`, `data-bind`, `data-text`, `data-show`, `data-class`, `data-attr` 等）
- **アクションリファレンス**: `@get()`, `@post()`, `@patch()`, `@delete()` 等のバックエンドアクション
- **SSE イベント仕様**: `datastar-patch-elements`, `datastar-patch-signals` のフォーマット
- **ベストプラクティス**: The Tao of Datastar
- **セキュリティ**: エスケープ、CSP 等

#### 3. Datastar Go SDK を使った統合パターン

SDK を使った SSE エンドポイントの実装方法を具体的に記載する:

- **SSE の初期化**: `datastar.NewSSE(w, r)` でヘッダー設定・Flush を自動化
- **templ コンポーネントのフラグメント返却**: `sse.PatchElementTempl(component, opts...)` で templ コンポーネントを直接パッチ
- **シグナルの読み取りと更新**: `datastar.ReadSignals(r, &input)` と `sse.MarshalAndPatchSignals(data)` によるシグナル操作
- **要素の削除**: `sse.RemoveElementByID(id)` による DOM 要素の削除
- **CSRF 対策**: SSE リクエストでの CSRF トークンの扱い

#### 4. 実装手順のチェックリスト

Datastar を使った機能を実装する際の標準的な手順:

1. HTML テンプレートに `data-*` 属性を追加
2. 必要に応じて SSE エンドポイントをハンドラーに追加
3. ハンドラーで `datastar.NewSSE(w, r)` を使い、`sse.PatchElementTempl()` で templ コンポーネントを返す
4. CSRF トークンの受け渡し確認
5. 動作確認

### 既存スキルとの関係

- 既存スキル（`/commit`, `/review`, `/sync`）は Wikino リポジトリの `.claude/commands/` に配置されている
- 本スキルは `shimbaco-skills` プラグインリポジトリで管理し、`shimbaco-skills:datastar` としてアクセスする
- 既存スキルの `.claude/commands/` から `.claude/skills/` への移行は本タスクのスコープ外とする

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### SSE の手動実装（SDK を使わない方式）

`net/http` の標準機能のみで SSE レスポンスを手動実装する方式を当初検討していたが、以下の理由から Datastar Go SDK の採用に変更した:

- **実装コストの差**: 手動実装では SSE ヘッダー設定、フォーマット構築、Flush 制御、templ レンダリング → バッファ → 文字列変換を毎回書く必要がある（30〜50 行）のに対し、SDK では `NewSSE(w, r)` + `PatchElementTempl()` で完結する（3〜5 行）
- **SSE プロトコルの正確性**: SDK が SSE プロトコルの詳細（ヘッダー、改行、フラッシュタイミング等）を正しく処理するため、手動実装でのバグリスクを排除できる
- **SDK の軽量さ**: ソースコードを調査した結果、SDK は約 1,579 行と非常に小さく、直接依存も 4 個と最小限であった。templ への直接依存もない（インターフェースベース設計）
- **YAGNI ではなくなった**: SSE エンドポイントをこれから積極的に増やす方針であり、SDK が提供する `PatchElementTempl()`, `MarshalAndPatchSignals()`, `ReadSignals()` 等の機能は実際に使用する

### Wikino リポジトリの `.claude/skills/` への直接配置

Wikino リポジトリの `.claude/skills/datastar/` にスキルを配置する方式を検討した。しかし、以下の理由からプラグインリポジトリ方式を採用する:

- **ライセンスの懸念**: 外部ライブラリのソースコードを Wikino リポジトリ（パブリック）に含めると、ライセンス上の懸念がある。プライベートリポジトリで管理することでリスクを低減できる
- **再利用性**: Datastar スキルは Wikino 固有ではなく、他の個人プロジェクトでも利用可能。プラグインとして分離することで複数プロジェクトで共有できる
- **外部リソースの一元管理**: 今後同様に外部リソースを伴うスキルを作成する場合も、`shimbaco-skills` リポジトリで一元管理できる

### `.claude/commands/` への配置

既存スキル（`commit.md`, `review.md`, `sync.md`）と同様に `.claude/commands/datastar.md` として配置する方式も検討した。しかし、以下の理由から不採用:

- **サポートファイル**: API リファレンスやソースコードを別ファイルに分離できない
- **フロントマター**: `description` による自動読み込み制御ができない
- **推奨形式**: Claude Code の公式ドキュメントで `.claude/skills/` が推奨されている

既存スキルは `.claude/commands/` のまま引き続き動作するため、移行は本タスクのスコープ外とする。

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

### フェーズ 1: Datastar Go SDK の導入とスキルファイルの作成

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
Go版/Rails版の両方を修正する場合は別タスクに分けてください
-->

- [x] **1-1**: [Go] Datastar Go SDK と Datastar JS の互換性調査
  - Datastar Go SDK v1.1.0（`tmp/datastar-go-1.1.0/`）と Datastar JS v1.0.0-RC.7 の互換性を調査
  - **結果**: 同じ SSE プロトコル（イベントタイプ `datastar-patch-elements`/`datastar-patch-signals`、データ行フォーマット）を使用しており互換性あり
  - **JS の更新**: 不要（v1.0.0-RC.7 は公式ドキュメントでも参照されている現行バージョン）
  - **head.templ の更新**: 不要（JS バージョンが変わらないためパス変更なし）
  - **既存パターンの動作確認**: `$isSubmitting` パターンはクライアントサイドのみの機能であり、SDK 導入とは独立して動作する（ビルド・全テスト PASS を確認済み）
  - **SDK の go.mod への追加**: CI が `go mod tidy` の差分を検証するため、未使用の依存関係を追加できない。タスク 2-1 で SSE エンドポイント実装時に `go get` で追加する
  - **対象リポジトリ**: Wikino（`wikinoapp/wikino`）
  - **変更ファイル数**: 0 ファイル（調査のみ、コード変更なし）

- [x] **1-2**: `shimbaco-skills` リポジトリの作成と Datastar スキルの配置
  - Datastar 公式ドキュメント（`tmp/datastar-docs.md` に配置済み）と Go SDK のソースコードを調査し、スキルファイルの内容を作成
  - GitHub にプライベートリポジトリ `shimbaco-skills` を作成
  - `.claude-plugin/plugin.json` を作成（プラグインマニフェスト: `name: "shimbaco-skills"`, `version`, `description` 等）
  - `skills/datastar/` ディレクトリを作成し、以下のファイルを配置:
  - `SKILL.md`: フロントマター、スキル概要、Go SDK 統合パターン、実装チェックリスト
  - `reference.md`: `tmp/datastar-docs.md` から不要セクション（Pro 機能、Rocket、Go 以外の SDK）を除外した Datastar 公式リファレンス（約 2,700 行）
  - `datastar-src-1.0.0-RC.7/`: `tmp/datastar-1.0.0-RC.7/library/src/` からコピー（Datastar JS コアソース、39 ファイル）
  - `datastar-go-src-1.1.0/`: `tmp/datastar-go-1.1.0/datastar/` からコピー（Go SDK ソース）
  - `docker-compose.override.yml` で `shimbaco-skills` をコンテナ内にマウントし、`claude --plugin-dir ./shimbaco-skills` で起動して `shimbaco-skills:datastar` でアクセスできることを確認
  - Wikino リポジトリ側にプロジェクト固有の Datastar 情報を追記（`go/CLAUDE.md` またはプロジェクトスキル）
  - **対象リポジトリ**: `shimbaco-skills`（新規）+ Wikino（プロジェクト固有情報追記）
  - **想定ファイル数**: 約 50 ファイル（`plugin.json` 1 + `SKILL.md` 1 + `reference.md` 1 + `datastar-src-1.0.0-RC.7/` 39 + `datastar-go-src-1.1.0/` 約 10）
  - **想定行数**: `SKILL.md` 約 100〜150 行 + `reference.md` 約 2,700 行 + ソースコードはコピーのため実質新規記述なし

### フェーズ 2: スキルの検証

- [ ] **2-1**: [Go] リンク一覧表示の Datastar 化で検証
  - タスク 8b-1 で実装した plain JS（`fetch` + `innerHTML`）によるリンク一覧更新を、Datastar + Go SDK パターンにリファクタリング
  - ハンドラーで `datastar.NewSSE(w, r)` + `sse.PatchElementTempl()` を使用する形に変更
  - `shimbaco-skills:datastar` スキルを使って実装し、スキルの有用性を検証
  - 必要に応じてスキルの内容を修正・補完
  - **想定ファイル数**: 約 4 ファイル（実装 4 + テスト 0）
  - **想定行数**: 約 80 行（実装 80 行 + テスト 0 行）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **Datastar Pro 機能のリファレンス**: Pro Attributes / Pro Actions は有料機能のため除外
- **Rocket（Web Components フレームワーク）のリファレンス**: プロジェクトでは使用しないため除外
- **既存のフォーム送信パターンのリファクタリング**: `$isSubmitting` パターンは正しく動作しているため、変更しない
- **SDK の圧縮機能の有効化**: SDK は gzip/brotli 等の圧縮をサポートするが、現時点では不要（必要になった段階で有効化する）

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Datastar 公式サイト](https://data-star.dev/)
- [Datastar GitHub リポジトリ](https://github.com/starfederation/datastar)
- [Datastar Go SDK（datastar-go）](https://github.com/starfederation/datastar-go) - v1.1.0 のソースコードを `tmp/datastar-go-1.1.0/` に配置して調査済み
- Datastar 公式ドキュメント - `tmp/datastar-docs.md` に配置済み（`reference.md` のソースとして使用）
- [Claude Code スキルのドキュメント](https://code.claude.com/docs/ja/skills)
