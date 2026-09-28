-- migrate:up

-- 公開APIの個人アクセストークン。トークンは1つのスペースに束縛し、持ち主はユーザーでは
-- なくスペースメンバーで持つ。有効性をメンバーシップとメンバー権限で判定するためである。
--
-- トークンの値そのものは保存せず、SHA-256のダイジェストだけを持つ。一覧で見分けられる
-- ように、値の末尾の数文字だけをtoken_last_charsに残す。scopesはspace_members.scopesと
-- 同じ形で持つ。
--
-- 持ち主を束縛先スペースのメンバーに限定するため、space_member_idはspace_idとの複合外部キーに
-- する。親のスペースやメンバーが消えたら一緒に消えるべきデータなので、外部キーはどちらも
-- ON DELETE CASCADEにする。

-- space_members.idは単独でも一意だが、複合外部キーの参照先にするため組み合わせにも一意制約を置く
ALTER TABLE space_members
    ADD CONSTRAINT space_members_id_space_id_key UNIQUE (id, space_id);

CREATE TABLE personal_access_tokens (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    space_member_id UUID NOT NULL,
    name VARCHAR NOT NULL,
    token_digest VARCHAR NOT NULL,
    token_last_chars VARCHAR NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT personal_access_tokens_space_member_id_space_id_fkey
        FOREIGN KEY (space_member_id, space_id)
        REFERENCES space_members(id, space_id) ON DELETE CASCADE
);

-- トークン認証はダイジェストで行を引く
CREATE UNIQUE INDEX idx_personal_access_tokens_token_digest ON personal_access_tokens(token_digest);
CREATE INDEX idx_personal_access_tokens_space_id ON personal_access_tokens(space_id);
CREATE INDEX idx_personal_access_tokens_space_member_id ON personal_access_tokens(space_member_id);

-- migrate:down

DROP INDEX IF EXISTS idx_personal_access_tokens_space_member_id;
DROP INDEX IF EXISTS idx_personal_access_tokens_space_id;
DROP INDEX IF EXISTS idx_personal_access_tokens_token_digest;
DROP TABLE IF EXISTS personal_access_tokens;

ALTER TABLE space_members
    DROP CONSTRAINT IF EXISTS space_members_id_space_id_key;
