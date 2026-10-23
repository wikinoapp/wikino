-- migrate:up

-- Wikinoの公式CLIを、全スペース共通 (space_idがNULL) のpublicクライアントとして登録する。
-- client_idはCLIに埋め込んで配布するため固定値にし、Goの定数 (model.OfficialCLIClientID) と
-- 一致させる。リダイレクトはループバック (RFC 8252) で受け、ポート番号は認可要求のたびに決まる。
--
-- db/schema.sqlはデータを持たず、開発とテストのデータベースはそこから作る。同じ行を入れるため、
-- このup節はシードとテストからも実行する (go/db/official_oauth_application.goが埋め込む)
INSERT INTO oauth_applications (space_id, name, client_id, client_type, redirect_uris)
VALUES (NULL, 'Wikino CLI', 'wikino-cli', 1, ARRAY['http://127.0.0.1/callback', 'http://[::1]/callback']);

-- migrate:down

DELETE FROM oauth_applications WHERE client_id = 'wikino-cli' AND space_id IS NULL;
