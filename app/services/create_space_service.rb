# typed: strict
# frozen_string_literal: true

class CreateSpaceService < ApplicationService
  class Result < T::Struct
    const :space, SpaceRecord
  end

  sig { params(user: UserRecord, identifier: String, name: String).returns(Result) }
  def call(user:, identifier:, name:)
    current_time = T.let(Time.current, ActiveSupport::TimeWithZone)

    space = ActiveRecord::Base.transaction do
      new_space = SpaceRecord.where(identifier:).first_or_create!(name:, plan: Plan::Free.serialize, joined_at: current_time)
      new_space.add_member!(user:, role: SpaceMemberRole::Owner, joined_at: current_time)
      new_space
    end

    Result.new(space:)
  end
end
