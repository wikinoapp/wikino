# typed: strict
# frozen_string_literal: true

# Space Ownerロール専用のPolicyクラス
# Space関連の全権限を持つ
class SpaceOwnerPolicy < ApplicationPolicy
  include SpacePermissions

  sig { params(user_record: UserRecord, space_member_record: SpaceMemberRecord).void }
  def initialize(user_record:, space_member_record:)
    super(user_record:)
    @space_member_record = space_member_record
  end
  # Space権限の実装

  # Ownerはスペース設定を変更可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    in_same_space?(space_record_id: space_record.id)
  end

  # Ownerはトピック作成可能
  sig { override.returns(T::Boolean) }
  def can_create_topic?
    joined_space?
  end

  # OwnerはSpaceを削除可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_delete_space?(space_record:)
    in_same_space?(space_record_id: space_record.id)
  end

  # OwnerはSpaceメンバーを管理可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_space_members?(space_record:)
    in_same_space?(space_record_id: space_record.id)
  end

  # Ownerはゴミ箱を閲覧可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Ownerは一括復元可能
  sig { override.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    active?
  end

  # Ownerはファイルアップロード可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Ownerはファイル管理画面にアクセス可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Ownerはエクスポート可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    in_same_space?(space_record_id: space_record.id)
  end

  # 閲覧可能なトピック（Ownerは全トピック閲覧可能）
  sig { override.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    space_member_record.space_record.not_nil!.topic_records.kept
  end

  # 閲覧可能なページ（Ownerは全ページ閲覧可能）
  sig { override.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    space_member_record.space_record.not_nil!.page_records.active
  end

  sig { override.returns(T::Boolean) }
  def joined_space?
    true # space_member_recordが非nilableなので常にtrue
  end

  sig { override.returns(T.any(TopicRecord::PrivateCollectionProxy, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    space_member_record.topic_records
  end

  # 共通ヘルパーメソッド
  sig { params(space_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  def in_same_space?(space_record_id:)
    space_member_record.space_id == space_record_id
  end

  sig { returns(T::Boolean) }
  def active?
    space_member_record.active?
  end

  # Topic/Page操作権限（互換性のため）
  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    in_same_space?(space_record_id: topic_record.space_id)
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    in_same_space?(space_record_id: topic_record.space_id)
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    in_same_space?(space_record_id: topic_record.space_id)
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    active? && in_same_space?(space_record_id: topic_record.space_id)
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    in_same_space?(space_record_id: page_record.space_id) || page_record.topic_record!.visibility_public?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_draft_page?(topic_record:)
    active? && in_same_space?(space_record_id: topic_record.space_id)
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  # 添付ファイル削除権限（Ownerは全ファイル削除可能）
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    active? && in_same_space?(space_record_id: attachment_record.space_id)
  end

  # 添付ファイル閲覧権限
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    in_same_space?(space_record_id: attachment_record.space_id) || attachment_record.all_referencing_pages_public?
  end

  sig { returns(SpaceMemberRecord) }
  attr_reader :space_member_record
  private :space_member_record

  # Topic権限への委譲メソッド
  sig { params(topic_record: TopicRecord).returns(T::Wikino::TopicPolicyInstance) }
  def topic_policy_for(topic_record:)
    # Space OwnerはすべてのTopicでAdmin権限を持つ
    # TopicAdminPolicyを返す
    topic_member_record = user_record&.topic_member_records&.find_by(topic_record:)
    if topic_member_record
      # 既存のTopicMemberRecordがある場合はそれを使用
      TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record:
      )
    else
      # TopicMemberRecordがなくても、Space OwnerはAdmin権限として扱う
      # 仮想的なTopicMemberRecordを作成
      virtual_topic_member = TopicMemberRecord.new(
        topic_record:,
        space_member_record:,
        role: TopicMemberRole::Admin.serialize
      )
      TopicAdminPolicy.new(
        user_record:,
        space_member_record:,
        topic_member_record: virtual_topic_member
      )
    end
  end
end
