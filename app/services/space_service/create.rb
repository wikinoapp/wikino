# typed: strict
# frozen_string_literal: true

module SpaceService
  class Create < ApplicationService
    class Result < T::Struct
      const :space_record, SpaceRecord
    end

    sig { params(user_record: UserRecord, identifier: String, name: String).returns(Result) }
    def call(user_record:, identifier:, name:)
      current_time = T.let(Time.current, ActiveSupport::TimeWithZone)

      space_record = with_transaction do
        new_space_record = SpaceRecord.create!(
          identifier:,
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
end
