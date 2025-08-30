# typed: strict
# frozen_string_literal: true

# Space Regular Memberロール専用のPolicyクラス
# Space関連の基本操作権限のみを持つ
class SpaceRegularMemberPolicy < BaseSpaceMemberPolicy
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
end
