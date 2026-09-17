-- migrate:up

-- actionをread / write / deleteに固定したスコープ名へ、保存済みの旧名を置き換える。
-- 旧名と正式名が同じ行に混在していると置換後に重複するため、最初に現れた位置を
-- 残して重複を除く。対象の旧名を含まない行は書き換えない。
UPDATE space_members
SET scopes = ARRAY(
    SELECT renamed.scope
    FROM (
        SELECT
            CASE s.scope
                WHEN 'page:trash' THEN 'page_trash:write'
                WHEN 'page:restore' THEN 'page_trash:delete'
                WHEN 'suggestion:apply' THEN 'suggestion_application:write'
                WHEN 'suggestion:close' THEN 'suggestion_closure:write'
                ELSE s.scope
            END AS scope,
            s.ord
        FROM unnest(space_members.scopes) WITH ORDINALITY AS s(scope, ord)
    ) AS renamed
    GROUP BY renamed.scope
    ORDER BY min(renamed.ord)
)
WHERE scopes && ARRAY['page:trash', 'page:restore', 'suggestion:apply', 'suggestion:close']::text[];

UPDATE topic_members
SET scopes = ARRAY(
    SELECT renamed.scope
    FROM (
        SELECT
            CASE s.scope
                WHEN 'page:trash' THEN 'page_trash:write'
                WHEN 'page:restore' THEN 'page_trash:delete'
                WHEN 'suggestion:apply' THEN 'suggestion_application:write'
                WHEN 'suggestion:close' THEN 'suggestion_closure:write'
                ELSE s.scope
            END AS scope,
            s.ord
        FROM unnest(topic_members.scopes) WITH ORDINALITY AS s(scope, ord)
    ) AS renamed
    GROUP BY renamed.scope
    ORDER BY min(renamed.ord)
)
WHERE scopes && ARRAY['page:trash', 'page:restore', 'suggestion:apply', 'suggestion:close']::text[];

-- migrate:down

-- 正式名を旧コードが解釈できる旧名へ戻す。page_trash:readには対応する旧名が無く、
-- 旧コードでは権限を与えない未知の値になるため配列から取り除く。
UPDATE space_members
SET scopes = ARRAY(
    SELECT renamed.scope
    FROM (
        SELECT
            CASE s.scope
                WHEN 'page_trash:write' THEN 'page:trash'
                WHEN 'page_trash:delete' THEN 'page:restore'
                WHEN 'suggestion_application:write' THEN 'suggestion:apply'
                WHEN 'suggestion_closure:write' THEN 'suggestion:close'
                ELSE s.scope
            END AS scope,
            s.ord
        FROM unnest(space_members.scopes) WITH ORDINALITY AS s(scope, ord)
        WHERE s.scope <> 'page_trash:read'
    ) AS renamed
    GROUP BY renamed.scope
    ORDER BY min(renamed.ord)
)
WHERE scopes && ARRAY['page_trash:read', 'page_trash:write', 'page_trash:delete', 'suggestion_application:write', 'suggestion_closure:write']::text[];

UPDATE topic_members
SET scopes = ARRAY(
    SELECT renamed.scope
    FROM (
        SELECT
            CASE s.scope
                WHEN 'page_trash:write' THEN 'page:trash'
                WHEN 'page_trash:delete' THEN 'page:restore'
                WHEN 'suggestion_application:write' THEN 'suggestion:apply'
                WHEN 'suggestion_closure:write' THEN 'suggestion:close'
                ELSE s.scope
            END AS scope,
            s.ord
        FROM unnest(topic_members.scopes) WITH ORDINALITY AS s(scope, ord)
        WHERE s.scope <> 'page_trash:read'
    ) AS renamed
    GROUP BY renamed.scope
    ORDER BY min(renamed.ord)
)
WHERE scopes && ARRAY['page_trash:read', 'page_trash:write', 'page_trash:delete', 'suggestion_application:write', 'suggestion_closure:write']::text[];
