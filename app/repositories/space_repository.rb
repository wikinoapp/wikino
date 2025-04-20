# typed: strict
# frozen_string_literal: true

class SpaceRepository < ApplicationRepository
  # sig { params(user: User).returns(T::Array[Space]) }
  # def active_spaces(user:)
  #   user_record = UserRecord.kept.find(user.id)

  #   user_record.active_space_records.map { _1.to_model(space_viewer:) }
  # end
end
