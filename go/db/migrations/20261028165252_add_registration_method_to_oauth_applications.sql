-- migrate:up
SET LOCAL lock_timeout = '5s';

-- クライアントの登録の方式。0は登録したクライアント (スペースのアプリ・公式クライアント)、
-- 1はClient ID Metadata Document (CIMD) のクライアントで、認可要求のときに `client_id` の
-- URLから取得した文書で作成・更新する。
-- metadata_expires_atはCIMDの文書のキャッシュの期限で、登録したクライアントではNULLになる。
-- CIMDのクライアントは事前に秘密を共有できないため、スペースに属さないpublicクライアントに限る。
ALTER TABLE oauth_applications
    ADD COLUMN registration_method INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN metadata_expires_at TIMESTAMP WITH TIME ZONE,
    ADD CONSTRAINT chk_oauth_applications_registration_method CHECK (registration_method IN (0, 1)),
    ADD CONSTRAINT chk_oauth_applications_client_id_metadata_document
        CHECK (registration_method = 0 OR (space_id IS NULL AND client_type = 1 AND client_secret_digest IS NULL));

-- migrate:down
SET LOCAL lock_timeout = '5s';

-- CIMDのクライアントの行を残すと、戻した後に公式クライアントとして扱われるため消す。
-- 許可とトークンは外部キーのON DELETE CASCADEで一緒に消える。
DELETE FROM oauth_applications WHERE registration_method = 1;
ALTER TABLE oauth_applications
    DROP COLUMN metadata_expires_at,
    DROP COLUMN registration_method;
