# typed: strict
# frozen_string_literal: true

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

    if mismatched_relations?
      raise ArgumentError, [
        "Mismatched relations.",
        "user_record.id: #{user_record&.id.inspect}",
        "space_member_record.user_id: #{space_member_record&.user_id.inspect}"
      ].join(" ")
    end
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_update_space?(space_record:)
    return false if space_member_record.nil?

    space_member_record!.space_id == space_record.id &&
      space_member_record!.permissions.include?(SpaceMemberPermission::UpdateSpace)
  end

  sig { returns(T::Boolean) }
  def can_create_topic?
    !space_member_record.nil?
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic_record:)
    return false if space_member_record.nil?

    space_member_record!.space_id == topic_record.space_id &&
      space_member_record!.permissions.include?(SpaceMemberPermission::UpdateTopic) &&
      space_member_record!.topic_records.where(id: topic_record.id).exists?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_draft_page?(page_record:)
    return false if space_member_record.nil?

    space_member_record!.active? &&
      space_member_record!.space_id == page_record.space_id &&
      space_member_record!.topic_records.where(id: page_record.topic_id).exists?
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_create_page?(topic_record:)
    return false if space_member_record.nil?

    space_member_record!.topic_records.where(id: topic_record.id).exists?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_update_page?(page_record:)
    return false if space_member_record.nil?

    space_member_record!.active? &&
      space_member_record!.joined_topic_records.where(id: page_record.topic_id).exists?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_show_page?(page_record:)
    if space_member_record.nil?
      return page_record.topic_record!.visibility_public?
    end

    space_member_record!.space_id == page_record.space_id &&
      space_member_record!.active?
  end

  sig { params(page_record: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page_record:)
    return false if space_member_record.nil?

    space_member_record!.active? &&
      space_member_record!.space_id == page_record.space_id
  end

  sig { returns(T::Boolean) }
  def can_create_bulk_restore_pages?
    return false if space_member_record.nil?

    space_member_record!.active?
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_show_trash?(space_record:)
    return false if space_member_record.nil?

    space_member_record!.active? &&
      space_member_record!.space_id == space_record.id
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def can_export_space?(space_record:)
    return false if space_member_record.nil?

    space_member_record!.space_id == space_record.id &&
      space_member_record!.permissions.include?(SpaceMemberPermission::ExportSpace)
  end

  sig { params(space_record: SpaceRecord).returns(TopicRecord::PrivateAssociationRelation) }
  def showable_topics(space_record:)
    if space_member_record
      return space_member_record!.space_record.not_nil!.topic_records.kept
    end

    space_record.topic_records.kept.visibility_public
  end

  sig { returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    space_member_record&.joined_topic_records.presence || TopicRecord.none
  end

  sig { params(space_record: SpaceRecord).returns(PageRecord::PrivateAssociationRelation) }
  def showable_pages(space_record:)
    if space_member_record
      return space_member_record!.space_record.not_nil!.page_records.active
    end

    space_record.page_records.active.topics_visibility_public
  end

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record
  private :user_record

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(SpaceMemberRecord) }
  private def space_member_record!
    space_member_record.not_nil!
  end

  sig { returns(T::Boolean) }
  private def mismatched_relations?
    if !user_record.nil? && !space_member_record.nil?
      user_record.not_nil!.id != space_member_record.not_nil!.user_id
    else
      false
    end
  end
end
