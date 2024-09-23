# typed: strict
# frozen_string_literal: true

class EmailConfirmationForm < ApplicationForm
  attribute :email_confirmation_id, :string
  attribute :confirmation_code, :string

  validate :valid_confirmation_code

  sig { returns(EmailConfirmation) }
  def email_confirmation!
    EmailConfirmation.find(email_confirmation_id.not_nil!)
  end

  sig { returns(T.nilable(EmailConfirmation)) }
  private def active_email_confirmation
    @active_email_confirmation ||= T.let(EmailConfirmation.active.find_by(id: email_confirmation_id), T.nilable(EmailConfirmation))
  end

  sig { void }
  private def valid_confirmation_code
    return if confirmation_code == active_email_confirmation&.code

    errors.add(:confirmation_code, :incorrect_or_expired)
  end
end
