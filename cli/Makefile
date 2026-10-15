.PHONY: help
help: ## ヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# リリースのビルドではVERSIONを渡して `wikino --version` の値を埋め込む。
# 渡さない手元のビルドは "dev" と表示する (cmd/wikino/main.goのresolveVersionを参照)。
VERSION ?=
LDFLAGS = $(if $(VERSION),-X main.version=$(VERSION))

.PHONY: build
build: ## バイナリをビルド (使用例: make build VERSION=v1.0.0)
	go build -ldflags "$(LDFLAGS)" -o bin/wikino ./cmd/wikino

.PHONY: test
test: ## テストを実行
	go test -race ./...

# goimportsとkoryluslintは、CLIのモジュールに開発ツールの依存を足さないよう、
# サーバー (go/) のモジュールで固定したバージョンを使う。CLIのgo.modの依存は
# go installで利用者の手元に取得されるため、実行に要るものだけに絞る。
GO_TOOLS_MODULE = ../go
KORYLUSLINT ?= /tmp/koryluslint

.PHONY: goimports
goimports: ## goimportsでimport文を整理
	go -C $(GO_TOOLS_MODULE) run golang.org/x/tools/cmd/goimports -w $(CURDIR)

.PHONY: goimports-check
goimports-check: ## goimportsでimport文をチェック (差分があればエラー)
	@unformatted=$$(go -C $(GO_TOOLS_MODULE) run golang.org/x/tools/cmd/goimports -l $(CURDIR)) || exit $$?; \
	if [ -n "$$unformatted" ]; then \
		echo "次のファイルのimport文が整理されていません:"; \
		echo "$$unformatted"; \
		echo ""; \
		echo "'make goimports' を実行してimport文を整理してください"; \
		exit 1; \
	fi

.PHONY: fmt
fmt: ## コードをフォーマット
	go fmt ./...
	@$(MAKE) goimports

# 公開APIのOpenAPI記述 (サーバーのgo/api/openapi.yaml) を正本として、クライアントと型、
# operationの分類の表をinternal/apigenに生成する。生成物はコミットし、CIで再生成の
# 差分が無いことを確かめる。oapi-codegenはサーバーのモジュールで固定したバージョンを使う。
OPENAPI_SPEC = ../go/api/openapi.yaml
APIGEN_DIR = internal/apigen

# テンプレートの相対パスをCLIのルートから解決するため、固定版の実行ファイルを取得して
# このディレクトリで起動する。Goのtool依存は引き続きサーバーのモジュールで管理する。
.PHONY: api-generate
api-generate: ## OpenAPI記述からAPIクライアントを生成
	"$$(go -C $(GO_TOOLS_MODULE) tool -n oapi-codegen)" -config $(CURDIR)/oapi-codegen.yaml -o $(CURDIR)/$(APIGEN_DIR)/apigen.gen.go $(CURDIR)/$(OPENAPI_SPEC)
	go run ./$(APIGEN_DIR)/genoperations -spec $(OPENAPI_SPEC) -o $(APIGEN_DIR)/operations.gen.go

.PHONY: api-generate-check
api-generate-check: api-generate ## APIクライアントの生成物がOpenAPI記述と一致するかをチェック
	@if ! git diff --exit-code --quiet -- $(APIGEN_DIR) || [ -n "$$(git ls-files --others --exclude-standard -- $(APIGEN_DIR))" ]; then \
		git status --short -- $(APIGEN_DIR); \
		echo ""; \
		echo "APIクライアントの生成物がOpenAPI記述と一致しません"; \
		echo "'make api-generate' を実行して生成物をコミットしてください"; \
		exit 1; \
	fi

# スキル (SKILL.md) に載せるコマンドの一覧表を、cobraのコマンド定義から生成する。
# 生成物はコミットし、コマンド定義とのずれはrootパッケージのテストで検出する。
SKILL_DIR = internal/skill

.PHONY: skill-generate
skill-generate: ## スキルのコマンドの一覧表をコマンド定義から生成
	go run ./internal/cmd/root/gencommandtable -dir $(SKILL_DIR)

# 開発用のサーバーに対して、ビルドしたCLIのe2eテスト (e2e/) を実行する。
# サーバーの起動とシードの投入は済ませておく。接続先のデータベースとポートは、
# 手元ではサーバーと同じくgo/.envからop runで得て、CIではジョブの環境変数で渡す。
ifeq ($(CI),true)
  E2E_ENV =
else
  E2E_ENV = cd ../go && APP_ENV=dev op run --env-file=".env" --
endif

.PHONY: e2e
e2e: ## 開発用のサーバーに対してe2eテストを実行 (サーバーの起動とシードの投入が要る)
	$(E2E_ENV) sh -c 'WIKINO_E2E_HOST="$${WIKINO_E2E_HOST:-http://localhost:$$WIKINO_PORT}" bash $(CURDIR)/scripts/e2e.sh'

.PHONY: lint-golangci
lint-golangci: ## golangci-lintを実行
	golangci-lint run --config=.golangci.yml ./...

.PHONY: lint-style
lint-style: ## 日本語テキストのスタイルをチェック (koryluslint)
	@go -C $(GO_TOOLS_MODULE) build -o $(KORYLUSLINT) github.com/korylus/tools/cmd/koryluslint
	$(KORYLUSLINT) comment

.PHONY: lint
lint: lint-golangci lint-style ## リントを実行 (golangci-lint + 日本語テキストのスタイルチェック)

.PHONY: clean
clean: ## ビルド成果物をクリーン
	rm -rf bin/
