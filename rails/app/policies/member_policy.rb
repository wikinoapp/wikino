# typed: strict
# frozen_string_literal: true

# スペースメンバー用の権限判定クラス。
# スペーススコープとトピックスコープの和集合を含意展開し、
# 有効スコープの集合として保持する。
class MemberPolicy
  extend T::Sig

  sig do
    params(
      space_scopes: T::Array[String],
      topic_scopes: T::Array[String]
    ).void
  end
  def initialize(space_scopes:, topic_scopes:)
    merged = space_scopes + topic_scopes
    expanded = ScopeExpander.expand(merged)
    @effective_scopes = T.let(
      expanded.to_set.freeze,
      T::Set[String]
    )
  end

  # トピック

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_show_topic?(topic_record:)
    if topic_record.visibility_public?
      return true
    end

    effective_scopes.include?(Scope::TOPIC_READ)
  end

  sig { returns(T::Boolean) }
  def can_update_topic?
    effective_scopes.include?(Scope::TOPIC_WRITE)
  end

  sig { returns(T::Boolean) }
  def can_delete_topic?
    effective_scopes.include?(Scope::TOPIC_DELETE)
  end

  # ページ

  sig { returns(T::Boolean) }
  def can_create_page?
    effective_scopes.include?(Scope::PAGE_WRITE)
  end

  sig { returns(T::Boolean) }
  def can_update_page?
    effective_scopes.include?(Scope::PAGE_WRITE)
  end

  sig { returns(T::Boolean) }
  def can_trash_page?
    effective_scopes.include?(Scope::PAGE_TRASH)
  end

  # 下書きページ

  sig { params(is_owner: T::Boolean).returns(T::Boolean) }
  def can_show_draft_page?(is_owner:)
    if !effective_scopes.include?(Scope::DRAFT_PAGE_READ)
      return false
    end

    is_owner || effective_scopes.include?(Scope::SPACE_ADMIN)
  end

  sig { params(is_owner: T::Boolean).returns(T::Boolean) }
  def can_update_draft_page?(is_owner:)
    if !effective_scopes.include?(Scope::DRAFT_PAGE_WRITE)
      return false
    end

    is_owner || effective_scopes.include?(Scope::SPACE_ADMIN)
  end

  # スペース

  sig { returns(T::Boolean) }
  def can_update_space?
    effective_scopes.include?(Scope::SPACE_WRITE)
  end

  sig { returns(T::Boolean) }
  def can_export_space?
    effective_scopes.include?(Scope::SPACE_WRITE)
  end

  # トピック作成

  sig { returns(T::Boolean) }
  def can_create_topic?
    effective_scopes.include?(Scope::TOPIC_WRITE)
  end

  # ゴミ箱

  sig { returns(T::Boolean) }
  def can_show_trash?
    effective_scopes.include?(Scope::PAGE_TRASH)
  end

  sig { returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    effective_scopes.include?(Scope::PAGE_RESTORE)
  end

  # 添付ファイル

  sig { returns(T::Boolean) }
  def can_upload_attachment?
    effective_scopes.include?(Scope::ATTACHMENT_WRITE)
  end

  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    effective_scopes.include?(Scope::ATTACHMENT_READ) ||
      attachment_record.all_referencing_pages_public?
  end

  sig { returns(T::Boolean) }
  def can_delete_attachment?
    effective_scopes.include?(Scope::ATTACHMENT_DELETE)
  end

  sig { returns(T::Set[String]) }
  attr_reader :effective_scopes
  private :effective_scopes
end
