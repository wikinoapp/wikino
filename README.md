# Wikino

## 開発環境のセットアップ

```
git clone git@github.com:wikinoapp/wikino.git
cd wikino
docker compose up
docker compose exec app bin/setup
docker compose exec app bin/dev
docker compose exec app bin/rails server
```
