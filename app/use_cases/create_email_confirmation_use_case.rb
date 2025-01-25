# typed: strict
# frozen_string_literal: true

class CreateEmailConfirmationUseCase < ApplicationUseCase
  class Result < T::Struct
    const :email_confirmation, EmailConfirmation
  end

  sig { params(email: String, event: EmailConfirmationEvent, locale: ViewerLocale).returns(Result) }
  def call(email:, event:, locale:)
    current_time = Time.current
    email_confirmation = EmailConfirmation.new(
      email:,
      event: event.serialize,
      code: EmailConfirmation.generate_code,
      started_at: current_time
    )

    ActiveRecord::Base.transaction do
      email_confirmation.save!
      email_confirmation.send_mail!(locale:)
    end

    Result.new(email_confirmation:)
  end
end
