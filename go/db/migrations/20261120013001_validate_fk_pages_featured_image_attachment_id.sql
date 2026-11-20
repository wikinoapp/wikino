-- migrate:up
SET LOCAL lock_timeout = '5s';

-- 外部キーの追加とは別のトランザクションで検証し、検証中の書き込みを妨げない。
ALTER TABLE pages VALIDATE CONSTRAINT fk_pages_featured_image_attachment_id;

-- migrate:down
SET LOCAL lock_timeout = '5s';

-- 検証中の書き込みを妨げないよう、新しい外部キーのDROPで強いロックを取る前に検証する。
ALTER TABLE pages VALIDATE CONSTRAINT fk_rails_89b3a4bafa;

ALTER TABLE pages DROP CONSTRAINT fk_pages_featured_image_attachment_id;
ALTER TABLE pages ADD CONSTRAINT fk_pages_featured_image_attachment_id
    FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id) ON DELETE SET NULL NOT VALID;
