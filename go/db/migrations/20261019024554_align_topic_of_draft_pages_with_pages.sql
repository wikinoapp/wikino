-- migrate:up
-- ロック待ちが長引いてページへの書き込みを止め続けないよう、取得できなければ失敗させる。
SET LOCAL lock_timeout = '5s';

-- ページの移動が終わってからトピックを読み、補正が終わるまで次の移動を待たせる。
LOCK TABLE pages IN SHARE MODE;

-- 下書きのトピックはページのトピックに従う。ページのトピックと食い違う下書きを、ページのトピックへ揃える。
-- 食い違う下書きが残ると、それが指すトピックの削除が外部キー (draft_pages.topic_id) の違反で失敗する。
UPDATE draft_pages dp
SET topic_id = p.topic_id
FROM pages p
WHERE p.id = dp.page_id
  AND p.space_id = dp.space_id
  AND dp.topic_id <> p.topic_id;

-- migrate:down
-- 揃える前のトピックは残していないため、戻さない。
