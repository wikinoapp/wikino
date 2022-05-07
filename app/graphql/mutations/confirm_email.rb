# frozen_string_literal: true

module Mutations
  class ConfirmEmail < Mutations::Base
    argument :email, String, required: true
    argument :event, Types::Enums::EmailConfirmationEvent, required: true
    argument :state, String, required: true

    field :email_confirmation, Types::Objects::EmailConfirmationType, null: true
    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve(email:, event:, state:)
      email_confirmation = EmailConfirmation.new(email:, state:, event: event.downcase)

      if email_confirmation.event_sign_in? && !email_confirmation.user_exists?
        return email_confirmation.graphql.user_not_found_error
      end

      if email_confirmation.invalid?
        return email_confirmation.graphql.invalid_error
      end

      ActiveRecord::Base.transaction do
        email_confirmation.save!
        EmailConfirmationMailer.confirmation(email_confirmation.id, email_confirmation.state).deliver_later!
      end

      email_confirmation.graphql.success
    end
  end
end
