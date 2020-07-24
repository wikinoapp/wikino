# frozen_string_literal: true

module Mutations
  class CreateNote < Mutations::Base
    argument :body, String, required: true

    field :note, Types::Objects::NoteType, null: true
    field :errors, [Types::Objects::MutationErrorType], null: false

    def resolve(body:)
      viewer = context[:viewer]

      note = viewer.notes.new(
        body: body
      )
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
