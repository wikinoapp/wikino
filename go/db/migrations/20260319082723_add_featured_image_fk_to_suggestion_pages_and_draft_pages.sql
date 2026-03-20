-- migrate:up
ALTER TABLE suggestion_pages ADD CONSTRAINT fk_suggestion_pages_featured_image_attachment_id FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id);
ALTER TABLE draft_pages ADD CONSTRAINT fk_draft_pages_featured_image_attachment_id FOREIGN KEY (featured_image_attachment_id) REFERENCES attachments(id);

-- migrate:down
ALTER TABLE draft_pages DROP CONSTRAINT fk_draft_pages_featured_image_attachment_id;
ALTER TABLE suggestion_pages DROP CONSTRAINT fk_suggestion_pages_featured_image_attachment_id;
