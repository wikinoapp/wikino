# typed: strict
# frozen_string_literal: true

# Topic Owner専用のポリシークラス
# Space OwnerがTopic操作を行う際に使用される
# TopicMemberRecordを必要とせず、SpaceMemberRecordのみで権限を判定
class TopicOwnerPolicy < ApplicationPolicy
  include TopicPermissions

  sig do
    params(
      user_record: UserRecord,
      space_member_record: SpaceMemberRecord
    ).void
  end
  def initialize(user_record:, space_member_record:)
    super(user_record:)
    @space_member_record = space_member_record

    if mismatched_relations?
      raise ArgumentError, [
        "Mismatched relations.",
        "user_record.id: #{user_record.id.inspect}",
        "space_member_record.user_id: #{space_member_record.user_id.inspect}"
      ].join(" ")
    end
  end
  # Topic Ownerは常にトピックを閲覧可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_show_topic?(topic_record:)
    active?
  end

  # Topic Ownerはトピックの基本情報を更新可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    active? && in_same_space?(space_record_id: topic_record.space_id)
  end

  # Topic Ownerはトピックを削除可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    active? && in_same_space?(space_record_id: topic_record.space_id)
  end

  # Topic Ownerはトピックメンバーを管理可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    active? && in_same_space?(space_record_id: topic_record.space_id)
  end

  # Topic Ownerは自分が管理するトピックにページを作成可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    active? && in_same_space?(space_record_id: topic_record.space_id)
  end

  # Topic Ownerはページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  # Topic Ownerはドラフトページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    can_update_page?(page_record:)
  end

  # Topic Ownerはページを閲覧可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    # 公開トピックのページは誰でも閲覧可能
    if page_record.topic_record.not_nil!.visibility_public?
      return true
    end

    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  # Topic Ownerはページをゴミ箱に移動可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  sig { returns(SpaceMemberRecord) }
  attr_reader :space_member_record
  private :space_member_record

  # 共通ヘルパーメソッド
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
    user_record.not_nil!.id != space_member_record.user_id
  end
end
