# typed: strict
# frozen_string_literal: true

module EmailConfirmations
  class CreateService < ApplicationService
    class Result < T::Struct
      const :email_confirmation, EmailConfirmationRecord
    end

    sig { params(email: String, event: EmailConfirmationEvent, locale: Locale).returns(Result) }
    def call(email:, event:, locale:)
      current_time = Time.current
      email_confirmation = EmailConfirmationRecord.new(
        email:,
        event: event.serialize,
        code: EmailConfirmationRecord.generate_code,
        started_at: current_time
      )

      ActiveRecord::Base.transaction do
        email_confirmation.save!
        email_confirmation.send_mail!(locale:)
      end

      Result.new(email_confirmation:)
    end
  end
end
