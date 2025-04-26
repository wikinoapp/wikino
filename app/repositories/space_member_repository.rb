# typed: strict
# frozen_string_literal: true

class SpaceMemberRepository < ApplicationRepository
  sig { params(space_member_record: SpaceMemberRecord).returns(SpaceMember) }
  def to_model(space_member_record:)
    space = SpaceRepository.new.to_model(space_record: space_member_record.space_record!)
    user = UserRepository.new.to_model(user_record: space_member_record.user_record!)

    SpaceMember.new(
      database_id: space_member_record.id,
      space:,
      user:
    )
  end
end
