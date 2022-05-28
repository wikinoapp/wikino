# typed: true
# frozen_string_literal: true

module Mutations
  class DeleteNote < Mutations::Base
    argument :id, ID, required: true

    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve(id:)
      note = NonotoSchema.object_from_id(id, context)

      note&.destroy!

      {
        errors: []
      }
    rescue ActiveRecord::RecordNotDestroyed => e
      {
        errors: e.record.errors.full_messages.map { |message| { message: message } }
      }
    end
  end
end
