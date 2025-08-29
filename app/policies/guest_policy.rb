# typed: strict
# frozen_string_literal: true

# ゲスト（非メンバー）用のPolicyクラス
# 公開コンテンツのみ閲覧可能
class GuestPolicy < ApplicationPolicy
  extend T::Sig

  # ゲストはトピック編集不可
  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    false
  end

  # ゲストはスペース設定変更不可
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    false
  end

  # ゲストはトピック作成不可
  sig { returns(T::Boolean) }
  def can_create_topic?
    false
  end

  # ゲストはページ作成不可
  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    false
  end

  # ゲストはページ編集不可
  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    false
  end

  # ゲストはドラフトページ編集不可
  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    false
  end

  # ゲストは公開トピックのページのみ閲覧可能
  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    page_record.topic_record!.visibility_public?
  end

  # ゲストはページ削除不可
  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    false
  end

  # ゲストはゴミ箱閲覧不可
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    false
  end

  # ゲストは一括復元不可
  sig { returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    false
  end

  # ゲストはファイルアップロード不可
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    false
  end

  # ゲストは公開ページで使用されているファイルのみ閲覧可能
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    attachment_record.all_referencing_pages_public?
  end

  # ゲストはファイル削除不可
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    false
  end

  # ゲストはファイル管理画面アクセス不可
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    false
  end

  # ゲストはエクスポート不可
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    false
  end

  # スペースに参加していない
  sig { returns(T::Boolean) }
  def joined_space?
    false
  end

  # 参加トピックなし
  sig { returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    TopicRecord.none
  end

  # 閲覧可能なトピック（公開トピックのみ）
  sig { params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    space_record.topic_records.kept.visibility_public
  end

  # 閲覧可能なページ（公開トピックのページのみ）
  sig { params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    space_record.page_records.active.topics_visibility_public
  end
end

