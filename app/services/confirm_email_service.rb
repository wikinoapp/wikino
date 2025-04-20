# typed: strict
# frozen_string_literal: true

class ConfirmEmailService < ApplicationService
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(email_confirmation: EmailConfirmation).returns(Result) }
  def call(email_confirmation:)
    ActiveRecord::Base.transaction do
      email_confirmation.success!

      current_user = T.let(Current.viewer, T.nilable(User))
      current_user&.run_after_email_confirmation_success!(email_confirmation:)
    end

    Result.new(email_confirmation:)
  end
end
