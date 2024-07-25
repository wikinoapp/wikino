# Nonoto

## 開発環境のセットアップ

```
git clone git@github.com:nonoto/nonoto.git
cd nonoto
docker compose up
docker compose exec app bin/setup
docker compose exec app bin/dev
docker compose exec app bin/rails server
```
