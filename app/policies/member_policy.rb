# typed: strict
# frozen_string_literal: true

# Space Memberロール専用のPolicyクラス
# Memberは基本操作権限のみを持つ
class MemberPolicy < BaseMemberPolicy
  # Memberは参加しているトピックのみ編集可能
  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    if space_member_record.nil?
      return false
    end

    in_same_space?(space_record_id: topic_record.space_id) &&
      joined_topic?(topic_record_id: topic_record.id)
  end

  # Memberはスペース設定を変更不可
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    false
  end

  # Memberはトピック作成可能
  sig { returns(T::Boolean) }
  def can_create_topic?
    joined_space?
  end

  # Memberは参加しているトピックにページ作成可能
  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    if space_member_record.nil?
      return false
    end

    joined_topic?(topic_record_id: topic_record.id)
  end

  # Memberは参加しているトピックのページを編集可能
  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    if space_member_record.nil?
      return false
    end

    active? && joined_topic?(topic_record_id: page_record.topic_id)
  end

  # Memberは参加しているトピックのドラフトページを編集可能
  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    if space_member_record.nil?
      return false
    end

    active? &&
      in_same_space?(space_record_id: page_record.space_id) &&
      joined_topic?(topic_record_id: page_record.topic_id)
  end

  # Memberはスペース内のページを閲覧可能
  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    if space_member_record.nil?
      return false
    end

    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  # Memberはページを削除可能
  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    if space_member_record.nil?
      return false
    end

    active? && in_same_space?(space_record_id: page_record.space_id)
  end

  # Memberはゴミ箱を閲覧可能
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    if space_member_record.nil?
      return false
    end

    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Memberは一括復元可能
  sig { returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    if space_member_record.nil?
      return false
    end

    active?
  end

  # Memberはファイルアップロード可能
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    if space_member_record.nil?
      return false
    end

    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Memberはファイル閲覧可能
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    if space_member_record.nil?
      return false
    end

    active? && in_same_space?(space_record_id: attachment_record.space_id)
  end

  # Memberは自分がアップロードしたファイルのみ削除可能
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    if space_member_record.nil?
      return false
    end

    active? &&
      in_same_space?(space_record_id: attachment_record.space_id) &&
      space_member_record!.id == attachment_record.attached_space_member_id
  end

  # Memberはファイル管理画面にアクセス不可
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    false
  end

  # Memberはエクスポート不可
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    false
  end

  # 閲覧可能なトピック（Memberは全トピック閲覧可能）
  sig { params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    if space_member_record.nil?
      return T.cast(TopicRecord.none, TopicRecord::PrivateAssociationRelation)
    end

    space_member_record!.space_record.not_nil!.topic_records.kept
  end

  # 閲覧可能なページ（Memberは全ページ閲覧可能）
  sig { params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    if space_member_record.nil?
      return T.cast(PageRecord.none, PageRecord::PrivateAssociationRelation)
    end

    space_member_record!.space_record.not_nil!.page_records.active
  end
end
