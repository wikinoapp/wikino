# typed: true
# frozen_string_literal: true

class EmailConfirmationMailer < ApplicationMailer
  sig { params(email_confirmation_id: Types::DatabaseId, locale: String).void }
  def email_confirmation(email_confirmation_id:, locale:)
    email_confirmation = EmailConfirmationRecord.active.find(email_confirmation_id)

    @email = email_confirmation.email
    @code = email_confirmation.code

    I18n.with_locale(locale) do
      subject = default_i18n_subject
      mail(to: @email, subject:)
    end
  end
end
