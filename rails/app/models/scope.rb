# typed: strict
# frozen_string_literal: true

# リソースに対する権限を表すスコープ定数。
# GitHub 風の "resource:action" 形式で命名する。
module Scope
  extend T::Sig

  # トピック関連スコープ
  TOPIC_READ = T.let("topic:read", String)
  TOPIC_WRITE = T.let("topic:write", String)
  TOPIC_DELETE = T.let("topic:delete", String)

  # トピックメンバー関連スコープ
  TOPIC_MEMBER_READ = T.let("topic_member:read", String)
  TOPIC_MEMBER_WRITE = T.let("topic_member:write", String)
  TOPIC_MEMBER_DELETE = T.let("topic_member:delete", String)

  # ページ関連スコープ
  PAGE_READ = T.let("page:read", String)
  PAGE_WRITE = T.let("page:write", String)
  PAGE_TRASH = T.let("page:trash", String)
  PAGE_RESTORE = T.let("page:restore", String)

  # 下書きページ関連スコープ
  DRAFT_PAGE_READ = T.let("draft_page:read", String)
  DRAFT_PAGE_WRITE = T.let("draft_page:write", String)
  DRAFT_PAGE_DELETE = T.let("draft_page:delete", String)

  # 編集提案関連スコープ
  SUGGESTION_READ = T.let("suggestion:read", String)
  SUGGESTION_WRITE = T.let("suggestion:write", String)
  SUGGESTION_APPLY = T.let("suggestion:apply", String)
  SUGGESTION_CLOSE = T.let("suggestion:close", String)

  # 編集提案コメント関連スコープ
  SUGGESTION_COMMENT_READ = T.let("suggestion_comment:read", String)
  SUGGESTION_COMMENT_WRITE = T.let("suggestion_comment:write", String)

  # スペースメンバー関連スコープ
  SPACE_MEMBER_READ = T.let("space_member:read", String)
  SPACE_MEMBER_WRITE = T.let("space_member:write", String)
  SPACE_MEMBER_DELETE = T.let("space_member:delete", String)

  # スペース関連スコープ
  SPACE_READ = T.let("space:read", String)
  SPACE_WRITE = T.let("space:write", String)
  SPACE_DELETE = T.let("space:delete", String)
  # 全スコープを包括する唯一の特別スコープ
  SPACE_ADMIN = T.let("space:admin", String)

  # 添付ファイル関連スコープ
  ATTACHMENT_READ = T.let("attachment:read", String)
  ATTACHMENT_WRITE = T.let("attachment:write", String)
  ATTACHMENT_DELETE = T.let("attachment:delete", String)
end
