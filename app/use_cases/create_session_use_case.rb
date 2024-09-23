# typed: strict
# frozen_string_literal: true

class CreateSessionUseCase < ApplicationUseCase
  class Result < T::Struct
    const :session, Session
  end

  sig do
    params(
      user: User,
      ip_address: T.nilable(String),
      user_agent: T.nilable(String)
    ).returns(Result)
  end
  def call(user:, ip_address:, user_agent:)
    session = ActiveRecord::Base.transaction do
      user.sessions.start!(space: user.space, ip_address:, user_agent:)
    end

    Result.new(session:)
  end
end
