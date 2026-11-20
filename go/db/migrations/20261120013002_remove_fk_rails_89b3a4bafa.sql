-- migrate:up
SET LOCAL lock_timeout = '5s';

-- カバー画像の削除を拒む古い外部キーを外す。同じ列はfk_pages_featured_image_attachment_idが参照を保つ。
ALTER TABLE pages DROP CONSTRAINT fk_rails_89b3a4bafa;

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE pages ADD CONSTRAINT fk_rails_89b3a4bafa
    FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id) NOT VALID;
