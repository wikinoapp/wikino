# frozen_string_literal: true

module Mutations
  class SignOut < Mutations::Base
    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve
      viewer = context[:viewer]

      viewer.access_token.destroy!

      {
        errors: []
      }
    end
  end
end
