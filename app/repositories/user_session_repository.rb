# typed: strict
# frozen_string_literal: true

class UserSessionRepository < ApplicationRepository
  sig { params(token: T.nilable(String)).returns(T.nilable(UserSession)) }
  def find_by_token(token)
    return unless token

    user_session_record = UserSessionRecord.find_by(token: token)
    return unless user_session_record

    user_session_record.to_model
  end
end
