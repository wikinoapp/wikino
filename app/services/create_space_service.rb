# typed: strict
# frozen_string_literal: true

class CreateSpaceService < ApplicationService
  class Result < T::Struct
    const :space_record, SpaceRecord
  end

  sig { params(user_record: UserRecord, identifier: String, name: String).returns(Result) }
  def call(user_record:, identifier:, name:)
    current_time = T.let(Time.current, ActiveSupport::TimeWithZone)

    space_record = ActiveRecord::Base.transaction do
      new_space_record = SpaceRecord.where(identifier:).first_or_create!(
        name:,
        plan: Plan::Free.serialize,
        joined_at: current_time
      )
      new_space_record.add_member!(user_record:, role: SpaceMemberRole::Owner, joined_at: current_time)
      new_space_record
    end

    Result.new(space_record:)
  end
end
