# typed: strict
# frozen_string_literal: true

class SpaceRepository < ApplicationRepository
  sig { params(user: User).returns(T::Array[Space]) }
  def active_spaces(user:)
    user_record = UserRecord.kept.find(user.id)

    user_record.active_spaces.map(:to_model)
  end
end
