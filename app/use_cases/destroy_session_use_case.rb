# typed: strict
# frozen_string_literal: true

class DestroySessionUseCase < ApplicationUseCase
  sig { params(session_token: String).void }
  def call(session_token:)
    Session.find_by(token: session_token)&.destroy!

    nil
  end
end
