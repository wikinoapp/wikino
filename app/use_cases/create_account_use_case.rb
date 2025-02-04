# typed: strict
# frozen_string_literal: true

class CreateAccountUseCase < ApplicationUseCase
  class Result < T::Struct
    const :user, User
  end

  sig do
    params(
      email: String,
      atname: String,
      locale: ViewerLocale,
      password: String,
      time_zone: String
    ).returns(Result)
  end
  def call(email:, atname:, locale:, password:, time_zone:)
    current_time = Time.current

    user = ActiveRecord::Base.transaction do
      User.create_initial_user!(email:, atname:, password:, locale:, time_zone:, current_time:)
    end

    Result.new(user:)
  end
end
