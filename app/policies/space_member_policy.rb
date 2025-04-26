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
  def can_export_space?(space_record:)
    return false if space_member_record.nil?

    space_member_record!.space_id == space_record.id &&
      space_member_record!.permissions.include?(SpaceMemberPermission::ExportSpace)
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
