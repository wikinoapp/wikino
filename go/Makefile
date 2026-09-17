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
	$(OP_RUN_DEV) go run ./cmd/wikino serve

.PHONY: seed
seed: ## 開発用データベースの中身をシードデータで作り直す (既存データは失われる)
	$(OP_RUN_DEV) go run ./cmd/wikino seed

# アカウントはatnameで指定する。これはアプリケーション自身が表示する識別子で
# あり、シードが作るアカウントに限らず開発用データベースにあるどのアカウントでも
# コードを生成できるようにするため。既定はサインインが2要素認証ステップを通る
# シードのアカウント。ほかのシードのアカウントはパスワードだけでサインインし、
# TOTP secretを持たない。
#
# 変数をコマンドの前に置くのにenvを使うのは、OP_RUN_DEVが `op run` とその実行対象
# コマンドを区切る `--` で終わっており、そこに代入を書くとコマンド自体として
# 読まれるため。
DEVTOTP_USER ?= seeduser2

.PHONY: devtotp
devtotp: ## 開発用アカウントの現在のTOTPコードを出力 (別アカウントはDEVTOTP_USER=<atname>)
	@$(OP_RUN_DEV) env WIKINO_DEVTOTP_ATNAME=$(DEVTOTP_USER) go run ./cmd/devtotp

.PHONY: browse-login
browse-login: ## devサイトへブラウザでログイン (別ユーザーはmake browse-login BROWSE_USER=collaborator)
	@$(OP_RUN_DEV) bash scripts/browse.sh login $(BROWSE_USER) || { echo "ログインに失敗しました。make browse-checkで環境を確認してください" >&2; exit 1; }

.PHONY: browse-check
browse-check: ## devサイトのブラウザ確認に必要な環境を検証 (別アカウントはBROWSE_USER=<役割>)
	$(OP_RUN_DEV) bash scripts/browse.sh check $(BROWSE_USER)

.PHONY: browse-shot
browse-shot: ## ログイン済みセッションでページを開きスクショ (例: make browse-shot URL=/)
	bash scripts/browse.sh shot "$(URL)"

.PHONY: browse-close
browse-close: ## ブラウザセッションを閉じ一時ファイルを削除
	bash scripts/browse.sh close

.PHONY: templ-generate
templ-generate: ## templファイルからGoコードを生成し、import文を整理
	@which templ > /dev/null || (echo "go.modのtemplをインストールしています..." && go install github.com/a-h/templ/cmd/templ)
	templ generate
	@$(MAKE) goimports

.PHONY: goimports
goimports: ## goimportsでimport文を整理
	@which goimports > /dev/null || (echo "go.modのgoimportsをインストールしています..." && go install golang.org/x/tools/cmd/goimports)
	goimports -w .

.PHONY: goimports-check
goimports-check: ## goimportsでimport文をチェック (差分があればエラー)
	@which goimports > /dev/null || (echo "go.modのgoimportsをインストールしています..." && go install golang.org/x/tools/cmd/goimports)
	@unformatted=$$(goimports -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "次のファイルのimport文が整理されていません:"; \
		echo "$$unformatted"; \
		echo ""; \
		echo "'make goimports' を実行してimport文を整理してください"; \
		exit 1; \
	fi

.PHONY: build
build: templ-generate ## バイナリをビルド (templ-generateがgoimportsも実行)
	go build -o bin/wikino ./cmd/wikino

.PHONY: test
test: db-setup-test ## テストを実行
	@which tparse > /dev/null || (echo "go.modのtparseをインストールしています..." && go install github.com/mfridman/tparse)
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./... | tparse -all'

.PHONY: test-pkg
test-pkg: db-setup-test ## 特定のパッケージのテストを実行 (使用例: make test-pkg PKG=internal/handler/sign_in)
	@which tparse > /dev/null || (echo "go.modのtparseをインストールしています..." && go install github.com/mfridman/tparse)
	@if [ -z "$(PKG)" ]; then \
		echo "エラー: PKG変数を指定してください"; \
		echo "使用例: make test-pkg PKG=internal/handler/sign_in"; \
		exit 1; \
	fi
	@echo "パッケージ$(PKG)のテストを実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./$(PKG) | tparse -all'

.PHONY: test-run
test-run: db-setup-test ## 特定のテストを実行 (使用例: make test-run PKG=internal/handler/sign_in RUN=TestCreate)
	@which tparse > /dev/null || (echo "go.modのtparseをインストールしています..." && go install github.com/mfridman/tparse)
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
	@echo "パッケージ$(PKG)のテスト$(RUN)を実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./$(PKG) -run $(RUN) | tparse -all'

.PHONY: test-verbose
test-verbose: db-setup-test ## 詳細ログ付きで全テストを実行 (デバッグ用)
	@which tparse > /dev/null || (echo "go.modのtparseをインストールしています..." && go install github.com/mfridman/tparse)
	@echo "ローカル環境: 詳細ログ付きで全テストを実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race -count=1 ./... | tparse -all'

.PHONY: test-js
test-js: ## フロントエンドのユニットテストを実行 (Vitest)
	pnpm test

.PHONY: typecheck-js
typecheck-js: ## フロントエンドのTypeScriptを型検査 (tsc --noEmit)
	pnpm typecheck

.PHONY: fmt
fmt: ## コードをフォーマット
	go fmt ./...
	@$(MAKE) goimports

.PHONY: lint-golangci
lint-golangci: ## golangci-lintを実行
	@echo "🔍 golangci-lintを実行中..."
	@golangci-lint run --config=.golangci.yml ./...

.PHONY: lint-style
lint-style: ## 日本語テキストのスタイルをチェック (koryluslint)
	@echo "🔍 日本語テキストのスタイルをチェック中..."
	@go tool koryluslint comment

.PHONY: lint
lint: lint-golangci lint-style sqlc-diff ## リントを実行 (golangci-lint + 日本語テキストのスタイルチェック + sqlc同期チェック)

.PHONY: clean
clean: ## ビルド成果物をクリーン
	rm -rf bin/
	find . -name "*_templ.go" -type f -delete

# データベースマイグレーションコマンド

# ダンプのうち、それを取得した環境を記述している行を削除する。pg_dump 17.6
# 以降が出力するランダムな \restrict / \unrestrictの対と、サーバーおよびpg_dumpの
# ビルドを記した見出しが対象。これらを残すとdb/schema.sqlが開発コンテナとCIで
# 食い違い、自分でダンプを取ってGit管理下のファイルと比較するdb-check-driftが
# 成立しなくなる。
# https://github.com/amacneil/dbmate/issues/678
NORMALIZE_SCHEMA_DUMP = sed -i '/^\\restrict /d;/^\\unrestrict /d;/^-- Dumped /d' db/schema.sql

.PHONY: db-migrate
db-migrate: ## データベースマイグレーションを実行
	$(OP_RUN_DEV) dbmate up
	@$(NORMALIZE_SCHEMA_DUMP)

# 開発用データベースを、既存データを捨てずに初期化する。`dbmate create` は
# データベースが存在しなかったときだけ成功するため、スキーマのロードは新規環境の
# 初回だけ行われ、以降の実行は未適用のマイグレーションを進めるだけになる。既存の
# データベースを作り直したいときはdb-setup-devを使う。
#
# 初回のスキーマロードに失敗した場合は、その実行で作成したデータベースを削除して
# からエラーを返す。これにより、同じコマンドを再実行すれば最初から構築し直せる。
# 後片付けは `dbmate create` が成功したときだけ行うため、実行前から存在していた
# データベースを削除することはない。
#
# --no-dump-schemaを付けるのはセットアップがdb/schema.sqlを書き戻さないように
# するため。dbmateはダンプのたびにpg_dumpのバージョン見出しを書き換えるため、
# 付けないと通常のセットアップがGit管理下のファイルを変更した状態にしてしまう。
# スキーマファイルを更新するコマンドはdb-migrateのままとする。
.PHONY: db-prepare-dev
db-prepare-dev: ## 開発用データベースが無ければ構築し、未適用のマイグレーションを適用する
	@if $(OP_RUN_DEV) dbmate create > /dev/null 2>&1; then \
		if ! $(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'; then \
			$(OP_RUN_DEV) dbmate drop; \
			exit 1; \
		fi; \
	fi
	$(OP_RUN_DEV) dbmate --no-dump-schema up

# データベースをdb/schema.sqlから作り直す。users / spaces / pagesなどの
# Rails期のテーブルはRailsのマイグレーションが作ったものでdb/migrationsに
# 対応するファイルが無いため、空のデータベースに `dbmate up` しても再現できない。
# 全テーブルと適用済みバージョンの双方を持つのはdb/schema.sqlだけであり、
# これを流せば `dbmate up` は未適用ゼロになる。
#
# スキーマのリセットに `dbmate drop` ではなくDROP SCHEMAを使うのは、他から
# 接続が張られていると `dbmate drop` が失敗するため (GUIクライアントを開いた
# ままにしているだけで起きる)。IF EXISTSを付けるのは、dropとcreateの間で
# 中断した場合でも再実行で復帰できるようにするため。
#
# `dbmate create` はデータベースが既にあると失敗し、CIランナーにはdbmate自体が
# 無いため、エラーは握り潰して実際の失敗は後続のリセットで報告する。スキーマの
# 適用にpsqlを使うのも同じ理由で、CIではデータベースがpostgresサービス
# コンテナから与えられ、使えるのはpsqlだけになる。
#
# AUTHORIZATION pg_database_ownerはinitdbがpublicスキーマに与える所有者を
# 再現するためのもの。これを付けずに作り直すと接続ロールが所有者になり、
# pg_dumpがpublicスキーマを出力対象とみなすため、次回のダンプで
# db/schema.sqlに不要な差分が入る。
DB_RESET_SCHEMA_SQL = SET client_min_messages TO warning; DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public AUTHORIZATION pg_database_owner;

.PHONY: db-setup-dev
db-setup-dev: ## 開発用データベースをdb/schema.sqlから作り直す (既存データは失われる)
	@$(OP_RUN_DEV) sh -c 'dbmate create > /dev/null 2>&1 || true'
	@$(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$(DB_RESET_SCHEMA_SQL)" > /dev/null'
	@$(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'

.PHONY: db-setup-test
db-setup-test: ## テスト用データベースをdb/schema.sqlから作り直す (既存データは失われる)
	@$(OP_RUN_TEST) sh -c 'dbmate create > /dev/null 2>&1 || true'
	@$(OP_RUN_TEST) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$(DB_RESET_SCHEMA_SQL)" > /dev/null'
	@$(OP_RUN_TEST) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'

.PHONY: db-rollback
db-rollback: ## 最後のマイグレーションをロールバック
	$(OP_RUN_DEV) dbmate down

.PHONY: db-new
db-new: ## 新しいマイグレーションファイルを作成 (使用法: make db-new name=create_users)
	dbmate new $(name)

.PHONY: db-dump
db-dump: ## データベーススキーマをdb/schema.sqlにダンプ
	$(OP_RUN_DEV) dbmate dump
	@$(NORMALIZE_SCHEMA_DUMP)

# テスト用データベースをdb/schema.sqlから作り直し、そのファイルが適用済みと
# して記録していないマイグレーションを適用して、結果のダンプが元と食い違えば失敗
# する。マイグレーションのSQLを実行する場所は他に無く、どのデータベースも
# db/schema.sqlをロードして作るため、ダンプし戻していないマイグレーションは本番で
# 初めて実行されることになる。変化しなかったデータベースをダンプすることは、
# pg_dumpが出力しない形へ手で編集されたdb/schema.sqlの報告も兼ねる。
#
# ダンプはdb/schema.sqlを上書きするため、終了前にファイルを元へ戻す。検査に期待
# するのはドリフトの報告であって解消ではない。実際に更新するときはdb-migrateを
# 実行する。
.PHONY: db-check-drift
db-check-drift: db-setup-test ## db/schema.sqlがマイグレーションと同期しているかをチェック
	@echo "🔍 マイグレーションとdb/schema.sqlの同期をチェック中..."
	@committed=$$(mktemp); \
	cp db/schema.sql "$$committed"; \
	status=0; \
	$(OP_RUN_TEST) dbmate up > /dev/null || status=$$?; \
	if [ "$$status" -eq 0 ]; then \
		$(NORMALIZE_SCHEMA_DUMP); \
		diff -u --label "db/schema.sql" --label "db/schema.sql (dumped)" "$$committed" db/schema.sql || status=1; \
	fi; \
	cp "$$committed" db/schema.sql; \
	rm -f "$$committed"; \
	if [ "$$status" -ne 0 ]; then \
		echo "db/schema.sqlがマイグレーションとずれています。make db-migrateを実行して差分をコミットしてください" >&2; \
	fi; \
	exit "$$status"

.PHONY: sqlc-generate
sqlc-generate: ## SQLクエリからGoコードを生成
	go tool sqlc generate

# 生成コードをdb/queries/とdb/schema.sqlと比較し、ずれていれば失敗する。
# `sqlc diff` は差分を表示するだけなのでlintから呼んでも安全であり、コードを
# 生成し直すのはsqlc-generateの責務のままとする。
.PHONY: sqlc-diff
sqlc-diff: ## 生成コードがクエリ・スキーマと同期しているかをチェック
	@echo "🔍 sqlc生成コードの同期をチェック中..."
	@go tool sqlc diff

# E2Eテストコマンド

.PHONY: e2e
e2e: db-setup-test ## E2Eテストを実行 (使用例: make e2e)
	@pnpm build:js
	@bash scripts/run-e2e.sh ""

.PHONY: e2e-file
e2e-file: db-setup-test ## 特定のE2Eテストファイルを実行 (使用例: make e2e-file FILE=tests/tab-indent.spec.ts)
	@if [ -z "$(FILE)" ]; then \
		echo "エラー: FILE変数を指定してください"; \
		echo "使用例: make e2e-file FILE=tests/tab-indent.spec.ts"; \
		exit 1; \
	fi
	@pnpm build:js
	@bash scripts/run-e2e.sh "$(FILE)"
