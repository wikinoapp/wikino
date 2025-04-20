# typed: strict
# frozen_string_literal: true

class UserSessionRepository < ApplicationRepository
  sig { params(token: T.nilable(String)).returns(T.nilable(UserSession)) }
  def find_by_token(token)
    return unless token

    user_session_record = UserSessionRecord.find_by(token:)
    return unless user_session_record

    build_model(user_session_record:)
  end

  sig { params(form: UserSessionForm).returns(UserSession) }
  def create!(form:)
    user_record = UserRecord.kept.find_by!(email: form.email)

    if user_record.nil?
      user_session.errors.add(:base, :unauthenticated)
      return user_session
    end

    unless user_record.user_password_record&.authenticate(user_session.password)
      user_session.errors.add(:password, :unauthenticated)
      binding.irb
      return user_session
    end

    user_session_record = user_record.user_session_records.start!(
      ip_address: user_session.ip_address,
      user_agent: user_session.user_agent
    )

    build_model(user_session_record:)
  end

  sig { params(user_session_record: UserSessionRecord).returns(UserSession) }
  def build_model(user_session_record:)
    UserSession.new(
      user: UserRepository.new.build_model(user_record: user_session_record.user_record.not_nil!),
      token: user_session_record.token,
      ip_address: user_session_record.ip_address,
      user_agent: user_session_record.user_agent,
      signed_in_at: user_session_record.signed_in_at
    )
  end
end
