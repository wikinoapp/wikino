# typed: strict
# frozen_string_literal: true

class CreateSpaceService < ApplicationService
  class Result < T::Struct
    const :space, Space
  end

  sig { params(user: User, identifier: String, name: String).returns(Result) }
  def call(user:, identifier:, name:)
    current_time = T.let(Time.current, ActiveSupport::TimeWithZone)

    space = ActiveRecord::Base.transaction do
      new_space = Space.where(identifier:).first_or_create!(name:, plan: Plan::Free.serialize, joined_at: current_time)
      new_space.add_member!(user:, role: SpaceMemberRole::Owner, joined_at: current_time)
      new_space
    end

    Result.new(space:)
  end
end
