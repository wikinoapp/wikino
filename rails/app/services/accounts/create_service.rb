# typed: strict
# frozen_string_literal: true

module Accounts
  class CreateService < ApplicationService
    class Result < T::Struct
      const :user, UserRecord
    end

    sig do
      params(
        email: String,
        atname: String,
        locale: Locale,
        password: String,
        time_zone: String
      ).returns(Result)
    end
    def call(email:, atname:, locale:, password:, time_zone:)
      current_time = Time.current

      user = ActiveRecord::Base.transaction do
        UserRecord.create_initial_user!(email:, atname:, password:, locale:, time_zone:, current_time:)
      end

      Result.new(user:)
    end
  end
end
