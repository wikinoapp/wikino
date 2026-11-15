-- migrate:up
SET LOCAL lock_timeout = '5s';

-- メールアドレス変更を始めたユーザー。新規登録・パスワードリセットと既存の確認はNULL。
ALTER TABLE email_confirmations ADD COLUMN user_id UUID;
ALTER TABLE email_confirmations ADD CONSTRAINT fk_email_confirmations_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE email_confirmations DROP COLUMN user_id;
