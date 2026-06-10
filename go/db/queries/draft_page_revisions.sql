-- name: CreateDraftPageRevision :one
-- 下書きページリビジョンを作成する
INSERT INTO draft_page_revisions (draft_page_id, space_id, space_member_id, title, body, body_html, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DeleteDraftPageRevisionsByDraftPageID :exec
-- 下書きページIDに紐づくリビジョンをすべて削除する
DELETE FROM draft_page_revisions WHERE draft_page_id = $1 AND space_id = $2;

-- name: CountDraftPageRevisionsByDraftPageID :one
-- 下書きページIDに紐づくリビジョン件数を返す
SELECT COUNT(*) FROM draft_page_revisions WHERE draft_page_id = $1 AND space_id = $2;

-- name: ListDraftPageRevisionsByDraftPageID :many
-- Returns revisions for a draft page, newest first, capped at the given limit.
-- The order is created_at DESC, id DESC so that revisions sharing a created_at
-- (e.g. created within the same instant) still have a stable total order. The
-- version-number calculation in the upper layers depends on this stable order.
--
-- [Ja] 下書きページのリビジョンを新しい順 (最大 limit 件) で返す。
-- 並び順は created_at DESC, id DESC とし、created_at が同一のリビジョン
-- (同一時刻に作成された場合など) でも安定した全順序になるようにする。
-- 上位層でのバージョン番号の算出がこの安定した順序に依存する。
SELECT * FROM draft_page_revisions
WHERE draft_page_id = $1 AND space_id = $2
ORDER BY created_at DESC, id DESC
LIMIT $3;

-- name: FindDraftPageRevisionByID :one
-- Returns a single draft page revision by ID, scoped by space_id.
-- [Ja] 下書きページリビジョンを ID で取得する (スペース ID でスコープ)。
SELECT * FROM draft_page_revisions WHERE id = $1 AND space_id = $2;

-- name: FindPreviousDraftPageRevision :one
-- Returns the revision immediately preceding the given one within the same draft
-- page, used as the comparison target for diffs. "Preceding" is defined by the
-- (created_at, id) total order that ListDraftPageRevisionsByDraftPageID also uses,
-- so the previous revision is the newest one strictly older than the target.
--
-- [Ja] 同一下書きページ内で対象リビジョンの直前のリビジョンを返す。差分の比較対象に使う。
-- 「直前」は ListDraftPageRevisionsByDraftPageID と同じ (created_at, id) の全順序で定義し、
-- 対象より厳密に古いもののうち最も新しいものを返す。
SELECT * FROM draft_page_revisions
WHERE draft_page_id = $1 AND space_id = $2
  AND (created_at < $3 OR (created_at = $3 AND id < $4))
ORDER BY created_at DESC, id DESC
LIMIT 1;
