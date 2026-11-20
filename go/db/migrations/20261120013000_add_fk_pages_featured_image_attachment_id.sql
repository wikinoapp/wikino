-- migrate:up
SET LOCAL lock_timeout = '5s';

-- 添付ファイルを削除すると、カバー画像にしていたページの列をNULLにする (draft_pages・suggestion_pagesと同じ形)。
-- 既存の行の検証は別のマイグレーションで行い、外部キーを足す間の書き込みを妨げない。
ALTER TABLE pages ADD CONSTRAINT fk_pages_featured_image_attachment_id
    FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id) ON DELETE SET NULL NOT VALID;

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE pages DROP CONSTRAINT fk_pages_featured_image_attachment_id;
