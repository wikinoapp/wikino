.PHONY: dev
dev: ## 全サービスの開発サーバーを起動
	hivemind Procfile.dev

.PHONY: help
help: ## ヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# 両サブプロジェクトをセットアップする。データベースのフェーズを分けているのは
# 所有者が異なるため。primaryはgo/db/schema.sqlとdbmateが管理するマイグレーション
# に従い、Railsが持つのはqueueだけである。それぞれを所有するサブプロジェクトの
# Makefileから初期化することで、各自の1Password環境ファイルが読み込まれる。
#
# db-setup-devではなくdb-prepare-devを使うのは、依存関係を入れ直すためにsetupを
# 実行し直しても開発データが残るようにするため。データベースを作り直すのは
# `make -C go db-setup-dev` を明示的に実行したときだけとする。
.PHONY: setup
setup: ## Railsの依存関係をインストールし、両方のデータベースをセットアップ
	$(MAKE) -C rails setup
	$(MAKE) -C go db-prepare-dev
	$(MAKE) -C rails db-setup

.PHONY: fmt
fmt: ## コードをフォーマット (Oxfmt)
	pnpm fmt

.PHONY: fmt-check
fmt-check: ## フォーマットチェック (Oxfmt)
	pnpm fmt:check

# koryluslintはkorylus-toolsが提供するKorylus共通リンタ。その `md`
# サブコマンドはMarkdownの句点改行 (semantic line break) を検査する。バージョンは
# go/go.modのtoolディレクティブで固定し、公開モジュールgithub.com/korylus/tools
# から取得する。
# その固定版バイナリをビルドしてリポジトリルートから実行することで、ルート直下の
# Markdown (README・docsなど) を走査する。
#
# mdを `go tool koryluslint` で直接実行できないのは、Goモジュールがgo/にネスト
# している一方、走査対象のMarkdownはリポジトリルートにあるため。`go tool` は
# モジュールのディレクトリを起点に走査するためgo/を見てしまう。固定版バイナリを
# ビルドすれば任意の作業ディレクトリから検査を実行できる。
KORYLUSLINT ?= /tmp/koryluslint

.PHONY: koryluslint-build
koryluslint-build:
	@go -C go build -o $(KORYLUSLINT) github.com/korylus/tools/cmd/koryluslint

.PHONY: lint-md
lint-md: koryluslint-build ## Markdownの句点改行をチェック (変更行のみ)
	@$(KORYLUSLINT) md

.PHONY: lint-md-base
lint-md-base: koryluslint-build ## BASE refとの差分行でMarkdownの句点改行をチェック (例: BASE=origin/main)
	@$(KORYLUSLINT) md -base=$(BASE)

.PHONY: lint-md-fix
lint-md-fix: koryluslint-build ## Markdownの句点改行を自動修正 (変更ファイル)
	@$(KORYLUSLINT) md --write
