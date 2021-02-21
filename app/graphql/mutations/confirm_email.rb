# frozen_string_literal: true

module Mutations
  class ConfirmEmail < Mutations::Base
    argument :email, String, required: true
    argument :event, Types::Enums::EmailConfirmationEvent, required: true
    argument :state, String, required: true

    field :email_confirmation, Types::Objects::EmailConfirmationType, null: true
    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve(email:, event:, state:)
      email_confirmation = EmailConfirmation.new(email: email, event: event.downcase, state: state).save_and_send_email

      if email_confirmation.invalid?
        return {
          email_confirmation: nil,
          errors: email_confirmation.errors.full_messages.map { |message| { message: message } }
        }
      end

      {
        email_confirmation: email_confirmation,
        errors: []
      }
    end
  end
end
