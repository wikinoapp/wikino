# typed: strict
# frozen_string_literal: true

# Topic Member専用のポリシークラス
# TopicのMember権限を持つユーザーの権限を定義
class TopicMemberPolicy < ApplicationPolicy
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

  # Topic Memberはトピックの基本情報を更新不可
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    false # Topic Memberは設定変更不可
  end

  # Topic Memberはトピックを削除不可
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    false
  end

  # Topic Memberはトピックメンバーを管理不可
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    false
  end

  # Topic Memberは自分が参加しているトピックにページを作成可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    return false unless space_member_record.active?

    # 自分が参加しているトピックにページ作成可能
    topic_member_record.topic_id == topic_record.id
  end

  # Topic Memberはページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    return false unless space_member_record.active?

    # 自分が参加しているトピックのページは更新可能
    topic_member_record.topic_id == page_record.topic_id
  end

  # Topic Memberはドラフトページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    can_update_page?(page_record:)
  end

  # Topic Memberはページを閲覧可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    # 公開トピックのページは誰でも閲覧可能
    topic_record = page_record.topic_record
    return true if topic_record&.visibility_public?

    return false unless space_member_record.active?

    # 参加しているトピックのページは閲覧可能
    topic_member_record.topic_id == page_record.topic_id
  end

  # Topic Memberはページを削除不可（ゴミ箱移動は可能）
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_delete_page?(page_record:)
    false # 完全削除は不可
  end

  # Topic Memberはページをゴミ箱に移動可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    return false unless space_member_record.active?

    # 自分が参加しているトピックのページはゴミ箱移動可能
    topic_member_record.topic_id == page_record.topic_id
  end

  # Topic Memberはドラフトページを作成可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_draft_page?(topic_record:)
    return false unless space_member_record.active?

    topic_member_record.topic_id == topic_record.id
  end

  # Topic Memberはファイルを閲覧可能
  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    # 公開ページで使用されているファイルは誰でも閲覧可能
    return true if attachment_record.all_referencing_pages_public?
    return false unless space_member_record.active?

    # 同じSpace内のファイルは閲覧可能
    attachment_record.space_id == space_member_record.space_id
  end

  # Topic Memberは自分がアップロードしたファイルのみ削除可能
  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    return false unless space_member_record.active?

    # 自分がアップロードしたファイルのみ削除可能
    attachment_record.space_id == space_member_record.space_id &&
      attachment_record.attached_space_member_id == space_member_record.id
  end

  sig { returns(SpaceMemberRecord) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(TopicMemberRecord) }
  attr_reader :topic_member_record
  private :topic_member_record
end
