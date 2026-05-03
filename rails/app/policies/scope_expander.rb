# typed: strict
# frozen_string_literal: true

# スコープの含意を展開するヘルパー。
# DB 保存時には展開せず、権限判定時にのみ使用する。
module ScopeExpander
  extend T::Sig

  # リソース内の含意ルール（上位スコープ -> 下位スコープ）
  IMPLICATIONS = T.let(
    {
      Scope::TOPIC_WRITE => [Scope::TOPIC_READ],
      Scope::TOPIC_MEMBER_WRITE => [Scope::TOPIC_MEMBER_READ],
      Scope::PAGE_WRITE => [Scope::PAGE_READ],
      Scope::DRAFT_PAGE_WRITE => [Scope::DRAFT_PAGE_READ],
      Scope::SUGGESTION_WRITE => [Scope::SUGGESTION_READ],
      Scope::SUGGESTION_COMMENT_WRITE => [Scope::SUGGESTION_COMMENT_READ],
      Scope::SPACE_WRITE => [Scope::SPACE_READ],
      Scope::SPACE_MEMBER_WRITE => [Scope::SPACE_MEMBER_READ],
      Scope::ATTACHMENT_WRITE => [Scope::ATTACHMENT_READ]
    }.freeze,
    T::Hash[String, T::Array[String]]
  )

  # space:admin が包括するすべてのリソーススコープ（space:admin 自体は含まない）
  ALL_RESOURCE_SCOPES = T.let(
    [
      # スペース
      Scope::SPACE_READ,
      Scope::SPACE_WRITE,
      Scope::SPACE_DELETE,
      # トピック
      Scope::TOPIC_READ,
      Scope::TOPIC_WRITE,
      Scope::TOPIC_DELETE,
      # トピックメンバー
      Scope::TOPIC_MEMBER_READ,
      Scope::TOPIC_MEMBER_WRITE,
      Scope::TOPIC_MEMBER_DELETE,
      # ページ
      Scope::PAGE_READ,
      Scope::PAGE_WRITE,
      Scope::PAGE_TRASH,
      Scope::PAGE_RESTORE,
      # 下書きページ
      Scope::DRAFT_PAGE_READ,
      Scope::DRAFT_PAGE_WRITE,
      Scope::DRAFT_PAGE_DELETE,
      # 編集提案
      Scope::SUGGESTION_READ,
      Scope::SUGGESTION_WRITE,
      Scope::SUGGESTION_APPLY,
      Scope::SUGGESTION_CLOSE,
      # 編集提案コメント
      Scope::SUGGESTION_COMMENT_READ,
      Scope::SUGGESTION_COMMENT_WRITE,
      # スペースメンバー
      Scope::SPACE_MEMBER_READ,
      Scope::SPACE_MEMBER_WRITE,
      Scope::SPACE_MEMBER_DELETE,
      # 添付ファイル
      Scope::ATTACHMENT_READ,
      Scope::ATTACHMENT_WRITE,
      Scope::ATTACHMENT_DELETE
    ].freeze,
    T::Array[String]
  )

  # スコープの含意を展開し、有効なスコープの配列を返す
  sig { params(scopes: T::Array[String]).returns(T::Array[String]) }
  def self.expand(scopes)
    expanded = scopes.dup

    # リソース内の含意展開（write -> read）
    scopes.each do |scope|
      implied = IMPLICATIONS[scope]
      if implied
        expanded.concat(implied)
      end
    end

    # space:admin は全リソーススコープを包括する（唯一の特別スコープ）
    if scopes.include?(Scope::SPACE_ADMIN)
      expanded.concat(ALL_RESOURCE_SCOPES)
    end

    expanded.uniq
  end
end
