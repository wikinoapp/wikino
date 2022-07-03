# typed: strict
# frozen_string_literal: true

module Mutations
  class DeleteNote < Mutations::Base
    extend T::Sig

    argument :id, ID, required: true

    field :errors, [Types::Unions::DeleteNoteError], null: false

    sig { params(id: String).returns(T::Hash[Symbol, T.untyped]) }
    def resolve(id:)
      note = NonotoSchema.object_from_id(id, context)
      form = NoteDestroyingForm.new(user: context[:viewer], note:)

      result = ActiveRecord::Base.transaction do
        DestroyNoteService.new(form:).call
      end

      {
        errors: result.errors
      }
    end
  end
end
