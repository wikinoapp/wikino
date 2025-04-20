# typed: strict
# frozen_string_literal: true

class SpaceMemberRepository < ApplicationRepository
  def find(user:, space:)
  end

  sig { params(user_record: UserRecord).returns(User) }
  def build_model(user_record:)
    User.new(
      database_id: user_record.id,
      atname: user_record.atname,
      name: user_record.name,
      description: user_record.description,
      serialized_locale: user_record.locale,
      time_zone: user_record.time_zone
    )
  end
end
