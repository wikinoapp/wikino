# typed: strict
# frozen_string_literal: true

class CreateUserSessionService < ApplicationService
  class Result < T::Struct
    const :user_session_record, UserSessionRecord
  end

  sig do
    params(
      user_record: UserRecord,
      ip_address: T.nilable(String),
      user_agent: T.nilable(String)
    ).returns(Result)
  end
  def call(user_record:, ip_address:, user_agent:)
    user_session_record = ActiveRecord::Base.transaction do
      user_record.user_session_records.start!(ip_address:, user_agent:)
    end

    Result.new(user_session_record:)
  end
end
