# typed: strict
# frozen_string_literal: true

# 権限チェックの基底クラス
# SpaceメンバーのPolicyクラスで共通して使用するロジックを提供
class BaseSpaceMemberPolicy < ApplicationPolicy
  extend T::Helpers
  abstract!

  sig do
    params(
      user_record: T.nilable(UserRecord),
      space_member_record: T.nilable(SpaceMemberRecord)
    ).void
  end
  def initialize(user_record:, space_member_record:)
    super(user_record:)
    @space_member_record = space_member_record

    # user_recordとspace_member_recordの関連性を検証
    if mismatched_relations?
      raise ArgumentError, [
        "Mismatched relations.",
        "user_record.id: #{user_record&.id.inspect}",
        "space_member_record.user_id: #{space_member_record&.user_id.inspect}"
      ].join(" ")
    end
  end

  # スペースに参加しているかどうか
  sig { override.returns(T::Boolean) }
  def joined_space?
    !space_member_record.nil?
  end

  # 指定されたスペースIDと同じスペースにいるかどうか
  sig { params(space_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  def in_same_space?(space_record_id:)
    space_member_record&.space_id == space_record_id
  end

  # アクティブなメンバーかどうか
  sig { returns(T::Boolean) }
  def active?
    space_member_record&.active? || false
  end

  # 参加しているトピックのレコードを取得
  sig { override.returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    if space_member_record.nil?
      return TopicRecord.none
    end

    space_member_record!.joined_topic_records
  end

  # トピックに参加しているかどうか
  sig { params(topic_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  def joined_topic?(topic_record_id:)
    if space_member_record.nil?
      return false
    end

    space_member_record!.joined_topic_records.where(id: topic_record_id).exists?
  end

  # 抽象メソッド - 子クラスで実装が必要
  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
  end

  sig { abstract.returns(T::Boolean) }
  def can_create_topic?
  end

  sig { abstract.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
  end

  sig { abstract.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
  end

  sig { abstract.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
  end

  sig { abstract.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
  end

  sig { abstract.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
  end

  sig { abstract.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
  end

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_reader :space_member_record
  protected :space_member_record

  # space_member_record の non-nil バージョン
  sig { returns(SpaceMemberRecord) }
  private def space_member_record!
    space_member_record.not_nil!
  end

  # user_recordとspace_member_recordの関連性が不整合かどうか
  sig { returns(T::Boolean) }
  private def mismatched_relations?
    if user_record.nil? || space_member_record.nil?
      return false
    end

    user_record.not_nil!.id != space_member_record.not_nil!.user_id
  end
end
