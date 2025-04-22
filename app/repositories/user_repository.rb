# typed: strict
# frozen_string_literal: true

class UserRepository < ApplicationRepository
  sig { params(user_record: UserRecord).returns(User) }
  def to_model(user_record:)
    User.new(
      database_id: user_record.id,
      atname: user_record.atname,
      name: user_record.name,
      description: user_record.description,
      locale: Locale.deserialize(user_record.locale),
      time_zone: user_record.time_zone
    )
  end
end
