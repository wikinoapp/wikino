# typed: strict
# frozen_string_literal: true

# Space Memberロール専用のPolicyクラス
# Space関連の基本操作権限のみを持つ
class SpaceMemberPolicy < ApplicationPolicy
  include SpacePermissions

  sig { params(user_record: T.nilable(UserRecord), space_member_record: T.nilable(SpaceMemberRecord)).void }
  def initialize(user_record:, space_member_record:)
    super(user_record:)
    @space_member_record = space_member_record
  end
  # Space権限の実装

  # Memberはスペース設定を変更不可
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    false
  end

  # Memberはトピック作成可能
  sig { override.returns(T::Boolean) }
  def can_create_topic?
    joined_space?
  end

  # MemberはSpaceを削除不可
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_delete_space?(space_record:)
    false
  end

  # MemberはSpaceメンバーを管理不可
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_space_members?(space_record:)
    false
  end

  # Memberはゴミ箱を閲覧可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    if space_member_record.nil?
      return false
    end

    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Memberは一括復元可能
  sig { override.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    if space_member_record.nil?
      return false
    end

    active?
  end

  # Memberはファイルアップロード可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    if space_member_record.nil?
      return false
    end

    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Memberはファイル管理画面にアクセス不可
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    false
  end

  # Memberはエクスポート不可
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    false
  end

  # 閲覧可能なトピック（Memberは全トピック閲覧可能）
  sig { override.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    if space_member_record.nil?
      return T.cast(TopicRecord.none, TopicRecord::PrivateAssociationRelation)
    end

    space_member_record!.space_record.not_nil!.topic_records.kept
  end

  # 閲覧可能なページ（Memberは全ページ閲覧可能）
  sig { override.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    if space_member_record.nil?
      return T.cast(PageRecord.none, PageRecord::PrivateAssociationRelation)
    end

    space_member_record!.space_record.not_nil!.page_records.active
  end

  sig { override.returns(T::Boolean) }
  def joined_space?
    !space_member_record.nil?
  end

  sig { override.returns(T.any(TopicRecord::PrivateCollectionProxy, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    space_member_record&.topic_records || TopicRecord.none
  end

  # 共通ヘルパーメソッド
  sig { params(space_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  def in_same_space?(space_record_id:)
    space_member_record&.space_id == space_record_id
  end

  sig { returns(T::Boolean) }
  def active?
    space_member_record&.active? || false
  end

  # Topic/Page操作権限（互換性のため）
  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    return false if space_member_record.nil?
    in_same_space?(space_record_id: topic_record.space_id) &&
      space_member_record!.topic_records.where(id: topic_record.id).exists?
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    false # Memberはトピック削除不可
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    false # Memberはトピックメンバー管理不可
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    return false if space_member_record.nil?
    active? &&
      in_same_space?(space_record_id: topic_record.space_id) &&
      space_member_record!.topic_records.where(id: topic_record.id).exists?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    return false if space_member_record.nil?
    active? &&
      in_same_space?(space_record_id: page_record.space_id) &&
      space_member_record!.topic_records.where(id: page_record.topic_id).exists?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    return true if page_record.topic_record!.visibility_public?
    return false if space_member_record.nil?
    in_same_space?(space_record_id: page_record.space_id)
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    return false if space_member_record.nil?
    active? &&
      in_same_space?(space_record_id: page_record.space_id) &&
      space_member_record!.topic_records.where(id: page_record.topic_id).exists?
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_draft_page?(topic_record:)
    return false if space_member_record.nil?
    active? &&
      in_same_space?(space_record_id: topic_record.space_id) &&
      space_member_record!.topic_records.where(id: topic_record.id).exists?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    return false if space_member_record.nil?
    active? &&
      in_same_space?(space_record_id: page_record.space_id) &&
      space_member_record!.topic_records.where(id: page_record.topic_id).exists?
  end

  # 添付ファイル削除権限（Memberは自分がアップロードしたファイルのみ削除可能）
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    return false if space_member_record.nil?
    active? &&
      in_same_space?(space_record_id: attachment_record.space_id) &&
      space_member_record!.id == attachment_record.attached_space_member_id
  end

  # 添付ファイル閲覧権限
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    return true if attachment_record.all_referencing_pages_public?
    return false if space_member_record.nil?
    in_same_space?(space_record_id: attachment_record.space_id)
  end

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(SpaceMemberRecord) }
  private def space_member_record!
    space_member_record.not_nil!
  end
end
