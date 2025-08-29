# typed: strict
# frozen_string_literal: true

# 既存のSpaceMemberPolicyクラス
# 後方互換性のためFactoryパターンを通じて適切なPolicyに処理を委譲
class SpaceMemberPolicy < ApplicationPolicy
  sig do
    params(
      user_record: T.nilable(UserRecord),
      space_member_record: T.nilable(SpaceMemberRecord)
    ).void
  end
  def initialize(user_record: nil, space_member_record: nil)
    @user_record = user_record
    @space_member_record = space_member_record
    # Factoryパターンを使用して適切なPolicyインスタンスを生成
    @delegate_policy = T.let(
      SpaceMemberPolicyFactory.build(user_record:, space_member_record:),
      T.any(OwnerPolicy, MemberPolicy, GuestPolicy)
    )
  end

  sig { override.returns(T::Boolean) }
  def joined_space?
    @delegate_policy.joined_space?
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    @delegate_policy.can_update_space?(space_record:)
  end

  sig { override.returns(T::Boolean) }
  def can_create_topic?
    @delegate_policy.can_create_topic?
  end

  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    @delegate_policy.can_update_topic?(topic_record:)
  end

  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    @delegate_policy.can_update_draft_page?(page_record:)
  end

  sig { override.params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    @delegate_policy.can_create_page?(topic_record:)
  end

  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    @delegate_policy.can_update_page?(page_record:)
  end

  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    @delegate_policy.can_show_page?(page_record:)
  end

  sig { override.params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    @delegate_policy.can_trash_page?(page_record:)
  end

  sig { override.returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    @delegate_policy.can_create_bulk_restore_pages?
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    @delegate_policy.can_show_trash?(space_record:)
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    @delegate_policy.can_export_space?(space_record:)
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_upload_attachment?(space_record:)
    @delegate_policy.can_upload_attachment?(space_record:)
  end

  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_view_attachment?(attachment_record:)
    @delegate_policy.can_view_attachment?(attachment_record:)
  end

  sig { override.params(attachment_record: AttachmentRecord).returns(T::Boolean) }
  def can_delete_attachment?(attachment_record:)
    @delegate_policy.can_delete_attachment?(attachment_record:)
  end

  sig { override.params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_manage_attachments?(space_record:)
    @delegate_policy.can_manage_attachments?(space_record:)
  end

  sig { override.params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    @delegate_policy.showable_topics(space_record:)
  end

  sig { override.returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    @delegate_policy.joined_topic_records
  end

  sig { override.params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    @delegate_policy.showable_pages(space_record:)
  end

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record
  private :user_record

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(T.any(OwnerPolicy, MemberPolicy, GuestPolicy)) }
  attr_reader :delegate_policy
  private :delegate_policy
end
