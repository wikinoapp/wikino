# typed: strict
# frozen_string_literal: true

# Topic Admin専用のポリシークラス
# TopicのAdmin権限を持つユーザーの権限を定義
class TopicAdminPolicy < BaseSpaceMemberPolicy
  extend T::Sig

  sig do
    params(
      user_record: T.nilable(UserRecord),
      space_member_record: T.nilable(SpaceMemberRecord),
      topic_member_record: T.nilable(TopicMemberRecord)
    ).void
  end
  def initialize(user_record:, space_member_record:, topic_member_record:)
    super(user_record:, space_member_record:)
    @topic_member_record = topic_member_record
  end

  # Topic Adminはトピックの基本情報を更新可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    return false unless active?
    return false unless in_same_space?(space_record_id: topic_record.space_id)

    # 自分がAdminであるトピックのみ更新可能
    topic_member_record&.topic_id == topic_record.id
  end

  # Topic Adminはトピックを削除可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_delete_topic?(topic_record:)
    return false unless active?
    return false unless in_same_space?(space_record_id: topic_record.space_id)

    # 自分がAdminであるトピックのみ削除可能
    topic_member_record&.topic_id == topic_record.id
  end

  # Topic Adminはトピックメンバーを管理可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_manage_topic_members?(topic_record:)
    return false unless active?
    return false unless in_same_space?(space_record_id: topic_record.space_id)

    # 自分がAdminであるトピックのメンバーのみ管理可能
    topic_member_record&.topic_id == topic_record.id
  end

  # Topic Adminはスペース設定を変更不可
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    false
  end

  # Topic Adminは新しいトピックを作成可能
  sig { override.returns(T::Boolean) }
  def can_create_topic?
    active?
  end

  # Topic Adminは自分が管理するトピックにページを作成可能
  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    return false unless active?
    return false unless in_same_space?(space_record_id: topic_record.space_id)

    # 自分がAdminであるトピック、または参加しているトピックにページ作成可能
    topic_member_record&.topic_id == topic_record.id || joined_topic?(topic_record_id: topic_record.id)
  end

  # Topic Adminはページを更新可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    return false unless active?
    return false unless in_same_space?(space_record_id: page_record.space_id)

    topic_record = page_record.topic_record
    return false if topic_record.nil?

    # 自分がAdminであるトピックのページは全て更新可能
    if topic_member_record&.topic_id == topic_record.id
      return true
    end

    # 参加しているトピックのページも更新可能
    joined_topic?(topic_record_id: topic_record.id)
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
    topic_record = page_record.topic_record
    return true if topic_record&.visibility_public?

    return false unless active?
    return false unless in_same_space?(space_record_id: page_record.space_id)

    true
  end

  # Topic Adminはページを削除可能
  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    return false unless active?
    return false unless in_same_space?(space_record_id: page_record.space_id)

    topic_record = page_record.topic_record
    return false if topic_record.nil?

    # 自分がAdminであるトピックのページは全て削除可能
    if topic_member_record&.topic_id == topic_record.id
      return true
    end

    # 自分が作成したページのみ削除可能（注：現在の実装では全ページ削除可能）
    # TODO: ページ作成者の記録機能が実装されたら、作成者チェックを追加
    true
  end

  # Topic Adminはゴミ箱を閲覧可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Topic Adminは一括復元可能
  sig { override.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    active?
  end

  # Topic Adminはファイルアップロード可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    active? && in_same_space?(space_record_id: space_record.id)
  end

  # Topic Adminはファイルを閲覧可能
  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    # 公開ページで使用されているファイルは誰でも閲覧可能
    return true if attachment_record.all_referencing_pages_public?

    active? && in_same_space?(space_record_id: attachment_record.space_id)
  end

  # Topic Adminは自分がアップロードしたファイルのみ削除可能
  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    return false unless active?
    return false unless in_same_space?(space_record_id: attachment_record.space_id)

    # 自分がアップロードしたファイルのみ削除可能
    attachment_record.attached_space_member_id == space_member_record&.id
  end

  # Topic Adminはファイル管理画面にアクセス不可（Space Owner専用）
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    false
  end

  # Topic Adminはスペースエクスポート不可
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    false
  end

  # 閲覧可能なトピック一覧
  sig { override.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    if space_member_record.nil?
      return T.cast(TopicRecord.none, TopicRecord::PrivateAssociationRelation)
    end

    # Topic Adminは全トピックを閲覧可能
    space_member_record!.space_record.not_nil!.topic_records.kept
  end

  # 閲覧可能なページ一覧
  sig { override.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    if space_member_record.nil?
      return T.cast(PageRecord.none, PageRecord::PrivateAssociationRelation)
    end

    # Topic Adminは全ページを閲覧可能
    space_member_record!.space_record.not_nil!.page_records.active
  end

  sig { returns(T.nilable(TopicMemberRecord)) }
  attr_reader :topic_member_record
  private :topic_member_record
end
