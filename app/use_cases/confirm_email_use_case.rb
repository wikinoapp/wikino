# typed: strict
# frozen_string_literal: true

class ConfirmEmailUseCase < ApplicationUseCase
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(email_confirmation: EmailConfirmation).returns(Result) }
  def call(email_confirmation:)
    ActiveRecord::Base.transaction do
      email_confirmation.success!
      Current.user&.run_after_email_confirmation_success!(email_confirmation:)
    end

    Result.new(email_confirmation:)
  end
end
