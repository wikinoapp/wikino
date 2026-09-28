-- migrate:up

-- 公開APIのページ一覧 (ListPublishedPagesByModifiedAt) は、スペースのページを更新日時の新しい順に
-- 並べ、(modified_at, id) のカーソルで続きを取る。並び順と同じ複合インデックスを逆順に走査させ、
-- 深いページでも先頭から数え直さずに続きを返せるようにする。
CREATE INDEX idx_pages_on_space_id_and_modified_at_and_id ON pages(space_id, modified_at, id);

-- migrate:down

DROP INDEX idx_pages_on_space_id_and_modified_at_and_id;
