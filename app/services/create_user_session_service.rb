# typed: strict
# frozen_string_literal: true

class CreateUserSessionService < ApplicationService
  class Result < T::Struct
    const :user_session, UserSessionRecord
  end

  sig do
    params(
      user: UserRecord,
      ip_address: T.nilable(String),
      user_agent: T.nilable(String)
    ).returns(Result)
  end
  def call(user:, ip_address:, user_agent:)
    user_session = ActiveRecord::Base.transaction do
      user.user_session_records.start!(ip_address:, user_agent:)
    end

    Result.new(user_session:)
  end
end
