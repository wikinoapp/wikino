-- migrate:up
SET LOCAL lock_timeout = '5s';

-- 列の追加とは別のトランザクションで検証し、検証中の書き込みを妨げない。
ALTER TABLE email_confirmations VALIDATE CONSTRAINT fk_email_confirmations_user_id;

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE email_confirmations DROP CONSTRAINT fk_email_confirmations_user_id;
ALTER TABLE email_confirmations ADD CONSTRAINT fk_email_confirmations_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
