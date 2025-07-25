# typed: strict
# frozen_string_literal: true

module EmailConfirmations
  class CheckForm < ApplicationForm
    attribute :email_confirmation_id, :string
    attribute :confirmation_code, :string

    validate :valid_confirmation_code

    sig { returns(EmailConfirmationRecord) }
    def email_confirmation_record!
      EmailConfirmationRecord.find(email_confirmation_id.not_nil!)
    end

    sig { returns(T.nilable(EmailConfirmationRecord)) }
    private def active_email_confirmation
      @active_email_confirmation ||= T.let(EmailConfirmationRecord.active.find_by(id: email_confirmation_id), T.nilable(EmailConfirmationRecord))
    end

    sig { void }
    private def valid_confirmation_code
      return if confirmation_code == active_email_confirmation&.code

      errors.add(:confirmation_code, :incorrect_or_expired)
    end
  end
end