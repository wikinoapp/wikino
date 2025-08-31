# typed: strict
# frozen_string_literal: true

# Space Memberロール専用のPolicyクラス
# Space関連の基本操作権限のみを持つ
class SpaceMemberPolicy < ApplicationPolicy
  include SpacePermissions
  include Memoizable
  include QueryOptimizer

  sig { params(user_record: UserRecord, space_member_record: SpaceMemberRecord).void }
  def initialize(user_record:, space_member_record:)
    super(user_record:)
    # 関連をプリロードして最適化
    @space_member_record = T.let(
      preload_space_member_associations(space_member_record).not_nil!,
      SpaceMemberRecord
    )

    if mismatched_relations?
      raise ArgumentError, [
        "Mismatched relations.",
        "user_record.id: #{user_record.id.inspect}",
        "space_member_record.user_id: #{space_member_record.user_id.inspect}"
      ].join(" ")
    end
  end
  # Space権限の実装

  # Memberはスペース設定を変更不可
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    false
  end

  # Memberはトピック作成可能
  sig { override.returns(T::Boolean) }
  def can_create_topic?
    active? && joined_space?
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
    memoize(:can_show_trash, { space_record_id: space_record.id }) do
      active? && in_same_space?(space_record_id: space_record.id)
    end
  end

  # Memberは一括復元可能
  sig { override.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    active?
  end

  # Memberはファイルアップロード可能
  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    memoize(:can_upload_attachment, { space_record_id: space_record.id }) do
      active? && in_same_space?(space_record_id: space_record.id)
    end
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
    space_member_record.space_record.not_nil!.topic_records.kept
  end

  # 閲覧可能なページ（Memberは全ページ閲覧可能）
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

  # 添付ファイル削除権限（Memberは自分がアップロードしたファイルのみ削除可能）
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    memoize(:can_delete_attachment, { attachment_id: attachment_record.id }) do
      active? &&
        in_same_space?(space_record_id: attachment_record.space_id) &&
        space_member_record.id == attachment_record.attached_space_member_id
    end
  end

  # 添付ファイル閲覧権限
  sig { params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    memoize(:can_view_attachment, { attachment_id: attachment_record.id }) do
      active? && (
        in_same_space?(space_record_id: attachment_record.space_id) ||
        attachment_record.all_referencing_pages_public?
      )
    end
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
    # Topic関連をプリロードして最適化
    optimized_topic_record = preload_topic_associations(topic_record)
    topic_member_record = user_record.not_nil!.topic_member_records
      .preload(:topic_record, :space_member_record)
      .find_by(topic_record: optimized_topic_record)

    if topic_member_record
      # TopicMemberのロールに応じて適切なPolicyを返す
      case topic_member_record.role
      when TopicMemberRole::Admin.serialize
        TopicAdminPolicy.new(
          user_record: user_record.not_nil!,
          space_member_record:,
          topic_member_record:
        )
      when TopicMemberRole::Member.serialize
        TopicMemberPolicy.new(
          user_record: user_record.not_nil!,
          space_member_record:,
          topic_member_record:
        )
      else
        # 想定外のロールの場合は例外を投げる
        raise ArgumentError, "Unexpected topic member role: #{topic_member_record.role.inspect}"
      end
    else
      # TopicMemberRecordがない場合はTopicGuestPolicyを返す（公開トピックのみ閲覧可能）
      TopicGuestPolicy.new(user_record: user_record.not_nil!)
    end
  end
end
