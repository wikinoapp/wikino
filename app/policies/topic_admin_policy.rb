# typed: strict
# frozen_string_literal: true

# Topic Admin専用のポリシークラス
# TopicのAdmin権限を持つユーザーの権限を定義
class TopicAdminPolicy < ApplicationPolicy
  include TopicPermissions
  include Memoizable
  include QueryOptimizer

  sig do
    params(
      user_record: UserRecord,
      space_member_record: SpaceMemberRecord,
      topic_member_record: TopicMemberRecord
    ).void
  end
  def initialize(user_record:, space_member_record:, topic_member_record:)
    super(user_record:)
    # 関連をプリロードして最適化
    @space_member_record = T.let(
      preload_space_member_associations(space_member_record).not_nil!,
      SpaceMemberRecord
    )
    @topic_member_record = T.let(
      preload_topic_member_associations(topic_member_record).not_nil!,
      TopicMemberRecord
    )

    if mismatched_relations?
      raise ArgumentError, [
        "Mismatched relations.",
        "user_record.id: #{user_record.id.inspect}",
        "space_member_record.user_id: #{space_member_record.user_id.inspect}",
        "topic_member_record.space_member_id: #{topic_member_record.space_member_id.inspect}",
        "space_member_record.id: #{space_member_record.id.inspect}"
      ].join(" ")
    end
  end
  # Topic Adminは常にトピックを閲覧可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_show_topic?(topic_record:)
    active?
  end

  # Topic Adminはトピックの基本情報を更新可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    memoize(:can_update_topic, { topic_record_id: topic_record.id }) do
      active? && in_same_topic?(topic_record_id: topic_record.id)
    end
  end

  # Topic Adminはトピックを削除可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    memoize(:can_delete_topic, { topic_record_id: topic_record.id }) do
      active? && in_same_topic?(topic_record_id: topic_record.id)
    end
  end

  # Topic Adminはトピックメンバーを管理可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    memoize(:can_manage_topic_members, { topic_record_id: topic_record.id }) do
      active? && in_same_topic?(topic_record_id: topic_record.id)
    end
  end

  # Topic Adminは自分が管理するトピックにページを作成可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    memoize(:can_create_page, { topic_record_id: topic_record.id }) do
      active? && in_same_topic?(topic_record_id: topic_record.id)
    end
  end

  # Topic Adminはページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    memoize(:can_update_page, { page_id: page_record.id }) do
      active? && in_same_topic?(topic_record_id: page_record.topic_id)
    end
  end

  # Topic Adminはドラフトページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    can_update_page?(page_record:)
  end

  # Topic Adminはページを閲覧可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    memoize(:can_show_page, { page_id: page_record.id }) do
      # 公開トピックのページは誰でも閲覧可能
      if page_record.topic_record.not_nil!.visibility_public?
        return true
      end

      active? && in_same_topic?(topic_record_id: page_record.topic_id)
    end
  end

  # Topic Adminはページをゴミ箱に移動可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    memoize(:can_trash_page, { page_id: page_record.id }) do
      active? && in_same_topic?(topic_record_id: page_record.topic_id)
    end
  end

  sig { returns(SpaceMemberRecord) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(TopicMemberRecord) }
  attr_reader :topic_member_record
  private :topic_member_record

  # 共通ヘルパーメソッド
  sig { params(topic_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  private def in_same_topic?(topic_record_id:)
    topic_member_record.topic_id == topic_record_id
  end

  sig { params(space_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  private def in_same_space?(space_record_id:)
    space_member_record.space_id == space_record_id
  end

  sig { returns(T::Boolean) }
  private def active?
    space_member_record.active?
  end

  sig { returns(T::Boolean) }
  private def mismatched_relations?
    user_record.not_nil!.id != space_member_record.user_id ||
      topic_member_record.space_member_id != space_member_record.id
  end
end
