# frozen_string_literal: true

module Mutations
  class SignIn < Mutations::Base
    argument :token, String, required: true

    field :user, Types::Objects::UserType, null: true
    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve(token:)
      email_confirmation = EmailConfirmation.after(Time.zone.now, field: :expires_at).find_by(token: token)

      unless email_confirmation
        return {
          user: nil,
          errors: [{ message: "Not found" }]
        }
      end

      user = nil
      ActiveRecord::Base.transaction do
        user = User.only_kept.where(email: email_confirmation.email).first_or_create!(signed_up_at: Time.zone.now)
        user.build_access_token.save!
        email_confirmation.destroy!
      end

      {
        user: user,
        errors: []
      }
    end
  end
end
