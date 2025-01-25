# typed: strict
# frozen_string_literal: true

class DestroySessionUseCase < ApplicationUseCase
  sig { params(user_session_token: String).void }
  def call(user_session_token:)
    UserSession.find_by(token: user_session_token)&.destroy!

    nil
  end
end
