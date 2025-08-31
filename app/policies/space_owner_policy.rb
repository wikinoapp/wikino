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

    if mismatched_relations?
      raise ArgumentError, [
        "Mismatched relations.",
        "user_record.id: #{user_record.id.inspect}",
        "space_member_record.user_id: #{space_member_record.user_id.inspect}"
      ].join(" ")
    end
  end
  # Space権限の実装

  # Ownerはスペース設定を変更可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Ownerはトピック作成可能
  sig { override.returns(T::Boolean) }
  def can_create_topic?
    active? && joined_space?
  end

  # OwnerはSpaceを削除可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_delete_space?(space_record:)
    active? && in_same_space?(space_record_id: space_record.id)
  end

  # OwnerはSpaceメンバーを管理可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_space_members?(space_record:)
    active? && in_same_space?(space_record_id: space_record.id)
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
    active? && in_same_space?(space_record_id: space_record.id)
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

  sig { override.returns(T.any(TopicRecord::PrivateCollectionProxy, TopicRecord::PrivateRelation, TopicRecord::PrivateAssociationRelation)) }
  def joined_topic_records
    space_member_record.joined_topic_records
  end

  # 添付ファイル削除権限（Ownerは全ファイル削除可能）
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    active? && in_same_space?(space_record_id: attachment_record.space_id)
  end

  # 添付ファイル閲覧権限
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    active? && (
      in_same_space?(space_record_id: attachment_record.space_id) ||
      attachment_record.all_referencing_pages_public?
    )
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

  # Topic権限への委譲メソッド
  sig { params(topic_record: TopicRecord).returns(T::Wikino::TopicPolicyInstance) }
  def topic_policy_for(topic_record:)
    # Space Ownerは常にTopicの全権限を持つため、TopicOwnerPolicyを返す
    TopicOwnerPolicy.new(
      user_record: user_record.not_nil!,
      space_member_record:
    )
  end
end
