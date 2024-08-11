# typed: strict
# frozen_string_literal: true

class CreateAccountUseCase < ApplicationUseCase
  class Result < T::Struct
    const :user, User
  end

  sig do
    params(email: String, locale: Locale, password: String, time_zone: String).returns(Result)
  end
  def call(email:, locale:, password:, time_zone:)
    current_time = Time.current

    user = ActiveRecord::Base.transaction do
      space = Space.create_initial_space!(current_time:, locale:)

      space.users.create_initial_user!(email:, password:, locale:, time_zone:, current_time:)
    end

    Result.new(user:)
  end
end
