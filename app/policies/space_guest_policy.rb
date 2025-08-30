# typed: strict
# frozen_string_literal: true

# ゲスト（非メンバー）用のPolicyクラス
# 公開コンテンツのみ閲覧可能
class SpaceGuestPolicy < ApplicationPolicy
  extend T::Helpers

  include SpacePermissions

  sig { params(user_record: T.nilable(UserRecord)).void }
  def initialize(user_record:)
    super
    @user_record = user_record
  end

  # Space権限（全て不可）
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    false
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_space_members?(space_record:)
    false
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    false
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_delete_space?(space_record:)
    false
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    false
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    false
  end

  # Space参加状態
  sig { override.returns(T::Boolean) }
  def joined_space?
    false
  end

  sig { override.returns(T.any(TopicRecord::PrivateCollectionProxy, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    TopicRecord.none
  end

  sig { override.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    # ゲストは公開トピックのみ閲覧可能
    space_record.topic_records.kept.visibility_public
  end

  sig { override.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    # ゲストは公開トピックのページのみ閲覧可能
    space_record.page_records.active.topics_visibility_public
  end

  sig { override.returns(T::Boolean) }
  def can_create_topic?
    false
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    false
  end

  sig { override.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    false
  end

  # Topic権限（全て不可）
  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    false
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    false
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    false
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    false
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    false
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    # 公開トピックのページのみ閲覧可能
    page_record.topic_record!.visibility_public?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    false
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_draft_page?(topic_record:)
    false
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    false
  end

  # 添付ファイル権限
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    # 公開ページで使用されているファイルのみ閲覧可能
    attachment_record.all_referencing_pages_public?
  end

  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    false
  end

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record
  private :user_record

  # Topic権限への委譲メソッド（ゲストはTopic権限を持たない）
  sig { params(topic_record: TopicRecord).returns(T.nilable(T::Wikino::TopicPolicyInstance)) }
  def topic_policy_for(topic_record:)
    nil # ゲストはTopic権限を持たない
  end
end
