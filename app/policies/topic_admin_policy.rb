# typed: strict
# frozen_string_literal: true

# Topic Admin専用のポリシークラス
# TopicのAdmin権限を持つユーザーの権限を定義
class TopicAdminPolicy < ApplicationPolicy
  include TopicPermissions

  sig do
    params(
      user_record: UserRecord,
      space_member_record: SpaceMemberRecord,
      topic_member_record: TopicMemberRecord
    ).void
  end
  def initialize(user_record:, space_member_record:, topic_member_record:)
    super(user_record:)
    @space_member_record = space_member_record
    @topic_member_record = topic_member_record
  end

  # Topic Adminはトピックの基本情報を更新可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    active? && in_same_topic?(topic_record_id: topic_record.id)
  end

  # Topic Adminはトピックを削除可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    active? && in_same_topic?(topic_record_id: topic_record.id)
  end

  # Topic Adminはトピックメンバーを管理可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    active? && in_same_topic?(topic_record_id: topic_record.id)
  end

  # Topic Adminは自分が管理するトピックにページを作成可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    active? && in_same_topic?(topic_record_id: topic_record.id)
  end

  # Topic Adminはページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    active? && in_same_topic?(topic_record_id: page_record.topic_id)
  end

  # Topic Adminはドラフトページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    can_update_page?(page_record:)
  end

  # Topic Adminはページを閲覧可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    # 公開トピックのページは誰でも閲覧可能
    if page_record.topic_record.not_nil!.visibility_public?
      return true
    end

    active? && in_same_topic?(topic_record_id: page_record.topic_id)
  end

  # Topic Adminはページをゴミ箱に移動可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    active? && in_same_topic?(topic_record_id: page_record.topic_id)
  end

  # Topic Adminはファイルを閲覧可能
  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    # 公開ページで使用されているファイルは誰でも閲覧可能
    return true if attachment_record.all_referencing_pages_public?

    active? && in_same_space?(space_record_id: attachment_record.space_id)
  end

  # Topic AdminはTopic内のファイルを削除可能
  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    active? && in_same_space?(space_record_id: attachment_record.space_id)
  end

  # Topic Admin固有のメソッド（Space権限は扱わない）
  # Space関連のメソッドはテスト目的で最小限の実装のみ提供
  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    false # Topic AdminはSpace設定を変更できない
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    false # Topic Adminはファイル管理画面にアクセスできない
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    false # Topic AdminはSpaceをエクスポートできない
  end

  sig { returns(T::Boolean) }
  def can_create_topic?
    # Topic Adminは新しいトピックを作成可能
    active?
  end

  sig { returns(SpaceMemberRecord) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(TopicMemberRecord) }
  attr_reader :topic_member_record
  private :topic_member_record

  # 共通ヘルパーメソッド
  sig { params(topic_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  private def in_same_topic?(topic_record_id:)
    topic_member_record.topic_id == topic_record_id
  end

  sig { params(space_record_id: T::Wikino::DatabaseId).returns(T::Boolean) }
  private def in_same_space?(space_record_id:)
    space_member_record.space_id == space_record_id
  end

  sig { returns(T::Boolean) }
  private def active?
    space_member_record.active?
  end
end
