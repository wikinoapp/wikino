# typed: strict
# frozen_string_literal: true

class UserSessionRepository < ApplicationRepository
  sig { params(user_session_record: UserSessionRecord).returns(UserSession) }
  def to_model(user_session_record:)
    UserSession.new(
      user: UserRepository.new.to_model(user_record: user_session_record.user_record.not_nil!),
      token: user_session_record.token,
      ip_address: user_session_record.ip_address,
      user_agent: user_session_record.user_agent,
      signed_in_at: user_session_record.signed_in_at
    )
  end
end
