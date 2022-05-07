# frozen_string_literal: true

class EmailConfirmationMailer < ApplicationMailer
  def confirmation(email_confirmation_id, state)
    email_confirmation = EmailConfirmation.find(email_confirmation_id)

    @email = email_confirmation.email
    @url = email_confirmation.url(state:)

    mail(to: @email, subject: email_confirmation.subject) do |format|
      format.html { render email_confirmation.event }
    end
  end
end
