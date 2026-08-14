.PHONY: test test-file setup console server lint lint-fix fmt zeitwerk sorbet sorbet-update db-migrate db-rollback db-setup db-setup-test help

# Selects which 1Password item .env.local resolves (wikino_env_$APP_ENV). It does
# not set RAILS_ENV, so overriding it leaves the application in development while
# pointing it at another item's credentials. Specs run through OP_RUN_TEST below
# instead, and db-migrate / db-rollback / db-setup reject any value but dev.
#
# [Ja] .env.local がどの 1Password 項目 (wikino_env_$APP_ENV) を解決するかを選ぶ。
# RAILS_ENV は設定しないため、上書きしてもアプリケーションは development のまま
# 別項目の接続情報を見にいくだけになる。spec は後述の OP_RUN_TEST 経由で実行し、
# db-migrate / db-rollback / db-setup は dev 以外の値を拒否する。
APP_ENV ?= dev

# 1Password CLI経由で環境変数を読み込んでコマンドを実行
OP_RUN := APP_ENV=$(APP_ENV) op run --env-file=".env.local" --
# テスト専用（常にtest環境）
OP_RUN_TEST := APP_ENV=test op run --env-file=".env.local" --

## テスト
test: db-setup-test ## Run all specs. [Ja] 全テストを実行
	$(OP_RUN_TEST) bin/rspec

test-file: db-setup-test ## Run a single spec file (e.g. make test-file FILE=spec/models/user_spec.rb). [Ja] 特定のテストファイルを実行 (例: make test-file FILE=spec/models/user_spec.rb)
	$(OP_RUN_TEST) bin/rspec $(FILE)

# bin/setup installs dependencies, clears logs and tempfiles, and restarts the
# app server. It opens no database connection, but its log:clear and tmp:clear
# tasks go through Rake, and loading the Rake tasks resolves
# config/database.yml. Without `op run` dotenv reads the unresolved op://
# reference from .env.local into DATABASE_URL and parsing that URI aborts the
# script, so the recipe stays wrapped. Database initialization itself lives in
# the setup target of the repository root Makefile, which drives both
# subprojects.
#
# [Ja] bin/setup が行うのは依存関係のインストール、ログ・一時ファイルの掃除、
# アプリケーションサーバーの再起動である。データベースへは接続しないが、
# `bin/rails log:clear tmp:clear` は Rake 経由で実行され、Rake タスクの読み込みで
# config/database.yml を解決する。`op run` を通さないと dotenv が .env.local の
# 未解決の op:// 参照を DATABASE_URL として読み込み、その URI の解析で
# スクリプトが中断するため、ラップしたままにする。データベースの初期化そのものは、
# 両サブプロジェクトをまとめて扱うリポジトリルートの Makefile の setup ターゲットにある。
setup: ## Install dependencies. [Ja] 依存関係をインストール
	$(OP_RUN) bin/setup

## 開発サーバー
console: ## Railsコンソールを起動
	$(OP_RUN) bin/rails console

server: ## Railsサーバーを起動
	$(OP_RUN) bin/rails server

## リント・フォーマット
lint: ## Rubyコードのリントを実行
	$(OP_RUN) bin/standardrb

lint-fix: ## Rubyコードのリントを実行して自動修正
	$(OP_RUN) bin/standardrb --fix

fmt: lint-fix ## lint-fixのエイリアス

## 型チェック
zeitwerk: ## Zeitwerk（オートロード）チェック
	$(OP_RUN) bin/rails zeitwerk:check

sorbet: ## Sorbet型チェック
	bin/srb tc

sorbet-update: ## Sorbet型定義を更新
	$(OP_RUN) bin/rails sorbet:update

## データベース
# Rails owns only the queue database, so both targets name it explicitly. The
# Go-side database targets apply the primary schema from go/db/schema.sql with
# psql, while dbmate manages pending primary migrations. Rails has no primary
# migrations left. Naming the database is also what a multiple database
# application requires: bare db:rollback aborts and tells the caller to pick one.
#
# Both targets refuse any APP_ENV other than dev, the same way db-setup does.
# config/database.yml declares a queue database for development and production
# only, and APP_ENV does not set RAILS_ENV, so APP_ENV=test would migrate the
# development environment against the credentials of the test 1Password item.
#
# [Ja] Rails が持つのは queue だけなので、両ターゲットとも queue を明示する。
# primary のスキーマは Go 側の DB ターゲットが go/db/schema.sql から psql で適用し、
# 未適用の primary マイグレーションは dbmate が管理する。Rails 側の primary
# マイグレーションはもう無い。データベースを明示するのは multiple database な
# アプリの要請でもある。素の db:rollback はどちらかを選ぶよう促して中断する。
#
# 両ターゲットは db-setup と同じく dev 以外の APP_ENV を拒否する。
# config/database.yml が queue データベースを定義しているのは development と
# production だけであり、APP_ENV は RAILS_ENV を設定しないため、APP_ENV=test では
# development 環境を test 項目の接続情報でマイグレーションすることになる。
db-migrate: ## Apply pending migrations to the queue database. [Ja] queue データベースのマイグレーションを実行
	@if [ "$(APP_ENV)" != "dev" ]; then \
		echo "エラー: db-migrate は APP_ENV=dev のみ対応しています"; \
		echo "queue データベースの定義があるのは development と production だけです"; \
		exit 1; \
	fi
	$(OP_RUN) bin/rails db:migrate:queue

db-rollback: ## Roll back the most recent queue migration (steps with STEP=n). [Ja] queue データベースの最後のマイグレーションをロールバック (STEP=n で段数指定)
	@if [ "$(APP_ENV)" != "dev" ]; then \
		echo "エラー: db-rollback は APP_ENV=dev のみ対応しています"; \
		echo "queue データベースの定義があるのは development と production だけです"; \
		exit 1; \
	fi
	$(OP_RUN) bin/rails db:rollback:queue

# Set up what Rails owns in the development environment. The primary database
# schema belongs to the Go side (db/schema.sql, applied from go/Makefile), so
# db:setup is never pointed at it: Rails keeps neither migrations nor a schema
# dump for the primary database. Only the queue database stays on Rails
# migrations, under db/queue_migrate.
#
# The target refuses any APP_ENV other than dev because both of its steps are
# development-only: config/database.yml declares no queue database for test, and
# the test database is built from the Go schema by `make -C ../go db-setup-test`
# without Rails taking part.
#
# db:environment:set fills in ar_internal_metadata on the primary database:
# db/schema.sql restores that table but none of its rows, and without them
# every Rails task that runs db:check_protected_environments aborts with
# NoEnvironmentInSchemaError.
#
# The queue database is migrated instead of loaded from db/queue_structure.sql
# because db:schema:load:queue invokes db:test:purge:queue, and the test
# environment in config/database.yml declares no queue database.
#
# [Ja] dev 環境で Rails が担当する分だけをセットアップする。primary のスキーマは
# Go 側 (db/schema.sql を go/Makefile から適用) のものなので、db:setup は primary へ
# 向けない。Rails は primary のマイグレーションもスキーマのダンプも持たない。
# Rails のマイグレーション (db/queue_migrate) で管理するのは queue だけである。
#
# dev 以外の APP_ENV を拒否するのは、2 つのステップがどちらも dev 専用のため。
# config/database.yml の test 環境には queue データベースの定義が無く、テスト用
# データベースは `make -C ../go db-setup-test` が Go 側のスキーマから構築して
# Rails は関与しない。
#
# db:environment:set は primary の ar_internal_metadata を埋める。db/schema.sql
# はこのテーブルを復元するが行は復元せず、行が無いと
# db:check_protected_environments を実行する Rails のタスクがすべて
# NoEnvironmentInSchemaError で中断する。
#
# queue を db/queue_structure.sql からロードせずマイグレーションするのは、
# db:schema:load:queue が db:test:purge:queue を呼ぶ一方、config/database.yml の
# test 環境に queue データベースの定義が無いため。
db-setup: ## Set up the dev databases Rails owns (the primary one belongs to the Go side). [Ja] dev で Rails が管理するデータベースをセットアップ (primary は Go 側が管理)
	@if [ "$(APP_ENV)" != "dev" ]; then \
		echo "エラー: db-setup は APP_ENV=dev のみ対応しています"; \
		echo "テスト用データベースは make -C ../go db-setup-test で構築してください"; \
		exit 1; \
	fi
	$(OP_RUN) bin/rails db:environment:set
	$(OP_RUN) bin/rails db:create:queue db:migrate:queue

# Rebuild the test database before running specs, by delegating to the Go side
# that owns it. The schema comes from go/db/schema.sql, and schema_dump: false
# in config/database.yml stops ActiveRecord::Migration.maintain_test_schema!
# from refreshing it, so without this nothing notices when the Go schema moves
# ahead of the local database and the specs keep running against a stale one.
#
# [Ja] spec の実行前に、テスト用データベースをそれを所有する Go 側へ委譲して
# 作り直す。スキーマは go/db/schema.sql が正本であり、config/database.yml の
# schema_dump: false によって ActiveRecord::Migration.maintain_test_schema! も
# 更新しなくなったため、これが無いと Go 側のスキーマが進んでもローカルの
# データベースが古いままであることに気づけず、古いスキーマのまま spec が走る。
db-setup-test: ## Rebuild the test database from the Go schema. [Ja] Go 側のスキーマからテスト用データベースを作り直す
	$(MAKE) -C ../go db-setup-test

## ヘルプ
help: ## このヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
