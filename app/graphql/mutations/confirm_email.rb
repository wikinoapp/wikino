# frozen_string_literal: true

module Mutations
  class ConfirmEmail < Mutations::Base
    argument :email, String, required: true
    argument :event, Types::Enums::EmailConfirmationEvent, required: true
    argument :state, String, required: true

    field :email_confirmation, Types::Objects::EmailConfirmationType, null: true
    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve(email:, event:, state:)
      if event == "SIGN_IN"
        user = User.only_kept.find_by(email: email)

        unless user
          return {
            email_confirmation: nil,
            errors: [{
              message: "Account not found. Sign up instead?",
              code: "ACCOUNT_NOT_FOUND"
            }]
          }
        end
      end

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
