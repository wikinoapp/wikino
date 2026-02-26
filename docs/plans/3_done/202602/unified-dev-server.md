# 統合開発サーバー起動 作業計画書

## 仕様書

- 新規作成予定: なし（開発環境の改善のため、仕様書は不要）

## 概要

現在、ローカル開発環境でサーバーを立ち上げるには以下の 4 つのコマンドをそれぞれ別ターミナルで実行する必要がある：

```
cd /workspace/go && pnpm watch       # Go版フロントエンドアセットの監視・自動再ビルド
cd /workspace/go && air              # Go版サーバーのホットリロード
cd /workspace/rails && bin/dev       # Rails版フロントエンドアセットの監視（foreman経由）
cd /workspace/rails && make server   # Railsサーバーの起動
```

これを 1 コマンドで起動できるようにし、開発者体験を向上させる。

## 要件

### 機能要件

- ワークスペースルートで 1 コマンドを実行するだけで、4 つのプロセスすべてが起動する
- 各プロセスのログ出力にプレフィックスが付き、どのプロセスのログか判別できる
- Ctrl+C で全プロセスが確実に終了する
- 既存の個別起動方法（`air`, `make server` 等）はそのまま維持する（統合起動はあくまで追加手段）

## 実装ガイドラインの参照

開発環境のインフラ変更であり、Go 版・Rails 版のアプリケーションコードには変更を加えない。

## 設計

### 現状の構成

| プロセス             | コマンド                             | 役割                                           |
| -------------------- | ------------------------------------ | ---------------------------------------------- |
| Go フロントエンド    | `cd /workspace/go && pnpm watch`     | CSS/JS の監視・自動再ビルド                    |
| Go サーバー          | `cd /workspace/go && air`            | Go サーバーのホットリロード                    |
| Rails フロントエンド | `cd /workspace/rails && bin/dev`     | CSS/JS の監視（内部で foreman + Procfile.dev） |
| Rails サーバー       | `cd /workspace/rails && make server` | Rails サーバーの起動                           |

#### Rails `bin/dev` の内部動作

`bin/dev` は foreman を使って `rails/Procfile.dev` を実行している：

```
css: pnpm build:css --watch
js: pnpm build --watch
```

つまり `bin/dev` 自体が 2 プロセス（CSS ウォッチ + JS ウォッチ）を起動するプロセスマネージャーである。

### 方針: ルートレベルの Procfile.dev + hivemind

ワークスペースルートに `Procfile.dev` を作成し、[hivemind](https://github.com/DarthSim/hivemind) で全プロセスを統合管理する。

#### hivemind の選定理由

- **外部依存なし**: 単体の Go バイナリのみで動作（tmux 等の追加依存が不要）
- **Procfile 互換**: foreman / overmind と同じ Procfile 形式を使用
- **PTY ベースの出力**: カラー出力が保持され、ログの視認性が良い
- **Ruby に依存しない**: Rails 版廃止後もそのまま使い続けられる

#### Dockerfile.dev への hivemind インストール

dbmate と同じパターンで、バイナリを直接ダウンロードする：

```dockerfile
# --- hivemind ---
RUN curl -fsSL -L -o /tmp/hivemind.gz \
      https://github.com/DarthSim/hivemind/releases/download/v1.1.0/hivemind-v1.1.0-linux-amd64.gz && \
    gunzip /tmp/hivemind.gz && \
    mv /tmp/hivemind /usr/local/bin/hivemind && \
    chmod +x /usr/local/bin/hivemind
```

#### `/workspace/Procfile.dev`

```
go-assets: cd go && pnpm watch
go-server: cd go && air
rails-css: cd rails && pnpm build:css --watch
rails-js: cd rails && pnpm build --watch
rails-server: cd rails && PORT=3000 make server
```

**ポイント**:

- `rails/bin/dev`（foreman の入れ子）は使わず、`rails/Procfile.dev` の内容を直接展開する
  - プロセスマネージャーを入れ子にすると、シグナル伝播やログ管理が複雑になるため
- `rails-server` は `make server` 経由で起動する
  - `APP_ENV` や `op run` の引数は Makefile 側で一元管理されるため、Procfile.dev での重複を避ける
- `rails-server` に `PORT=3000` を明示的に指定する
  - hivemind は Heroku の Procfile 規約に準拠し、各プロセスに `PORT` 環境変数を自動設定する（ベースポート 5000 から 100 ずつ増加）
  - 5 番目のプロセスである `rails-server` には `PORT=5400` が暗黙的に設定されるため、Rails のデフォルトポート 3000 で上書きする必要がある

#### `/workspace/Makefile` への追加

```makefile
.PHONY: dev
dev: ## 全サービスの開発サーバーを起動
	hivemind Procfile.dev
```

### 起動方法

```sh
# ワークスペースルートで
make dev
```

## 採用しなかった方針

### foreman の採用

foreman は Ruby gem であり、以下の理由で採用しない：

- Ruby（Rails）への依存があり、Rails 版廃止後に不要になる
- 現状 `rails/bin/dev` 実行時に gem install されるが、事前にインストールされている保証がない
- Go 版への移行を進めている中で、Ruby 依存のツールを新たに正式採用するのは方向性に合わない

### overmind の採用

overmind は hivemind の上位互換（同じ作者が開発）で、個別プロセスの再起動や tmux 統合など高機能だが、以下の理由で採用しない：

- tmux への依存があり、Dockerfile.dev に tmux のインストールも追加する必要がある
- 現時点の要件（一括起動・停止）には hivemind で十分
- Procfile 形式は互換性があるため、必要になった場合は後から overmind に移行可能

### シェルスクリプト + バックグラウンドプロセス

`&` でバックグラウンド化する方法は以下の問題がある：

- Ctrl+C で子プロセスが残る可能性がある
- ログのプレフィックス付与が手動実装になる
- プロセス管理の再発明になる

### docker-compose への統合

プロセスを docker-compose のサービスとして定義する方法もあるが：

- 現在は単一の `app` コンテナで開発しており、サービスを分割するとコンテナ間通信の設定が必要になる
- 開発者はコンテナ内でコマンドを実行する運用のため、docker-compose レベルの変更は不適切

## タスクリスト

### フェーズ 1: hivemind のインストール

- [x] **1-1**: Dockerfile.dev に hivemind のインストールを追加
  - `Dockerfile.dev` に hivemind のバイナリダウンロード・配置を追加
  - Docker イメージの再ビルドが必要（ホスト側で `docker compose build app --no-cache`）
  - **想定ファイル数**: 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 5 行（実装 5 行 + テスト 0 行）

### フェーズ 2: 統合開発サーバーの構築

- [x] **2-1**: ルートレベルの Procfile.dev と Makefile ターゲットの追加
  - `/workspace/Procfile.dev` を作成（5 プロセスを定義）
  - `/workspace/Makefile` に `dev` ターゲットを追加
  - 動作確認（`make dev` で全プロセスが起動・終了できること）
  - **想定ファイル数**: 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 10 行（実装 10 行 + テスト 0 行）

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **個別プロセスの再起動機能**: overmind 等への移行時に検討
- **プロセスのヘルスチェック**: 現時点では不要（クラッシュはログで確認可能）
- **環境ごとの Procfile 分離**: 開発環境のみの用途のため不要
