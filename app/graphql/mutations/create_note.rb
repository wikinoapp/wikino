# typed: strict
# frozen_string_literal: true

module Mutations
  class CreateNote < Mutations::Base
    extend T::Sig

    argument :title, String, required: true
    argument :body, String, required: false

    field :note, Types::Objects::Note, null: true
    field :errors, [Types::Unions::CreateNoteError], null: false

    sig { params(title: String, body: T.nilable(String)).returns(T::Hash[Symbol, T.untyped]) }
    def resolve(title:, body: nil)
      user = context[:viewer]
      form = NoteCreatingForm.new(user:, title:, body:)

      result = ActiveRecord::Base.transaction do
        CreateNoteService.new(form:).call
      end

      {
        note: result.note,
        errors: result.errors
      }
    end
  end
end
