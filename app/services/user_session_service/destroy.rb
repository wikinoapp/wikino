# typed: strict
# frozen_string_literal: true

module UserSessionService
  class Destroy < ApplicationService
    sig { params(user_session_token: String).void }
    def call(user_session_token:)
      UserSessionRecord.find_by(token: user_session_token)&.destroy!

      nil
    end
  end
end
