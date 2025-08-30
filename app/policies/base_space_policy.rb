# typed: strict
# frozen_string_literal: true

# 権限チェックの基底クラス
# SpaceメンバーのPolicyクラスで共通して使用するロジックを提供
class BaseSpacePolicy < ApplicationPolicy
  extend T::Helpers

  include SpacePermissions

  abstract!

  sig { params(user_record: T.nilable(UserRecord), space_member_record: T.nilable(SpaceMemberRecord)).void }
  def initialize(user_record:, space_member_record:)
    super(user_record:)
    @space_member_record = space_member_record
  end

  sig { override.returns(T::Boolean) }
  def joined_space?
    !space_member_record.nil?
  end

  sig { params(space_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  def in_same_space?(space_record_id:)
    space_member_record&.space_id == space_record_id
  end

  sig { returns(T::Boolean) }
  def active?
    space_member_record&.active? || false
  end

  sig { override.returns(T.any(TopicRecord::PrivateCollectionProxy, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    space_member_record&.topic_records || TopicRecord.none
  end

  sig { override.params(space_record: SpaceRecord).returns(TopicRecord::PrivateCollectionProxy) }
  def showable_topics(space_record:)
    # デフォルト実装：Spaceメンバーは全トピック閲覧可能
    space_record.topic_records
  end

  sig { override.params(space_record: SpaceRecord).returns(PageRecord::PrivateCollectionProxy) }
  def showable_pages(space_record:)
    # デフォルト実装：Spaceメンバーは全ページ閲覧可能
    space_record.page_records
  end

  sig { override.returns(T::Boolean) }
  def can_create_topic?
    active?
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    in_same_space?(space_record_id: space_record.id)
  end

  sig { override.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    active?
  end

  # Topic/Page操作権限（互換性のため）
  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    false # デフォルト実装
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    false # デフォルト実装
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    false # デフォルト実装
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    false # デフォルト実装
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    false # デフォルト実装
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_draft_page?(topic_record:)
    false # デフォルト実装
  end

  sig { params(draft_page_record: DraftPageRecord).returns(T::Boolean) }
  def can_update_draft_page?(draft_page_record:)
    false # デフォルト実装
  end

  # 添付ファイル削除権限（個別ファイル）
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    # デフォルト実装：サブクラスでオーバーライド
    false
  end

  # 添付ファイル閲覧権限
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    # デフォルト実装：サブクラスでオーバーライド
    false
  end

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(SpaceMemberRecord) }
  private def space_member_record!
    space_member_record.not_nil!
  end
end
