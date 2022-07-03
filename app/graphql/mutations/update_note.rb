# typed: strict
# frozen_string_literal: true

module Mutations
  class UpdateNote < Mutations::Base
    extend T::Sig

    argument :id, ID, "The ID of the note to modify.", required: true
    argument :title, String, "The title of the note.", required: true
    argument :body, String, "The body of the note.", required: true

    field :note, Types::Objects::Note, null: true
    field :errors, [Types::Unions::UpdateNoteError], null: false

    sig { params(id: String, title: String, body: String).returns(T::Hash[Symbol, T.untyped]) }
    def resolve(id:, title:, body:)
      note = NonotoSchema.object_from_id(id, context)
      form = NoteUpdatingForm.new(user: context[:viewer], note:, title:, body:)

      result = ActiveRecord::Base.transaction do
        UpdateNoteService.new(form:).call
      end

      {
        note: result.note,
        errors: result.errors
      }
    end
  end
end
