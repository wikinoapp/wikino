# typed: strict
# frozen_string_literal: true

class ConfirmEmailUseCase < ApplicationUseCase
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(viewer: T.nilable(User), email_confirmation: EmailConfirmation).returns(Result) }
  def call(viewer:, email_confirmation:)
    ActiveRecord::Base.transaction do
      email_confirmation.success!
      viewer&.run_after_email_confirmation_success!(email_confirmation:)
    end

    Result.new(email_confirmation:)
  end
end
