ifeq ($(CI),true)
  OP_RUN_DEV = APP_ENV=dev
  OP_RUN_TEST = APP_ENV=test
else
  OP_RUN_DEV = APP_ENV=dev op run --env-file=".env" --
  OP_RUN_TEST = APP_ENV=test op run --env-file=".env" --
endif

.PHONY: help
help: ## ヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## サーバーを起動
	$(OP_RUN_DEV) go run cmd/server/main.go

.PHONY: browse-login
browse-login: ## dev サイトへブラウザでログイン (別ユーザーは make browse-login BROWSE_USER=2)
	$(OP_RUN_DEV) bash scripts/browse.sh login $(BROWSE_USER)

.PHONY: browse-shot
browse-shot: ## ログイン済みセッションでページを開きスクショ (例: make browse-shot URL=/)
	bash scripts/browse.sh shot "$(URL)"

.PHONY: browse-close
browse-close: ## ブラウザセッションを閉じ一時ファイルを削除
	bash scripts/browse.sh close

.PHONY: templ-generate
templ-generate: ## templファイルからGoコードを生成し、import文を整理
	@which templ > /dev/null || (echo "Installing templ from go.mod..." && go install github.com/a-h/templ/cmd/templ)
	templ generate
	@$(MAKE) goimports

.PHONY: goimports
goimports: ## goimportsでimport文を整理
	@which goimports > /dev/null || (echo "Installing goimports from go.mod..." && go install golang.org/x/tools/cmd/goimports)
	goimports -w .

.PHONY: goimports-check
goimports-check: ## goimportsでimport文をチェック（差分があればエラー）
	@which goimports > /dev/null || (echo "Installing goimports from go.mod..." && go install golang.org/x/tools/cmd/goimports)
	@unformatted=$$(goimports -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files have incorrect imports:"; \
		echo "$$unformatted"; \
		echo ""; \
		echo "Please run 'make goimports' to fix imports"; \
		exit 1; \
	fi

.PHONY: build
build: templ-generate ## バイナリをビルド（templ-generateがgoimportsも実行）
	go build -o bin/server cmd/server/main.go

.PHONY: test
test: db-setup-test ## テストを実行
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./... | tparse -all'

.PHONY: test-pkg
test-pkg: db-setup-test ## 特定のパッケージのテストを実行（使用例: make test-pkg PKG=internal/handler/sign_in）
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@if [ -z "$(PKG)" ]; then \
		echo "エラー: PKG変数を指定してください"; \
		echo "使用例: make test-pkg PKG=internal/handler/sign_in"; \
		exit 1; \
	fi
	@echo "パッケージ $(PKG) のテストを実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./$(PKG) | tparse -all'

.PHONY: test-run
test-run: db-setup-test ## 特定のテストを実行（使用例: make test-run PKG=internal/handler/sign_in RUN=TestCreate）
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@if [ -z "$(PKG)" ]; then \
		echo "エラー: PKG変数を指定してください"; \
		echo "使用例: make test-run PKG=internal/handler/sign_in RUN=TestCreate"; \
		exit 1; \
	fi
	@if [ -z "$(RUN)" ]; then \
		echo "エラー: RUN変数を指定してください"; \
		echo "使用例: make test-run PKG=internal/handler/sign_in RUN=TestCreate"; \
		exit 1; \
	fi
	@echo "パッケージ $(PKG) のテスト $(RUN) を実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./$(PKG) -run $(RUN) | tparse -all'

.PHONY: test-verbose
test-verbose: db-setup-test ## 詳細ログ付きで全テストを実行（デバッグ用）
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@echo "ローカル環境: 詳細ログ付きで全テストを実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race -count=1 ./... | tparse -all'

.PHONY: test-js
test-js: ## Run frontend unit tests (Vitest). [Ja] フロントエンドのユニットテストを実行 (Vitest)
	pnpm test

.PHONY: typecheck-js
typecheck-js: ## Type-check frontend TypeScript (tsc --noEmit). [Ja] フロントエンドの TypeScript を型検査 (tsc --noEmit)
	pnpm typecheck

.PHONY: fmt
fmt: ## コードをフォーマット
	go fmt ./...
	@$(MAKE) goimports

.PHONY: lint-golangci
lint-golangci: ## golangci-lintを実行
	@echo "🔍 golangci-lintを実行中..."
	@golangci-lint run --config=.golangci.yml ./...

.PHONY: lint-comments
lint-comments: ## Check [Ja]-marker misuse in code comments (koryluslint). [Ja] コードコメントの [Ja] マーカー誤用をチェック (koryluslint)
	@echo "🔍 コメントの英日併記をチェック中..."
	@go tool koryluslint comment

.PHONY: lint
lint: lint-golangci lint-comments sqlc-diff ## Run all linters (golangci-lint + bilingual comment check + sqlc sync check). [Ja] リントを実行 (golangci-lint + コメント併記チェック + sqlc 同期チェック)

.PHONY: clean
clean: ## ビルド成果物をクリーン
	rm -rf bin/
	find . -name "*_templ.go" -type f -delete

# データベースマイグレーションコマンド

# Strip the random \restrict / \unrestrict lines that pg_dump >= 17.6 emits into
# db/schema.sql, which dbmate writes on both up and dump. Shared by db-migrate
# and db-dump so the workaround lives in one place.
# https://github.com/amacneil/dbmate/issues/678
#
# [Ja] dbmate が up / dump の双方で db/schema.sql へ書き込む、pg_dump 17.6 以降が
# 出力するランダムな \restrict / \unrestrict 行を削除する。db-migrate と db-dump で
# 共有し、回避策を 1 箇所にまとめる。
STRIP_SCHEMA_RESTRICT = sed -i '/^\\restrict /d;/^\\unrestrict /d' db/schema.sql

.PHONY: db-migrate
db-migrate: ## データベースマイグレーションを実行
	$(OP_RUN_DEV) dbmate up
	@$(STRIP_SCHEMA_RESTRICT)

# Initialize the development database without discarding data. `dbmate create`
# succeeds only when the database did not exist, so the schema is loaded exactly
# once, on a fresh environment, and every later run only applies the migrations
# that are still pending. Use db-setup-dev instead to rebuild an existing
# database.
#
# If the initial schema load fails, the database created by that invocation is
# dropped before the error is returned. This lets the same command start cleanly
# on retry. The cleanup runs only after `dbmate create` succeeds, so it can never
# remove a database that existed before the command started.
#
# --no-dump-schema keeps setup from writing db/schema.sql back: dbmate rewrites
# the pg_dump version header on every dump, so without the flag a routine setup
# leaves a tracked file modified. db-migrate remains the command that updates
# the schema file.
#
# [Ja] 開発用データベースを、既存データを捨てずに初期化する。`dbmate create` は
# データベースが存在しなかったときだけ成功するため、スキーマのロードは新規環境の
# 初回だけ行われ、以降の実行は未適用のマイグレーションを進めるだけになる。既存の
# データベースを作り直したいときは db-setup-dev を使う。
#
# 初回のスキーマロードに失敗した場合は、その実行で作成したデータベースを削除して
# からエラーを返す。これにより、同じコマンドを再実行すれば最初から構築し直せる。
# 後片付けは `dbmate create` が成功したときだけ行うため、実行前から存在していた
# データベースを削除することはない。
#
# --no-dump-schema を付けるのはセットアップが db/schema.sql を書き戻さないように
# するため。dbmate はダンプのたびに pg_dump のバージョン見出しを書き換えるため、
# 付けないと通常のセットアップが Git 管理下のファイルを変更した状態にしてしまう。
# スキーマファイルを更新するコマンドは db-migrate のままとする。
.PHONY: db-prepare-dev
db-prepare-dev: ## Initialize the development database if it does not exist, then apply pending migrations. [Ja] 開発用データベースが無ければ構築し、未適用のマイグレーションを適用する
	@if $(OP_RUN_DEV) dbmate create > /dev/null 2>&1; then \
		if ! $(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'; then \
			$(OP_RUN_DEV) dbmate drop; \
			exit 1; \
		fi; \
	fi
	$(OP_RUN_DEV) dbmate --no-dump-schema up

# Rebuild a database from db/schema.sql. The Rails-era tables (users, spaces,
# pages, ...) were created by Rails migrations and have no counterpart under
# db/migrations, so `dbmate up` alone cannot bootstrap an empty database. Only
# db/schema.sql carries the full schema together with every applied version,
# which also leaves `dbmate up` with nothing pending.
#
# The schema is reset with DROP SCHEMA rather than `dbmate drop`, which fails
# whenever anything else holds a connection to the database (a GUI client left
# open is enough). IF EXISTS keeps the reset re-runnable after a run that was
# interrupted between the drop and the create.
#
# `dbmate create` fails once the database exists, and the CI runners have no
# dbmate at all, so its error is ignored; the reset that follows is what reports
# a real failure. The schema is applied with psql for the same reason: on CI the
# database comes from the postgres service container, and only psql is present.
#
# AUTHORIZATION pg_database_owner reproduces the ownership initdb gives the
# public schema. Recreating it without that clause leaves it owned by the
# connecting role, which pg_dump then reports as a schema worth emitting, and
# db/schema.sql picks up a spurious diff on the next dump.
#
# [Ja] データベースを db/schema.sql から作り直す。users / spaces / pages などの
# Rails 期のテーブルは Rails のマイグレーションが作ったもので db/migrations に
# 対応するファイルが無いため、空のデータベースに `dbmate up` しても再現できない。
# 全テーブルと適用済みバージョンの双方を持つのは db/schema.sql だけであり、
# これを流せば `dbmate up` は未適用ゼロになる。
#
# スキーマのリセットに `dbmate drop` ではなく DROP SCHEMA を使うのは、他から
# 接続が張られていると `dbmate drop` が失敗するため (GUI クライアントを開いた
# ままにしているだけで起きる)。IF EXISTS を付けるのは、drop と create の間で
# 中断した場合でも再実行で復帰できるようにするため。
#
# `dbmate create` はデータベースが既にあると失敗し、CI ランナーには dbmate 自体が
# 無いため、エラーは握り潰して実際の失敗は後続のリセットで報告する。スキーマの
# 適用に psql を使うのも同じ理由で、CI ではデータベースが postgres サービス
# コンテナから与えられ、使えるのは psql だけになる。
#
# AUTHORIZATION pg_database_owner は initdb が public スキーマに与える所有者を
# 再現するためのもの。これを付けずに作り直すと接続ロールが所有者になり、
# pg_dump が public スキーマを出力対象とみなすため、次回のダンプで
# db/schema.sql に不要な差分が入る。
DB_RESET_SCHEMA_SQL = SET client_min_messages TO warning; DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public AUTHORIZATION pg_database_owner;

.PHONY: db-setup-dev
db-setup-dev: ## Recreate the development database from db/schema.sql (drops existing data). [Ja] 開発用データベースを db/schema.sql から作り直す (既存データは失われる)
	@$(OP_RUN_DEV) sh -c 'dbmate create > /dev/null 2>&1 || true'
	@$(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$(DB_RESET_SCHEMA_SQL)" > /dev/null'
	@$(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'

.PHONY: db-setup-test
db-setup-test: ## Recreate the test database from db/schema.sql (drops existing data). [Ja] テスト用データベースを db/schema.sql から作り直す (既存データは失われる)
	@$(OP_RUN_TEST) sh -c 'dbmate create > /dev/null 2>&1 || true'
	@$(OP_RUN_TEST) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$(DB_RESET_SCHEMA_SQL)" > /dev/null'
	@$(OP_RUN_TEST) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'

.PHONY: db-rollback
db-rollback: ## 最後のマイグレーションをロールバック
	$(OP_RUN_DEV) dbmate down

.PHONY: db-new
db-new: ## 新しいマイグレーションファイルを作成（使用法: make db-new name=create_users）
	dbmate new $(name)

.PHONY: db-dump
db-dump: ## データベーススキーマをdb/schema.sqlにダンプ
	$(OP_RUN_DEV) dbmate dump
	@$(STRIP_SCHEMA_RESTRICT)

.PHONY: sqlc-generate
sqlc-generate: ## SQLクエリからGoコードを生成
	go tool sqlc generate

# Compare the generated code with db/queries/ and db/schema.sql and fail when they
# have drifted apart. `sqlc diff` only prints the differences, so this stays safe to
# run from lint: regenerating the code remains the job of sqlc-generate.
#
# [Ja] 生成コードを db/queries/ と db/schema.sql と比較し、ずれていれば失敗する。
# `sqlc diff` は差分を表示するだけなので lint から呼んでも安全であり、コードを
# 生成し直すのは sqlc-generate の責務のままとする。
.PHONY: sqlc-diff
sqlc-diff: ## Check that the generated code is in sync with the queries and schema. [Ja] 生成コードがクエリ・スキーマと同期しているかをチェック
	@echo "🔍 sqlc 生成コードの同期をチェック中..."
	@go tool sqlc diff

# E2Eテストコマンド

.PHONY: e2e
e2e: db-setup-test ## E2Eテストを実行（使用例: make e2e）
	@pnpm build:js
	@bash scripts/run-e2e.sh ""

.PHONY: e2e-file
e2e-file: db-setup-test ## 特定のE2Eテストファイルを実行（使用例: make e2e-file FILE=tests/tab-indent.spec.ts）
	@if [ -z "$(FILE)" ]; then \
		echo "エラー: FILE変数を指定してください"; \
		echo "使用例: make e2e-file FILE=tests/tab-indent.spec.ts"; \
		exit 1; \
	fi
	@pnpm build:js
	@bash scripts/run-e2e.sh "$(FILE)"
