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
lint: lint-golangci lint-comments ## リントを実行（golangci-lint + コメント併記チェック）

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

.PHONY: db-setup-test
db-setup-test: ## テスト用データベースのスキーマをセットアップ
	@if [ "$$CI" = "true" ]; then \
		echo "CI環境: テストDBをリセットしてスキーマを適用"; \
		psql -h localhost -U postgres -d wikino_test -c 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;' > /dev/null 2>&1; \
		psql -h localhost -U postgres -d wikino_test -f db/schema.sql > /dev/null; \
	else \
		echo "ローカル環境: テストDBをリセットしてスキーマを適用"; \
		psql -h postgresql -U postgres -d wikino_test -c 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;' > /dev/null 2>&1; \
		psql -h postgresql -U postgres -d wikino_test -f db/schema.sql > /dev/null; \
	fi

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
	sqlc generate

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
