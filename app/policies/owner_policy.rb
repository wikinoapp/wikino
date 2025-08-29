# typed: strict
# frozen_string_literal: true

# Space Ownerロール専用のPolicyクラス
# Ownerは全権限を持つ
class OwnerPolicy < BaseMemberPolicy
  # Ownerは全トピックを編集可能
  # TopicMemberRecordの有無に関わらず、Space内の全トピックで編集権限を持つ
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    in_same_space?(space_record_id: topic_record.space_id)
  end

  # Ownerは全トピックを削除可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    in_same_space?(space_record_id: topic_record.space_id)
  end

  # Ownerは全トピックのメンバー管理可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    in_same_space?(space_record_id: topic_record.space_id)
  end

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

  # Ownerはページ作成可能
  # TopicMemberRecordの有無に関わらず、全トピックでページ作成権限を持つ
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    in_same_space?(space_record_id: topic_record.space_id)
  end

  # Ownerは全ページを編集可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  # Ownerは全ドラフトページを編集可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  # Ownerはページ閲覧可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  # Ownerはページを削除可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    active? && in_same_space?(space_record_id: page_record.space_id)
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

  # Ownerはファイル閲覧可能
  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    active? && in_same_space?(space_record_id: attachment_record.space_id)
  end

  # Ownerは全ファイルを削除可能
  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    active? && in_same_space?(space_record_id: attachment_record.space_id)
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
    if space_member_record.nil?
      return T.cast(TopicRecord.none, TopicRecord::PrivateAssociationRelation)
    end

    space_member_record!.space_record.not_nil!.topic_records.kept
  end

  # 閲覧可能なページ（Ownerは全ページ閲覧可能）
  sig { override.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    if space_member_record.nil?
      return T.cast(PageRecord.none, PageRecord::PrivateAssociationRelation)
    end

    space_member_record!.space_record.not_nil!.page_records.active
  end
end
