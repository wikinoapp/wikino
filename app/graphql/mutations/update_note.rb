# frozen_string_literal: true

module Mutations
  class UpdateNote < Mutations::Base
    argument :id, ID, required: true
    argument :body, String, required: true

    field :note, Types::Objects::NoteType, null: false
    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve(id:, body:)
      note = NonotoSchema.object_from_id(id, context)
      note.body = body
      note.set_title!

      unless note.valid?
        return {
          note: nil,
          errors: note.errors.full_messages.map { |msg| { message: msg } }
        }
      end

      note.save

      {
        note: note,
        errors: []
      }
    end
  end
end
